package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// GET /v1/reviews sorts the reviews of all restaurants. Its default sort is
// -created. The index on (restaurant, created, id) does not serve it.
func init() {
	m.Register(func(app core.App) error {
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		reviews.AddIndex("idx_reviews_created", false, "created, id", "")
		return app.Save(reviews)
	}, func(app core.App) error {
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		reviews.RemoveIndex("idx_reviews_created")
		return app.Save(reviews)
	})
}
