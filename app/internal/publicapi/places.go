package publicapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
)

const (
	// maxPlaces is the maxItems of PlaceList in the spec.
	maxPlaces        = 10
	placeCacheTTL    = 24 * time.Hour
	placeCacheSize   = 1000
	placeTimeout     = 5 * time.Second
	maxPlaceBody     = 1 << 20
	maxAddressLength = 255
)

// placeKinds are the OpenStreetMap amenity values of one restaurant. Only
// these places get a place ID. An address can have many restaurants.
var placeKinds = map[string]bool{
	"restaurant": true, "cafe": true, "fast_food": true, "bar": true, "pub": true, "biergarten": true, "ice_cream": true,
}

// placeSearch gets places from a Photon geocoder and keeps the answers.
type placeSearch struct {
	baseURL   string
	userAgent string
	client    *http.Client
	// upstream counts the requests of all users to the geocoder.
	upstream *rateLimiter

	mu    sync.Mutex
	cache map[string]placeAnswer
	now   func() time.Time
}

type placeAnswer struct {
	places  []reviewsapi.Place
	expires time.Time
}

func newPlaceSearch(baseURL, userAgent string, limit int, window time.Duration) *placeSearch {
	return &placeSearch{
		baseURL:   strings.TrimRight(baseURL, "/"),
		userAgent: userAgent,
		client:    &http.Client{Timeout: placeTimeout},
		upstream:  newRateLimiter(limit, window),
		cache:     map[string]placeAnswer{},
		now:       time.Now,
	}
}

// errPlaceSearchBusy means that all users together reached the limit of
// requests to the geocoder.
var errPlaceSearchBusy = errors.New("the geocoder request limit is reached")

// search returns the places for the text. The point is optional. Its
// coordinates have one decimal, so near users share a cache entry.
func (p *placeSearch) search(ctx context.Context, text string, point *[2]float64) ([]reviewsapi.Place, error) {
	query := url.Values{"q": {strings.Join(strings.Fields(text), " ")}, "limit": {strconv.Itoa(maxPlaces)}}
	if point != nil {
		query.Set("lat", strconv.FormatFloat(math.Round(point[0]*10)/10, 'f', 1, 64))
		query.Set("lon", strconv.FormatFloat(math.Round(point[1]*10)/10, 'f', 1, 64))
	}
	key := strings.ToLower(query.Encode())

	p.mu.Lock()
	answer, ok := p.cache[key]
	p.mu.Unlock()
	if ok && p.now().Before(answer.expires) {
		return answer.places, nil
	}
	if wait, _ := p.upstream.allow(""); wait > 0 {
		return nil, errPlaceSearchBusy
	}
	places, err := p.fetch(ctx, query)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	if len(p.cache) >= placeCacheSize {
		clear(p.cache)
	}
	p.cache[key] = placeAnswer{places: places, expires: p.now().Add(placeCacheTTL)}
	p.mu.Unlock()
	return places, nil
}

func (p *placeSearch) fetch(ctx context.Context, query url.Values) ([]reviewsapi.Place, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", p.userAgent)
	res, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the geocoder answered %d", res.StatusCode)
	}
	var body photonAnswer
	if err := json.NewDecoder(io.LimitReader(res.Body, maxPlaceBody)).Decode(&body); err != nil {
		return nil, err
	}
	places := []reviewsapi.Place{}
	seen := map[string]bool{}
	for _, f := range body.Features {
		place, ok := f.place()
		if !ok {
			continue
		}
		key := place.Location.Address
		if place.Name != nil {
			key += "\n" + *place.Name
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		if places = append(places, place); len(places) == maxPlaces {
			break
		}
	}
	return places, nil
}

// photonAnswer is the GeoJSON answer of the Photon /api endpoint.
type photonAnswer struct {
	Features []photonFeature `json:"features"`
}

