// Package recordvalues gets the API values of the fields of a restaurant or
// a recipe. The Public API and the history use it. See decision 00004.
package recordvalues

import (
	"maps"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

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
// An empty array is also nil. The API has one value for a list without items.
func OptionalList(r *core.Record, field string) *[]string {
	var items []string
	if err := r.UnmarshalJSONField(field, &items); err != nil || len(items) == 0 {
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

// LocationOf gets the location of a restaurant.
func LocationOf(r *core.Record) Location {
	return Location{Latitude: r.GetFloat("latitude"), Longitude: r.GetFloat("longitude"), Address: r.GetString("address"), PlaceID: r.GetString("place_id")}
}

// ExpandTags loads the tag records of the tags relation of records. A record
// that lost its tags then also loses the old expand.
func ExpandTags(app core.App, records ...*core.Record) error {
	if len(records) == 0 {
		return nil
	}
	for _, r := range records {
		e := maps.Clone(r.Expand())
		delete(e, "tags")
		r.SetExpand(e)
	}
	for _, err := range app.ExpandRecords(records, []string{"tags"}, nil) {
		return err
	}
	return nil
}

// Tags returns the names of the tags that ExpandTags loaded, in the order of
// the relation. It returns nil for a record without tags.
func Tags(r *core.Record) *[]string {
	tags := r.ExpandedAll("tags")
	if len(tags) == 0 {
		return nil
	}
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.GetString("name")
	}
	return &names
}

// ParentsWithTag gets the records of the collection table that have the tag
// with the ID tagID. The tag table tags has a row for each of them.
func ParentsWithTag(app core.App, table, parent, tags, tagID string) ([]*core.Record, error) {
	return app.FindAllRecords(table, dbx.NewExp(
		"[[id]] IN (SELECT [["+parent+"]] FROM {{"+tags+"}} WHERE [[tag]] = {:tag})", dbx.Params{"tag": tagID},
	))
}
