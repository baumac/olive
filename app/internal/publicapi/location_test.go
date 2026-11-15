package publicapi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func locationJSON(lat, lon float64, address, placeID string) string {
	place := ""
	if placeID != "" {
		place = fmt.Sprintf(`,"place_id":%q`, placeID)
	}
	return fmt.Sprintf(`{"latitude":%v,"longitude":%v,"address":%q%s}`, lat, lon, address, place)
}

func TestRestaurantLocation(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	const review = `{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"Great"}`
	barceloneta := locationJSON(41.3766, 2.1893, "Carrer de Sant Carles 4, 08003 Barcelona", "osm:N1")

	r := e.createRestaurant(ana.Token, `{"name":"Can Sole","location":`+barceloneta+`}`)
	id := r["id"].(string)
	path := "/v1/restaurants/" + id

	t.Run("create and get", func(t *testing.T) {
		for _, got := range []map[string]any{r, e.expect(request{method: "GET", path: path}, http.StatusOK).object(t)} {
			l, _ := got["location"].(map[string]any)
			if l["latitude"] != 41.3766 || l["longitude"] != 2.1893 || l["address"] != "Carrer de Sant Carles 4, 08003 Barcelona" || l["place_id"] != "osm:N1" {
				t.Errorf("location %v", got["location"])
			}
		}
		plain := e.createRestaurant(ana.Token, `{"name":"Plain"}`)
		if _, ok := plain["location"].(map[string]any)["place_id"]; ok {
			t.Errorf("a location without a place ID has one: %v", plain["location"])
		}
	})

	t.Run("create errors", func(t *testing.T) {
		for body, want := range map[string]string{
			`{"name":"A"}`:                 "location",
			`{"name":"A","location":null}`: "location",
			`{"name":"A","location":{}}`:   "location.address,location.latitude,location.longitude",
			`{"name":"A","location":{"latitude":91,"longitude":-181,"address":""}}`:                             "location.address,location.latitude,location.longitude",
			`{"name":"A","location":{"latitude":1,"longitude":1,"address":"A","place_id":"no colon"}}`:          "location.place_id",
			`{"name":"A","location":{"latitude":"1","longitude":1,"address":"A​"}}`:                             "location.address,location.latitude",
			`{"name":"A","location":{"latitude":1,"longitude":1,"address":"A","extra":1}}`:                      "location",
			`{"name":"A","location":{"latitude":1,"longitude":1,"address":"` + strings.Repeat("a", 256) + `"}}`: "location.address",
		} {
			res := e.expect(request{method: "POST", path: "/v1/restaurants", body: body, token: ana.Token}, http.StatusUnprocessableEntity)
			if got := strings.Join(errorProperties(t, res), ","); got != want {
				t.Errorf("%s: properties %q, want %q", body, got, want)
			}
		}
	})

	t.Run("a second restaurant with the place gets 409", func(t *testing.T) {
		res := e.expect(request{method: "POST", path: "/v1/restaurants",
			body: `{"name":"Can Solé","location":` + locationJSON(41.3767, 2.1894, "Sant Carles 4", "osm:N1") + `}`, token: joe.Token}, http.StatusConflict)
		if got := res.object(t); got["restaurant"] != id || got["property"] != "location" {
			t.Errorf("problem %v, want restaurant %s and property location", got, id)
		}
		other := e.createRestaurant(joe.Token, `{"name":"Next door","location":`+locationJSON(41.3767, 2.1894, "Sant Carles 6", "osm:N2")+`}`)
		res = e.expect(request{method: "PATCH", path: "/v1/restaurants/" + other["id"].(string),
			body: `{"location":` + barceloneta + `}`, token: joe.Token}, http.StatusConflict)
		if got := res.object(t)["restaurant"]; got != id {
			t.Errorf("patch: restaurant %v, want %s", got, id)
		}
		// The same address without a place ID is not a conflict.
		e.createRestaurant(joe.Token, `{"name":"Upstairs","location":`+locationJSON(41.3766, 2.1893, "Carrer de Sant Carles 4, 08003 Barcelona", "")+`}`)
	})

	t.Run("patch", func(t *testing.T) {
		same := e.expect(request{method: "PATCH", path: path, body: `{"location":` + barceloneta + `}`, token: ana.Token}, http.StatusOK).object(t)
		if same["updated"] != r["updated"] {
			t.Errorf("an equal location changed updated")
		}
		e.expect(request{method: "PATCH", path: path, body: `{"location":null}`, token: ana.Token}, http.StatusUnprocessableEntity)
		e.expect(request{method: "PATCH", path: path, body: `{"location":{"latitude":1}}`, token: ana.Token}, http.StatusUnprocessableEntity)
		e.expect(request{method: "PATCH", path: path, body: `{"location":` + barceloneta + `}`, token: joe.Token}, http.StatusForbidden)

		moved := e.expect(request{method: "PATCH", path: path,
			body: `{"location":` + locationJSON(41.38, 2.19, "Passeig de Joan de Borbó 1", "") + `}`, token: ana.Token}, http.StatusOK).object(t)
		l := moved["location"].(map[string]any)
		if _, hasPlace := l["place_id"]; hasPlace || l["address"] != "Passeig de Joan de Borbó 1" || l["latitude"] != 41.38 {
			t.Errorf("moved location %v", l)
		}
		// The place ID is free again.
		e.expect(request{method: "PATCH", path: path, body: `{"location":` + barceloneta + `}`, token: ana.Token}, http.StatusOK)
	})

	t.Run("reviews of other users lock the location", func(t *testing.T) {
		e.createReview(ana.Token, id, review)
		e.expect(request{method: "PATCH", path: path, body: `{"location":` + locationJSON(41.3766, 2.1893, "Sant Carles 4", "osm:N1") + `}`, token: ana.Token}, http.StatusOK)
		e.createReview(joe.Token, id, review)
		res := e.expect(request{method: "PATCH", path: path, body: `{"location":` + barceloneta + `}`, token: ana.Token}, http.StatusConflict)
		if got := res.object(t); got["restaurant"] != nil || got["property"] != "location" {
			t.Errorf("lock conflict %s", res.body)
		}
		res = e.expect(request{method: "PATCH", path: path, body: `{"name":"Other"}`, token: ana.Token}, http.StatusConflict)
		if got := res.object(t)["property"]; got != "name" {
			t.Errorf("name conflict %s", res.body)
		}
		e.expect(request{method: "PATCH", path: path, body: `{"description":"x","location":` + locationJSON(41.3766, 2.1893, "Sant Carles 4", "osm:N1") + `}`, token: ana.Token}, http.StatusOK)
	})

	t.Run("a restaurant from before locations can get one", func(t *testing.T) {
		old := e.createRestaurant(ana.Token, `{"name":"Old"}`)["id"].(string)
		rec, err := e.app.FindRecordById("restaurants", old)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range []string{"latitude", "longitude"} {
			rec.Set(f, 0)
		}
		rec.Set("address", "")
		if err := e.app.Save(rec); err != nil {
			t.Fatal(err)
		}
		got := e.expect(request{method: "GET", path: "/v1/restaurants/" + old}, http.StatusOK).object(t)
		if l, ok := got["location"]; !ok || l != nil {
			t.Errorf("location %v, want null", l)
		}
		e.createReview(joe.Token, old, review)
		e.expect(request{method: "PATCH", path: "/v1/restaurants/" + old, body: `{"location":` + testLocation + `}`, token: ana.Token}, http.StatusOK)
		e.expect(request{method: "PATCH", path: "/v1/restaurants/" + old, body: `{"location":` + locationJSON(1, 1, "Other", "") + `}`, token: ana.Token}, http.StatusConflict)
	})
}

