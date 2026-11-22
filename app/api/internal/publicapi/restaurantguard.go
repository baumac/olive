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

// createSchema gets a create schema of the spec. Its properties have no null
// option. Each error then names the failed rule.
func createSchema(name string) func() (*openapi3.Schema, error) {
	return sync.OnceValues(func() (*openapi3.Schema, error) {
		doc, err := embeddedSpec()
		if err != nil {
			return nil, err
		}
		ref := doc.Components.Schemas[name]
		if ref == nil || ref.Value == nil {
			return nil, errors.New("the spec has no " + name + " schema")
		}
		return ref.Value, nil
	})
}

// fieldGuard gives a save through the PocketBase API the checks of the
// Public API body. Only superusers can use this API for the collection. A
// value from the dashboard then cannot make the responses fail.
type fieldGuard struct {
	collection string
	noun       string
	schema     func() (*openapi3.Schema, error)
	// properties gets the properties that a response would have.
	properties func(r *core.Record, schema *openapi3.Schema) (map[string]json.RawMessage, error)
	// required adds the errors of a create without a required property.
	// The schema check sees only the properties that the record has.
	required func(props map[string]json.RawMessage, errs validation.Errors)
	// set stores a property that x-nfc or x-trim changed.
	set func(r *core.Record, name string, value json.RawMessage) error
}

func (g fieldGuard) register(app core.App) {
	app.OnRecordCreateRequest(g.collection).BindFunc(func(e *core.RecordRequestEvent) error {
		if err := g.check(e.Record, nil); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest(g.collection).BindFunc(func(e *core.RecordRequestEvent) error {
		if err := g.check(e.Record, e.Record.Original()); err != nil {
			return err
		}
		return e.Next()
	})
}

// check changes the record like the Public API changes a body. It applies
// x-nfc before the checks and x-trim after them. With an original record, it
// checks only the changed properties.
func (g fieldGuard) check(r, original *core.Record) error {
	schema, err := g.schema()
	if err != nil {
		return err
	}
	props, err := g.properties(r, schema)
	if err != nil {
		return err
	}
	stored := make(map[string]json.RawMessage, len(props))
	for name, value := range props {
		stored[name] = slices.Clone(value)
	}
	var old map[string]json.RawMessage
	if original != nil {
		if old, err = g.properties(original, schema); err != nil {
			return err
		}
	}

	changeProperties(schema, props, "x-nfc", norm.NFC.String)
	errs := schemaFieldErrors(schema, props, func(name string) bool {
		return original == nil || !bytes.Equal(stored[name], old[name])
	})
	if original == nil && g.required != nil {
		g.required(props, errs)
	}
	if len(errs) > 0 {
		return apis.NewBadRequestError("The "+g.noun+" has values that the Public API refuses.", errs)
	}
	changeProperties(schema, props, "x-trim", trimWhitespace)

	for name, value := range props {
		if bytes.Equal(value, stored[name]) {
			continue
		}
		if err := g.set(r, name, value); err != nil {
			return err
		}
	}
	return nil
}

// setText stores a property that is a JSON string.
func setText(r *core.Record, name string, value json.RawMessage) error {
	var text string
	if err := json.Unmarshal(value, &text); err != nil {
		return err
	}
	r.Set(name, text)
	return nil
}

var restaurantGuard = fieldGuard{
	collection: "restaurants",
	noun:       "restaurant",
	schema:     createSchema("RestaurantCreate"),
	properties: restaurantProperties,
	set: func(r *core.Record, name string, value json.RawMessage) error {
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
			return setText(r, name, value)
		}
		return nil
	},
}

// restaurantProperties gets the properties that a response would have. The
// tags are the stored JSON. A stored string or object then gets an error.
// A location without an address gets an error of the address.
func restaurantProperties(r *core.Record, schema *openapi3.Schema) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(toRestaurant(r, nil, nil, nil, viewerData{}))
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
	props := map[string]json.RawMessage{}
	for name, value := range all {
		if _, ok := schema.Properties[name]; ok && len(value) > 0 && string(value) != "null" {
			props[name] = value
		}
	}
	return props, nil
}

// schemaFieldErrors keys each error with the record field. An error of the
// location names its address, latitude, longitude, or place_id field.
func schemaFieldErrors(schema *openapi3.Schema, props map[string]json.RawMessage, check func(string) bool) validation.Errors {
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
