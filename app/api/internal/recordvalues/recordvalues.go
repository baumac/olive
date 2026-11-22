// Package recordvalues gets the API values of the fields of a restaurant or
// a recipe. The Public API and the history use it. See decision 00004.
package recordvalues

import "github.com/pocketbase/pocketbase/core"

// OptionalText returns nil for the empty string that PocketBase stores for a
// text without a value.
func OptionalText(r *core.Record, field string) *string {
	if v := r.GetString(field); v != "" {
		return &v
	}
	return nil
}

// OptionalNumber returns nil for the 0 that PocketBase stores for a number
// without a value.
func OptionalNumber(r *core.Record, field string) *int {
	if v := r.GetInt(field); v != 0 {
		return &v
	}
	return nil
}

// OptionalList returns nil for a field that is not a JSON array of strings.
func OptionalList(r *core.Record, field string) *[]string {
	var items []string
	if err := r.UnmarshalJSONField(field, &items); err != nil || items == nil {
		return nil
	}
	return &items
}

// List returns an empty list for a field that is not a JSON array of
// strings.
func List(r *core.Record, field string) []string {
	if items := OptionalList(r, field); items != nil {
		return *items
	}
	return []string{}
}

// Location has the JSON shape of the Location schema.
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
	PlaceID   string  `json:"place_id,omitempty"`
}

// LocationOf returns nil for a restaurant without an address. Such a
// restaurant is from before the location field.
func LocationOf(r *core.Record) *Location {
	address := r.GetString("address")
	if address == "" {
		return nil
	}
	return &Location{Latitude: r.GetFloat("latitude"), Longitude: r.GetFloat("longitude"), Address: address, PlaceID: r.GetString("place_id")}
}
