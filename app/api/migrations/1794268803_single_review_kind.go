package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Reviews have only the space, service, food, and price scores. A basic
// review gets two times its 1 to 5 rating in each score.
func init() {
	m.Register(func(app core.App) error {
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		if _, err := app.DB().NewQuery(
			"UPDATE {{reviews}} SET [[space]] = [[rating]] * 2, [[service]] = [[rating]] * 2," +
				" [[food]] = [[rating]] * 2, [[price]] = [[rating]] * 2 WHERE [[kind]] = 'basic'",
		).Execute(); err != nil {
			return err
		}
		reviews.Fields.RemoveByName("kind")
		reviews.Fields.RemoveByName("rating")
		for _, name := range []string{"space", "service", "food", "price"} {
			reviews.Fields.GetByName(name).(*core.NumberField).Required = true
		}
		return app.Save(reviews)
	}, func(app core.App) error {
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		for _, name := range []string{"space", "service", "food", "price"} {
			reviews.Fields.GetByName(name).(*core.NumberField).Required = false
		}
		reviews.Fields.AddAt(2, &core.SelectField{Name: "kind", Values: []string{"basic", "joc_de_cartes"}, MaxSelect: 1, Required: true})
		reviews.Fields.AddAt(3, score("rating", 5))
		if err := app.Save(reviews); err != nil {
			return err
		}
		_, err = app.DB().Update("reviews", dbx.Params{"kind": "joc_de_cartes"}, nil).Execute()
		return err
	})
}
