package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// A restaurant without a location has an empty address. The migration makes
// restaurants_fts again for its new address column.
func init() {
	m.Register(func(app core.App) error {
		restaurants, err := app.FindCollectionByNameOrId("restaurants")
		if err != nil {
			return err
		}
		at := len(restaurants.Fields)
		for i, f := range restaurants.Fields {
			if f.GetName() == "tags" {
				at = i + 1
			}
		}
		restaurants.Fields.AddAt(at,
			&core.NumberField{Name: "latitude"},
			&core.NumberField{Name: "longitude"},
			&core.TextField{Name: "address", Max: 255},
			&core.TextField{Name: "place_id", Max: 80},
		)
		restaurants.AddIndex("idx_restaurants_place_id", true, "place_id", "place_id != ''")
		restaurants.AddIndex("idx_restaurants_latitude", false, "latitude, longitude", "")
		if err := app.Save(restaurants); err != nil {
			return err
		}
		return rebuildRestaurantIndex(app, true)
	}, func(app core.App) error {
		if err := rebuildRestaurantIndex(app, false); err != nil {
			return err
		}
		restaurants, err := app.FindCollectionByNameOrId("restaurants")
		if err != nil {
			return err
		}
		restaurants.RemoveIndex("idx_restaurants_place_id")
		restaurants.RemoveIndex("idx_restaurants_latitude")
		for _, name := range []string{"latitude", "longitude", "address", "place_id"} {
			restaurants.Fields.RemoveByName(name)
		}
		return app.Save(restaurants)
	})
}

// rebuildRestaurantIndex makes restaurants_fts again and fills it from the
// records. The address column needs the address field of restaurants.
func rebuildRestaurantIndex(app core.App, withAddress bool) error {
	columns, values := "name, description, tags", "[[name]], [[description]], "+tagText("restaurants")
	if withAddress {
		columns += ", address"
		values += ", [[address]]"
	}
	for _, q := range []string{
		"DROP TABLE {{restaurants_fts}}",
		"CREATE VIRTUAL TABLE {{restaurants_fts}} USING fts5(id UNINDEXED, " + columns + ", tokenize = 'unicode61 remove_diacritics 2', prefix = '2 3')",
		"INSERT INTO {{restaurants_fts}} (id, " + columns + ") SELECT [[id]], " + values + " FROM {{restaurants}}",
	} {
		if _, err := app.DB().NewQuery(q).Execute(); err != nil {
			return err
		}
	}
	return nil
}
