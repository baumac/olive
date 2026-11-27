package publicapi

import (
	"maps"
	"slices"
	"testing"

	"github.com/baumac/olive/app/api/internal/history"
)

// TestHistoryHasTheCreateProperties fails when a create schema gets a
// property that the history does not keep.
func TestHistoryHasTheCreateProperties(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		schema string
		kind   history.Kind
	}{{"RestaurantCreate", history.Restaurants}, {"RecipeCreate", history.Recipes}} {
		schema, err := createSchema(tc.schema)()
		if err != nil {
			t.Fatal(err)
		}
		want := slices.Sorted(maps.Keys(schema.Properties))
		var got []string
		for _, p := range tc.kind.Properties {
			got = append(got, p.Name)
		}
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("history of %s has %v, want %v", tc.schema, got, want)
		}
	}
}
