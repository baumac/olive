package migrations

import (
	"slices"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// cspReportRule limits the CSP reports of decision 00005 per IP address.
// Anybody can send them. The limit keeps the log small.
var cspReportRule = core.RateLimitRule{Label: "POST /csp-report", MaxRequests: 20, Duration: 60}

func init() {
	m.Register(func(app core.App) error {
		settings := app.Settings()
		settings.RateLimits.Rules = append(settings.RateLimits.Rules, cspReportRule)
		return app.Save(settings)
	}, func(app core.App) error {
		settings := app.Settings()
		settings.RateLimits.Rules = slices.DeleteFunc(settings.RateLimits.Rules, func(r core.RateLimitRule) bool {
			return r.Label == cspReportRule.Label
		})
		return app.Save(settings)
	})
}
