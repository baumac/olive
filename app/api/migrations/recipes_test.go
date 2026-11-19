package migrations

import (
	"database/sql"
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/baumac/reviews-app/app/api/internal/testapp"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func recipeReview(user, recipe *core.Record, taste int) map[string]any {
	return map[string]any{"recipe": recipe.Id, "creator": user.Id, "title": "T", "taste": taste, "ease": 8, "instructions": 8, "value": 8}
}

// storedRecipeOverall returns false when the recipe has no row.
func storedRecipeOverall(t *testing.T, app core.App, recipe string) (overall, ranking sql.NullFloat64, ok bool) {
	t.Helper()
	err := app.DB().NewQuery("SELECT [[overall]], [[ranking]] FROM {{recipe_overall}} WHERE [[recipe]] = {:id}").
		Bind(dbx.Params{"id": recipe}).Row(&overall, &ranking)
	if errors.Is(err, sql.ErrNoRows) {
		return overall, ranking, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return overall, ranking, true
}

func TestRecipeTriggers(t *testing.T) {
	t.Parallel()
	app := newIndexApp(t)
	ana := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	joe := newRecord(t, app, "users", map[string]any{"email": "joe@example.com", "password": "correct-horse", "name": "Joe"})
	dal := newRecord(t, app, "recipes", map[string]any{"name": "Dal", "ingredients": []string{"lentils"}, "steps": []string{"Boil."}, "tags": []string{"dal", "dal", "vegan"}})

	if overall, ranking, ok := storedRecipeOverall(t, app, dal.Id); !ok || overall.Valid || ranking.Valid {
		t.Errorf("new recipe: overall %v, ranking %v, row %v", overall, ranking, ok)
	}

	first := newRecord(t, app, "recipe_reviews", recipeReview(ana, dal, 9))
	newRecord(t, app, "recipe_reviews", recipeReview(joe, dal, 6))
	// The overall values are 8.25 and 7.5.
	overall, ranking, _ := storedRecipeOverall(t, app, dal.Id)
	if overall.Float64 != 7.9 || math.Abs(ranking.Float64-(5*5.5+8.25+7.5)/7) > 1e-9 {
		t.Errorf("two reviews: overall %v, ranking %v", overall, ranking)
	}
	var column float64
	if err := app.DB().Select("overall").From("recipe_reviews").Where(dbx.HashExp{"id": first.Id}).Row(&column); err != nil || column != 8.25 {
		t.Errorf("overall column %v, error %v", column, err)
	}

	first.Set("taste", 1)
	if err := app.Save(first); err != nil {
		t.Fatal(err)
	}
	if overall, _, _ := storedRecipeOverall(t, app, dal.Id); overall.Float64 != 6.9 {
		t.Errorf("changed review: overall %v, want 6.9 of 6.25 and 7.5", overall)
	}

	var tags []string
	if err := app.DB().Select("tag").From("recipe_tags").Where(dbx.HashExp{"recipe": dal.Id}).OrderBy("tag").Column(&tags); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(tags, []string{"dal", "vegan"}) {
		t.Errorf("tags %v", tags)
	}
	dal.Set("tags", []string{"lentils"})
	if err := app.Save(dal); err != nil {
		t.Fatal(err)
	}
	tags = nil
	if err := app.DB().Select("tag").From("recipe_tags").Where(dbx.HashExp{"recipe": dal.Id}).Column(&tags); err != nil || !slices.Equal(tags, []string{"lentils"}) {
		t.Errorf("changed tags %v, error %v", tags, err)
	}

	if err := app.Delete(joe); err != nil {
		t.Fatal(err)
	}
	if err := app.Delete(first); err != nil {
		t.Fatal(err)
	}
	if overall, ranking, ok := storedRecipeOverall(t, app, dal.Id); !ok || overall.Valid || ranking.Valid {
		t.Errorf("no reviews: overall %v, ranking %v, row %v", overall, ranking, ok)
	}
	if err := app.Delete(dal); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := storedRecipeOverall(t, app, dal.Id); ok {
		t.Error("deleted recipe still has a row")
	}
}

func TestRecipesMigrationDown(t *testing.T) {
	t.Parallel()
	app := newIndexApp(t)
	items, err := app.FindCollectionByNameOrId("list_recipes")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Delete(items); err != nil {
		t.Fatal(err)
	}
	if err := dropRecipes(app); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"recipes", "recipe_reviews", "recipe_review_photos", "recipe_overall", "recipe_tags", "recipes_fts", "recipe_reviews_fts"} {
		if app.HasTable(name) {
			t.Errorf("table %s stays", name)
		}
	}
	if err := addRecipes(app); err != nil {
		t.Fatal(err)
	}
}

func TestTrimStoredStepsMigration(t *testing.T) {
	t.Parallel()
	app, err := tests.NewTestApp(testapp.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	redo := downTo(t, app, "1794268820_trim_stored_steps.go")
	padded := newRecord(t, app, "recipes", map[string]any{
		"name": "Shakshuka", "ingredients": []string{"6 eggs"},
		"steps": []string{"\n\tCook the onion.\r\n", "Add the tomatoes.", " Break the eggs.\n\n"},
	})
	plain := newRecord(t, app, "recipes", map[string]any{
		"name": "Hummus", "ingredients": []string{"1 can of chickpeas"},
		"steps": []string{"Blend\nthe chickpeas.", "Serve\u00a0"},
	})
	updated := padded.GetString("updated")
	// A save without validation can store steps that are not an array of
	// text. The migration keeps them.
	odd := map[string]string{}
	for _, steps := range []string{`"  Mix. "`, `[" Mix. ",3]`, `{"a":" b "}`, `not json`} {
		r := newRecord(t, app, "recipes", map[string]any{"name": "Odd", "ingredients": []string{"1 egg"}, "steps": []string{"x"}})
		if _, err := app.DB().Update("recipes", dbx.Params{"steps": steps}, dbx.HashExp{"id": r.Id}).Execute(); err != nil {
			t.Fatal(err)
		}
		odd[r.Id] = steps
	}
	redo()

	for id, want := range odd {
		var got string
		if err := app.DB().NewQuery("SELECT [[steps]] FROM {{recipes}} WHERE [[id]] = {:id}").Bind(dbx.Params{"id": id}).Row(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("steps %s, want %s", got, want)
		}
	}
	for id, want := range map[string][]string{
		padded.Id: {"Cook the onion.", "Add the tomatoes.", "Break the eggs."},
		plain.Id:  {"Blend\nthe chickpeas.", "Serve\u00a0"},
	} {
		r, err := app.FindRecordById("recipes", id)
		if err != nil {
			t.Fatal(err)
		}
		var steps []string
		if err := r.UnmarshalJSONField("steps", &steps); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(steps, want) {
			t.Errorf("steps %q, want %q", steps, want)
		}
	}
	if r, _ := app.FindRecordById("recipes", padded.Id); r.GetString("updated") != updated {
		t.Errorf("the migration changed updated: %s, want %s", r.GetString("updated"), updated)
	}
}