func TestListRestaurantsNear(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	// One degree of latitude is about 111 km. 0.0005 degrees are about 56 m.
	for name, l := range map[string][2]float64{
		"Center": {41.3766, 2.1893}, "North 56 m": {41.3771, 2.1893}, "East 62 m": {41.3766, 2.19005},
		"North 110 m": {41.3776, 2.1893}, "Madrid": {40.4168, -3.7038},
		"Date line east": {0, 179.9999}, "Date line west": {0, -179.9999},
	} {
		e.createRestaurant(ana.Token, fmt.Sprintf(`{"name":%q,"location":%s}`, name, locationJSON(l[0], l[1], "Qwx 1", "")))
	}
	none := e.createRestaurant(ana.Token, `{"name":"No location"}`)["id"].(string)
	if _, err := e.app.DB().NewQuery("UPDATE restaurants SET address = '', latitude = 41.3766, longitude = 2.1893 WHERE id = {:id}").
		Bind(map[string]any{"id": none}).Execute(); err != nil {
		t.Fatal(err)
	}

	names := func(query string) string {
		t.Helper()
		var page struct {
			Items []map[string]any `json:"items"`
		}
		e.expect(request{method: "GET", path: "/v1/restaurants?sort=name&" + query}, http.StatusOK).decode(t, &page)
		var out []string
		for _, it := range page.Items {
			out = append(out, it["name"].(string))
		}
		return strings.Join(out, ",")
	}
	near := func(lat, lon float64, radius int) string {
		return fmt.Sprintf("near[latitude]=%v&near[longitude]=%v&near[radius]=%d", lat, lon, radius)
	}
	for query, want := range map[string]string{
		near(41.3766, 2.1893, 1):     "Center",
		near(41.3766, 2.1893, 75):    "Center,East 62 m,North 56 m",
		near(41.3766, 2.1893, 150):   "Center,East 62 m,North 110 m,North 56 m",
		near(41.3766, 2.1893, 50000): "Center,East 62 m,North 110 m,North 56 m",
		near(0, 180, 100):            "Date line east,Date line west",
		near(10, 10, 100):            "",
	} {
		if got := names(query); got != want {
			t.Errorf("%s: %q, want %q", query, got, want)
		}
	}

	var first struct {
		NextCursor *string `json:"next_cursor"`
	}
	e.expect(request{method: "GET", path: "/v1/restaurants?limit=1&" + near(41.3766, 2.1893, 150)}, http.StatusOK).decode(t, &first)
	if first.NextCursor == nil {
		t.Fatal("no next cursor")
	}
	cursor := "&cursor=" + url.QueryEscape(*first.NextCursor)
	e.expect(request{method: "GET", path: "/v1/restaurants?limit=1&" + near(41.3766, 2.1893, 150) + cursor}, http.StatusOK)
	e.expect(request{method: "GET", path: "/v1/restaurants?limit=1&" + near(41.3766, 2.1893, 75) + cursor}, http.StatusBadRequest)

	for _, query := range []string{
		"near[latitude]=1", "near[latitude]=1&near[radius]=5", "near[latitude]=91&near[longitude]=1&near[radius]=5",
		"near[latitude]=1&near[longitude]=1&near[radius]=0", "near[latitude]=1&near[longitude]=1&near[radius]=50001",
		"near[latitude]=1&near[longitude]=1&near[radius]=5&near[extra]=1", "near=1",
	} {
		e.expect(request{method: "GET", path: "/v1/restaurants?" + query}, http.StatusBadRequest)
	}
}

