package publicapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/baumac/reviews-app/app/api/internal/testapp"
	"github.com/pocketbase/pocketbase/tests"
	"golang.org/x/crypto/bcrypt"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return newTestServerWith(t, Options{SignInLimit: 10, SignInWindow: time.Minute})
}

func newTestServerWith(t *testing.T, opts Options) *Server {
	t.Helper()
	app, err := tests.NewTestApp(testapp.LowCostDir(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	s, err := New(app, opts)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestDummyHashUsesThePasswordCost checks both costs. A sign in with an
// unknown email then takes as long as one with a wrong password.
func TestDummyHashUsesThePasswordCost(t *testing.T) {
	for _, c := range []struct {
		name string
		dir  string
		want int
	}{
		{"production schema", testapp.Dir(t), bcrypt.DefaultCost},
		{"low cost schema", testapp.LowCostDir(t), bcrypt.MinCost},
	} {
		t.Run(c.name, func(t *testing.T) {
			app, err := tests.NewTestApp(c.dir)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(app.Cleanup)
			s, err := New(app, Options{SignInLimit: 10, SignInWindow: time.Minute})
			if err != nil {
				t.Fatal(err)
			}
			if got, err := bcrypt.Cost(s.dummyHash); err != nil || got != c.want {
				t.Errorf("cost %d, %v, want %d", got, err, c.want)
			}
		})
	}
}

func serveProblem(t *testing.T, s *Server, h http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+basePath+"/restaurants", h)
	w := httptest.NewRecorder()
	s.prepare(mux).ServeHTTP(w, httptest.NewRequest("GET", basePath+"/restaurants", nil))
	var p struct{ Status int }
	if w.Code != http.StatusInternalServerError || w.Header().Get("Content-Type") != problemContentType {
		t.Fatalf("status %d, content type %q", w.Code, w.Header().Get("Content-Type"))
	}
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil || p.Status != 500 {
		t.Errorf("body %s", w.Body)
	}
	return w
}

func TestPanicGetsProblemDetails(t *testing.T) {
	serveProblem(t, newTestServer(t), func(http.ResponseWriter, *http.Request) { panic("boom") })
}

// The buffered response has not reached the client at the panic. The
// client then gets only the 500 response.
func TestPanicAfterWriteGetsOnlyProblemDetails(t *testing.T) {
	w := serveProblem(t, newTestServer(t), func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Partial", "yes")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[`))
		panic("boom")
	})
	if w.Header().Get("X-Partial") != "" {
		t.Error("a header of the partial response reached the client")
	}
}

func TestInvalidResponseGetsProblemDetails(t *testing.T) {
	w := serveProblem(t, newTestServer(t), func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":"not an array"}`))
	})
	if w.Header().Get("Content-Type") != problemContentType {
		t.Errorf("content type %q", w.Header().Get("Content-Type"))
	}
}

func TestBufferedWriterWritesOnlyOnFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Vary", "Origin")
	b := newBufferedWriter(rec)
	b.Header().Set("Content-Type", "application/json")
	b.WriteHeader(http.StatusCreated)
	b.WriteHeader(http.StatusTeapot)
	_, _ = b.Write([]byte(`{}`))
	if rec.Body.Len() != 0 || rec.Header().Get("Content-Type") != "" {
		t.Fatal("the writer wrote before Flush")
	}
	if err := b.Flush(); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated || rec.Body.String() != `{}` {
		t.Errorf("status %d, body %q", rec.Code, rec.Body)
	}
	if rec.Header().Get("Content-Type") != "application/json" || rec.Header().Get("Vary") != "Origin" {
		t.Errorf("headers %v", rec.Header())
	}
}

func TestSkipResponseValidation(t *testing.T) {
	s := newTestServerWith(t, Options{SignInLimit: 10, SignInWindow: time.Minute, SkipResponseValidation: true})
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+basePath+"/restaurants", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":"not an array"}`))
	})
	w := httptest.NewRecorder()
	s.prepare(mux).ServeHTTP(w, httptest.NewRequest("GET", basePath+"/restaurants", nil))
	if w.Code != http.StatusOK || w.Body.String() != `{"items":"not an array"}` {
		t.Errorf("status %d, body %q", w.Code, w.Body)
	}
}

func TestResponseProblemLeavesOutValues(t *testing.T) {
	s := newTestServer(t)
	op := s.spec.operations["GET "+basePath+"/restaurants"]
	in, err := op.input(httptest.NewRequest("GET", basePath+"/restaurants", nil))
	if err != nil {
		t.Fatal(err)
	}
	header := http.Header{"Content-Type": {"application/json"}}
	err = checkResponse(t.Context(), in, http.StatusOK, header, []byte(`{"items":"secret-token-value"}`))
	if err == nil {
		t.Fatal("the invalid response passed")
	}
	got := responseProblem(err)
	if strings.Contains(got, "secret-token-value") || !strings.Contains(got, "/items") {
		t.Errorf("problem %q", got)
	}
}

func TestResponseAcceptsEachStoredEmail(t *testing.T) {
	s := newTestServer(t)
	op := s.spec.operations["POST "+basePath+"/users/tokens"]
	in, err := op.input(httptest.NewRequest("POST", basePath+"/users/tokens", nil))
	if err != nil {
		t.Fatal(err)
	}
	body := `{"token":"t","expires":"2026-11-09T18:36:48.504Z","user":{"id":"k3x9m2p7q1w8z4a",` +
		`"email":"Ana <ana@example.com>","name":"Ana","approved":true,"created":"2026-11-09T18:36:48.504Z","updated":"2026-11-09T18:36:48.504Z"}}`
	header := http.Header{"Content-Type": {"application/json"}}
	if err := checkResponse(t.Context(), in, http.StatusOK, header, []byte(body)); err != nil {
		t.Errorf("the stored email failed the response check: %s", responseProblem(err))
	}
}
