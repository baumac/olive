package publicapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func TestPanicGetsProblemDetails(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	s := &Server{app: app}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+basePath+"/restaurants", func(http.ResponseWriter, *http.Request) { panic("boom") })

	w := httptest.NewRecorder()
	s.prepare(mux).ServeHTTP(w, httptest.NewRequest("GET", basePath+"/restaurants", nil))

	var p struct{ Status int }
	if w.Code != http.StatusInternalServerError || w.Header().Get("Content-Type") != problemContentType {
		t.Fatalf("status %d, content type %q", w.Code, w.Header().Get("Content-Type"))
	}
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil || p.Status != 500 {
		t.Errorf("body %s", w.Body)
	}
}

func TestPanicAfterWriteDoesNotWriteAgain(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	s := &Server{app: app}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+basePath+"/restaurants", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		panic("boom")
	})

	w := httptest.NewRecorder()
	defer func() {
		if v := recover(); v != http.ErrAbortHandler {
			t.Errorf("recover %v, want http.ErrAbortHandler", v)
		}
		if w.Code != http.StatusOK || w.Body.Len() != 0 {
			t.Errorf("status %d, body %q", w.Code, w.Body)
		}
	}()
	s.prepare(mux).ServeHTTP(w, httptest.NewRequest("GET", basePath+"/restaurants", nil))
}
