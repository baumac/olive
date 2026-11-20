package migrations

import (
	"slices"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const manyReviewsFile = "1794268821_allow_many_reviews.go"

func TestAllowManyReviewsSetsTheDate(t *testing.T) {
	t.Parallel()
	app := newIndexApp(t)
	redo := downTo(t, app, manyReviewsFile)
	ana := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	luigi := newRecord(t, app, "restaurants", map[string]any{"name": "Luigi's"})
	dal := newRecord(t, app, "recipes", map[string]any{"name": "Dal", "ingredients": []string{"lentils"}, "steps": []string{"Boil."}})
	old := newRecord(t, app, "reviews", review(ana, luigi, scores(8)))
	oldRecipe := newRecord(t, app, "recipe_reviews", recipeReview(ana, dal, 8))
	for table, id := range map[string]string{"reviews": old.Id, "recipe_reviews": oldRecipe.Id} {
		if _, err := app.DB().Update(table, dbx.Params{"created": "2024-03-31 23:59:59.999Z"}, dbx.HashExp{"id": id}).Execute(); err != nil {
			t.Fatal(err)
		}
	}
	redo()

	for table, id := range map[string]string{"reviews": old.Id, "recipe_reviews": oldRecipe.Id} {
		r, err := app.FindRecordById(table, id)
		if err != nil {
			t.Fatal(err)
		}
		if got := r.GetString("date"); got != "2024-03-31" {
			t.Errorf("%s: date %q, want 2024-03-31", table, got)
		}
	}
	newRecord(t, app, "reviews", review(ana, luigi, scores(6)))
	newRecord(t, app, "recipe_reviews", recipeReview(ana, dal, 6))

	bad := review(ana, luigi, scores(6))
	bad["date"] = "6 Nov 2024"
	c, err := app.FindCollectionByNameOrId("reviews")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(bad)
	if err := app.Save(r); err == nil {
		t.Error("a date that does not match the pattern was saved")
	}
}

func TestAllowManyReviewsDownKeepsTheLatestReview(t *testing.T) {
	t.Parallel()
	app := newIndexApp(t)
	ana := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	joe := newRecord(t, app, "users", map[string]any{"email": "joe@example.com", "password": "correct-horse", "name": "Joe"})
	luigi := newRecord(t, app, "restaurants", map[string]any{"name": "Luigi's"})
	dal := newRecord(t, app, "recipes", map[string]any{"name": "Dal", "ingredients": []string{"lentils"}, "steps": []string{"Boil."}})

	dated := func(data map[string]any, date string) map[string]any {
		data["date"] = date
		return data
	}
	january := newRecord(t, app, "reviews", dated(review(ana, luigi, scores(2)), "2024-01-01"))
	march := newRecord(t, app, "reviews", dated(review(ana, luigi, scores(9)), "2024-03-01"))
	february := newRecord(t, app, "reviews", dated(review(ana, luigi, scores(4)), "2024-02-01"))
	joes := newRecord(t, app, "reviews", dated(review(joe, luigi, scores(7)), "2024-01-01"))
	// Two reviews with one date keep the review with the later created.
	firstCook := newRecord(t, app, "recipe_reviews", recipeReview(ana, dal, 3))
	secondCook := newRecord(t, app, "recipe_reviews", recipeReview(ana, dal, 9))
	if _, err := app.DB().Update("recipe_reviews", dbx.Params{"created": "2024-01-01 00:00:00.000Z"}, dbx.HashExp{"id": firstCook.Id}).Execute(); err != nil {
		t.Fatal(err)
	}

	photo := "p" + january.Id[1:]
	if _, err := app.DB().Insert("review_photos", dbx.Params{
		"id": photo, "review": january.Id, "alt": "", "width": 1, "height": 1, "full": "", "thumb": "", "created": "2024-01-01 00:00:00.000Z",
	}).Execute(); err != nil {
		t.Fatal(err)
	}
	for _, r := range []*core.Record{january, march, february, joes} {
		if _, err := app.DB().Insert("reviews_fts", dbx.Params{"id": r.Id, "title": "T"}).Execute(); err != nil {
			t.Fatal(err)
		}
	}

	redo := downTo(t, app, manyReviewsFile)

	if got := ids(t, app, "reviews"); !sameIDs(got, march.Id, joes.Id) {
		t.Errorf("reviews %v, want %s and %s", got, march.Id, joes.Id)
	}
	if got := ids(t, app, "recipe_reviews"); !sameIDs(got, secondCook.Id) {
		t.Errorf("recipe reviews %v, want %s", got, secondCook.Id)
	}
	if got := ids(t, app, "review_photos"); len(got) != 0 {
		t.Errorf("photos %v stay", got)
	}
	if got := ids(t, app, "reviews_fts"); !sameIDs(got, march.Id, joes.Id) {
		t.Errorf("search rows %v, want %s and %s", got, march.Id, joes.Id)
	}
	var overall float64
	if err := app.DB().NewQuery("SELECT [[overall]] FROM {{restaurant_overall}} WHERE [[restaurant]] = {:id}").
		Bind(dbx.Params{"id": luigi.Id}).Row(&overall); err != nil || overall != 8 {
		t.Errorf("overall %v, error %v, want 8", overall, err)
	}

	c, err := app.FindCollectionByNameOrId("reviews")
	if err != nil {
		t.Fatal(err)
	}
	if c.Fields.GetByName("date") != nil {
		t.Error("down kept the date field")
	}
	r := core.NewRecord(c)
	r.Load(review(ana, luigi, scores(5)))
	if err := app.Save(r); err == nil {
		t.Error("down allowed a second review of one user")
	}

	redo()
	if r, err := app.FindRecordById("reviews", march.Id); err != nil || r.GetString("date") == "" {
		t.Errorf("up again: date %q, error %v", r.GetString("date"), err)
	}
}

func ids(t *testing.T, app core.App, table string) []string {
	t.Helper()
	var out []string
	if err := app.DB().Select("id").From(table).Column(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func sameIDs(got []string, want ...string) bool {
	slices.Sort(got)
	slices.Sort(want)
	return slices.Equal(got, want)
}
