package webui

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	robotsPath  = "robots.txt"
	sitemapPath = "sitemap.xml"
)

// maxSitemapURLs is the limit of the sitemap protocol for one file.
const maxSitemapURLs = 50000

// crawlerCacheControl lets a cache keep robots.txt and the sitemap for one
// hour. The sitemap reads all restaurants and reviews.
const crawlerCacheControl = "public, max-age=3600"

// robotsTxt keeps crawlers out of the dashboard, the PocketBase API, and
// the screens that need an account. The photo files under /v1 stay open
// for link previews. It has no Sitemap line without a baseURL.
func robotsTxt(baseURL string) string {
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	for _, path := range []string{"/_/", "/api/", "/sign-in", "/sign-up", "/my-reviews", "/account", "/confirm-email", "/forgot-password", "/reset-password"} {
		b.WriteString("Disallow: " + path + "\n")
	}
	if baseURL != "" {
		b.WriteString("\nSitemap: " + baseURL + "/" + sitemapPath + "\n")
	}
	return b.String()
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemapFile struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// sitemap lists the restaurants list, the restaurants, the reviews, and the
// lists that have a restaurant and an approved creator.
// The records with the newest change come first. It stops at
// maxSitemapURLs.
func sitemap(app core.App, baseURL string) ([]byte, error) {
	file := sitemapFile{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: []sitemapURL{{Loc: baseURL + "/restaurants"}}}
	var rows []struct {
		Kind    string         `db:"kind"`
		ID      string         `db:"id"`
		Parent  string         `db:"parent"`
		Updated types.DateTime `db:"updated"`
	}
	err := app.DB().
		NewQuery("SELECT 'restaurant' AS [[kind]], [[id]], '' AS [[parent]], [[updated]] FROM {{restaurants}} " +
			"UNION ALL SELECT 'review', [[id]], [[restaurant]], [[updated]] FROM {{reviews}} " +
			"UNION ALL SELECT 'list', [[id]], '', [[updated]] FROM {{lists}} " +
			"WHERE EXISTS (SELECT 1 FROM {{list_items}} WHERE [[list_items.list]] = [[lists.id]]) " +
			"AND [[creator]] IN (SELECT [[id]] FROM {{users}} WHERE [[approved]] = TRUE) " +
			"ORDER BY [[updated]] DESC, [[id]] LIMIT {:limit}").
		Bind(map[string]any{"limit": maxSitemapURLs - len(file.URLs)}).
		All(&rows)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		path := "/restaurants/" + row.ID
		switch row.Kind {
		case "review":
			path = "/restaurants/" + row.Parent + "/reviews/" + row.ID
		case "list":
			path = "/lists/" + row.ID
		}
		u := sitemapURL{Loc: baseURL + path}
		if !row.Updated.IsZero() {
			u.LastMod = row.Updated.Time().UTC().Format(time.RFC3339)
		}
		file.URLs = append(file.URLs, u)
	}
	var b bytes.Buffer
	b.WriteString(xml.Header)
	if err := xml.NewEncoder(&b).Encode(file); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// serveCrawlerFile returns false for a path that is not robots.txt or the
// sitemap. The sitemap needs a public Application URL. Its URLs must be
// absolute.
func serveCrawlerFile(e *core.RequestEvent, path string) (bool, error) {
	if path != robotsPath && path != sitemapPath {
		return false, nil
	}
	baseURL := publicBaseURL(e.App.Settings().Meta.AppURL)
	if path == robotsPath {
		e.Response.Header().Set("Cache-Control", crawlerCacheControl)
		return true, e.String(http.StatusOK, robotsTxt(baseURL))
	}
	if baseURL == "" {
		return true, e.NotFoundError("", nil)
	}
	data, err := sitemap(e.App, baseURL)
	if err != nil {
		return true, e.InternalServerError("", err)
	}
	e.Response.Header().Set("Cache-Control", crawlerCacheControl)
	return true, e.Blob(http.StatusOK, "application/xml; charset=utf-8", data)
}
