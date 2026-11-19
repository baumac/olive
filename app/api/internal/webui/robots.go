package webui

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

const robotsPath = "robots.txt"

const robotsCacheControl = "public, max-age=3600"

// robotsTxt keeps crawlers out of the dashboard, the PocketBase API, and
// the screens that need an account. The photo files under /v1 stay open
// for link previews.
const robotsTxt = `User-agent: *
Disallow: /_/
Disallow: /api/
Disallow: /sign-in
Disallow: /sign-up
Disallow: /my-reviews
Disallow: /account
Disallow: /confirm-email
Disallow: /forgot-password
Disallow: /reset-password
`

func serveRobots(e *core.RequestEvent) error {
	e.Response.Header().Set("Cache-Control", robotsCacheControl)
	return e.String(http.StatusOK, robotsTxt)
}
