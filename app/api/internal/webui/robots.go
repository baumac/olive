package webui

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

const robotsPath = "robots.txt"

const robotsCacheControl = "public, max-age=3600"

// robotsTxt keeps all crawlers out of the site. The pages are for the
// people who get a link. See decision 00012.
const robotsTxt = `User-agent: *
Disallow: /
`

func serveRobots(e *core.RequestEvent) error {
	e.Response.Header().Set("Cache-Control", robotsCacheControl)
	return e.String(http.StatusOK, robotsTxt)
}
