package publicapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"sync"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"golang.org/x/text/unicode/norm"
)

// restaurantSchema is the RestaurantCreate schema. Its properties have no
// null option. Each error then names the failed rule.
var restaurantSchema = sync.OnceValues(func() (*openapi3.Schema, error) {
	doc, err := embeddedSpec()
	if err != nil {
		return nil, err
	}
	ref := doc.Components.Schemas["RestaurantCreate"]
	if ref == nil || ref.Value == nil {
		return nil, errors.New("the spec has no RestaurantCreate schema")
	}
	return ref.Value, nil
})

// guardRestaurantFields gives a save through the PocketBase API the checks
// of the Public API body. Only superusers can use this API for restaurants.
// A value from the dashboard then cannot make the responses fail.
func guardRestaurantFields(app core.App) {
	app.OnRecordCreateRequest("restaurants").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := checkRestaurantFields(e.Record, nil); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("restaurants").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := checkRestaurantFields(e.Record, e.Record.Original()); err != nil {
			return err
		}
		return e.Next()
	})
}

// checkRestaurantFields changes the record like the Public API changes a
// body. It applies x-nfc before the checks and x-trim after them. With an
// original record, it checks only the changed properties.
func checkRestaurantFields(r, original *core.Record) error {
	schema, err := restaurantSchema()
	if err != nil {
		return err
	}
	props, err := restaurantProperties(r, schema)
	if err != nil {
		return err
	}
	stored := make(map[string]json.RawMessage, len(props))
	for name, value := range props {
		stored[name] = slices.Clone(value)
	}
	var old map[string]json.RawMessage
	if original != nil {
		if old, err = restaurantProperties(original, schema); err != nil {
			return err
		}
	}

	changeProperties(schema, props, "x-nfc", norm.NFC.String)
	errs := restaurantFieldErrors(schema, props, func(name string) bool {
		return original == nil || !bytes.Equal(stored[name], old[name])
	})
	if _, ok := props["location"]; !ok && original == nil {
		errs["address"] = validation.NewError("validation_public_api", "is required")
	}
	if len(errs) > 0 {
		return apis.NewBadRequestError("The restaurant has values that the Public API refuses.", errs)
	}
	changeProperties(schema, props, "x-trim", trimSpaces)

	for name, value := range props {
		if bytes.Equal(value, stored[name]) {
			continue
		}
		switch name {
		case "tags":
			r.Set("tags", types.JSONRaw(value))
		case "location":
			var l reviewsapi.Location
			if err := json.Unmarshal(value, &l); err != nil {
				return err
			}
			setLocation(r, l)
		default:
			var text string
			if err := json.Unmarshal(value, &text); err != nil {
				return err
			}
			r.Set(name, text)
		}
	}
	return nil
}

// restaurantProperties gets the properties that a response would have. The
// tags are the stored JSON. A stored string or object then gets an error.
// A location without an address gets an error of the address.
func restaurantProperties(r *core.Record, schema *openapi3.Schema) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(toRestaurant(r, nil, nil, nil, nil))
	if err != nil {
		return nil, err
	}
	all, err := decodeObject(raw)
	if err != nil {
		return nil, err
	}
	if tags, ok := r.Get("tags").(types.JSONRaw); ok {
		all["tags"] = json.RawMessage(tags)
	}
	if partial := partialLocation(r); partial != nil {
		if all["location"], err = json.Marshal(partial); err != nil {
			return nil, err
		}
	}
	props := map[string]json.RawMessage{}
	for name, value := range all {
		if _, ok := schema.Properties[name]; ok && len(value) > 0 && string(value) != "null" {
			props[name] = value
		}
	}
	return props, nil
}

// partialLocation returns the location fields of a record without an
// address. It returns nil when these fields are empty too.
func partialLocation(r *core.Record) *reviewsapi.Location {
	if r.GetString("address") != "" {
		return nil
	}
	l := reviewsapi.Location{Latitude: r.GetFloat("latitude"), Longitude: r.GetFloat("longitude")}
	if id := r.GetString("place_id"); id != "" {
		l.PlaceId = &id
	}
	if l.Latitude == 0 && l.Longitude == 0 && l.PlaceId == nil {
		return nil
	}
	return &l
}

// restaurantFieldErrors keys each error with the record field. An error of
// the location names its address, latitude, longitude, or place_id field.
func restaurantFieldErrors(schema *openapi3.Schema, props map[string]json.RawMessage, check func(string) bool) validation.Errors {
	opts := append([]openapi3.SchemaValidationOption{openapi3.MultiErrors()}, validationOptions.SchemaValidationOptions...)
	errs := validation.Errors{}
	for name, raw := range props {
		if !check(name) {
			continue
		}
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			errs[name] = validation.NewError("validation_public_api", "must be valid JSON")
			continue
		}
		var schemaErrs []*openapi3.SchemaError
		collectSchemaErrors(schema.Properties[name].Value.VisitJSON(value, opts...), &schemaErrs)
		for _, e := range schemaErrs {
			field := name
			if ptr := e.JSONPointer(); name == "location" && len(ptr) > 0 {
				field = ptr[0]
			}
			if _, ok := errs[field]; !ok {
				errs[field] = validation.NewError("validation_public_api", schemaMessage(e))
			}
		}
	}
	return errs
}
