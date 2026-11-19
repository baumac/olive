package publicapi_test

import (
	"net/http"
	"net/url"
	"testing"
)

func TestRestaurantGallery(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	bob := e.signUp("bob@example.com", "Bob")
	created := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)
	if created["cover"] != nil {
		t.Errorf("new restaurant cover %v", created["cover"])
	}
	restaurant := created["id"].(string)
	empty := e.createRestaurant(ana.Token, `{"name":"Empty","location":{"latitude":1,"longitude":1,"address":"Qwx 2"}}`)["id"].(string)
	base := "/v1/restaurants/" + restaurant

	addPhoto := func(token, review string) string {
		t.Helper()
		return e.expect(request{method: "POST", path: base + "/reviews/" + review + "/photos",
			body: string(testJPEG(t, 40, 20, 0)), contentType: "image/jpeg", token: token}, http.StatusCreated).object(t)["id"].(string)
	}
	// Ana and Bob have the overall 7.6. Joe has 9. Bob is newer than Ana.
	anaReview := e.createReview(ana.Token, restaurant, reviewBody)["id"].(string)
	a1, a2 := addPhoto(ana.Token, anaReview), addPhoto(ana.Token, anaReview)
	joeReview := e.createReview(joe.Token, restaurant, `{"space":9,"service":9,"food":9,"price":9,"cleanliness":9,"title":"Best"}`)["id"].(string)
	j1 := addPhoto(joe.Token, joeReview)
	bobReview := e.createReview(bob.Token, restaurant, reviewBody)["id"].(string)
	b1 := addPhoto(bob.Token, bobReview)
	e.createReview(ana.Token, empty, reviewBody)

	gallery := func(query string) ([]map[string]any, any) {
		t.Helper()
		var page struct {
			Items      []map[string]any `json:"items"`
			NextCursor any              `json:"next_cursor"`
		}
		e.expect(request{method: "GET", path: base + "/photos" + query}, http.StatusOK).decode(t, &page)
		return page.Items, page.NextCursor
	}
	ids := func(items []map[string]any) []string {
		out := make([]string, len(items))
		for i, item := range items {
			out[i] = item["id"].(string)
		}
		return out
	}
	equal := func(name string, got, want []string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s %v, want %v", name, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("%s %v, want %v", name, got, want)
			}
		}
	}

	items, next := gallery("")
	equal("gallery", ids(items), []string{j1, b1, a1, a2})
	if next != nil {
		t.Errorf("next_cursor %v", next)
	}
	first := items[0]
	if first["review"] != joeReview || first["creator"].(map[string]any)["name"] != "Joe" {
		t.Errorf("first photo %v", first)
	}
	if urlOf(first, "thumb") != base+"/reviews/"+joeReview+"/photos/"+j1+"/thumb" {
		t.Errorf("urls %v", first["urls"])
	}

	page1, next := gallery("?limit=3")
	equal("page 1", ids(page1), []string{j1, b1, a1})
	cursor, ok := next.(string)
	if !ok {
		t.Fatalf("next_cursor %v", next)
	}
	page2, next := gallery("?limit=3&cursor=" + url.QueryEscape(cursor))
	equal("page 2", ids(page2), []string{a2})
	if next != nil {
		t.Errorf("last next_cursor %v", next)
	}

	cover := func() any {
		t.Helper()
		return e.expect(request{method: "GET", path: base}, http.StatusOK).object(t)["cover"]
	}
	if c, ok := cover().(map[string]any); !ok || c["id"] != j1 || c["review"] != joeReview {
		t.Errorf("cover %v", c)
	}
	var list struct {
		Items []map[string]any `json:"items"`
	}
	e.expect(request{method: "GET", path: "/v1/restaurants"}, http.StatusOK).decode(t, &list)
	for _, r := range list.Items {
		c, _ := r["cover"].(map[string]any)
		switch r["id"] {
		case restaurant:
			if c == nil || c["id"] != j1 {
				t.Errorf("list cover of %s: %v", restaurant, r["cover"])
			}
		case empty:
			if r["cover"] != nil {
				t.Errorf("list cover of a restaurant without photos: %v", r["cover"])
			}
		}
	}

	e.expect(request{method: "PATCH", path: base + "/reviews/" + joeReview, body: `{"food":1,"space":1}`, token: joe.Token}, http.StatusOK)
	if c, ok := cover().(map[string]any); !ok || c["id"] != b1 {
		t.Errorf("cover after a lower score %v", c)
	}
	e.expect(request{method: "DELETE", path: base + "/reviews/" + bobReview, token: bob.Token}, http.StatusNoContent)
	items, _ = gallery("")
	equal("gallery after a delete", ids(items), []string{a1, a2, j1})
	updated := e.expect(request{method: "PATCH", path: base, body: `{"description":"New"}`, token: ana.Token}, http.StatusOK).object(t)
	if c, ok := updated["cover"].(map[string]any); !ok || c["id"] != a1 {
		t.Errorf("update response cover %v", updated["cover"])
	}

	e.expect(request{method: "GET", path: "/v1/restaurants/" + empty + "/photos"}, http.StatusOK)
	e.expect(request{method: "GET", path: "/v1/restaurants/aaaaaaaaaaaaaaa/photos"}, http.StatusNotFound)
	e.expect(request{method: "GET", path: "/v1/restaurants/" + empty + "/photos?cursor=" + url.QueryEscape(cursor)}, http.StatusBadRequest)
}
