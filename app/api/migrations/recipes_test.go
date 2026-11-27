package migrations

import (
	"database/sql"
	"errors"
	"slices"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func recipeReview(user, recipe *core.Record, taste int) map[string]any {
	return map[string]any{"recipe": recipe.Id, "creator": user.Id, "title": "T", "date": "2024-11-06", "taste": taste, "ease": 8, "instructions": 8, "value": 8}
}

// storedRecipeOverall returns false when the recipe has no row.
func storedRecipeOverall(t *testing.T, app core.App, recipe string) (overall sql.NullFloat64, reviewers int, ok bool) {
	t.Helper()
	err := app.DB().NewQuery("SELECT [[overall]], [[reviewers]] FROM {{recipe_overall}} WHERE [[recipe]] = {:id}").
		Bind(dbx.Params{"id": recipe}).Row(&overall, &reviewers)
	if errors.Is(err, sql.ErrNoRows) {
		return overall, reviewers, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return overall, reviewers, true
}

func TestRecipeTriggers(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	ana := newUser(t, app, "ana")
	joe := newUser(t, app, "joe")
	dalTag, vegan, lentils := newTag(t, app, "dal"), newTag(t, app, "vegan"), newTag(t, app, "lentils")
	dal := newRecord(t, app, "recipes", map[string]any{"name": "Dal", "ingredients": []string{"lentils"}, "steps": []string{"Boil."}, "tags": []string{dalTag, dalTag, vegan}})

	if overall, reviewers, ok := storedRecipeOverall(t, app, dal.Id); !ok || overall.Valid || reviewers != 0 {
		t.Errorf("new recipe: overall %v, reviewers %d, row %v", overall, reviewers, ok)
	}

	first := newRecord(t, app, "recipe_reviews", recipeReview(ana, dal, 9))
	newRecord(t, app, "recipe_reviews", recipeReview(joe, dal, 6))
	// The overall values are 8.25 and 7.5.
	if overall, reviewers, _ := storedRecipeOverall(t, app, dal.Id); overall.Float64 != 7.9 || reviewers != 2 {
		t.Errorf("two reviews: overall %v, reviewers %d", overall, reviewers)
	}
	// Ana has 8.25 and 6.25. Joe has 7.5.
	again := newRecord(t, app, "recipe_reviews", recipeReview(ana, dal, 1))
	if overall, reviewers, _ := storedRecipeOverall(t, app, dal.Id); overall.Float64 != 7.4 || reviewers != 2 {
		t.Errorf("second review of Ana: overall %v, reviewers %d", overall, reviewers)
	}
	if err := app.Delete(again); err != nil {
		t.Fatal(err)
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
	if want := sorted(dalTag, vegan); !slices.Equal(tags, want) {
		t.Errorf("tags %v", tags)
	}
	dal.Set("tags", []string{lentils})
	if err := app.Save(dal); err != nil {
		t.Fatal(err)
	}
	tags = nil
	if err := app.DB().Select("tag").From("recipe_tags").Where(dbx.HashExp{"recipe": dal.Id}).Column(&tags); err != nil || !slices.Equal(tags, []string{lentils}) {
		t.Errorf("changed tags %v, error %v", tags, err)
	}

	if err := app.Delete(joe); err != nil {
		t.Fatal(err)
	}
	if err := app.Delete(first); err != nil {
		t.Fatal(err)
	}
	if overall, reviewers, ok := storedRecipeOverall(t, app, dal.Id); !ok || overall.Valid || reviewers != 0 {
		t.Errorf("no reviews: overall %v, reviewers %d, row %v", overall, reviewers, ok)
	}
	if err := app.Delete(dal); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := storedRecipeOverall(t, app, dal.Id); ok {
		t.Error("deleted recipe still has a row")
	}
}
