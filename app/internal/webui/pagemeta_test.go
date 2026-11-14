package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	_ "github.com/baumac/reviews-app/app/migrations"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const testIndex = `<!doctype html><html><head><meta property="og:site_name" content="Sponga47 Reviews" />
<!-- page-meta --><title>Sponga47 Reviews</title><meta name="description" content="Default" />
<meta property="og:image" content="https://site.example.com/og-image.png" /><meta property="og:image:width" content="1200" />
<meta property="og:image:height" content="630" /><meta property="og:image:alt" content="Logo &amp; tagline" /><!-- /page-meta -->
</head><body><div id="app"></div></body></html>`

func TestIndexHasMetaBlock(t *testing.T) {
	data, err := os.ReadFile("../../ui/index.html")
	if err != nil {
		t.Fatal(err)
	}
	index, ok := readIndex(fstest.MapFS{"index.html": {Data: data}})
	if !ok {
		t.Fatal("app/ui/index.html has no page-meta block")
	}
	block := strings.TrimPrefix(string(data), string(index.before))
	block = strings.TrimSuffix(block, string(index.after))
	for _, tag := range []string{"<title>", `name="description"`, `property="og:title"`, `property="og:image"`, `name="twitter:card"`} {
		if !strings.Contains(block, tag) {
			t.Errorf("the page-meta block has no %s", tag)
		}
	}
	if index.siteName != "Sponga47 Reviews" || index.siteImage.ImageWidth != 1200 || index.siteImage.ImageHeight != 630 ||
		!strings.HasSuffix(index.siteImage.Image, "/og-image.png") || index.siteImage.ImageAlt == "" {
		t.Errorf("the default tags are %q %+v", index.siteName, index.siteImage)
	}
	page, err := index.render(pageMeta{Title: "T", URL: "https://x.example.com/"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{"<title>", `name="description"`, `property="og:title"`, `property="og:image"`, `name="twitter:card"`} {
		if n := strings.Count(string(page), tag); n != 1 {
			t.Errorf("the rendered page has %s %d times", tag, n)
		}
	}
}

func TestReadIndexWithoutMetaBlock(t *testing.T) {
	if _, ok := readIndex(placeholder); ok {
		t.Error("the placeholder has a page-meta block")
	}
	noImage := "<head><!-- page-meta --><title>T</title><!-- /page-meta --></head>"
	if _, ok := readIndex(fstest.MapFS{"index.html": {Data: []byte(noImage)}}); ok {
		t.Error("a block without og:image is used")
	}
}

func TestBuiltIndexHasMetaBlock(t *testing.T) {
	files := Files()
	if _, built := files.(noDotFiles); !built {
		t.Skip("the UI is not built")
	}
	index, ok := readIndex(files)
	if !ok || index.siteName == "" || index.siteImage.ImageWidth == 0 {
		t.Errorf("the UI build has no usable page-meta block: %v %+v", ok, index)
	}
}

func TestReadIndexAttributes(t *testing.T) {
	data := `<head><!-- page-meta --><TITLE> Site &amp; Co </TITLE>
<meta content='https://site.example.com/a.png' property=og:image>
<meta
  content="640" property="og:image:width" /><!-- /page-meta --></head>`
	index, ok := readIndex(fstest.MapFS{"index.html": {Data: []byte(data)}})
	if !ok || index.siteName != "Site & Co" || index.siteImage.Image != "https://site.example.com/a.png" || index.siteImage.ImageWidth != 640 {
		t.Errorf("%v %q %+v", ok, index.siteName, index.siteImage)
	}
}

func TestPublicBaseURL(t *testing.T) {
	for in, want := range map[string]string{
		"https://reviews.example.com/":           "https://reviews.example.com",
		"https://reviews.example.com":            "https://reviews.example.com",
		"http://203.0.113.7:8090":                "http://203.0.113.7:8090",
		"http://localhost:8090":                  "",
		"http://LOCALHOST:8080/":                 "",
		"http://app.localhost":                   "",
		"http://127.0.0.1:8090":                  "",
		"http://[::1]:8090":                      "",
		"http://0.0.0.0:8090":                    "",
		"http://192.168.1.10:8090":               "",
		"http://10.0.0.5":                        "",
		"http://[fd00::1]":                       "",
		"http://169.254.1.1":                     "",
		"https://user:pass@reviews.example.com/": "https://reviews.example.com",
		"https://reviews.example.com/?ref=x#top": "https://reviews.example.com",
		"https://example.com/reviews/":           "https://example.com/reviews",
		"javascript:alert(1)":                    "",
		"":                                       "",
		"reviews.example.com":                    "",
	} {
		if got := publicBaseURL(in); got != want {
			t.Errorf("publicBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestShorten(t *testing.T) {
	long := strings.Repeat("word ", 60)
	wordEndsAtCut := "abcde" + strings.Repeat(" abcd", 39)
	for in, want := range map[string]string{
		wordEndsAtCut + " tail":  wordEndsAtCut + "…",
		"":                       "",
		"  Good\n\n food.\t":     "Good food.",
		long:                     strings.TrimSpace(strings.Repeat("word ", 40)) + "…",
		strings.Repeat("é", 250): strings.Repeat("é", 200) + "…",
	} {
		if got := shorten(in); got != want {
			t.Errorf("shorten(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPageMeta(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://reviews.example.com/"

	user := save(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	restaurant := save(t, app, "restaurants", map[string]any{
		"name": `Luigi's <script>alert(1)</script> & "Co"`, "description": "Pasta\nand pizza.", "creator": user.Id,
		"address": "Carrer Nou 1, Barcelona", "latitude": 41.39, "longitude": 2.16,
	})
	plain := save(t, app, "restaurants", map[string]any{"name": "Plain", "creator": user.Id})
	noReviews := save(t, app, "restaurants", map[string]any{"name": "No Reviews", "creator": user.Id})
	scores := map[string]any{"space": 7, "service": 7, "food": 7, "price": 7, "cleanliness": 7}
	review := save(t, app, "reviews", with(scores, map[string]any{
		"restaurant": restaurant.Id, "creator": user.Id, "title": "Great <b>food</b>", "feedback": "The best pasta.",
		"cleanliness": 9,
	}))
	other := save(t, app, "reviews", with(scores, map[string]any{"restaurant": plain.Id, "creator": user.Id, "title": "Fine"}))
	photo := savePhoto(t, app, review.Id)

	h := newHandler(t, app, fstest.MapFS{"index.html": {Data: []byte(testIndex)}})
	restaurantPath := "/restaurants/" + restaurant.Id
	reviewPath := restaurantPath + "/reviews/" + review.Id
	otherPath := "/restaurants/" + plain.Id + "/reviews/" + other.Id

	for _, tc := range []struct {
		method, path string
		status       int
		want         []string
	}{
		{"GET", restaurantPath, 200, []string{
			`<title>Luigi&#39;s &lt;script&gt;alert(1)&lt;/script&gt; &amp; &#34;Co&#34; · Sponga47 Reviews</title>`,
			`<meta name="description" content="Pasta and pizza." />`,
			`<link rel="canonical" href="https://reviews.example.com` + restaurantPath + `" />`,
			`<meta property="og:url" content="https://reviews.example.com` + restaurantPath + `" />`,
			`<meta property="og:type" content="website" />`,
			`<meta property="og:image" content="https://site.example.com/og-image.png" />`,
			`<meta property="og:image:alt" content="Logo &amp; tagline" />`,
			`<meta property="og:site_name" content="Sponga47 Reviews" />`,
			`<script type="application/ld+json">{"@context":"https://schema.org","@type":"Restaurant",` +
				`"address":"Carrer Nou 1, Barcelona",` +
				`"aggregateRating":{"@type":"AggregateRating","bestRating":10,"ratingValue":7.4,"reviewCount":1,"worstRating":1},` +
				`"description":"Pasta and pizza.","geo":{"@type":"GeoCoordinates","latitude":41.39,"longitude":2.16},` +
				`"name":"Luigi's \u003cscript\u003ealert(1)\u003c/script\u003e \u0026 \"Co\"",` +
				`"url":"https://reviews.example.com` + restaurantPath + `"}</script>`,
		}},
		{"GET", restaurantPath + "/", 200, []string{`href="https://reviews.example.com` + restaurantPath + `"`}},
		{"HEAD", restaurantPath, 200, nil},
		{"GET", "/restaurants/" + plain.Id, 200, []string{
			`<meta name="description" content="Reviews of Plain on Sponga47 Reviews." />`,
		}},
		{"GET", reviewPath, 200, []string{
			`<title>Great &lt;b&gt;food&lt;/b&gt; · Sponga47 Reviews</title>`,
			`<meta property="og:title" content="Great &lt;b&gt;food&lt;/b&gt; · Luigi&#39;s`,
			`<meta name="twitter:description" content="The best pasta." />`,
			`<meta property="og:type" content="article" />`,
			`<meta property="og:image" content="https://reviews.example.com/v1` + reviewPath + `/photos/` + photo.Id + `/full" />`,
			`<meta property="og:image:width" content="640" />`,
			`<meta property="og:image:alt" content="A plate of pasta" />`,
			`<script type="application/ld+json">{"@context":"https://schema.org","@type":"Review",` +
				`"author":{"@type":"Person","name":"Ana","url":"https://reviews.example.com/reviewers/` + user.Id + `"},`,
			`"itemReviewed":{"@type":"Restaurant","address":"Carrer Nou 1, Barcelona","name":"Luigi's`,
			`"name":"Great \u003cb\u003efood\u003c/b\u003e","reviewBody":"The best pasta.",` +
				`"reviewRating":{"@type":"Rating","bestRating":10,"ratingValue":7.4,"worstRating":1},` +
				`"url":"https://reviews.example.com` + reviewPath + `"}</script>`,
		}},
		{"GET", otherPath, 200, []string{
			`<meta name="description" content="A review of Plain on Sponga47 Reviews." />`,
			`<meta property="og:image" content="https://site.example.com/og-image.png" />`,
		}},
		{"GET", "/Restaurants/" + plain.Id, 200, []string{`<title>Plain · Sponga47 Reviews</title>`}},
		{"GET", "/restaurants/" + noReviews.Id, 200, []string{
			`<script type="application/ld+json">{"@context":"https://schema.org","@type":"Restaurant","name":"No Reviews",` +
				`"url":"https://reviews.example.com/restaurants/` + noReviews.Id + `"}</script>`,
		}},
		{"GET", "/restaurants/aaaaaaaaaaaaaaa", 404, []string{`content="Default"`}},
		{"GET", restaurantPath + "/reviews/aaaaaaaaaaaaaaa", 404, []string{`content="Default"`}},
		{"GET", restaurantPath + "/reviews/" + other.Id, 404, []string{`content="Default"`}},
		{"GET", "/restaurants/new", 200, []string{`content="Default"`}},
		{"GET", restaurantPath + "/edit", 200, []string{`content="Default"`}},
		{"GET", reviewPath + "/edit", 200, []string{`content="Default"`}},
		{"GET", "/restaurants", 200, []string{`content="Default"`}},
		{"GET", "/reviewers/" + user.Id, 200, []string{
			`<title>Ana · Sponga47 Reviews</title>`,
			`<meta name="description" content="Reviews by Ana on Sponga47 Reviews." />`,
			`<meta property="og:type" content="profile" />`,
			`<meta property="og:url" content="https://reviews.example.com/reviewers/` + user.Id + `" />`,
			`<meta property="og:image" content="https://site.example.com/og-image.png" />`,
		}},
		{"GET", "/Reviewers/" + user.Id + "/", 200, []string{`<title>Ana · Sponga47 Reviews</title>`}},
		{"GET", "/reviewers/aaaaaaaaaaaaaaa", 404, []string{`content="Default"`}},
		{"GET", "/reviewers/BAD", 200, []string{`content="Default"`}},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		body := w.Body.String()
		if w.Code != tc.status {
			t.Errorf("%s %s: status %d, want %d", tc.method, tc.path, w.Code, tc.status)
		}
		if w.Header().Get("Content-Security-Policy") != contentSecurityPolicy("") || w.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("%s %s: headers %v", tc.method, tc.path, w.Header())
		}
		if !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
			t.Errorf("%s %s: Content-Type %q", tc.method, tc.path, w.Header().Get("Content-Type"))
		}
		for _, want := range tc.want {
			if !strings.Contains(body, want) {
				t.Errorf("%s %s: no %s in\n%s", tc.method, tc.path, want, body)
			}
		}
		if strings.Contains(body, "<script>") || strings.Contains(body, "<b>") {
			t.Errorf("%s %s: a value is not escaped:\n%s", tc.method, tc.path, body)
		}
		if strings.Contains(body, user.GetString("email")) {
			t.Errorf("%s %s: the page has the email address", tc.method, tc.path)
		}
	}

	app.Settings().Meta.AppURL = "http://localhost:8090"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", reviewPath, nil))
	body := w.Body.String()
	if w.Code != 200 || strings.Contains(body, "canonical") || strings.Contains(body, "og:url") || strings.Contains(body, "localhost") {
		t.Errorf("default Application URL: %d\n%s", w.Code, body)
	}
	if !strings.Contains(body, `<meta property="og:image" content="https://site.example.com/og-image.png" />`) ||
		!strings.Contains(body, `<title>Great &lt;b&gt;food&lt;/b&gt; · Sponga47 Reviews</title>`) {
		t.Errorf("default Application URL: no site image or title\n%s", body)
	}
}

func newHandler(t *testing.T, app core.App, files fstest.MapFS) http.Handler {
	t.Helper()
	if err := register(app, files, Options{}); err != nil {
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
	return h
}

func save(t *testing.T, app core.App, collection string, fields map[string]any) *core.Record {
	t.Helper()
	c, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(fields)
	if collection == "users" {
		r.SetPassword(fields["password"].(string))
	}
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

// savePhoto skips the validation. The files are then not real JPEG files.
func savePhoto(t *testing.T, app core.App, reviewID string) *core.Record {
	t.Helper()
	c, err := app.FindCollectionByNameOrId("review_photos")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	for _, name := range []string{"full", "thumb"} {
		f, err := filesystem.NewFileFromBytes([]byte("jpeg"), name+".jpg")
		if err != nil {
			t.Fatal(err)
		}
		r.Set(name, f)
	}
	r.Load(map[string]any{"review": reviewID, "alt": "A plate of pasta", "width": 640, "height": 480})
	if err := app.SaveNoValidate(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func with(a, b map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func TestCrawlerFiles(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://reviews.example.com/"

	user := save(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	restaurant := save(t, app, "restaurants", map[string]any{"name": "Luigi & Co", "creator": user.Id})
	review := save(t, app, "reviews", map[string]any{
		"restaurant": restaurant.Id, "creator": user.Id, "title": "Fine",
		"space": 7, "service": 7, "food": 7, "price": 7, "cleanliness": 7,
	})
	h := newHandler(t, app, fstest.MapFS{"index.html": {Data: []byte(testIndex)}})
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}

	robots := get("/robots.txt")
	wantRobots := "User-agent: *\nDisallow: /_/\nDisallow: /api/\nDisallow: /sign-in\nDisallow: /sign-up\nDisallow: /my-reviews\n" +
		"\nSitemap: https://reviews.example.com/sitemap.xml\n"
	if robots.Code != 200 || robots.Body.String() != wantRobots || !strings.HasPrefix(robots.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("robots.txt: %d %q %q", robots.Code, robots.Header().Get("Content-Type"), robots.Body.String())
	}

	sitemap := get("/sitemap.xml")
	body := sitemap.Body.String()
	if sitemap.Code != 200 || sitemap.Header().Get("Content-Type") != "application/xml; charset=utf-8" ||
		sitemap.Header().Get("Cache-Control") != crawlerCacheControl {
		t.Errorf("sitemap.xml: %d %v", sitemap.Code, sitemap.Header())
	}
	for _, want := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`,
		`<url><loc>https://reviews.example.com/restaurants</loc></url>`,
		`<url><loc>https://reviews.example.com/restaurants/` + restaurant.Id + `</loc><lastmod>` +
			restaurant.GetDateTime("updated").Time().UTC().Format(time.RFC3339) + `</lastmod></url>`,
		`<loc>https://reviews.example.com/restaurants/` + restaurant.Id + `/reviews/` + review.Id + `</loc>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("sitemap.xml: no %s in\n%s", want, body)
		}
	}
	if strings.Count(body, "<url>") != 3 || strings.Contains(body, user.Id) {
		t.Errorf("sitemap.xml has other URLs:\n%s", body)
	}

	app.Settings().Meta.AppURL = "http://localhost:8090"
	if w := get("/sitemap.xml"); w.Code != 404 {
		t.Errorf("sitemap.xml without a public URL: %d", w.Code)
	}
	if w := get("/robots.txt"); w.Code != 200 || strings.Contains(w.Body.String(), "Sitemap") {
		t.Errorf("robots.txt without a public URL: %d %q", w.Code, w.Body.String())
	}
}
