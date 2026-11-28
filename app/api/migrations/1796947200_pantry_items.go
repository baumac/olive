package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(pantryItemsUp, pantryItemsDown)
}

// pantryItemsUp makes the collections and the tables of the pantry items of
// decision 00012.
func pantryItemsUp(app core.App) error {
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	lists, err := app.FindCollectionByNameOrId("lists")
	if err != nil {
		return err
	}
	tags, err := app.FindCollectionByNameOrId("tags")
	if err != nil {
		return err
	}
	return pantryItemKindWith(tags).create(app, users, lists)
}

func pantryItemsDown(app core.App) error {
	return pantryItemKind.drop(app)
}

func pantryItemKindWith(tags *core.Collection) subjectKind {
	k := pantryItemKind
	k.newParent = func(users *core.Collection) *core.Collection { return newPantryItems(users, tags) }
	return k
}
