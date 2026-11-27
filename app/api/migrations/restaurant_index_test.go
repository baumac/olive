package migrations

import (
	"database/sql"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func scores(n int) map[string]any {
	return map[string]any{"space": n, "service": n, "food": n, "price": n}
}

func review(user, restaurant *core.Record, s map[string]any) map[string]any {
	data := map[string]any{"restaurant": restaurant.Id, "creator": user.Id, "title": "T", "date": "2024-11-06"}
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

func wantReviewers(t *testing.T, app core.App, restaurant string, want int) {
	t.Helper()
	var got int
	err := app.DB().Select("reviewers").From("restaurant_overall").Where(dbx.HashExp{"restaurant": restaurant}).Row(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("reviewers of %s: %d, want %d", restaurant, got, want)
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
	t.Parallel()
	app := newApp(t)
	ana := newUser(t, app, "ana")
	joe := newUser(t, app, "joe")
	a := newRecord(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "A"})
	b := newRecord(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "B"})
	wantNoOverall(t, app, a.Id, "new restaurant")
	wantReviewers(t, app, a.Id, 0)

	first := newRecord(t, app, "reviews", review(ana, a, scores(8)))
	newRecord(t, app, "reviews", review(joe, a, map[string]any{"space": 7, "service": 7, "food": 7, "price": 3}))
	wantOverall(t, app, a.Id, 7)
	wantReviewers(t, app, a.Id, 2)

	first.Set("food", 4)
	if err := app.Save(first); err != nil {
		t.Fatal(err)
	}
	wantOverall(t, app, a.Id, 6.5)

	first.Set("restaurant", b.Id)
	if err := app.Save(first); err != nil {
		t.Fatal(err)
	}
	wantOverall(t, app, a.Id, 6)
	wantOverall(t, app, b.Id, 7)
	wantReviewers(t, app, a.Id, 1)
	wantReviewers(t, app, b.Id, 1)

	if err := app.Delete(first); err != nil {
		t.Fatal(err)
	}
	wantNoOverall(t, app, b.Id, "last review deleted")
	wantReviewers(t, app, b.Id, 0)

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

func TestOverallCountsEachReviewerOnce(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	ana := newUser(t, app, "ana")
	ben := newUser(t, app, "ben")
	a := newRecord(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "A"})
	var visits []*core.Record
	for range 4 {
		visits = append(visits, newRecord(t, app, "reviews", review(ana, a, scores(10))))
	}
	newRecord(t, app, "reviews", review(ben, a, map[string]any{"space": 6, "service": 6, "food": 2, "price": 10}))
	wantOverall(t, app, a.Id, 8)
	wantReviewers(t, app, a.Id, 2)
	var count int
	var space, food, price float64
	err := app.DB().Select("review_count", "space", "food", "price").From("restaurant_overall").Where(dbx.HashExp{"restaurant": a.Id}).Row(&count, &space, &food, &price)
	if err != nil || count != 5 || space != 8 || food != 6 || price != 10 {
		t.Errorf("review_count %d, space %v, food %v, price %v, error %v", count, space, food, price, err)
	}

	// Ana has 10, 10, and 10. Ben has 10 and 6.
	visits[0].Set("creator", ben.Id)
	if err := app.Save(visits[0]); err != nil {
		t.Fatal(err)
	}
	wantOverall(t, app, a.Id, 9)
	wantReviewers(t, app, a.Id, 2)

	if err := app.Delete(ben); err != nil {
		t.Fatal(err)
	}
	wantOverall(t, app, a.Id, 10)
	wantReviewers(t, app, a.Id, 1)
}

func TestFailedSaveKeepsOverall(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	ana := newUser(t, app, "ana")
	joe := newUser(t, app, "joe")
	a := newRecord(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "A"})
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
	t.Parallel()
	app := newApp(t)
	cheap := newTag(t, app, "cheap")
	if _, err := app.DB().Insert("restaurants", dbx.Params{"id": "rawrestaurant01", "name": "Raw", "tags": `["` + cheap + `"]`}).Execute(); err != nil {
		t.Fatal(err)
	}
	wantNoOverall(t, app, "rawrestaurant01", "raw insert")
	wantReviewers(t, app, "rawrestaurant01", 0)
	if got := storedTags(t, app, "rawrestaurant01"); !slices.Equal(got, []string{cheap}) {
		t.Errorf("raw insert tags: %v", got)
	}
}

func TestTagTriggers(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	pasta, italian, pizza, cheap := newTag(t, app, "pasta"), newTag(t, app, "italian"), newTag(t, app, "pizza"), newTag(t, app, "cheap")
	a := newRecord(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "A", "tags": []string{pasta, italian, pasta}})
	if got := storedTags(t, app, a.Id); !slices.Equal(got, sorted(italian, pasta)) {
		t.Errorf("create: %v", got)
	}

	a.Set("tags", []string{pizza})
	if err := app.Save(a); err != nil {
		t.Fatal(err)
	}
	if got := storedTags(t, app, a.Id); !slices.Equal(got, []string{pizza}) {
		t.Errorf("update: %v", got)
	}

	a.Set("tags", []string{})
	if err := app.Save(a); err != nil {
		t.Fatal(err)
	}
	if got := storedTags(t, app, a.Id); len(got) != 0 {
		t.Errorf("no tags: %v", got)
	}

	a.Set("tags", []string{cheap})
	if err := app.Save(a); err != nil {
		t.Fatal(err)
	}
	a.Set("name", "B")
	if err := app.Save(a); err != nil {
		t.Fatal(err)
	}
	if got := storedTags(t, app, a.Id); !slices.Equal(got, []string{cheap}) {
		t.Errorf("name update: %v", got)
	}

	if err := app.Delete(a); err != nil {
		t.Fatal(err)
	}
	if got := storedTags(t, app, a.Id); len(got) != 0 {
		t.Errorf("delete: %v", got)
	}
}

func TestRatingOfReviewersUsesTheCreatorIndex(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	for _, c := range []struct {
		kind subjectKind
		want string
	}{
		{restaurantKind, "SEARCH reviews USING INDEX idx_reviews_restaurant_creator (restaurant=?)"},
		{recipeKind, "SEARCH recipe_reviews USING INDEX idx_recipe_reviews_recipe_creator (recipe=?)"},
	} {
		var rows []struct {
			Detail string `db:"detail"`
		}
		q := app.DB().NewQuery("EXPLAIN QUERY PLAN " + c.kind.byReviewer("", "{:id}")).Bind(dbx.Params{"id": "k3x9m2p7q1w8z4a"})
		if err := q.All(&rows); err != nil {
			t.Fatal(err)
		}
		var plan []string
		for _, r := range rows {
			plan = append(plan, r.Detail)
		}
		if got := strings.Join(plan, "\n"); !strings.Contains(got, c.want) {
			t.Errorf("%s plan has no %q:\n%s", c.kind.reviews, c.want, got)
		}
	}
}
