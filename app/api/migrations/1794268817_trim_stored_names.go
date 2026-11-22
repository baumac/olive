package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The API removes the ASCII spaces around these values of a request. The
// values from before that change get the same rule. See decision 00001.
func init() {
	m.Register(func(app core.App) error {
		return execAll(app,
			"UPDATE {{users}} SET [[name]] = TRIM([[name]]) WHERE [[name]] != TRIM([[name]])",
			"UPDATE {{restaurants}} SET [[name]] = TRIM([[name]]) WHERE [[name]] != TRIM([[name]])",
			"UPDATE {{restaurants}} SET [[address]] = TRIM([[address]]) WHERE [[address]] != TRIM([[address]])",
			"UPDATE {{reviews}} SET [[title]] = TRIM([[title]]) WHERE [[title]] != TRIM([[title]])",
			"UPDATE {{review_photos}} SET [[alt]] = TRIM([[alt]]) WHERE [[alt]] != TRIM([[alt]])",
			// The SQL updates skip the record hooks of the search index.
			"UPDATE {{restaurants_fts}} SET [[name]] = TRIM([[name]]), [[address]] = TRIM([[address]])"+
				" WHERE [[name]] != TRIM([[name]]) OR [[address]] != TRIM([[address]])",
			"UPDATE {{reviews_fts}} SET [[title]] = TRIM([[title]]), [[restaurant_name]] = TRIM([[restaurant_name]])"+
				" WHERE [[title]] != TRIM([[title]]) OR [[restaurant_name]] != TRIM([[restaurant_name]])",
		)
	}, func(core.App) error {
		return nil
	})
}
