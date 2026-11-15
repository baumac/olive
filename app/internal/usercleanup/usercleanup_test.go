package usercleanup_test

import (
	"testing"
	"time"

	"github.com/baumac/reviews-app/app/internal/usercleanup"
	_ "github.com/baumac/reviews-app/app/migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func save(t *testing.T, app core.App, collection string, data map[string]any) *core.Record {
	t.Helper()
	c, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(data)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRun(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	user := func(name string, approved bool) *core.Record {
		return save(t, app, "users", map[string]any{"email": name + "@example.com", "password": "correct-horse", "name": name, "approved": approved})
	}
	approved := user("approved", true)
	waiting := user("waiting", false)
	reviewer := user("reviewer", false)
	owner := user("owner", false)
	lister := user("lister", false)
	restaurant := save(t, app, "restaurants", map[string]any{"name": "Luigi's", "creator": owner.Id})
	save(t, app, "reviews", map[string]any{
		"restaurant": restaurant.Id, "creator": reviewer.Id, "title": "Good",
		"space": 7, "service": 7, "food": 7, "price": 7, "cleanliness": 7,
	})
	list := save(t, app, "lists", map[string]any{"creator": lister.Id, "name": "Favorites", "kind": "favorites"})
	save(t, app, "list_items", map[string]any{"list": list.Id, "restaurant": restaurant.Id})
	save(t, app, "lists", map[string]any{"creator": waiting.Id, "name": "Favorites", "kind": "favorites"})

	exists := func(r *core.Record) bool {
		_, err := app.FindRecordById("users", r.Id)
		return err == nil
	}
	now := time.Now()

	if n, err := usercleanup.Run(app, now.Add(usercleanup.MaxAge-time.Hour)); err != nil || n != 0 {
		t.Fatalf("before the maximum age: %d users, error %v", n, err)
	}
	n, err := usercleanup.Run(app, now.Add(usercleanup.MaxAge+time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("after the maximum age: %d users, error %v, want 1", n, err)
	}
	if exists(waiting) {
		t.Error("the user without approval and without data stays")
	}
	for _, r := range []*core.Record{approved, reviewer, owner, lister} {
		if !exists(r) {
			t.Errorf("%s was deleted", r.GetString("name"))
		}
	}
	if lists, err := app.CountRecords("lists"); err != nil || lists != 1 {
		t.Errorf("%d lists (%v), want only the list of the lister", lists, err)
	}
}
