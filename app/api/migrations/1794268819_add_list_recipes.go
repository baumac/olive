package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// list_recipes has the fields and the indexes of list_items with a recipe in
// the place of the restaurant. See decision 00004.
func init() {
	m.Register(func(app core.App) error {
		lists, err := app.FindCollectionByNameOrId("lists")
		if err != nil {
			return err
		}
		recipes, err := app.FindCollectionByNameOrId("recipes")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection("list_recipes")
		c.Fields.Add(
			&core.RelationField{Name: "list", CollectionId: lists.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.RelationField{Name: "recipe", CollectionId: recipes.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		c.AddIndex("idx_list_recipes_list_recipe", true, "list, recipe", "")
		c.AddIndex("idx_list_recipes_list_created", false, "list, created, recipe", "")
		c.AddIndex("idx_list_recipes_recipe", false, "recipe, list", "")
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("list_recipes")
		if err != nil {
			return err
		}
		return app.Delete(c)
	})
}
