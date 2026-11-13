package publicapi

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
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
