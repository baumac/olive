package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The users that exist before this migration have a confirmed email
// address. They signed in without a confirmation. See decision 00037.
func init() {
	m.Register(func(app core.App) error {
		return execAll(app, "UPDATE {{users}} SET [[verified]] = TRUE")
	}, func(core.App) error {
		return nil
	})
}
