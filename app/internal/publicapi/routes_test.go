package publicapi

import (
	"net/http"
	"testing"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
)

type patternRecorder struct {
	*http.ServeMux
	patterns []string
}

func (p *patternRecorder) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) {
	p.patterns = append(p.patterns, pattern)
	p.ServeMux.HandleFunc(pattern, h)
}

// A generated operation without a spec operation gets 500 from prepare.
func TestEveryOperationIsInTheSpec(t *testing.T) {
	spec, err := loadSpec()
	if err != nil {
		t.Fatal(err)
	}
	rec := &patternRecorder{ServeMux: http.NewServeMux()}
	reviewsapi.HandlerWithOptions(reviewsapi.NewStrictHandler(&Server{}, nil),
		reviewsapi.StdHTTPServerOptions{BaseURL: basePath, BaseRouter: rec})
	if len(rec.patterns) != len(spec.operations) {
		t.Errorf("%d generated operations, %d spec operations", len(rec.patterns), len(spec.operations))
	}
	for _, p := range rec.patterns {
		if _, ok := spec.operations[p]; !ok {
			t.Errorf("no spec operation for %q", p)
		}
	}
}

func TestAuthAndRateLimitComeFromTheSpec(t *testing.T) {
	spec, err := loadSpec()
	if err != nil {
		t.Fatal(err)
	}
	for pattern, want := range map[string]struct {
		auth    authMode
		limited bool
	}{
		"POST " + basePath + "/users":                        {authIgnored, true},
		"POST " + basePath + "/users/tokens":                 {authIgnored, true},
		"POST " + basePath + "/users/tokens/refresh":         {authRequired, false},
		"GET " + basePath + "/restaurants":                   {authOptional, false},
		"DELETE " + basePath + "/restaurants/{restaurantId}": {authRequired, false},
	} {
		op := spec.operations[pattern]
		if op == nil || op.auth != want.auth || op.limited != want.limited {
			t.Errorf("%s: %+v, want %+v", pattern, op, want)
		}
	}
}
