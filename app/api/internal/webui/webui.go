// Package webui serves the build of the Public UI from the PocketBase app.
// See decision 00005.
package webui

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing/fstest"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// dist has only .gitkeep in git. make build-ui writes the UI build into it.
//
//go:embed all:dist
var dist embed.FS

// contentSecurityPolicy allows the files of this origin and the tiles origin.
// Decisions 00005, 00008, and 00018 explain the directives.
func contentSecurityPolicy(tiles string) string {
	if tiles != "" {
		tiles = " " + tiles
	}
	// The hash allows the inline style of Reka UI SelectViewport. A Reka UI
	// update can change its text. The end-to-end tests then fail.
	return "default-src 'self'; script-src 'self'; " +
		"style-src 'self' 'sha256-60LHlRjW/B3CtzIoE/Lf1/NEDvko9efWMFaGVhHu/cs='; img-src 'self' blob:" + tiles + "; " +
		"font-src 'self'; connect-src 'self'; manifest-src 'self'; object-src 'none'; base-uri 'none'; " +
		"form-action 'self'; frame-ancestors 'none'; report-uri /csp-report; report-to csp"
}

// DefaultTileURL is the tile server of OpenStreetMap. Its usage policy has
// rules for the map. Decision 00018 lists them.
const DefaultTileURL = "https://tile.openstreetmap.org/{z}/{x}/{y}.png"

// mapConfigPath gives the UI the settings of the map.
const mapConfigPath = "map-config.json"

// Options are the settings of the UI.
type Options struct {
	// TileURL is the URL template of the map tiles with {z}, {x}, and {y}.
	// An empty value turns the maps off.
	TileURL string
	// TileAttribution is the credit of a tile server that is not from
	// OpenStreetMap. The map shows it before the OpenStreetMap credit.
	TileAttribution string
}

// tilePlaceholders are the parts of a tile URL that Leaflet replaces. {r}
// is the suffix for screens with a high resolution.
var tilePlaceholders = strings.NewReplacer("{z}", "0", "{x}", "0", "{y}", "0", "{r}", "")

