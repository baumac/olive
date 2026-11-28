package usercleanup_test

import (
	"testing"
	"time"

	"github.com/baumac/olive/app/api/internal/testapp"
	"github.com/baumac/olive/app/api/internal/usercleanup"
	_ "github.com/baumac/olive/app/api/migrations"
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
	t.Parallel()
	app, err := tests.NewTestApp(testapp.Dir(t))
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
	cook := user("cook", false)
	taster := user("taster", false)
	saver := user("saver", false)
	cellar := user("cellar", false)
	sipper := user("sipper", false)
	keeper := user("keeper", false)
	restaurant := save(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "Luigi's", "creator": owner.Id})
	save(t, app, "reviews", map[string]any{
		"restaurant": restaurant.Id, "creator": reviewer.Id, "date": "2024-11-06", "title": "Good",
		"space": 7, "service": 7, "food": 7, "price": 7,
	})
	recipe := save(t, app, "recipes", map[string]any{"name": "Dal", "creator": cook.Id, "ingredients": []string{"lentils"}, "steps": []string{"Boil."}})
	save(t, app, "recipe_reviews", map[string]any{
		"recipe": recipe.Id, "creator": taster.Id, "date": "2024-11-06", "title": "Good",
		"taste": 7, "ease": 7, "instructions": 7, "value": 7,
	})
	list := save(t, app, "lists", map[string]any{"creator": lister.Id, "name": "Favorites", "kind": "favorites"})
	save(t, app, "list_items", map[string]any{"list": list.Id, "restaurant": restaurant.Id})
	save(t, app, "lists", map[string]any{"creator": waiting.Id, "name": "Favorites", "kind": "favorites"})
	saved := save(t, app, "lists", map[string]any{"creator": saver.Id, "name": "Favorites", "kind": "favorites"})
	save(t, app, "list_recipes", map[string]any{"list": saved.Id, "recipe": recipe.Id})
	item := save(t, app, "pantry_items", map[string]any{"name": "Gouda", "category": "cheese", "creator": cellar.Id})
	save(t, app, "pantry_item_reviews", map[string]any{
		"pantry_item": item.Id, "creator": sipper.Id, "date": "2024-11-06", "title": "Good",
		"taste": 7, "aroma": 7, "texture": 7, "value": 7,
	})
	kept := save(t, app, "lists", map[string]any{"creator": keeper.Id, "name": "Favorites", "kind": "favorites"})
	save(t, app, "list_pantry_items", map[string]any{"list": kept.Id, "pantry_item": item.Id})

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
	for _, r := range []*core.Record{approved, reviewer, owner, lister, cook, taster, saver, cellar, sipper, keeper} {
		if !exists(r) {
			t.Errorf("%s was deleted", r.GetString("name"))
		}
	}
	if lists, err := app.CountRecords("lists"); err != nil || lists != 3 {
		t.Errorf("%d lists (%v), want only the lists of the lister, the saver, and the keeper", lists, err)
	}
}

func runJob(t *testing.T, app core.App) {
	t.Helper()
	for _, job := range app.Cron().Jobs() {
		if job.Id() == usercleanup.JobID {
			job.Run()
			return
		}
	}
	t.Fatalf("no job %s", usercleanup.JobID)
}

func TestRegisteredJob(t *testing.T) {
	t.Parallel()
	app, err := tests.NewTestApp(testapp.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	usercleanup.Register(app)

	oldUser := func(name string) *core.Record {
		r := save(t, app, "users", map[string]any{"email": name + "@example.com", "password": "correct-horse", "name": name})
		if _, err := app.DB().NewQuery("UPDATE users SET created = '2000-01-01 00:00:00.000Z' WHERE id = {:id}").
			Bind(map[string]any{"id": r.Id}).Execute(); err != nil {
			t.Fatal(err)
		}
		return r
	}
	exists := func(r *core.Record) bool {
		_, err := app.FindRecordById("users", r.Id)
		return err == nil
	}

	waiting := oldUser("waiting")
	runJob(t, app)
	if exists(waiting) {
		t.Error("the job did not delete the old user without approval")
	}

	// The job logs the error. It deletes no user.
	other := oldUser("other")
	if _, err := app.DB().NewQuery("DROP TABLE list_items").Execute(); err != nil {
		t.Fatal(err)
	}
	runJob(t, app)
	if !exists(other) {
		t.Error("the job deleted a user after a failed query")
	}
}
