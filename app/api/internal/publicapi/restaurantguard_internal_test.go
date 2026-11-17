package publicapi

import (
	"errors"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestRestaurantGuardRefusesBrokenStoredValues(t *testing.T) {
	s := newTestServer(t)
	collection, err := s.app.FindCollectionByNameOrId("restaurants")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(collection)
	r.Set("name", "Cafe")
	r.Set("place_id", "osm:node:1")
	r.Set("latitude", 41.4)
	r.Set("longitude", 2.2)
	r.Set("tags", types.JSONRaw("[1,"))

	var apiErr *router.ApiError
	if err := checkRestaurantFields(r, nil); !errors.As(err, &apiErr) {
		t.Fatalf("error %v, want an API error", err)
	}
	for _, field := range []string{"address", "tags"} {
		if _, ok := apiErr.Data[field]; !ok {
			t.Errorf("no error for %s: %v", field, apiErr.Data)
		}
	}
}
