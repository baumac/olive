package publicapi

import (
	"net/http"
	"reflect"
	"strings"
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
	photos := basePath + "/restaurants/{restaurantId}/reviews/{reviewId}/photos"
	for pattern, want := range map[string]struct {
		auth  authMode
		limit *rateLimit
	}{
		"POST " + basePath + "/users":                        {authIgnored, &rateLimit{"ip", "sign-in"}},
		"POST " + basePath + "/users/tokens":                 {authIgnored, &rateLimit{"ip", "sign-in"}},
		"POST " + basePath + "/users/tokens/refresh":         {authRequired, nil},
		"GET " + basePath + "/restaurants":                   {authOptional, nil},
		"POST " + basePath + "/restaurants":                  {authRequired, &rateLimit{"user", "create"}},
		"DELETE " + basePath + "/restaurants/{restaurantId}": {authRequired, &rateLimit{"user", "edit"}},
		"POST " + photos:                                     {authRequired, &rateLimit{"user", "photo-upload"}},
		"GET " + photos + "/{photoId}/{variant}":             {authOptional, nil},
	} {
		op := spec.operations[pattern]
		if op == nil || op.auth != want.auth || !reflect.DeepEqual(op.limit, want.limit) {
			t.Errorf("%s: %+v, want %+v", pattern, op, want)
		}
	}
}

// TestWriteOperationsHaveARateLimit finds a new write operation without
// x-rate-limit. The operations with x-allow-unapproved change no data.
func TestWriteOperationsHaveARateLimit(t *testing.T) {
	spec, err := loadSpec()
	if err != nil {
		t.Fatal(err)
	}
	for pattern, op := range spec.operations {
		write := !strings.HasPrefix(pattern, "GET ") && !strings.HasPrefix(pattern, "HEAD ")
		if write && op.approval && op.limit == nil {
			t.Errorf("%s has no x-rate-limit", pattern)
		}
	}
}

func TestFileOperationsComeFromTheSpec(t *testing.T) {
	spec, err := loadSpec()
	if err != nil {
		t.Fatal(err)
	}
	for pattern, op := range spec.operations {
		upload := strings.HasSuffix(pattern, "/photos") && strings.HasPrefix(pattern, "POST ")
		file := strings.HasSuffix(pattern, "/{variant}")
		if op.fileBody != upload || op.fileResponse != file {
			t.Errorf("%s: fileBody %v, fileResponse %v", pattern, op.fileBody, op.fileResponse)
		}
	}
}
