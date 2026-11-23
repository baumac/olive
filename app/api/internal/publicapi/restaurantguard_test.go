package publicapi_test

import (
	"net/http"
	"regexp"
	"testing"

	reviewsapi "github.com/baumac/reviews-app/specs/api/sdk/server/go"
	"github.com/pocketbase/pocketbase/core"
)

func (e *env) superuserToken(email string) string {
	e.t.Helper()
	token, err := e.addSuperuser(email).NewAuthToken()
	if err != nil {
		e.t.Fatal(err)
	}
	return token
}

// dashboard sends a request to the PocketBase API. Its errors are not
// Problem Details.
func (e *env) dashboard(req request, status int) response {
	e.t.Helper()
	req.contentType = "application/json"
	res := e.do(req)
	if res.status != status {
		e.t.Fatalf("%s %s: status %d, want %d: %s", req.method, req.path, res.status, status, res.body)
	}
	return res
}

func TestDashboardRestaurantEdit(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	bo := e.signUp("bo@example.com", "Bo")
	root := e.superuserToken("root@example.com")
	id := e.createRestaurant(ana.Token, `{"name":"Cafe","location":{"latitude":41.4,"longitude":2.2,"address":"Carrer 1"}}`)["id"].(string)
	e.createReview(bo.Token, id, `{"date":"2024-11-06","space":7,"service":9,"food":8,"price":6,"title":"T"}`)
	records := "/api/collections/restaurants/records"

	t.Run("a superuser changes the fields of a restaurant of another user", func(t *testing.T) {
		e.dashboard(request{method: http.MethodPatch, path: records + "/" + id, token: root,
			body: `{"name":"  Café Nou ","tags":["cafe\u0301","brunch"],"address":" Carrer 2 ","website":"https://cafe.example"}`}, http.StatusOK)
		got := e.expect(request{method: http.MethodGet, path: "/v1/restaurants/" + id, token: ana.Token}, http.StatusOK).object(t)
		if got["name"] != "Café Nou" {
			t.Errorf("name %q, want the trimmed name", got["name"])
		}
		if tags := got["tags"].([]any); len(tags) != 2 || tags[0] != "café" {
			t.Errorf("tags %v, want the NFC tag café", tags)
		}
		if l := got["location"].(map[string]any); l["address"] != "Carrer 2" {
			t.Errorf("address %q, want the trimmed address", l["address"])
		}
		if got["website"] != "https://cafe.example" {
			t.Errorf("website %q, want the new website", got["website"])
		}
	})

	for name, tc := range map[string]struct{ body, field string }{
		"an uppercase tag":     {`{"tags":["Pasta"]}`, "tags"},
		"repeated tags":        {`{"tags":["pasta","pasta"]}`, "tags"},
		"tags as a string":     {`{"tags":"pasta"}`, "tags"},
		"a tag with a hyphen":  {`{"tags":["-pasta"]}`, "tags"},
		"a blank name":         {`{"name":"   "}`, "name"},
		"a zero-width space":   {`{"name":"Cafe\u200b"}`, "name"},
		"a blank address":      {`{"address":"   "}`, "address"},
		"a far latitude":       {`{"latitude":91}`, "latitude"},
		"a bad place ID":       {`{"place_id":"no colon"}`, "place_id"},
		"a hidden description": {`{"description":"a\u200eb"}`, "description"},
		"an empty address":     {`{"address":""}`, "address"},
	} {
		t.Run("the dashboard refuses "+name, func(t *testing.T) {
			res := e.dashboard(request{method: http.MethodPatch, path: records + "/" + id, token: root, body: tc.body}, http.StatusBadRequest)
			data, _ := res.object(t)["data"].(map[string]any)
			if _, ok := data[tc.field]; !ok {
				t.Errorf("errors %s, want an error of %s", res.body, tc.field)
			}
			e.expect(request{method: http.MethodGet, path: "/v1/restaurants/" + id, token: ana.Token}, http.StatusOK)
		})
	}

	t.Run("the dashboard creates a restaurant with the checks", func(t *testing.T) {
		location := `"latitude":41.5,"longitude":2.3,"address":"Carrer 3"`
		e.dashboard(request{method: http.MethodPost, path: records, token: root, body: `{"name":"New","tags":["X"],` + location + `}`}, http.StatusBadRequest)
		res := e.dashboard(request{method: http.MethodPost, path: records, token: root, body: `{"name":"New"}`}, http.StatusBadRequest)
		if data, _ := res.object(t)["data"].(map[string]any); data["address"] == nil {
			t.Errorf("errors %s, want an error of address", res.body)
		}
		got := e.dashboard(request{method: http.MethodPost, path: records, token: root, body: `{"name":" New ","tags":["new"],` + location + `}`}, http.StatusOK).object(t)
		if got["name"] != "New" {
			t.Errorf("name %q, want the trimmed name", got["name"])
		}
	})

	t.Run("the dashboard changes a field next to an old invalid value", func(t *testing.T) {
		r, err := e.app.FindRecordById("restaurants", id)
		if err != nil {
			t.Fatal(err)
		}
		r.Set("tags", []string{"Old"})
		if err := e.app.Save(r); err != nil {
			t.Fatalf("a save outside the PocketBase API ran the checks: %v", err)
		}
		e.dashboard(request{method: http.MethodPatch, path: records + "/" + id, token: root, body: `{"price_level":2}`}, http.StatusOK)
		e.dashboard(request{method: http.MethodPatch, path: records + "/" + id, token: root, body: `{"tags":["Old","new"]}`}, http.StatusBadRequest)
	})
}

// TestRestaurantFieldPatterns fails when a field pattern of the migrations
// and its spec pattern differ. Saves outside the PocketBase API get only the
// field patterns.
func TestRestaurantFieldPatterns(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	doc, err := reviewsapi.GetSwagger()
	if err != nil {
		t.Fatal(err)
	}
	restaurants, err := e.app.FindCollectionByNameOrId("restaurants")
	if err != nil {
		t.Fatal(err)
	}
	escape := regexp.MustCompile(`\\u([0-9A-F]{4})`)
	for field, schema := range map[string]string{"website": "Website", "phone": "Phone", "opening_hours": "OpeningHours"} {
		f, ok := restaurants.Fields.GetByName(field).(*core.TextField)
		if !ok {
			t.Fatalf("restaurants has no text field %s", field)
		}
		want := escape.ReplaceAllString(doc.Components.Schemas[schema].Value.Pattern, `\x{$1}`)
		if f.Pattern != want {
			t.Errorf("pattern of %s is %s, want the pattern of %s: %s", field, f.Pattern, schema, want)
		}
	}
}
