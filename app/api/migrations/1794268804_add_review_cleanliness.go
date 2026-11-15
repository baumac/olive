package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Existing reviews get their space score as cleanliness. The space score
// included cleanliness before this field.
func init() {
	m.Register(func(app core.App) error {
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		cleanliness := score("cleanliness", 10)
		reviews.Fields.AddAt(7, cleanliness)
		if err := app.Save(reviews); err != nil {
			return err
		}
		if _, err := app.DB().NewQuery("UPDATE {{reviews}} SET [[cleanliness]] = [[space]]").Execute(); err != nil {
			return err
		}
		cleanliness.Required = true
		return app.Save(reviews)
	}, func(app core.App) error {
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		reviews.Fields.RemoveByName("cleanliness")
		return app.Save(reviews)
	})
}
