package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Each user gets a favorites list. A unique index keeps it to one for each
// user. See decision 00006.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		restaurants, err := app.FindCollectionByNameOrId("restaurants")
		if err != nil {
			return err
		}

		lists := core.NewBaseCollection("lists")
		lists.Fields.Add(
			&core.RelationField{Name: "creator", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.SelectField{Name: "kind", Values: []string{"favorites", "custom"}, MaxSelect: 1, Required: true},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		lists.AddIndex("idx_lists_creator_created", false, "creator, created, id", "")
		lists.AddIndex("idx_lists_creator_name", true, "creator, name COLLATE NOCASE", "")
		lists.AddIndex("idx_lists_favorites", true, "creator", "kind = 'favorites'")
		if err := app.Save(lists); err != nil {
			return err
		}

		items := core.NewBaseCollection("list_items")
		items.Fields.Add(
			&core.RelationField{Name: "list", CollectionId: lists.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.RelationField{Name: "restaurant", CollectionId: restaurants.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		items.AddIndex("idx_list_items_list_restaurant", true, "list, restaurant", "")
		items.AddIndex("idx_list_items_list_created", false, "list, created, restaurant", "")
		items.AddIndex("idx_list_items_restaurant", false, "restaurant, list", "")
		if err := app.Save(items); err != nil {
			return err
		}

		// The id column has a default value.
		return execAll(app, "INSERT INTO {{lists}} ([[creator]], [[name]], [[kind]], [[created]], [[updated]]) "+
			"SELECT [[id]], 'Favorites', 'favorites', strftime('%Y-%m-%d %H:%M:%fZ'), strftime('%Y-%m-%d %H:%M:%fZ') FROM {{users}}")
	}, func(app core.App) error {
		for _, name := range []string{"list_items", "lists"} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		return nil
	})
}
