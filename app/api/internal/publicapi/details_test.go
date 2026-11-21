package publicapi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRestaurantDetails(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	const details = `"website":"https://cansole.example/menu?lang=en","phone":"+34 932 21 50 12","opening_hours":"Tu-Sa 13:00-16:00\nSu 13:00-16:00","price_level":3`

	r := e.createRestaurant(ana.Token, `{"name":"Can Sole",`+details+`}`)
	path := "/v1/restaurants/" + r["id"].(string)
	want := map[string]any{
		"website":       "https://cansole.example/menu?lang=en",
		"phone":         "+34 932 21 50 12",
		"opening_hours": "Tu-Sa 13:00-16:00\nSu 13:00-16:00",
		"price_level":   float64(3),
	}

	t.Run("create and get", func(t *testing.T) {
		for _, got := range []map[string]any{r, e.expect(request{method: "GET", path: path}, http.StatusOK).object(t)} {
			for k, v := range want {
				if got[k] != v {
					t.Errorf("%s = %v, want %v", k, got[k], v)
				}
			}
		}
		plain := e.createRestaurant(ana.Token, `{"name":"Plain"}`)
		for k := range want {
			if _, ok := plain[k]; ok {
				t.Errorf("a restaurant without details has %s", k)
			}
		}
	})

	t.Run("errors", func(t *testing.T) {
		for body, want := range map[string]string{
			`{"name":"A","website":"cansole.example"}`:                        "website",
			`{"name":"A","website":"javascript:alert(1)"}`:                    "website",
			`{"name":"A","website":"https:///path"}`:                          "website",
			`{"name":"A","website":"https://can sole.example"}`:               "website",
			`{"name":"A","website":"https://cansole.example/​"}`:              "website",
			`{"name":"A","phone":"12"}`:                                       "phone",
			`{"name":"A","phone":"call 932 21 50 12"}`:                        "phone",
			`{"name":"A","phone":"` + strings.Repeat("1", 21) + `"}`:          "phone",
			`{"name":"A","opening_hours":"` + strings.Repeat("a", 501) + `"}`: "opening_hours",
			`{"name":"A","opening_hours":"Mo\u0007"}`:                         "opening_hours",
			`{"name":"A","price_level":0}`:                                    "price_level",
			`{"name":"A","price_level":5}`:                                    "price_level",
			`{"name":"A","price_level":2.5}`:                                  "price_level",
		} {
			res := e.expect(request{method: "POST", path: "/v1/restaurants", body: strings.TrimSuffix(body, "}") + `,"location":` + testLocation + "}", token: ana.Token}, http.StatusUnprocessableEntity)
			if got := strings.Join(errorProperties(t, res), ","); got != want {
				t.Errorf("%s: properties %q, want %q", body, got, want)
			}
		}
		e.expect(request{method: "PATCH", path: path, body: `{"price_level":"2"}`, token: ana.Token}, http.StatusUnprocessableEntity)
	})

	t.Run("patch", func(t *testing.T) {
		same := e.expect(request{method: "PATCH", path: path, body: `{` + details + `}`, token: ana.Token}, http.StatusOK).object(t)
		if same["updated"] != r["updated"] {
			t.Error("equal details changed updated")
		}
		byJoe := e.expect(request{method: "PATCH", path: path, body: `{"phone":"932215013"}`, token: joe.Token}, http.StatusOK).object(t)
		if byJoe["phone"] != "932215013" {
			t.Errorf("phone of another user %v", byJoe["phone"])
		}

		changed := e.expect(request{method: "PATCH", path: path, body: `{"phone":"932215012","price_level":1}`, token: ana.Token}, http.StatusOK).object(t)
		if changed["phone"] != "932215012" || changed["price_level"] != float64(1) || changed["website"] != want["website"] {
			t.Errorf("changed %v", changed)
		}
		cleared := e.expect(request{method: "PATCH", path: path,
			body: `{"website":null,"phone":null,"opening_hours":null,"price_level":null}`, token: ana.Token}, http.StatusOK).object(t)
		for k := range want {
			if _, ok := cleared[k]; ok {
				t.Errorf("null keeps %s", k)
			}
		}
	})

	t.Run("reviews of other users do not lock the details", func(t *testing.T) {
		e.createReview(joe.Token, r["id"].(string), `{"date":"2024-11-06","space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"Great"}`)
		e.expect(request{method: "PATCH", path: path, body: `{"price_level":2}`, token: ana.Token}, http.StatusOK)
	})
}

