package publicapi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRestaurantDetails(t *testing.T) {
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
		e.expect(request{method: "PATCH", path: path, body: `{"phone":"932215012"}`, token: joe.Token}, http.StatusForbidden)

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
		e.createReview(joe.Token, r["id"].(string), `{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"Great"}`)
		e.expect(request{method: "PATCH", path: path, body: `{"price_level":2}`, token: ana.Token}, http.StatusOK)
	})
}

func TestSearchPlacesDetails(t *testing.T) {
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
