package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestCacheControl(t *testing.T) {
	files := fstest.MapFS{"index.html": {}, "assets/app-1a2b.js": {}}
	for path, want := range map[string]string{
		"":                   "no-cache",
		"restaurants/abc":    "no-cache",
		"assets/app-1a2b.js": "public, max-age=31536000, immutable",
		"assets/missing.js":  "no-cache",
	} {
		if got := cacheControl(files, path); got != want {
			t.Errorf("%q: %q, want %q", path, got, want)
		}
	}
}

func TestNoDotFiles(t *testing.T) {
	files := noDotFiles{fstest.MapFS{"index.html": {}, ".gitkeep": {}, "assets/.env": {}}}
	for _, name := range []string{".gitkeep", "assets/.env"} {
		if _, err := files.Open(name); err == nil {
			t.Errorf("%s is served", name)
		}
	}
	if _, err := files.Open("index.html"); err != nil {
		t.Error(err)
	}
}

func TestRoutes(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	Register(app)
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.Any("/v1/{path...}", func(e *core.RequestEvent) error {
			return e.String(http.StatusTeapot, "api")
		})
		return se.Next()
	})
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.OnServe().Trigger(&core.ServeEvent{App: app, Router: router}, func(*core.ServeEvent) error { return nil }); err != nil {
		t.Fatal(err)
	}
	h, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		method, path string
		status       int
		body         string
	}{
		{"GET", "/", 200, "<html"},
		{"GET", "/restaurants/k3x9m2p7q1w8z4a", 200, "<html"},
		{"GET", "/v1/restaurants", http.StatusTeapot, "api"},
		{"POST", "/restaurants", http.StatusMethodNotAllowed, ""},
		{"POST", "/csp-report", http.StatusNoContent, ""},
		{"GET", "/api/does-not-exist", http.StatusNotFound, `"status":404`},
		{"POST", "/_/unknown", http.StatusNotFound, `"status":404`},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.body) {
			t.Errorf("%s %s: %d %q", tc.method, tc.path, w.Code, w.Body.String())
		}
		if tc.status == 200 && w.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("%s: Cache-Control %q", tc.path, w.Header().Get("Cache-Control"))
		}
		if tc.status == 200 && w.Header().Get("Reporting-Endpoints") != `csp="/csp-report"` {
			t.Errorf("%s: Reporting-Endpoints %q", tc.path, w.Header().Get("Reporting-Endpoints"))
		}
		if tc.status == 200 && w.Header().Get("Content-Security-Policy") != contentSecurityPolicy {
			t.Errorf("%s: Content-Security-Policy %q", tc.path, w.Header().Get("Content-Security-Policy"))
		}
		if strings.HasPrefix(tc.path, "/v1") && w.Header().Get("Content-Security-Policy") != "" {
			t.Errorf("%s: the API has the UI policy", tc.path)
		}
	}
}