type photonFeature struct {
	Geometry struct {
		// Coordinates has the longitude first.
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		Name        string `json:"name"`
		Street      string `json:"street"`
		HouseNumber string `json:"housenumber"`
		Postcode    string `json:"postcode"`
		City        string `json:"city"`
		Locality    string `json:"locality"`
		District    string `json:"district"`
		County      string `json:"county"`
		State       string `json:"state"`
		Country     string `json:"country"`
		OSMKey      string `json:"osm_key"`
		OSMValue    string `json:"osm_value"`
		OSMType     string `json:"osm_type"`
		OSMID       int64  `json:"osm_id"`
	} `json:"properties"`
}

// place returns false for a feature without a point or without text.
func (f photonFeature) place() (reviewsapi.Place, bool) {
	c := f.Geometry.Coordinates
	if len(c) != 2 || math.Abs(c[1]) > 90 || math.Abs(c[0]) > 180 {
		return reviewsapi.Place{}, false
	}
	p := f.Properties
	name := cleanPlaceText(p.Name)
	town := firstText(p.City, p.Locality, p.District, p.County)
	var parts []string
	for _, part := range []string{
		strings.TrimSpace(p.Street + " " + p.HouseNumber),
		strings.TrimSpace(p.Postcode + " " + town),
		p.State,
		p.Country,
	} {
		// A city state has the name of its city.
		if part = cleanPlaceText(part); part != "" && (len(parts) == 0 || !strings.HasSuffix(parts[len(parts)-1], part)) {
			parts = append(parts, part)
		}
	}
	address := cleanPlaceText(strings.Join(parts, ", "))
	if address == "" {
		address = name
	}
	if address == "" {
		return reviewsapi.Place{}, false
	}
	place := reviewsapi.Place{Location: reviewsapi.Location{Latitude: c[1], Longitude: c[0], Address: address}}
	if name != "" {
		place.Name = &name
	}
	if p.OSMKey == "amenity" && placeKinds[p.OSMValue] && len(p.OSMType) == 1 && p.OSMID > 0 {
		id := fmt.Sprintf("osm:%s%d", p.OSMType, p.OSMID)
		place.Location.PlaceId = &id
	}
	return place, true
}

func firstText(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// cleanPlaceText removes the characters that the Address schema refuses. It
// also cuts the text to the length of the schema.
func cleanPlaceText(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return ' '
		case hiddenRune(r):
			return -1
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	if runes := []rune(s); len(runes) > maxAddressLength {
		s = strings.TrimSpace(string(runes[:maxAddressLength]))
	}
	return s
}

// hiddenRune has the characters of the pattern of the Name schema.
func hiddenRune(r rune) bool {
	switch {
	case r <= 0x1F, r >= 0x7F && r <= 0x9F:
		return true
	case r >= 0x202A && r <= 0x202E, r >= 0x2060 && r <= 0x206F, r >= 0xFFF9 && r <= 0xFFFB:
		return true
	}
	return strings.ContainsRune("\u00AD\u061C\u115F\u1160\u180E\u200B\u200E\u200F\u3164\uFEFF\uFFA0", r)
}

// SearchPlaces gets the places from the geocoder. A request that the
// geocoder cannot answer does not count for the rate limit of the user.
func (s *Server) SearchPlaces(ctx context.Context, req reviewsapi.SearchPlacesRequestObject) (reviewsapi.SearchPlacesResponseObject, error) {
	p := req.Params
	if (p.Latitude == nil) != (p.Longitude == nil) {
		return reviewsapi.SearchPlaces400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: badRequest("The latitude and longitude parameters need each other."),
		}, nil
	}
	unavailable := func(detail string) (reviewsapi.SearchPlacesResponseObject, error) {
		stateFrom(ctx).giveBackLimit()
		return reviewsapi.SearchPlaces503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: reviewsapi.ServiceUnavailableApplicationProblemPlusJSONResponse(
				newProblem(http.StatusServiceUnavailable, detail)),
		}, nil
	}
	if s.places == nil {
		return unavailable("The place search is off.")
	}
	var point *[2]float64
	if p.Latitude != nil {
		point = &[2]float64{*p.Latitude, *p.Longitude}
	}
	places, err := s.places.search(ctx, p.Search, point)
	if err != nil {
		s.app.Logger().Warn("Place search failed", "error", err)
		return unavailable("The place search does not answer. Try again later.")
	}
	return reviewsapi.SearchPlaces200JSONResponse{Items: places}, nil
}
