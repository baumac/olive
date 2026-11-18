package searchindex_test

import (
	"database/sql"
	"errors"
	"slices"
	"testing"

	"github.com/baumac/reviews-app/app/api/internal/searchindex"
	"github.com/baumac/reviews-app/app/api/internal/testapp"
	_ "github.com/baumac/reviews-app/app/api/migrations"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

type env struct {
	app  *tests.TestApp
	user *core.Record
}

// newEnv starts a PocketBase app with an empty database, the migrations of
// this project, and the hooks of searchindex.
func newEnv(t *testing.T) *env {
	t.Helper()
	app, err := tests.NewTestApp(testapp.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	searchindex.Register(app)

	e := &env{app: app}
	e.user = e.save(t, "users", map[string]any{
		"email": "ana@example.com", "password": "correct-horse-battery", "name": "Ana",
	})
	return e
}

func (e *env) trySave(t *testing.T, collection string, fields map[string]any) (*core.Record, error) {
	t.Helper()
	c, err := e.app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(fields)
	return r, e.app.Save(r)
}

func (e *env) save(t *testing.T, collection string, fields map[string]any) *core.Record {
	t.Helper()
	r, err := e.trySave(t, collection, fields)
	if err != nil {
		t.Fatalf("save %s: %v", collection, err)
	}
	return r
}

func restaurantFields(name, description string, tags any) map[string]any {
	return map[string]any{"name": name, "description": description, "tags": tags}
}

func (e *env) reviewFields(restaurant *core.Record, title, feedback string) map[string]any {
	return map[string]any{
		"restaurant": restaurant.Id, "creator": e.user.Id, "title": title, "feedback": feedback,
		"space": 8, "service": 8, "food": 8, "price": 8, "cleanliness": 8,
	}
}

func (e *env) restaurant(t *testing.T, name, description string, tags any) *core.Record {
	t.Helper()
	return e.save(t, "restaurants", restaurantFields(name, description, tags))
}

func (e *env) review(t *testing.T, restaurant *core.Record, title, feedback string) *core.Record {
	t.Helper()
	return e.save(t, "reviews", e.reviewFields(restaurant, title, feedback))
}

func (e *env) update(t *testing.T, r *core.Record, fields map[string]any) {
	t.Helper()
	r.Load(fields)
	if err := e.app.Save(r); err != nil {
		t.Fatalf("update %s: %v", r.Id, err)
	}
}

func (e *env) stored(t *testing.T, r *core.Record) *core.Record {
	t.Helper()
	stored, err := e.app.FindRecordById(r.Collection(), r.Id)
	if err != nil {
		t.Fatalf("find %s: %v", r.Id, err)
	}
	return stored
}

type restaurantRow struct {
	RowID       int    `db:"rowid"`
	Name        string `db:"name"`
	Description string `db:"description"`
	Tags        string `db:"tags"`
}

type reviewRow struct {
	RowID          int    `db:"rowid"`
	Title          string `db:"title"`
	Feedback       string `db:"feedback"`
	RestaurantName string `db:"restaurant_name"`
}

// row reads the index row of id into dest. It returns false if no row exists.
func (e *env) row(t *testing.T, table, id string, dest any) bool {
	t.Helper()
	err := e.app.DB().Select("rowid", "*").From(table).Where(dbx.HashExp{"id": id}).One(dest)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	if err != nil {
		t.Fatalf("read %s row %s: %v", table, id, err)
	}
	return true
}

func (e *env) mustRow(t *testing.T, table, id string, dest any) {
	t.Helper()
	if !e.row(t, table, id, dest) {
		t.Fatalf("no %s row for %s", table, id)
	}
}

func (e *env) count(t *testing.T, table string) int {
	t.Helper()
	var n int
	if err := e.app.DB().Select("COUNT(*)").From(table).Row(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// match returns the IDs of the index rows that match the FTS5 query q.
func (e *env) match(t *testing.T, table, q string) []string {
	t.Helper()
	var ids []string
	err := e.app.DB().NewQuery("SELECT [[id]] FROM {{" + table + "}} WHERE {{" + table + "}} MATCH {:q} ORDER BY [[id]]").
		Bind(dbx.Params{"q": q}).Column(&ids)
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

// breakIndex renames table until the test ends. Writes to the index then fail.
func (e *env) breakIndex(t *testing.T, table string) {
	t.Helper()
	rename := func(from, to string) error {
		_, err := e.app.DB().NewQuery("ALTER TABLE {{" + from + "}} RENAME TO {{" + to + "}}").Execute()
		return err
	}
	if err := rename(table, table+"_off"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := rename(table+"_off", table); err != nil {
			t.Error(err)
		}
	})
}

func TestCreate(t *testing.T) {
	e := newEnv(t)

	t.Run("restaurant", func(t *testing.T) {
		r := e.restaurant(t, "Casa Pepe", "Tapas near the beach", []string{"date-night", "cheap"})
		var row restaurantRow
		e.mustRow(t, searchindex.Restaurants, r.Id, &row)
		want := restaurantRow{RowID: row.RowID, Name: "Casa Pepe", Description: "Tapas near the beach", Tags: "date-night cheap"}
		if row != want {
			t.Errorf("row %+v, want %+v", row, want)
		}
	})

	for _, tc := range []struct {
		name string
		tags any
	}{
		{"restaurant without tags", nil},
		{"restaurant with tags that are not a list of strings", map[string]any{"a": 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := e.restaurant(t, tc.name, "", tc.tags)
			var row restaurantRow
			e.mustRow(t, searchindex.Restaurants, r.Id, &row)
			if row.Tags != "" {
				t.Errorf("tags %q, want empty", row.Tags)
			}
		})
	}

	t.Run("review", func(t *testing.T) {
		cafe := e.restaurant(t, "Café Olé", "", nil)
		rv := e.review(t, cafe, "Nice tea", "Green tea with mint")
		var row reviewRow
		e.mustRow(t, searchindex.Reviews, rv.Id, &row)
		want := reviewRow{RowID: row.RowID, Title: "Nice tea", Feedback: "Green tea with mint", RestaurantName: "Café Olé"}
		if row != want {
			t.Errorf("row %+v, want %+v", row, want)
		}
	})
}

func TestSearchTerms(t *testing.T) {
	e := newEnv(t)
	pepe := e.restaurant(t, "Casa Pepe", "", []string{"date-night"})
	cafe := e.restaurant(t, "Café Olé", "", nil)
	rv := e.review(t, cafe, "Nice tea", "")

	for _, tc := range []struct {
		name, table, q string
		want           []string
	}{
		{"a tag splits into words", searchindex.Restaurants, "night", []string{pepe.Id}},
		{"accents do not matter", searchindex.Restaurants, "cafe", []string{cafe.Id}},
		{"case does not matter", searchindex.Restaurants, "PEPE", []string{pepe.Id}},
		{"a prefix matches", searchindex.Restaurants, "pep*", []string{pepe.Id}},
		{"a review matches its restaurant name", searchindex.Reviews, "ole", []string{rv.Id}},
		{"no match", searchindex.Restaurants, "sushi", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := e.match(t, tc.table, tc.q)
			if !slices.Equal(got, tc.want) {
				t.Errorf("%q: %v, want %v", tc.q, got, tc.want)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	e := newEnv(t)
	cafe := e.restaurant(t, "Café Olé", "Tea", []string{"quiet"})
	rv := e.review(t, cafe, "Nice tea", "Green tea")

	t.Run("a text change replaces the restaurant row", func(t *testing.T) {
		e.update(t, cafe, map[string]any{"description": "Tea and cake", "tags": []string{"quiet", "cozy"}})
		var row restaurantRow
		e.mustRow(t, searchindex.Restaurants, cafe.Id, &row)
		if row.Description != "Tea and cake" || row.Tags != "quiet cozy" {
			t.Errorf("row %+v", row)
		}
		if n := e.count(t, searchindex.Restaurants); n != 1 {
			t.Errorf("%d restaurant rows, want 1", n)
		}
	})

	t.Run("tags that are not a list of strings", func(t *testing.T) {
		e.update(t, cafe, map[string]any{"tags": map[string]any{"a": 1}})
		var row restaurantRow
		e.mustRow(t, searchindex.Restaurants, cafe.Id, &row)
		if row.Tags != "" {
			t.Errorf("tags %q, want empty", row.Tags)
		}
	})

	t.Run("a text change replaces the review row", func(t *testing.T) {
		e.update(t, rv, map[string]any{"title": "Great tea", "feedback": "Black tea"})
		var row reviewRow
		e.mustRow(t, searchindex.Reviews, rv.Id, &row)
		if row.Title != "Great tea" || row.Feedback != "Black tea" {
			t.Errorf("row %+v", row)
		}
		if n := e.count(t, searchindex.Reviews); n != 1 {
			t.Errorf("%d review rows, want 1", n)
		}
	})

	t.Run("a change back to the loaded value with the same record", func(t *testing.T) {
		loaded := e.stored(t, rv)
		e.update(t, loaded, map[string]any{"title": "Nice tea"})
		e.update(t, loaded, map[string]any{"title": "Great tea"})
		var row reviewRow
		e.mustRow(t, searchindex.Reviews, rv.Id, &row)
		if row.Title != "Great tea" {
			t.Errorf("title %q, want Great tea", row.Title)
		}
		e.update(t, rv, map[string]any{"title": "Nice tea"})
		e.mustRow(t, searchindex.Reviews, rv.Id, &row)
		if row.Title != "Nice tea" {
			t.Errorf("title %q, want Nice tea", row.Title)
		}
	})

	t.Run("a change of other fields keeps the row", func(t *testing.T) {
		// FTS5 gives a new row the largest rowid plus 1. A later row makes a
		// replaced row get a new rowid.
		e.review(t, e.restaurant(t, "Later", "", nil), "Later", "")
		var before, after reviewRow
		e.mustRow(t, searchindex.Reviews, rv.Id, &before)
		e.update(t, rv, map[string]any{"food": 3})
		e.mustRow(t, searchindex.Reviews, rv.Id, &after)
		if after.RowID != before.RowID {
			t.Errorf("the review row changed from %d to %d", before.RowID, after.RowID)
		}

		var rBefore, rAfter restaurantRow
		e.mustRow(t, searchindex.Restaurants, cafe.Id, &rBefore)
		e.update(t, cafe, map[string]any{"creator": e.user.Id})
		e.mustRow(t, searchindex.Restaurants, cafe.Id, &rAfter)
		if rAfter.RowID != rBefore.RowID {
			t.Errorf("the restaurant row changed from %d to %d", rBefore.RowID, rAfter.RowID)
		}
	})

	t.Run("a rename updates the reviews of the restaurant only", func(t *testing.T) {
		other := e.restaurant(t, "Other", "", nil)
		otherReview := e.review(t, other, "Fine", "")
		e.update(t, cafe, map[string]any{"name": "Teahouse"})

		var row, otherRow reviewRow
		e.mustRow(t, searchindex.Reviews, rv.Id, &row)
		if row.RestaurantName != "Teahouse" {
			t.Errorf("restaurant name %q, want Teahouse", row.RestaurantName)
		}
		e.mustRow(t, searchindex.Reviews, otherReview.Id, &otherRow)
		if otherRow.RestaurantName != "Other" {
			t.Errorf("other restaurant name %q, want Other", otherRow.RestaurantName)
		}
	})

	t.Run("a move to another restaurant changes the restaurant name", func(t *testing.T) {
		pepe := e.restaurant(t, "Casa Pepe", "", nil)
		e.update(t, rv, map[string]any{"restaurant": pepe.Id})
		var row reviewRow
		e.mustRow(t, searchindex.Reviews, rv.Id, &row)
		if row.RestaurantName != "Casa Pepe" {
			t.Errorf("restaurant name %q, want Casa Pepe", row.RestaurantName)
		}
	})
}

func TestDelete(t *testing.T) {
	e := newEnv(t)

	t.Run("review", func(t *testing.T) {
		rv := e.review(t, e.restaurant(t, "Café Olé", "", nil), "Nice tea", "")
		if err := e.app.Delete(rv); err != nil {
			t.Fatal(err)
		}
		if e.row(t, searchindex.Reviews, rv.Id, &reviewRow{}) {
			t.Error("the index row stays")
		}
	})

	t.Run("restaurant", func(t *testing.T) {
		r := e.restaurant(t, "No Reviews", "", nil)
		if err := e.app.Delete(r); err != nil {
			t.Fatal(err)
		}
		if e.row(t, searchindex.Restaurants, r.Id, &restaurantRow{}) {
			t.Error("the index row stays")
		}
	})

	t.Run("cascade delete of the creator", func(t *testing.T) {
		e.review(t, e.restaurant(t, "Casa Pepe", "", nil), "Good", "")
		e.review(t, e.restaurant(t, "Sushi Bar", "", nil), "Bad", "")
		if err := e.app.Delete(e.user); err != nil {
			t.Fatal(err)
		}
		if n := e.count(t, searchindex.Reviews); n != 0 {
			t.Errorf("%d review rows after the cascade delete", n)
		}
	})
}

func TestFailedIndexChangeKeepsTheRecord(t *testing.T) {
	t.Run("restaurant create", func(t *testing.T) {
		e := newEnv(t)
		e.breakIndex(t, searchindex.Restaurants)
		if _, err := e.trySave(t, "restaurants", restaurantFields("Not Indexed", "", nil)); err == nil {
			t.Error("a save without the index works")
		}
		if n, err := e.app.CountRecords("restaurants"); err != nil || n != 0 {
			t.Errorf("%d restaurants without an index row: %v", n, err)
		}
	})

	t.Run("review create", func(t *testing.T) {
		e := newEnv(t)
		cafe := e.restaurant(t, "Café Olé", "", nil)
		e.breakIndex(t, searchindex.Reviews)
		if _, err := e.trySave(t, "reviews", e.reviewFields(cafe, "Not Indexed", "")); err == nil {
			t.Error("a save without the index works")
		}
		if n, err := e.app.CountRecords("reviews"); err != nil || n != 0 {
			t.Errorf("%d reviews without an index row: %v", n, err)
		}
	})

	t.Run("restaurant update", func(t *testing.T) {
		e := newEnv(t)
		cafe := e.restaurant(t, "Café Olé", "", nil)
		e.breakIndex(t, searchindex.Restaurants)
		cafe.Set("description", "Tea")
		if err := e.app.Save(cafe); err == nil {
			t.Error("an update without the index works")
		}
		if got := e.stored(t, cafe).GetString("description"); got != "" {
			t.Errorf("stored description %q, want empty", got)
		}
	})

	t.Run("review update", func(t *testing.T) {
		e := newEnv(t)
		rv := e.review(t, e.restaurant(t, "Café Olé", "", nil), "Nice tea", "")
		e.breakIndex(t, searchindex.Reviews)
		rv.Set("title", "Great tea")
		if err := e.app.Save(rv); err == nil {
			t.Error("an update without the index works")
		}
		if got := e.stored(t, rv).GetString("title"); got != "Nice tea" {
			t.Errorf("stored title %q, want Nice tea", got)
		}
	})

	t.Run("rename in the reviews", func(t *testing.T) {
		e := newEnv(t)
		cafe := e.restaurant(t, "Café Olé", "", nil)
		e.review(t, cafe, "Nice tea", "")
		e.breakIndex(t, searchindex.Reviews)
		cafe.Set("name", "Teahouse")
		if err := e.app.Save(cafe); err == nil {
			t.Error("a rename without the review index works")
		}
		if got := e.stored(t, cafe).GetString("name"); got != "Café Olé" {
			t.Errorf("stored name %q, want Café Olé", got)
		}
		var row restaurantRow
		e.mustRow(t, searchindex.Restaurants, cafe.Id, &row)
		if row.Name != "Café Olé" {
			t.Errorf("restaurant row name %q, want Café Olé", row.Name)
		}
	})

	t.Run("review delete", func(t *testing.T) {
		e := newEnv(t)
		rv := e.review(t, e.restaurant(t, "Café Olé", "", nil), "Nice tea", "")
		e.breakIndex(t, searchindex.Reviews)
		if err := e.app.Delete(rv); err == nil {
			t.Error("a delete without the index works")
		}
		e.stored(t, rv)
	})

	t.Run("restaurant delete", func(t *testing.T) {
		e := newEnv(t)
		r := e.restaurant(t, "No Reviews", "", nil)
		e.breakIndex(t, searchindex.Restaurants)
		if err := e.app.Delete(r); err == nil {
			t.Error("a delete without the index works")
		}
		e.stored(t, r)
	})
}

func TestPut(t *testing.T) {
	e := newEnv(t)
	cafe := e.restaurant(t, "Café Olé", "", nil)
	rv := e.review(t, cafe, "Nice tea", "")

	t.Run("replaces the row", func(t *testing.T) {
		if _, err := e.app.DB().Delete(searchindex.Restaurants, nil).Execute(); err != nil {
			t.Fatal(err)
		}
		for range 2 {
			if err := searchindex.Put(e.app, cafe); err != nil {
				t.Fatal(err)
			}
		}
		if n := e.count(t, searchindex.Restaurants); n != 1 {
			t.Errorf("%d rows, want 1", n)
		}
	})

	t.Run("review of a missing restaurant keeps the old row", func(t *testing.T) {
		moved := e.stored(t, rv)
		moved.Set("restaurant", "missing00000000")
		moved.Set("title", "Moved")
		if err := searchindex.Put(e.app, moved); err == nil {
			t.Error("no error")
		}
		var row reviewRow
		e.mustRow(t, searchindex.Reviews, rv.Id, &row)
		if row.Title != "Nice tea" {
			t.Errorf("title %q, want Nice tea", row.Title)
		}
	})
	t.Run("record of a collection without an index", func(t *testing.T) {
		users, err := e.app.FindCachedCollectionByNameOrId("users")
		if err != nil {
			t.Fatal(err)
		}
		if err := searchindex.Put(e.app, core.NewRecord(users)); err == nil {
			t.Error("no error")
		}
	})
}

func TestRecipeRows(t *testing.T) {
	e := newEnv(t)
	dal := e.save(t, "recipes", map[string]any{
		"name": "Dal", "description": "Red lentils", "tags": []string{"vegan"},
		"ingredients": []string{"1 cup of lentils", "2 onions"}, "steps": []string{"Boil."},
	})
	rv := e.save(t, "recipe_reviews", map[string]any{
		"recipe": dal.Id, "creator": e.user.Id, "title": "Warm", "feedback": "Good on rice",
		"taste": 8, "ease": 8, "instructions": 8, "value": 8,
	})
	var recipe struct {
		Name        string `db:"name"`
		Tags        string `db:"tags"`
		Ingredients string `db:"ingredients"`
	}
	e.mustRow(t, "recipes_fts", dal.Id, &recipe)
	if recipe.Name != "Dal" || recipe.Tags != "vegan" || recipe.Ingredients != "1 cup of lentils 2 onions" {
		t.Errorf("recipe row %+v", recipe)
	}

	var review struct {
		Title      string `db:"title"`
		RecipeName string `db:"recipe_name"`
	}
	e.mustRow(t, "recipe_reviews_fts", rv.Id, &review)
	if review.Title != "Warm" || review.RecipeName != "Dal" {
		t.Errorf("review row %+v", review)
	}

	e.update(t, dal, map[string]any{"name": "Tarka dal", "ingredients": []string{"lentils"}})
	e.mustRow(t, "recipes_fts", dal.Id, &recipe)
	e.mustRow(t, "recipe_reviews_fts", rv.Id, &review)
	if recipe.Ingredients != "lentils" || review.RecipeName != "Tarka dal" {
		t.Errorf("after the change: recipe %+v, review %+v", recipe, review)
	}

	if err := e.app.Delete(rv); err != nil {
		t.Fatal(err)
	}
	if e.row(t, "recipe_reviews_fts", rv.Id, &review) {
		t.Error("the row of the deleted review stays")
	}
}
