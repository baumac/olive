package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The API removes the ASCII spaces around these values of a request. The
// values from before that change get the same rule. See decision 00040.
func init() {
	m.Register(func(app core.App) error {
		return execAll(app,
			"UPDATE {{users}} SET [[name]] = TRIM([[name]]) WHERE [[name]] != TRIM([[name]])",
			"UPDATE {{restaurants}} SET [[name]] = TRIM([[name]]) WHERE [[name]] != TRIM([[name]])",
			"UPDATE {{restaurants}} SET [[address]] = TRIM([[address]]) WHERE [[address]] != TRIM([[address]])",
			"UPDATE {{reviews}} SET [[title]] = TRIM([[title]]) WHERE [[title]] != TRIM([[title]])",
			"UPDATE {{review_photos}} SET [[alt]] = TRIM([[alt]]) WHERE [[alt]] != TRIM([[alt]])",
		)
	}, func(core.App) error {
		return nil
	})
}
