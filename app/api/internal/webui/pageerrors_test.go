package webui

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/baumac/reviews-app/app/api/internal/testapp"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(testapp.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	app.Settings().Meta.AppURL = "https://reviews.example.com/"
	return app
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestPagesOfUsersWithoutApproval(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	pending := save(t, app, "users", map[string]any{"email": "bo@example.com", "password": "correct-horse", "name": "Bo"})
	list := save(t, app, "lists", map[string]any{"creator": pending.Id, "name": "Date night", "kind": "custom"})
	h := newHandler(t, app, fstest.MapFS{"index.html": {Data: []byte(testIndex)}})

	for _, path := range []string{"/reviewers/" + pending.Id, "/lists/" + list.Id} {
		if w := get(h, path); w.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", path, w.Code)
		}
	}
}

func TestReviewPhotoWithoutAlt(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	user := save(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana", "approved": true})
	restaurant := save(t, app, "restaurants", map[string]any{"name": "Luigi's", "creator": user.Id})
	review := save(t, app, "reviews", map[string]any{
		"restaurant": restaurant.Id, "creator": user.Id, "date": "2024-11-06", "title": "Good",
		"space": 7, "service": 7, "food": 7, "price": 7,
	})
	photo := savePhoto(t, app, "review_photos", review.Id)
	photo.Set("alt", "")
	if err := app.SaveNoValidate(photo); err != nil {
		t.Fatal(err)
	}
	h := newHandler(t, app, fstest.MapFS{"index.html": {Data: []byte(testIndex)}})

	w := get(h, "/restaurants/"+restaurant.Id+"/reviews/"+review.Id)
	if want := `<meta property="og:image:alt" content="A photo of the review Good" />`; w.Code != http.StatusOK || !strings.Contains(w.Body.String(), want) {
		t.Errorf("status %d, want %s in\n%s", w.Code, want, w.Body)
	}
}

// A database error gives the default page. The UI then gets the data from
// the Public API.
func TestPageMetaDatabaseError(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	user := save(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana", "approved": true})
	restaurant := save(t, app, "restaurants", map[string]any{"name": "Luigi's", "creator": user.Id})
	list := save(t, app, "lists", map[string]any{"creator": user.Id, "name": "Date night", "kind": "custom"})
	h := newHandler(t, app, fstest.MapFS{"index.html": {Data: []byte(testIndex)}})
	for _, table := range []string{"list_items", "reviews"} {
		if _, err := app.DB().NewQuery("DROP TABLE " + table).Execute(); err != nil {
			t.Fatal(err)
		}
	}

	for _, path := range []string{"/lists/" + list.Id, "/restaurants/" + restaurant.Id + "/reviews/aaaaaaaaaaaaaaa"} {
		w := get(h, path)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "<title>Sponga47 Reviews</title>") {
			t.Errorf("%s: status %d, want the default page:\n%s", path, w.Code, w.Body)
		}
	}
}

func TestReadIndexWithoutTitleOrFile(t *testing.T) {
	t.Parallel()
	noTitle := `<head><!-- page-meta --><meta property="og:image" content="https://x.example/a.png" /><!-- /page-meta --></head>`
	for name, files := range map[string]fstest.MapFS{
		"no title": {"index.html": {Data: []byte(noTitle)}},
		"no file":  {},
	} {
		if _, ok := readIndex(files); ok {
			t.Errorf("%s: the index has a page-meta block", name)
		}
	}
}

// TestWarningOfBuildWithoutMetaBlock has no t.Parallel. It changes the
// package variable warnNoMetaBlock.
func TestWarningOfBuildWithoutMetaBlock(t *testing.T) {
	warned := 0
	defer func(warn func(core.App)) { warnNoMetaBlock = warn }(warnNoMetaBlock)
	warnNoMetaBlock = func(core.App) { warned++ }

	for _, tc := range []struct {
		name  string
		files fs.FS
		want  int
	}{
		{"a build without the block", noDotFiles{placeholder}, 1},
		{"a build with the block", noDotFiles{fstest.MapFS{"index.html": {Data: []byte(testIndex)}}}, 0},
		{"the placeholder", placeholder, 0},
	} {
		warned = 0
		h := newHandler(t, newTestApp(t), tc.files)
		if warned != tc.want {
			t.Errorf("%s: %d warnings, want %d", tc.name, warned, tc.want)
		}
		if w := get(h, "/"); w.Code != http.StatusOK {
			t.Errorf("%s: status %d", tc.name, w.Code)
		}
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func TestBrokenReportGetsBadRequest(t *testing.T) {
	t.Parallel()
	h := newHandler(t, newTestApp(t), placeholder)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, reportPath, failingReader{}))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", w.Code)
	}
}
