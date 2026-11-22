package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// The recipes get the tables, the triggers, and the indexes of the
// restaurants. A recipe has four scores. See decision 00047.
func init() {
	m.Register(addRecipes, dropRecipes)
}

var recipeScores = newReviewScores("recipe", "taste", "ease", "instructions", "value")

func addRecipes(app core.App) error {
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	recipes := newRecipes(users)
	if err := app.Save(recipes); err != nil {
		return err
	}
	reviews := newRecipeReviews(users, recipes)
	if err := app.Save(reviews); err != nil {
		return err
	}
	if err := app.Save(newRecipeReviewPhotos(reviews)); err != nil {
		return err
	}
	return execAll(app, recipeTables()...)
}

func dropRecipes(app core.App) error {
	err := execAll(app,
		"DROP TRIGGER {{recipe_index_insert}}",
		"DROP TRIGGER {{recipe_index_update}}",
		"DROP TRIGGER {{recipe_index_delete}}",
		"DROP TRIGGER {{recipe_review_overall_insert}}",
		"DROP TRIGGER {{recipe_review_overall_update}}",
		"DROP TRIGGER {{recipe_review_overall_delete}}",
		"DROP TABLE {{recipe_overall}}",
		"DROP TABLE {{recipe_tags}}",
		"DROP TABLE {{recipes_fts}}",
		"DROP TABLE {{recipe_reviews_fts}}",
	)
	if err != nil {
		return err
	}
	for _, name := range []string{"recipe_review_photos", "recipe_reviews", "recipes"} {
		c, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			return err
		}
		if err := app.Delete(c); err != nil {
			return err
		}
	}
	return nil
}

// newRecipes has the Website pattern of the spec for source. A record hook
// of the Public API sets source_key from source on each save.
func newRecipes(users *core.Collection) *core.Collection {
	c := core.NewBaseCollection("recipes")
	c.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 255},
		&core.TextField{Name: "description", Max: 10000},
		&core.JSONField{Name: "tags"},
		&core.JSONField{Name: "ingredients", Required: true},
		&core.JSONField{Name: "steps", Required: true},
		&core.TextField{Name: "source", Max: 2048, Pattern: websitePattern},
		&core.TextField{Name: "source_key", Max: 2048},
		&core.NumberField{Name: "total_time", OnlyInt: true, Min: types.Pointer(1.0), Max: types.Pointer(10080.0)},
		&core.NumberField{Name: "servings", OnlyInt: true, Min: types.Pointer(1.0), Max: types.Pointer(100.0)},
		&core.RelationField{Name: "creator", CollectionId: users.Id, MaxSelect: 1},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_recipes_name", false, "name COLLATE NOCASE, id", "")
	c.AddIndex("idx_recipes_created", false, "created, id", "")
	c.AddIndex("idx_recipes_creator_created", false, "creator, created, id", "")
	c.AddIndex("idx_recipes_source_key", true, "source_key", "source_key != ''")
	return c
}

