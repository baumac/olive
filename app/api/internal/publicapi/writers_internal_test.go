package publicapi

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/pocketbase/pocketbase/core"
)

const photoFilePath = basePath + "/restaurants/aaaaaaaaaaaaaaa/reviews/bbbbbbbbbbbbbbb/photos/ccccccccccccccc/thumb"

func serveWith(s *Server, pattern string, h http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	mux := http.NewServeMux()
	mux.HandleFunc(pattern, h)
	s.prepare(mux).ServeHTTP(w, r)
}

const photoFilePattern = "GET " + basePath + "/restaurants/{restaurantId}/reviews/{reviewId}/photos/{photoId}/{variant}"

// failingWriter counts the calls of the server. Each write fails like a
// closed connection.
type failingWriter struct {
	*httptest.ResponseRecorder
	headers, writes int
}

func (f *failingWriter) WriteHeader(status int) {
	f.headers++
	f.ResponseRecorder.WriteHeader(status)
}

func (f *failingWriter) Write([]byte) (int, error) {
	f.writes++
	return 0, errors.New("connection closed")
}

func TestPrepare(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)

	t.Run("an operation without spec gets 500", func(t *testing.T) {
		w := httptest.NewRecorder()
		serveWith(s, "GET "+basePath+"/other", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}, w, httptest.NewRequest(http.MethodGet, basePath+"/other", nil))
		if w.Code != http.StatusInternalServerError {
			t.Errorf("status %d, want 500", w.Code)
		}
	})

	t.Run("the headers of a file response are checked", func(t *testing.T) {
		w := httptest.NewRecorder()
		serveWith(s, photoFilePattern, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("not an image"))
		}, w, httptest.NewRequest(http.MethodGet, photoFilePath, nil))
		if w.Code != http.StatusInternalServerError || w.Body.String() == "not an image" {
			t.Errorf("status %d, body %q", w.Code, w.Body)
		}
	})

	// A file response goes to the client without a buffer. After a panic,
	// only a closed connection can end it.
	t.Run("a panic after a file write stops the connection", func(t *testing.T) {
		defer func() {
			if v := recover(); v != http.ErrAbortHandler {
				t.Errorf("panic %v, want http.ErrAbortHandler", v)
			}
		}()
		serveWith(s, photoFilePattern, func(w http.ResponseWriter, _ *http.Request) {
			writeProblem(w, newProblem(http.StatusNotFound, "The photo does not exist."))
			panic("boom")
		}, httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, photoFilePath, nil))
	})

	// The headers already went to the client. A 500 would be a second response.
	t.Run("a failed write gets no second response", func(t *testing.T) {
		w := &failingWriter{ResponseRecorder: httptest.NewRecorder()}
		serveWith(s, "GET "+basePath+"/restaurants", func(w http.ResponseWriter, _ *http.Request) {
			writeProblem(w, newProblem(http.StatusBadRequest, "The limit is not valid."))
		}, w, httptest.NewRequest(http.MethodGet, basePath+"/restaurants", nil))
		if w.Code != http.StatusBadRequest || w.headers != 1 || w.writes != 1 {
			t.Errorf("status %d, %d headers, %d writes, want 400 with one header and one write", w.Code, w.headers, w.writes)
		}
	})
}

func TestHeaderCheckWriterDropsARefusedResponse(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	hw := &headerCheckWriter{ResponseWriter: rec, check: func(int, http.Header) error { return errors.New("no ETag") }}
	hw.Header().Set("X-Partial", "yes")
	if n, err := hw.Write([]byte("abc")); n != 3 || err != nil {
		t.Errorf("write %d, %v", n, err)
	}
	if rec.Body.Len() != 0 || rec.Header().Get("X-Partial") != "" || hw.err == nil {
		t.Errorf("body %q, header %v, error %v", rec.Body, rec.Header(), hw.err)
	}
}

func TestWritersWithoutStatus(t *testing.T) {
	t.Parallel()
	if got := newBufferedWriter(httptest.NewRecorder()).Status(); got != http.StatusOK {
		t.Errorf("buffered status %d", got)
	}
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec}
	_, _ = sw.Write([]byte("x"))
	if sw.status != http.StatusOK || sw.Unwrap() != rec {
		t.Errorf("status writer %d", sw.status)
	}
	tw := &trackingWriter{ResponseWriter: rec}
	if tw.Unwrap() != rec {
		t.Error("the tracking writer does not unwrap")
	}
	if stateFrom(context.Background()).user != nil {
		t.Error("a context without state has a user")
	}
}

func TestClientIPWithoutPocketBase(t *testing.T) {
	t.Parallel()
	for addr, want := range map[string]string{"192.0.2.1:1234": "192.0.2.1", "pipe": "pipe"} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = addr
		if got := clientIP(r); got != want {
			t.Errorf("%s: client IP %q, want %q", addr, got, want)
		}
	}
}

func TestParamErrorDetail(t *testing.T) {
	t.Parallel()
	for err, want := range map[error]string{
		&reviewsapi.TooManyValuesForParamError{ParamName: "limit"}: "The query parameter limit can occur only one time.",
		errors.New("other"): "The request has a parameter that is not valid.",
	} {
		if got := paramErrorDetail(err); got != want {
			t.Errorf("detail %q, want %q", got, want)
		}
	}
}

func TestSecureWithTLS(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.TLS = &tls.ConnectionState{}
	if !(&Server{}).secure(r) {
		t.Error("a TLS request is not secure")
	}
}

func newRecord(t *testing.T, app core.App, collection string, set func(r *core.Record)) *core.Record {
	t.Helper()
	c, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	set(r)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

// Without the PocketBase router, the server reads the bearer token itself.
func TestBearerTokenWithoutPocketBase(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	user := newRecord(t, s.app, "users", func(r *core.Record) {
		r.SetEmail("ana@example.com")
		r.SetPassword("correct-horse-battery")
		r.SetVerified(true)
		r.Set("name", "Ana")
		r.Set("approved", true)
	})
	root := newRecord(t, s.app, core.CollectionNameSuperusers, func(r *core.Record) {
		r.SetEmail("root@example.com")
		r.SetPassword("correct-horse-battery")
	})
	token := func(r *core.Record) string {
		tok, err := r.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	for header, want := range map[string]int{
		"Bearer " + token(user): http.StatusOK,
		"Bearer " + token(root): http.StatusUnauthorized,
		"Bearer not-a-token":    http.StatusUnauthorized,
		"Basic YTpi":            http.StatusUnauthorized,
	} {
		r := httptest.NewRequest(http.MethodGet, basePath+"/users/me", nil)
		r.Header.Set("Authorization", header)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		if w.Code != want {
			t.Errorf("%.12s: status %d, want %d: %s", header, w.Code, want, w.Body)
		}
	}
}