// TileOrigin returns the origin of a tile URL template for the Content
// Security Policy. It returns an error for a URL that the map cannot use.
func TileOrigin(tileURL string) (string, error) {
	// A part in the host would give a different host for each tile.
	if _, rest, _ := strings.Cut(tileURL, "://"); strings.Contains(strings.SplitN(rest, "/", 2)[0], "{") {
		return "", errors.New("the tile URL must have a fixed host")
	}
	for _, part := range []string{"{z}", "{x}", "{y}"} {
		if !strings.Contains(tileURL, part) {
			return "", fmt.Errorf("the tile URL has no %s", part)
		}
	}
	u, err := url.Parse(tilePlaceholders.Replace(tileURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || strings.ContainsAny(u.String(), "{}") {
		return "", errors.New("the tile URL must be an http or https URL with only the {z}, {x}, {y}, and {r} parts")
	}
	// A space or a semicolon would add a source or a directive to the policy.
	if strings.ContainsAny(u.Host, " ;,'\"") {
		return "", errors.New("the tile URL has a host that is not valid")
	}
	return u.Scheme + "://" + u.Host, nil
}

// reportPath gets the violation reports of the browsers. The log then shows
// a violation that the tests did not find.
const reportPath = "/csp-report"

// maxReportBytes is far above one report. A larger body is cut.
const maxReportBytes = 16 << 10

var placeholder = fstest.MapFS{
	"index.html": {Data: []byte(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>Sponga47 Reviews</title></head>
<body><p>The UI is not built. Run <code>make build-ui</code>, then start the app again.</p></body></html>
`)},
}

// Files returns the UI build, or a page that says to build the UI.
func Files() fs.FS {
	files, err := fs.Sub(dist, "dist")
	if err != nil {
		return placeholder
	}
	if _, err := fs.Stat(files, "index.html"); err != nil {
		return placeholder
	}
	return noDotFiles{files}
}

// noDotFiles hides files and folders whose names start with a dot, such as
// .gitkeep. The embed needs all: for .gitkeep. all: also takes other
// dotfiles.
type noDotFiles struct {
	fs.FS
}

func (f noDotFiles) Open(name string) (fs.File, error) {
	for part := range strings.SplitSeq(name, "/") {
		if strings.HasPrefix(part, ".") && part != "." {
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
	}
	return f.FS.Open(name)
}

// Register serves the UI on all paths that no other route has. A path that
// is not a file gets index.html. The UI router then shows the screen.
func Register(app core.App, opts Options) error {
	return register(app, Files(), opts)
}

func register(app core.App, files fs.FS, opts Options) error {
	tileOrigin := ""
	if opts.TileURL != "" {
		var err error
		if tileOrigin, err = TileOrigin(opts.TileURL); err != nil {
			return err
		}
	}
	policy := contentSecurityPolicy(tileOrigin)
	mapConfig, err := json.Marshal(map[string]string{"tiles": opts.TileURL, "attribution": opts.TileAttribution})
	if err != nil {
		return err
	}
	index, hasMetaBlock := readIndex(files)
	if _, built := files.(noDotFiles); built && !hasMetaBlock {
		app.OnServe().BindFunc(func(se *core.ServeEvent) error {
			warnNoMetaBlock(se.App)
			return se.Next()
		})
	}
	if hasMetaBlock {
		watchAppURL(app, warnAppURL)
	}
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		static := apis.Static(files, true)
		se.Router.POST(reportPath, logReport)
		// The route has no method. A GET route would conflict with the /v1
		// route of the Public API, which has no method.
		se.Router.Any("/{path...}", func(e *core.RequestEvent) error {
			path := e.Request.PathValue(apis.StaticWildcardParam)
			if isPocketBasePath(path) {
				return e.NotFoundError("", nil)
			}
			if e.Request.Method != http.MethodGet && e.Request.Method != http.MethodHead {
				e.Response.Header().Set("Allow", "GET, HEAD")
				return e.NoContent(http.StatusMethodNotAllowed)
			}
			e.Response.Header().Set("Cache-Control", cacheControl(files, path))
			e.Response.Header().Set("Content-Security-Policy", policy)
			e.Response.Header().Set("Reporting-Endpoints", `csp="`+reportPath+`"`)
			if path == mapConfigPath {
				return e.Blob(http.StatusOK, "application/json", mapConfig)
			}
			if done, err := serveCrawlerFile(e, path); done {
				return err
			}
			if hasMetaBlock {
				if page, status, ok := renderPage(e, index, path); ok {
					return e.HTML(status, string(page))
				}
			}
			return static(e)
		}).Bind(apis.Gzip())
		return se.Next()
	})
	return nil
}

// watchAppURL warns when the app starts to serve with a private Application
// URL. After the start, it warns when a settings save changes the URL to
// such a URL. CLI commands do not serve, so they do not warn.
func watchAppURL(app core.App, warn func(core.App, string)) {
	var mu sync.Mutex
	serving, last := false, ""
	check := func(app core.App, force bool) {
		appURL := app.Settings().Meta.AppURL
		mu.Lock()
		changed := appURL != last
		last = appURL
		mu.Unlock()
		if (force || changed) && publicBaseURL(appURL) == "" {
			warn(app, appURL)
		}
	}
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		mu.Lock()
		serving = true
		mu.Unlock()
		check(se.App, true)
		return se.Next()
	})
	app.OnSettingsReload().BindFunc(func(e *core.SettingsReloadEvent) error {
		mu.Lock()
		started := serving
		mu.Unlock()
		if started {
			check(e.App, false)
		}
		return e.Next()
	})
}

func warnAppURL(app core.App, appURL string) {
	app.Logger().Warn("The Application URL is not public. Pages get no canonical URL. See decision 00013.", "appURL", appURL)
}

// warnNoMetaBlock is a variable for the tests. PocketBase writes its log
// later. A test cannot wait for it.
var warnNoMetaBlock = func(app core.App) {
	app.Logger().Warn("index.html of the UI build has no page-meta block. Pages get no link preview. See decision 00013.")
}

// renderPage returns false for a path without its own page meta. A missing
// record gets the default page with 404. The UI then shows its not found
// screen. On a database error, the app logs it and sends the default page.
func renderPage(e *core.RequestEvent, index indexPage, path string) ([]byte, int, bool) {
	meta, ok, err := index.metaFor(e.App, e.App.Settings().Meta.AppURL, path)
	if !ok {
		return nil, 0, false
	}
	if errors.Is(err, errNoRecord) {
		return index.whole, http.StatusNotFound, true
	}
	var page []byte
	if err == nil {
		page, err = index.render(meta)
	}
	if err != nil {
		e.App.Logger().Error("Cannot make the page meta", "path", path, "error", err)
		return nil, 0, false
	}
	return page, http.StatusOK, true
}

// cacheControl lets browsers keep the hashed files in assets/ for a year.
// Other files get a new check on each load. A new deploy then shows at once.
func cacheControl(files fs.FS, path string) string {
	if strings.HasPrefix(path, "assets/") {
		if _, err := fs.Stat(files, path); err == nil {
			return "public, max-age=31536000, immutable"
		}
	}
	return "no-cache"
}

// isPocketBasePath is true for the paths of the PocketBase API and
// dashboard. An unknown path there keeps the JSON 404 of PocketBase.
func isPocketBasePath(path string) bool {
	for _, prefix := range []string{"api", "_"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

// logReport logs a CSP violation report. Anybody can send one. The log
// keeps at most maxReportBytes of it.
func logReport(e *core.RequestEvent) error {
	body, err := io.ReadAll(io.LimitReader(e.Request.Body, maxReportBytes))
	if err != nil {
		return e.BadRequestError("", nil)
	}
	e.App.Logger().Warn("Content Security Policy violation",
		"contentType", e.Request.Header.Get("Content-Type"),
		"report", string(body),
		"userAgent", e.Request.UserAgent(),
	)
	return e.NoContent(http.StatusNoContent)
}