func TestListRestaurantsByDistance(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	for name, l := range map[string][2]float64{
		"Center": {41.3766, 2.1893}, "North 56 m": {41.3771, 2.1893}, "East 62 m": {41.3766, 2.19005},
		"North 110 m": {41.3776, 2.1893}, "Madrid": {40.4168, -3.7038},
		"Date line east": {0, 179.9999}, "Date line west": {0, -179.9998},
	} {
		e.createRestaurant(ana.Token, fmt.Sprintf(`{"name":%q,"location":%s}`, name, locationJSON(l[0], l[1], "Qwx 1", "")))
	}
	// Two restaurants at one point have the same distance. The id is then
	// the order.
	e.createRestaurant(ana.Token, `{"name":"Center","location":`+locationJSON(41.3766, 2.1893, "Qwx 2", "")+`}`)

	none := e.createRestaurant(ana.Token, `{"name":"No location"}`)["id"].(string)
	if _, err := e.app.DB().NewQuery("UPDATE restaurants SET address = '', latitude = 41.3766, longitude = 2.1893 WHERE id = {:id}").
		Bind(map[string]any{"id": none}).Execute(); err != nil {
		t.Fatal(err)
	}

	const point = "near[latitude]=41.3766&near[longitude]=2.1893"
	const barceloneta = point + "&near[radius]=150"
	names := func(query string, limit int) string {
		t.Helper()
		var out []string
		cursor := ""
		for {
			var page struct {
				Items      []map[string]any `json:"items"`
				NextCursor *string          `json:"next_cursor"`
			}
			e.expect(request{method: "GET", path: fmt.Sprintf("/v1/restaurants?limit=%d&%s%s", limit, query, cursor)}, http.StatusOK).decode(t, &page)
			for _, it := range page.Items {
				out = append(out, it["name"].(string))
			}
			if page.NextCursor == nil {
				return strings.Join(out, ",")
			}
			cursor = "&cursor=" + url.QueryEscape(*page.NextCursor)
		}
	}
	for query, want := range map[string]string{
		"sort=distance&" + barceloneta:                                        "Center,Center,North 56 m,East 62 m,North 110 m",
		"sort=-distance&" + barceloneta:                                       "North 110 m,East 62 m,North 56 m,Center,Center",
		"sort=name,distance&" + barceloneta:                                   "Center,Center,East 62 m,North 110 m,North 56 m",
		"sort=distance&search=north&" + barceloneta:                           "North 56 m,North 110 m",
		"sort=distance&near[latitude]=0&near[longitude]=180&near[radius]=100": "Date line east,Date line west",
		// Without a radius, near is only the point of the sort.
		"sort=distance&" + point:  "Center,Center,North 56 m,East 62 m,North 110 m,Madrid,Date line east,Date line west,No location",
		"sort=-distance&" + point: "Date line west,Date line east,Madrid,North 110 m,East 62 m,North 56 m,Center,Center,No location",
	} {
		for _, limit := range []int{1, 2, 100} {
			if got := names(query, limit); got != want {
				t.Errorf("%s with limit %d: %q, want %q", query, limit, got, want)
			}
		}
	}

	for _, query := range []string{"sort=distance", "sort=-distance", "sort=name,distance", "sort=distance,distance&" + barceloneta} {
		e.expect(request{method: "GET", path: "/v1/restaurants?" + query}, http.StatusBadRequest)
	}
}

