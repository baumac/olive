package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The overall column of reviews is not a field of the collection. A
// migration that drops a score column must first drop it and its indexes.
// See decision 00005.
func init() {
	m.Register(func(app core.App) error {
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		// idx_reviews_creator_overall also serves the queries of this index.
		reviews.RemoveIndex("idx_reviews_creator")
		if err := app.Save(reviews); err != nil {
			return err
		}
		return execAll(app,
			"ALTER TABLE {{reviews}} ADD COLUMN [[overall]] REAL AS "+fiveRestaurantScores.column(),
			"CREATE INDEX {{idx_reviews_restaurant_overall}} ON {{reviews}} ([[restaurant]], [[overall]], [[id]])",
			"CREATE INDEX {{idx_reviews_creator_overall}} ON {{reviews}} ([[creator]], [[overall]], [[id]])",
			"CREATE INDEX {{idx_reviews_overall}} ON {{reviews}} ([[overall]], [[id]])",
		)
	}, func(app core.App) error {
		err := execAll(app,
			"DROP INDEX {{idx_reviews_restaurant_overall}}",
			"DROP INDEX {{idx_reviews_creator_overall}}",
			"DROP INDEX {{idx_reviews_overall}}",
			"ALTER TABLE {{reviews}} DROP COLUMN [[overall]]",
		)
		if err != nil {
			return err
		}
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		reviews.AddIndex("idx_reviews_creator", false, "creator", "")
		return app.Save(reviews)
	})
}
