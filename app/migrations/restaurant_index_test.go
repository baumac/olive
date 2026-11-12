package migrations

import (
	"database/sql"
	"errors"
	"slices"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newIndexApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

func scores(n int) map[string]any {
	return map[string]any{"space": n, "service": n, "food": n, "price": n, "cleanliness": n}
}

func review(user, restaurant *core.Record, s map[string]any) map[string]any {
	data := map[string]any{"restaurant": restaurant.Id, "creator": user.Id, "title": "T"}
	for k, v := range s {
		data[k] = v
	}
	return data
}

// storedOverall returns false when the restaurant has no row.
func storedOverall(t *testing.T, app core.App, restaurant string) (sql.NullFloat64, bool) {
	t.Helper()
	var v sql.NullFloat64
	err := app.DB().Select("overall").From("restaurant_overall").Where(dbx.HashExp{"restaurant": restaurant}).Row(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return v, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return v, true
}

func wantOverall(t *testing.T, app core.App, restaurant string, want float64) {
	t.Helper()
	got, ok := storedOverall(t, app, restaurant)
	if !ok || !got.Valid || got.Float64 != want {
		t.Errorf("overall of %s is %v (row %v), want %v", restaurant, got, ok, want)
	}
}

func wantNoOverall(t *testing.T, app core.App, restaurant, when string) {
	t.Helper()
	if got, ok := storedOverall(t, app, restaurant); !ok || got.Valid {
		t.Errorf("%s: overall is %v (row %v), want NULL", when, got, ok)
	}
}

func storedTags(t *testing.T, app core.App, restaurant string) []string {
	t.Helper()
	var tags []string
	err := app.DB().Select("tag").From("restaurant_tags").Where(dbx.HashExp{"restaurant": restaurant}).OrderBy("tag").Column(&tags)
	if err != nil {
		t.Fatal(err)
	}
	return tags
}

func TestOverallTriggers(t *testing.T) {
	app := newIndexApp(t)
	ana := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	joe := newRecord(t, app, "users", map[string]any{"email": "joe@example.com", "password": "correct-horse", "name": "Joe"})
	a := newRecord(t, app, "restaurants", map[string]any{"name": "A"})
	b := newRecord(t, app, "restaurants", map[string]any{"name": "B"})
	wantNoOverall(t, app, a.Id, "new restaurant")

	first := newRecord(t, app, "reviews", review(ana, a, scores(8)))
	newRecord(t, app, "reviews", review(joe, a, map[string]any{"space": 7, "service": 7, "food": 7, "price": 7, "cleanliness": 6}))
	wantOverall(t, app, a.Id, 7.4)

	first.Set("food", 3)
	if err := app.Save(first); err != nil {
		t.Fatal(err)
	}
	wantOverall(t, app, a.Id, 6.9)

	first.Set("restaurant", b.Id)
	if err := app.Save(first); err != nil {
		t.Fatal(err)
	}
	wantOverall(t, app, a.Id, 6.8)
	wantOverall(t, app, b.Id, 7)

	if err := app.Delete(first); err != nil {
		t.Fatal(err)
	}
	wantNoOverall(t, app, b.Id, "last review deleted")

	// The reviews of a user go with a cascade delete.
	if err := app.Delete(joe); err != nil {
		t.Fatal(err)
	}
	wantNoOverall(t, app, a.Id, "user deleted")

	if err := app.Delete(a); err != nil {
		t.Fatal(err)
	}
	if _, ok := storedOverall(t, app, a.Id); ok {
		t.Error("deleted restaurant still has a row")
	}
}

func TestFailedSaveKeepsOverall(t *testing.T) {
	app := newIndexApp(t)
	ana := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	joe := newRecord(t, app, "users", map[string]any{"email": "joe@example.com", "password": "correct-horse", "name": "Joe"})
	a := newRecord(t, app, "restaurants", map[string]any{"name": "A"})
	newRecord(t, app, "reviews", review(ana, a, scores(8)))

	err := app.RunInTransaction(func(tx core.App) error {
		c, err := tx.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		r := core.NewRecord(c)
		r.Load(review(joe, a, scores(2)))
		if err := tx.Save(r); err != nil {
			return err
		}
		return errors.New("fail")
	})
	if err == nil {
		t.Fatal("transaction did not fail")
	}
	wantOverall(t, app, a.Id, 8)
}

func TestRawInsertGetsOverallRow(t *testing.T) {
	app := newIndexApp(t)
	if _, err := app.DB().Insert("restaurants", dbx.Params{"id": "rawrestaurant01", "name": "Raw", "tags": `["cheap"]`}).Execute(); err != nil {
		t.Fatal(err)
	}
	wantNoOverall(t, app, "rawrestaurant01", "raw insert")
	if got := storedTags(t, app, "rawrestaurant01"); !slices.Equal(got, []string{"cheap"}) {
		t.Errorf("raw insert tags: %v", got)
	}
}

func TestTagTriggers(t *testing.T) {
	app := newIndexApp(t)
	a := newRecord(t, app, "restaurants", map[string]any{"name": "A", "tags": []string{"pasta", "italian", "pasta"}})
	if got := storedTags(t, app, a.Id); !slices.Equal(got, []string{"italian", "pasta"}) {
		t.Errorf("create: %v", got)
	}

	a.Set("tags", []any{"pizza", 3, map[string]any{"x": "y"}})
	if err := app.Save(a); err != nil {
		t.Fatal(err)
	}
	if got := storedTags(t, app, a.Id); !slices.Equal(got, []string{"pizza"}) {
		t.Errorf("update: %v", got)
	}

	a.Set("tags", "pizza")
	if err := app.Save(a); err != nil {
		t.Fatal(err)
	}
	if got := storedTags(t, app, a.Id); len(got) != 0 {
		t.Errorf("tags that are not an array: %v", got)
	}

	a.Set("tags", []string{"cheap"})
	if err := app.Save(a); err != nil {
		t.Fatal(err)
	}
	a.Set("name", "B")
	if err := app.Save(a); err != nil {
		t.Fatal(err)
	}
	if got := storedTags(t, app, a.Id); !slices.Equal(got, []string{"cheap"}) {
		t.Errorf("name update: %v", got)
	}

	if err := app.Delete(a); err != nil {
		t.Fatal(err)
	}
	if got := storedTags(t, app, a.Id); len(got) != 0 {
		t.Errorf("delete: %v", got)
	}
}