// newRecipeReviews has no cascade delete of the recipe, like reviews. The
// Public API deletes the review of the creator with the recipe.
func newRecipeReviews(users, recipes *core.Collection) *core.Collection {
	c := core.NewBaseCollection("recipe_reviews")
	c.Fields.Add(
		&core.RelationField{Name: "recipe", CollectionId: recipes.Id, MaxSelect: 1, Required: true},
		&core.RelationField{Name: "creator", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
		recipeScore("taste"),
		recipeScore("ease"),
		recipeScore("instructions"),
		recipeScore("value"),
		&core.TextField{Name: "title", Required: true, Max: 255},
		&core.TextField{Name: "feedback", Max: 10000},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_recipe_reviews_recipe_creator", true, "recipe, creator", "")
	c.AddIndex("idx_recipe_reviews_recipe_created", false, "recipe, created, id", "")
	c.AddIndex("idx_recipe_reviews_title", false, "title COLLATE NOCASE, id", "")
	c.AddIndex("idx_recipe_reviews_created", false, "created, id", "")
	return c
}

func recipeScore(name string) *core.NumberField {
	return &core.NumberField{Name: name, OnlyInt: true, Required: true, Min: types.Pointer(1.0), Max: types.Pointer(10.0)}
}

// newRecipeReviewPhotos has no API rules, like review_photos.
func newRecipeReviewPhotos(reviews *core.Collection) *core.Collection {
	c := core.NewBaseCollection("recipe_review_photos")
	c.Fields.Add(
		&core.RelationField{Name: "review", CollectionId: reviews.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
		&core.TextField{Name: "alt", Max: 255},
		&core.NumberField{Name: "width", OnlyInt: true, Required: true},
		&core.NumberField{Name: "height", OnlyInt: true, Required: true},
		photoFile("full"),
		photoFile("thumb"),
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	c.AddIndex("idx_recipe_review_photos_review_created", false, "review, created, id", "")
	return c
}

// recipeTables makes the generated overall column of decision 00031, the
// tables of decision 00012, and the FTS5 tables of decision 00004.
func recipeTables() []string {
	recipeValues := recipeScores.overall() + ", " + recipeScores.ranking()
	ofReviews := func(values, recipe string) string {
		return "SELECT " + values + " FROM {{recipe_reviews}} WHERE [[recipe_reviews.recipe]] = " + recipe
	}
	update := func(row string) string {
		recipe := row + ".[[recipe]]"
		return "UPDATE {{recipe_overall}} SET ([[overall]], [[ranking]]) = (" + ofReviews(recipeValues, recipe) +
			") WHERE [[recipe]] = " + recipe + ";"
	}
	insertTags := func(row string) string {
		return "INSERT OR IGNORE INTO {{recipe_tags}} ([[tag]], [[recipe]]) SELECT [[value]], " + row + ".[[id]] FROM " +
			stringTags(row+".[[tags]]") + ";"
	}
	return []string{
		"ALTER TABLE {{recipe_reviews}} ADD COLUMN [[overall]] REAL AS " + recipeScores.column(),
		"CREATE INDEX {{idx_recipe_reviews_recipe_overall}} ON {{recipe_reviews}} ([[recipe]], [[overall]], [[id]])",
		"CREATE INDEX {{idx_recipe_reviews_creator_overall}} ON {{recipe_reviews}} ([[creator]], [[overall]], [[id]])",
		"CREATE INDEX {{idx_recipe_reviews_overall}} ON {{recipe_reviews}} ([[overall]], [[id]])",

		"CREATE TABLE {{recipe_overall}} ([[recipe]] TEXT PRIMARY KEY NOT NULL, [[overall]] REAL, " +
			"[[unrated]] INTEGER AS ([[overall]] IS NULL), [[rated]] INTEGER AS ([[overall]] IS NOT NULL), " +
			"[[ranking]] REAL, [[ranking_or_0]] REAL AS (COALESCE([[ranking]], 0))) WITHOUT ROWID",
		"CREATE INDEX {{idx_recipe_overall_asc}} ON {{recipe_overall}} ([[unrated]], [[ranking_or_0]], [[recipe]])",
		"CREATE INDEX {{idx_recipe_overall_desc}} ON {{recipe_overall}} ([[rated]], [[ranking_or_0]], [[recipe]])",
		"CREATE INDEX {{idx_recipe_overall_value}} ON {{recipe_overall}} ([[overall]])",
		"CREATE TABLE {{recipe_tags}} ([[tag]] TEXT NOT NULL, [[recipe]] TEXT NOT NULL, PRIMARY KEY ([[tag]], [[recipe]])) WITHOUT ROWID",
		"CREATE INDEX {{idx_recipe_tags_recipe}} ON {{recipe_tags}} ([[recipe]])",

		"CREATE TRIGGER {{recipe_index_insert}} AFTER INSERT ON {{recipes}} BEGIN " +
			"INSERT INTO {{recipe_overall}} ([[recipe]], [[overall]], [[ranking]]) " +
			ofReviews("NEW.[[id]], "+recipeValues, "NEW.[[id]]") + "; " +
			insertTags("NEW") + " END",
		"CREATE TRIGGER {{recipe_index_update}} AFTER UPDATE OF [[tags]] ON {{recipes}} WHEN OLD.[[tags]] IS NOT NEW.[[tags]] BEGIN " +
			"DELETE FROM {{recipe_tags}} WHERE [[recipe]] = OLD.[[id]]; " +
			insertTags("NEW") + " END",
		"CREATE TRIGGER {{recipe_index_delete}} AFTER DELETE ON {{recipes}} BEGIN " +
			"DELETE FROM {{recipe_overall}} WHERE [[recipe]] = OLD.[[id]]; " +
			"DELETE FROM {{recipe_tags}} WHERE [[recipe]] = OLD.[[id]]; END",
		"CREATE TRIGGER {{recipe_review_overall_insert}} AFTER INSERT ON {{recipe_reviews}} BEGIN " + update("NEW") + " END",
		"CREATE TRIGGER {{recipe_review_overall_update}} AFTER UPDATE OF " + strings.Join(recipeScores.triggerColumns, ", ") +
			" ON {{recipe_reviews}} WHEN " + anyChanged(recipeScores.triggerColumns) + " BEGIN " + update("NEW") + " " + update("OLD") + " END",
		"CREATE TRIGGER {{recipe_review_overall_delete}} AFTER DELETE ON {{recipe_reviews}} BEGIN " + update("OLD") + " END",

		"CREATE VIRTUAL TABLE {{recipes_fts}} USING fts5(id UNINDEXED, name, description, tags, ingredients, " +
			"tokenize = 'unicode61 remove_diacritics 2', prefix = '2 3')",
		"CREATE VIRTUAL TABLE {{recipe_reviews_fts}} USING fts5(id UNINDEXED, title, feedback, recipe_name, " +
			"tokenize = 'unicode61 remove_diacritics 2', prefix = '2 3')",
	}
}
