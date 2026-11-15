package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The reviewer page lists the restaurants of one creator by -created. The
// new index also serves the queries of idx_restaurants_creator. See decision
// 00032.
func init() {
	m.Register(func(app core.App) error {
		restaurants, err := app.FindCollectionByNameOrId("restaurants")
		if err != nil {
			return err
		}
		restaurants.RemoveIndex("idx_restaurants_creator")
		restaurants.AddIndex("idx_restaurants_creator_created", false, "creator, created, id", "")
		return app.Save(restaurants)
	}, func(app core.App) error {
		restaurants, err := app.FindCollectionByNameOrId("restaurants")
		if err != nil {
			return err
		}
		restaurants.RemoveIndex("idx_restaurants_creator_created")
		restaurants.AddIndex("idx_restaurants_creator", false, "creator", "")
		return app.Save(restaurants)
	})
}
