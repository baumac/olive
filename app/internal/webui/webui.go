// Package webui serves the build of the Public UI from the PocketBase app.
// See decision 00005.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
	"testing/fstest"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// dist has only .gitkeep in git. make build-ui writes the UI build into it.
//
//go:embed all:dist
var dist embed.FS

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
func Register(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		files := Files()
		static := apis.Static(files, true)
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
			return static(e)
		}).Bind(apis.Gzip())
		return se.Next()
	})
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
