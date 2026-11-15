package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The rate limits of PocketBase are off by default. Its default rules limit
// the auth routes of /api. These include the superuser sign in of the
// dashboard. The rules do not apply to /v1. The Public API has its own limit.
func init() {
	m.Register(func(app core.App) error {
		return setRateLimits(app, true)
	}, func(app core.App) error {
		return setRateLimits(app, false)
	})
}

func setRateLimits(app core.App, enabled bool) error {
	settings := app.Settings()
	settings.RateLimits.Enabled = enabled
	return app.Save(settings)
}