func TestSearchFindsTheAddress(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	id := e.createRestaurant(ana.Token, `{"name":"Can Sole","location":`+locationJSON(41.3766, 2.1893, "Carrer de Sant Carles 4, Barcelona", "")+`}`)["id"].(string)
	e.createRestaurant(ana.Token, `{"name":"Sant Pau","description":"Far from Carles","location":`+locationJSON(41.4, 2.2, "Carrer Nou 1, Sant Pol de Mar", "")+`}`)

	found := func(search string) string {
		t.Helper()
		var page struct {
			Items []map[string]any `json:"items"`
		}
		e.expect(request{method: "GET", path: "/v1/restaurants?search=" + url.QueryEscape(search)}, http.StatusOK).decode(t, &page)
		var out []string
		for _, it := range page.Items {
			out = append(out, it["name"].(string))
		}
		return strings.Join(out, ",")
	}
	if got := found("barcelona"); got != "Can Sole" {
		t.Errorf("barcelona: %q", got)
	}
	// The name counts more than the address. The address counts more than
	// the description.
	if got := found("sant"); got != "Sant Pau,Can Sole" {
		t.Errorf("sant: %q", got)
	}
	if got := found("carles"); got != "Can Sole,Sant Pau" {
		t.Errorf("carles: %q", got)
	}
	e.expect(request{method: "PATCH", path: "/v1/restaurants/" + id,
		body: `{"location":` + locationJSON(41.3766, 2.1893, "Rambla 1, Girona", "") + `}`, token: ana.Token}, http.StatusOK)
	if got := found("barcelona") + "|" + found("girona"); got != "|Can Sole" {
		t.Errorf("after the move: %q", got)
	}
}