func TestSearchPlacesDetails(t *testing.T) {
	t.Parallel()
	photon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"features":[
 {"geometry":{"coordinates":[2.1893,41.3766]},"properties":{"name":"Can Solé","street":"Carrer de Sant Carles","osm_key":"amenity","osm_value":"restaurant","osm_type":"N","osm_id":1,
  "extra":{"contact:website":"https://cansole.example","phone":"+34 932 21 50 12; +34 600 00 00 00","opening_hours":"Tu-Sa 13:00-16:00; Su 13:00-16:00"}}},
 {"geometry":{"coordinates":[2.19,41.377]},"properties":{"name":"Bad tags","street":"Carrer Nou","osm_key":"amenity","osm_value":"bar","osm_type":"N","osm_id":2,
  "extra":{"website":"www.bad.example","phone":"ask at the bar","opening_hours":7}}}
]}`)
	}))
	defer photon.Close()
	opts := testOptions
	opts.PlacesURL = photon.URL
	e := newEnv(t, opts)
	ana := e.signUp("ana@example.com", "Ana")

	var list struct {
		Items []map[string]any
	}
	e.expect(request{method: "GET", path: "/v1/places?search=can+sole", token: ana.Token}, http.StatusOK).decode(t, &list)
	if len(list.Items) != 2 {
		t.Fatalf("%d places", len(list.Items))
	}
	first := list.Items[0]
	if first["website"] != "https://cansole.example" || first["phone"] != "+34 932 21 50 12" || first["opening_hours"] != "Tu-Sa 13:00-16:00\nSu 13:00-16:00" {
		t.Errorf("first place %v", first)
	}
	for _, k := range []string{"website", "phone", "opening_hours"} {
		if v, ok := list.Items[1][k]; ok {
			t.Errorf("an invalid %s is in the answer: %v", k, v)
		}
	}
}

func TestPriceLevelFilter(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	ids := map[string]string{}
	for i, name := range []string{"L1", "L2", "L3", "L4"} {
		ids[name] = e.createRestaurant(ana.Token, fmt.Sprintf(`{"name":%q,"price_level":%d}`, name, i+1))["id"].(string)
	}
	e.createRestaurant(ana.Token, `{"name":"None"}`)

	page := func(query string) (string, string) {
		t.Helper()
		var p struct {
			Items      []struct{ Name string } `json:"items"`
			NextCursor *string                 `json:"next_cursor"`
		}
		e.expect(request{method: "GET", path: "/v1/restaurants?sort=name&" + query}, http.StatusOK).decode(t, &p)
		var names []string
		for _, r := range p.Items {
			names = append(names, r.Name)
		}
		cursor := ""
		if p.NextCursor != nil {
			cursor = *p.NextCursor
		}
		return strings.Join(names, ","), cursor
	}

	// A restaurant without a price level matches no comparison.
	for query, want := range map[string]string{
		"price_level[lte]=2":                    "L1,L2",
		"price_level[lt]=2":                     "L1",
		"price_level[eq]=3":                     "L3",
		"price_level[gte]=3":                    "L3,L4",
		"price_level[gt]=1&price_level[lt]=4":   "L2,L3",
		"price_level[gte]=3&price_level[lte]=2": "",
		"":                                      "L1,L2,L3,L4,None",
	} {
		if got, _ := page(query); got != want {
			t.Errorf("%q: %q, want %q", query, got, want)
		}
	}

	// The restaurants page sends the overall sort, near, and within.
	e.createReview(ana.Token, ids["L2"], `{"date":"2024-11-06","space":9,"service":9,"food":9,"price":9,"cleanliness":9,"title":"Top"}`)
	e.createReview(ana.Token, ids["L3"], `{"date":"2024-11-06","space":5,"service":5,"food":5,"price":5,"cleanliness":5,"title":"Fine"}`)
	box := "within[west]=2&within[south]=41&within[east]=3&within[north]=42"
	for query, want := range map[string]string{
		"sort=-overall&price_level[lte]=2":                                                             "L2,L1",
		"sort=-overall&price_level[gte]=2&" + box:                                                      "L2,L3,L4",
		"sort=distance&near[latitude]=41.3766&near[longitude]=2.1893&price_level[eq]=1":                "L1",
		"sort=name&price_level[lte]=4&within[west]=3&within[south]=41&within[east]=4&within[north]=42": "",
	} {
		var p struct {
			Items []struct{ Name string } `json:"items"`
		}
		e.expect(request{method: "GET", path: "/v1/restaurants?" + query}, http.StatusOK).decode(t, &p)
		var names []string
		for _, r := range p.Items {
			names = append(names, r.Name)
		}
		if got := strings.Join(names, ","); got != want {
			t.Errorf("%q: %q, want %q", query, got, want)
		}
	}

	first, cursor := page("price_level[lte]=2&limit=1")
	if next, _ := page("price_level[lte]=2&limit=1&cursor=" + url.QueryEscape(cursor)); first != "L1" || next != "L2" {
		t.Errorf("pages %q and %q, want L1 and L2", first, next)
	}
	e.expect(request{method: "GET", path: "/v1/restaurants?sort=name&price_level[lte]=3&cursor=" + url.QueryEscape(cursor)}, http.StatusBadRequest)

	for _, query := range []string{"price_level[lte]=0", "price_level[lte]=5", "price_level[lte]=x", "price_level[lte]=1.5", "price_level[in]=1", "price_level=1"} {
		e.expect(request{method: "GET", path: "/v1/restaurants?" + query}, http.StatusBadRequest)
	}
}
