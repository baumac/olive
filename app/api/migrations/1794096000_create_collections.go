// Package migrations has the PocketBase migrations of the collections that
// the Public API uses.
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/dbutils"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(up, down)
}

func up(app core.App) error {
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	if err := setupUsers(app, users); err != nil {
		return err
	}

	restaurants := newRestaurants(users)
	if err := app.Save(restaurants); err != nil {
		return err
	}
	return app.Save(newReviews(users, restaurants))
}

func down(app core.App) error {
	for _, name := range []string{"reviews", "restaurants"} {
		c, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			return err
		}
		if err := app.Delete(c); err != nil {
			return err
		}
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	return restoreUsers(app, users)
}

// setupUsers removes all API rules of users. A nil AuthRule also stops the
// PocketBase auth endpoints. All sign ins then go through the Public API.
func setupUsers(app core.App, users *core.Collection) error {
	users.ListRule = nil
	users.ViewRule = nil
	users.CreateRule = nil
	users.UpdateRule = nil
	users.DeleteRule = nil
	users.AuthRule = nil
	users.PasswordAuth.Enabled = true
	users.PasswordAuth.IdentityFields = []string{core.FieldNameEmail}

	name, ok := users.Fields.GetByName("name").(*core.TextField)
	if !ok {
		name = &core.TextField{Name: "name"}
		users.Fields.Add(name)
	}
	name.Required = true
	name.Max = 255

	setEmailIndex(users, "email COLLATE NOCASE")
	return app.Save(users)
}

// restoreUsers sets the users collection back to the PocketBase defaults.
func restoreUsers(app core.App, users *core.Collection) error {
	owner := types.Pointer("id = @request.auth.id")
	users.ListRule, users.ViewRule, users.UpdateRule, users.DeleteRule = owner, owner, owner, owner
	users.CreateRule = types.Pointer("")
	users.AuthRule = types.Pointer("")
	if name, ok := users.Fields.GetByName("name").(*core.TextField); ok {
		name.Required = false
	}
	setEmailIndex(users, "email")
	return app.Save(users)
}

// setEmailIndex replaces the unique email index. With COLLATE NOCASE, email
// uniqueness and the sign in lookup ignore ASCII case.
func setEmailIndex(users *core.Collection, columns string) {
	name := "idx_email_" + users.Id
	if idx, ok := dbutils.FindSingleColumnUniqueIndex(users.Indexes, core.FieldNameEmail); ok {
		name = idx.IndexName
		users.RemoveIndex(name)
	}
	users.AddIndex(name, true, columns, "email != ''")
}

func newRestaurants(users *core.Collection) *core.Collection {
	c := core.NewBaseCollection("restaurants")
	c.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 255},
		&core.TextField{Name: "description", Max: 10000},
		&core.JSONField{Name: "tags"},
		&core.RelationField{Name: "creator", CollectionId: users.Id, MaxSelect: 1},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_restaurants_name", false, "name COLLATE NOCASE, id", "")
	c.AddIndex("idx_restaurants_created", false, "created, id", "")
	c.AddIndex("idx_restaurants_creator", false, "creator", "")
	return c
}

func newReviews(users, restaurants *core.Collection) *core.Collection {
	c := core.NewBaseCollection("reviews")
	c.Fields.Add(
		&core.RelationField{Name: "restaurant", CollectionId: restaurants.Id, MaxSelect: 1, Required: true},
		&core.RelationField{Name: "creator", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
		&core.SelectField{Name: "kind", Values: []string{"basic", "joc_de_cartes"}, MaxSelect: 1, Required: true},
		score("rating", 5),
		score("space", 10),
		score("service", 10),
		score("food", 10),
		score("price", 10),
		&core.TextField{Name: "title", Required: true, Max: 255},
		&core.TextField{Name: "feedback", Max: 10000},
		&core.JSONField{Name: "tags"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_reviews_restaurant_creator", true, "restaurant, creator", "")
	c.AddIndex("idx_reviews_restaurant_created", false, "restaurant, created, id", "")
	c.AddIndex("idx_reviews_title", false, "title COLLATE NOCASE, id", "")
	c.AddIndex("idx_reviews_creator", false, "creator", "")
	return c
}

// score makes a number field. PocketBase stores 0 when the review kind does
// not have the score. Min and max apply only to values that are not 0.
func score(name string, maxValue float64) *core.NumberField {
	return &core.NumberField{Name: name, OnlyInt: true, Min: types.Pointer(1.0), Max: types.Pointer(maxValue)}
}
