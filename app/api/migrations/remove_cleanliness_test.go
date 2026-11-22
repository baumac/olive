package migrations

import (
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

const removeCleanlinessFile = "1794268822_remove_review_cleanliness.go"

func hasReviewColumn(t *testing.T, app core.App, name string) bool {
	t.Helper()
	columns, err := app.TableColumns("reviews")
	if err != nil {
		t.Fatal(err)
	}
	return slices.Contains(columns, name)
}

func TestRemoveCleanlinessComputesTheOverallAgain(t *testing.T) {
	t.Parallel()
	app := newIndexApp(t)
	redo := downTo(t, app, removeCleanlinessFile)
	ana := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	joe := newRecord(t, app, "users", map[string]any{"email": "joe@example.com", "password": "correct-horse", "name": "Joe"})
	rated := newRecord(t, app, "restaurants", map[string]any{"name": "Rated"})
	unrated := newRecord(t, app, "restaurants", map[string]any{"name": "Unrated"})
	first := newRecord(t, app, "reviews", review(ana, rated, map[string]any{"space": 3, "service": 9, "food": 8, "price": 6, "cleanliness": 4}))
	newRecord(t, app, "reviews", review(joe, rated, map[string]any{"space": 10, "service": 10, "food": 10, "price": 8, "cleanliness": 1}))
	wantOverall(t, app, rated.Id, 6.9)
	if !hasReviewColumn(t, app, "cleanliness") {
		t.Fatal("down: reviews has no cleanliness column")
	}

	redo()
	if hasReviewColumn(t, app, "cleanliness") {
		t.Error("reviews still has the cleanliness column")
	}
	c, err := app.FindCollectionByNameOrId("reviews")
	if err != nil {
		t.Fatal(err)
	}
	if c.Fields.GetByName("cleanliness") != nil {
		t.Error("reviews still has the cleanliness field")
	}
	if got := reviewOverall(t, app, first.Id); got != 6.5 {
		t.Errorf("overall of the review is %v, want 6.5", got)
	}
	wantOverall(t, app, rated.Id, 8)
	wantReviewers(t, app, rated.Id, 2)
	wantNoOverall(t, app, unrated.Id, "up")

	first, err = app.FindRecordById("reviews", first.Id)
	if err != nil {
		t.Fatal(err)
	}
	first.Set("space", 7)
	if err := app.Save(first); err != nil {
		t.Fatal(err)
	}
	wantOverall(t, app, rated.Id, 8.5)
	if err := app.Delete(first); err != nil {
		t.Fatal(err)
	}
	wantOverall(t, app, rated.Id, 9.5)
	wantReviewers(t, app, rated.Id, 1)
	added := newRecord(t, app, "restaurants", map[string]any{"name": "Added"})
	wantNoOverall(t, app, added.Id, "new restaurant")
}

func TestRemoveCleanlinessDownCopiesSpace(t *testing.T) {
	t.Parallel()
	app := newIndexApp(t)
	ana := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	luigi := newRecord(t, app, "restaurants", map[string]any{"name": "Luigi's"})
	r := newRecord(t, app, "reviews", review(ana, luigi, map[string]any{"space": 3, "service": 9, "food": 8, "price": 6}))

	redo := downTo(t, app, removeCleanlinessFile)
	r, err := app.FindRecordById("reviews", r.Id)
	if err != nil {
		t.Fatal(err)
	}
	if v := r.GetInt("cleanliness"); v != 3 {
		t.Errorf("cleanliness is %d, want 3", v)
	}
	if got := reviewOverall(t, app, r.Id); got != 5.8 {
		t.Errorf("overall of the review is %v, want 5.8", got)
	}
	wantOverall(t, app, luigi.Id, 5.8)
	r.Set("cleanliness", 8)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	wantOverall(t, app, luigi.Id, 6.8)
	wantRanking(t, app, luigi.Id, (5*5.5+6.8)/6)

	redo()
	wantOverall(t, app, luigi.Id, 6.5)
}
