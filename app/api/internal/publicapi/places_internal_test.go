package publicapi

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPlaceSearchCache(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	agents := make(chan string, 10)
	photon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		agents <- r.UserAgent()
		<-release
		fmt.Fprint(w, `{"features":[]}`)
	}))
	defer photon.Close()

	url := "https://one.example.com"
	p := newPlaceSearch(photon.URL, func() string { return "reviews-app (" + url + ")" }, 100, time.Minute)
	now := time.Unix(0, 0)
	p.now = func() time.Time { return now }

	// Equal searches at one time send one request.
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			if _, err := p.search(context.Background(), "can sole", nil); err != nil {
				t.Error(err)
			}
		})
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	if calls.Load() != 1 {
		t.Errorf("%d geocoder calls for equal searches, want 1", calls.Load())
	}

	url = "https://two.example.com"
	if _, err := p.search(context.Background(), "other", nil); err != nil {
		t.Fatal(err)
	}
	<-agents
	if got := <-agents; got != "reviews-app (https://two.example.com)" {
		t.Errorf("User-Agent %q after a settings change", got)
	}

	searches := slices.Collect(maps.Keys(p.cache))
	if len(searches) != 2 {
		t.Fatalf("%d answers after two searches", len(searches))
	}

	// A full cache removes the answer that expires first.
	now = now.Add(time.Second)
	for i := range placeCacheSize - 2 {
		p.store(fmt.Sprint(i), nil)
	}
	p.store("new", nil)
	if len(p.cache) != placeCacheSize {
		t.Errorf("%d answers, want %d", len(p.cache), placeCacheSize)
	}
	_, first := p.cached(searches[0])
	_, second := p.cached(searches[1])
	if first == second {
		t.Errorf("the older answers are both in the cache: %v", first)
	}
	if _, ok := p.cached("new"); !ok {
		t.Error("the new answer is not in the cache")
	}

	// A full cache removes all expired answers.
	now = now.Add(placeCacheTTL)
	if _, ok := p.cached("new"); ok {
		t.Error("an expired answer is in the cache")
	}
	p.store("latest", nil)
	if len(p.cache) != 1 {
		t.Errorf("%d answers after all expired, want 1", len(p.cache))
	}
}

func TestPlaceDetails(t *testing.T) {
	text := func(v *string) string {
		if v == nil {
			return "<nil>"
		}
		return *v
	}
	for in, want := range map[string]string{
		"https://cansole.example":                          "https://cansole.example",
		" http://cansole.example/a ":                       "http://cansole.example/a",
		"cansole.example":                                  "<nil>",
		"https://":                                         "<nil>",
		"https://cansole.example/​":                        "<nil>",
		"https://" + strings.Repeat("a", maxWebsiteLength): "<nil>",
	} {
		if got := text(placeWebsite(in)); got != want {
			t.Errorf("placeWebsite(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{
		"+34 932 21 50 12;+34 600": "+34 932 21 50 12",
		"(030) 123-45/67":          "(030) 123-45/67",
		"12":                       "<nil>",
		"tel. 123456":              "<nil>",
	} {
		if got := text(placePhone(in)); got != want {
			t.Errorf("placePhone(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{
		"Mo-Fr 12:00-15:00; Sa 12:00-23:00":   "Mo-Fr 12:00-15:00\nSa 12:00-23:00",
		"24/7":                                "24/7",
		" ; ":                                 "<nil>",
		strings.Repeat("Mo 12:00-13:00;", 40): "<nil>",
	} {
		if got := text(placeOpeningHours(in)); got != want {
			t.Errorf("placeOpeningHours(%q) = %q, want %q", in, got, want)
		}
	}
}