const photonBody = `{"features":[
 {"geometry":{"coordinates":[2.1893,41.3766]},"properties":{"name":"Can Solé","street":"Carrer de Sant Carles","housenumber":"4","postcode":"08003","city":"Barcelona","state":"Catalunya","country":"España","osm_key":"amenity","osm_value":"restaurant","osm_type":"N","osm_id":1234567890}},
 {"geometry":{"coordinates":[2.1893,41.3766]},"properties":{"name":"Can Solé","street":"Carrer de Sant Carles","housenumber":"4","postcode":"08003","city":"Barcelona","state":"Catalunya","country":"España","osm_key":"amenity","osm_value":"restaurant","osm_type":"N","osm_id":1234567890}},
 {"geometry":{"coordinates":[2.19,41.377]},"properties":{"street":"Carrer de Sant Carles","housenumber":"6","postcode":"08003","city":"Barcelona","country":"España","osm_key":"building","osm_value":"yes","osm_type":"W","osm_id":5}},
 {"geometry":{"coordinates":[13.3888599,52.5170365]},"properties":{"name":"Ber​lin","state":"Berlin","country":"Germany","osm_key":"place","osm_value":"city","osm_type":"N","osm_id":240109189}},
 {"geometry":{"coordinates":[1]},"properties":{"name":"No point"}},
 {"geometry":{"coordinates":[1,2]},"properties":{}}
]}`

func TestSearchPlaces(t *testing.T) {
	var calls atomic.Int32
	var lastQuery atomic.Value
	status := http.StatusOK
	photon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		lastQuery.Store(r.URL.Path + "?" + r.URL.RawQuery + " " + r.UserAgent())
		w.WriteHeader(status)
		fmt.Fprint(w, photonBody)
	}))
	defer photon.Close()

	opts := testOptions
	opts.PlacesURL = photon.URL
	opts.PlaceSearchLimit = 12
	e := newEnv(t, opts)
	ana := e.signUp("ana@example.com", "Ana")

	e.expect(request{method: "GET", path: "/v1/places?search=can+sole"}, http.StatusUnauthorized)
	for _, query := range []string{"", "search=ab", "search=abc&latitude=1", "search=abc&latitude=91&longitude=1", "search=abc&limit=1"} {
		e.expect(request{method: "GET", path: "/v1/places?" + query, token: ana.Token}, http.StatusBadRequest)
	}
	if calls.Load() != 0 {
		t.Fatalf("%d geocoder calls for invalid requests", calls.Load())
	}

	var list struct {
		Items []struct {
			Name     *string
			Location struct {
				Latitude, Longitude float64
				Address             string
				PlaceID             *string `json:"place_id"`
			}
		}
	}
	e.expect(request{method: "GET", path: "/v1/places?search=Can%20%20Sole&latitude=41.3874&longitude=2.1686", token: ana.Token}, http.StatusOK).decode(t, &list)
	if len(list.Items) != 3 {
		t.Fatalf("%d places: %+v", len(list.Items), list.Items)
	}
	first := list.Items[0]
	if *first.Name != "Can Solé" || first.Location.Address != "Carrer de Sant Carles 4, 08003 Barcelona, Catalunya, España" ||
		first.Location.Latitude != 41.3766 || first.Location.Longitude != 2.1893 || *first.Location.PlaceID != "osm:N1234567890" {
		t.Errorf("first place %+v", first)
	}
	if house := list.Items[1]; house.Name != nil || house.Location.PlaceID != nil || house.Location.Address != "Carrer de Sant Carles 6, 08003 Barcelona, España" {
		t.Errorf("house %+v", house)
	}
	if city := list.Items[2]; *city.Name != "Berlin" || city.Location.Address != "Berlin, Germany" || city.Location.PlaceID != nil {
		t.Errorf("city %+v", city)
	}
	if got := lastQuery.Load().(string); !strings.HasPrefix(got, "/api?lat=41.4&limit=10&lon=2.2&q=Can+Sole reviews-app (") {
		t.Errorf("geocoder request %q", got)
	}

	// An equal search with a near point uses the cache.
	e.expect(request{method: "GET", path: "/v1/places?search=can+sole&latitude=41.41&longitude=2.21", token: ana.Token}, http.StatusOK)
	if calls.Load() != 1 {
		t.Errorf("%d geocoder calls, want 1", calls.Load())
	}

	// The limit counted 5 invalid requests and 2 searches. A failed search
	// does not count.
	status = http.StatusBadGateway
	for range 10 {
		e.expect(request{method: "GET", path: "/v1/places?search=other", token: ana.Token}, http.StatusServiceUnavailable)
	}
	status = http.StatusOK
	for range 5 {
		e.expect(request{method: "GET", path: "/v1/places?search=can+sole", token: ana.Token}, http.StatusOK)
	}
	res := e.expect(request{method: "GET", path: "/v1/places?search=can+sole", token: ana.Token}, http.StatusTooManyRequests)
	if res.header.Get("Retry-After") == "" {
		t.Error("no Retry-After")
	}
}

