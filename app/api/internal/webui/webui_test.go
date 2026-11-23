package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/baumac/reviews-app/app/api/internal/testapp"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestCacheControl(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

// TestRegister uses the embedded build or the placeholder.
func TestRegister(t *testing.T) {
	t.Parallel()
	app, err := tests.NewTestApp(testapp.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	if err := Register(app, Options{TileURL: DefaultTileURL}); err != nil {
		t.Fatal(err)
	}
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
	for _, path := range []string{"/", "/.gitkeep"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "<title>Sponga47 Food</title>") {
			t.Errorf("%s: %d %q", path, w.Code, w.Body.String())
		}
	}
}

func TestWatchAppURL(t *testing.T) {
	t.Parallel()
	app, err := tests.NewTestApp(testapp.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	var warned []string
	watchAppURL(app, func(_ core.App, appURL string) { warned = append(warned, appURL) })
	if err := app.ReloadSettings(); err != nil {
		t.Fatal(err)
	}
	if len(warned) != 0 {
		t.Fatalf("a reload before serve warns: %v", warned)
	}
	if err := app.OnServe().Trigger(&core.ServeEvent{App: app}, func(*core.ServeEvent) error { return nil }); err != nil {
		t.Fatal(err)
	}
	for _, appURL := range []string{"http://localhost:8090", "https://reviews.example.com", "http://127.0.0.1:8090"} {
		app.Settings().Meta.AppURL = appURL
		if err := app.Save(app.Settings()); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"http://localhost:8090", "http://127.0.0.1:8090"}
	if strings.Join(warned, " ") != strings.Join(want, " ") {
		t.Errorf("warnings for %v, want %v", warned, want)
	}
}

func TestRoutes(t *testing.T) {
	t.Parallel()
	app, err := tests.NewTestApp(testapp.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	if err := register(app, fstest.MapFS{"index.html": {Data: []byte(testIndex)}}, Options{TileURL: "https://tiles.example.com/{z}/{x}/{y}{r}.png?key=k", TileAttribution: "© Example"}); err != nil {
		t.Fatal(err)
	}
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
		{"GET", "/restaurants/new", 200, "<html"},
		{"GET", "/map-config.json", 200, `{"attribution":"© Example","tiles":"https://tiles.example.com/{z}/{x}/{y}{r}.png?key=k"}`},
		{"GET", "/restaurants/k3x9m2p7q1w8z4a", http.StatusNotFound, "<html"},
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
		if csp := w.Header().Get("Content-Security-Policy"); tc.status == 200 && !strings.Contains(csp, "img-src 'self' blob: https://tiles.example.com;") {
			t.Errorf("%s: Content-Security-Policy %q", tc.path, csp)
		}
		if tc.status == 200 && w.Header().Get("Reporting-Endpoints") != `csp="/csp-report"` {
			t.Errorf("%s: Reporting-Endpoints %q", tc.path, w.Header().Get("Reporting-Endpoints"))
		}
		if tc.status == 200 && w.Header().Get("Content-Security-Policy") != contentSecurityPolicy("https://tiles.example.com") {
			t.Errorf("%s: Content-Security-Policy %q", tc.path, w.Header().Get("Content-Security-Policy"))
		}
		// The tile server of the maps needs the Referer header. See decision
		// 00008.
		if got := w.Header().Get("Referrer-Policy"); strings.Contains(got, "no-referrer") && got != "no-referrer-when-downgrade" {
			t.Errorf("%s: Referrer-Policy %q", tc.path, got)
		}
		if strings.HasPrefix(tc.path, "/v1") && w.Header().Get("Content-Security-Policy") != "" {
			t.Errorf("%s: the API has the UI policy", tc.path)
		}
	}
}

func TestTileOrigin(t *testing.T) {
	t.Parallel()
	for tileURL, want := range map[string]string{
		DefaultTileURL: "https://tile.openstreetmap.org",
		"http://127.0.0.1:8096/tiles/{z}/{x}/{y}.png":           "http://127.0.0.1:8096",
		"https://tiles.example.com/{z}/{x}/{y}{r}.png?key=a1b2": "https://tiles.example.com",
	} {
		if got, err := TileOrigin(tileURL); err != nil || got != want {
			t.Errorf("%s: %q, %v, want %q", tileURL, got, err, want)
		}
	}
	for _, tileURL := range []string{
		"", "https://tiles.example.com/", "https://tiles.example.com/{z}/{x}.png", "tiles.example.com/{z}/{x}/{y}.png",
		"ftp://tiles.example.com/{z}/{x}/{y}.png", "https://{s}.tiles.example.com/{z}/{x}/{y}.png",
		"https://tiles.example.com;script-src */{z}/{x}/{y}.png", "https://a b/{z}/{x}/{y}.png",
		"https://tiles.example.com;x/{z}/{x}/{y}.png",
		"https://{z}.tiles.example.com/{x}/{y}.png", "https://tiles{x}.example.com/{z}/{y}.png", "{z}://example.com/{x}/{y}.png",
	} {
		if got, err := TileOrigin(tileURL); err == nil {
			t.Errorf("%q: origin %q, want an error", tileURL, got)
		}
	}
	if got := contentSecurityPolicy(""); !strings.Contains(got, "img-src 'self' blob:; ") {
		t.Errorf("policy without tiles: %q", got)
	}
	app, err := tests.NewTestApp(testapp.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err := Register(app, Options{TileURL: "https://tiles.example.com/"}); err == nil {
		t.Error("Register accepts a tile URL without the parts")
	}
}
