package publicapi

import (
	"net/http"
	"testing"

	reviewsapi "github.com/baumac/reviews-app/sdk/go"
)

type patternRecorder struct {
	*http.ServeMux
	patterns []string
}

func (p *patternRecorder) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) {
	p.patterns = append(p.patterns, pattern)
	p.ServeMux.HandleFunc(pattern, h)
}

// A generated operation without a routes entry gets 500 from prepare.
func TestEveryOperationHasARoute(t *testing.T) {
	rec := &patternRecorder{ServeMux: http.NewServeMux()}
	reviewsapi.HandlerWithOptions(reviewsapi.NewStrictHandler(&Server{}, nil),
		reviewsapi.StdHTTPServerOptions{BaseURL: basePath, BaseRouter: rec})
	if len(rec.patterns) != len(routes) {
		t.Errorf("%d generated operations, %d routes entries", len(rec.patterns), len(routes))
	}
	for _, p := range rec.patterns {
		if _, ok := routes[p]; !ok {
			t.Errorf("no routes entry for %q", p)
		}
	}
}