func TestSearchPlacesOffAndBusy(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	e.expect(request{method: "GET", path: "/v1/places?search=can+sole", token: ana.Token}, http.StatusServiceUnavailable)

	var calls atomic.Int32
	photon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprint(w, `{"features":[]}`)
	}))
	defer photon.Close()
	opts := testOptions
	opts.PlacesURL = photon.URL
	opts.PlaceUpstreamLimit = 2
	busy := newEnv(t, opts)
	joe := busy.signUp("joe@example.com", "Joe")
	for i, want := range []int{http.StatusOK, http.StatusOK, http.StatusServiceUnavailable, http.StatusOK} {
		// The last search is in the cache.
		search := fmt.Sprintf("place%d", i%3)
		busy.expect(request{method: "GET", path: "/v1/places?search=" + search, token: joe.Token}, want)
	}
	if calls.Load() != 2 {
		t.Errorf("%d geocoder calls, want 2", calls.Load())
	}
}

func TestListRestaurantsWithin(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	for name, l := range map[string][2]float64{
		"Barcelona": {41.3874, 2.1686}, "Madrid": {40.4168, -3.7038}, "Girona": {41.9794, 2.8214},
		"Fiji": {-17.7134, 178.065}, "Samoa": {-13.8333, -171.75}, "Edge": {41.3, 2.1},
	} {
		e.createRestaurant(ana.Token, fmt.Sprintf(`{"name":%q,"location":%s}`, name, locationJSON(l[0], l[1], "Qwx 1", "")))
	}
	none := e.createRestaurant(ana.Token, `{"name":"No location"}`)["id"].(string)
	if _, err := e.app.DB().NewQuery("UPDATE restaurants SET address = '', latitude = 41.35, longitude = 2.15 WHERE id = {:id}").
		Bind(map[string]any{"id": none}).Execute(); err != nil {
		t.Fatal(err)
	}

	box := func(west, south, east, north float64) string {
		return fmt.Sprintf("within[west]=%v&within[south]=%v&within[east]=%v&within[north]=%v", west, south, east, north)
	}
	for query, want := range map[string]string{
		box(2.1, 41.3, 2.2, 41.4):                       "Barcelona,Edge",
		box(-10, 35, 5, 45):                             "Barcelona,Edge,Girona,Madrid",
		box(170, -30, -160, 0):                          "Fiji,Samoa",
		box(-180, -90, 180, 90):                         "Barcelona,Edge,Fiji,Girona,Madrid,Samoa",
		box(2.1, 41.3, 2.2, 41.4) + "&search=barcelona": "Barcelona",
		box(0, 0, 1, 1):                                 "",
	} {
		var page struct {
			Items []map[string]any `json:"items"`
		}
		e.expect(request{method: "GET", path: "/v1/restaurants?sort=name&" + query}, http.StatusOK).decode(t, &page)
		var names []string
		for _, it := range page.Items {
			names = append(names, it["name"].(string))
		}
		if got := strings.Join(names, ","); got != want {
			t.Errorf("%s: %q, want %q", query, got, want)
		}
	}
	for _, query := range []string{
		"within[west]=1&within[south]=1&within[east]=2", box(1, 2, 2, 1), box(-181, 1, 2, 2), box(1, 1, 2, 91),
		box(1, 1, 2, 2) + "&within[up]=1",
	} {
		e.expect(request{method: "GET", path: "/v1/restaurants?" + query}, http.StatusBadRequest)
	}
}
