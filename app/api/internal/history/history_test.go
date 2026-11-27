package history_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/baumac/olive/app/api/internal/history"
	"github.com/baumac/olive/app/api/internal/testapp"
	_ "github.com/baumac/olive/app/api/migrations"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(testapp.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

func newRecord(t *testing.T, app core.App, collection string, fields map[string]any) *core.Record {
	t.Helper()
	c, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(fields)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

// newTag gets the ID of a new tag. The tags field of a parent has tag IDs.
func newTag(t *testing.T, app core.App, name string) string {
	t.Helper()
	return newRecord(t, app, "tags", map[string]any{"name": name}).Id
}

func saveAs(t *testing.T, app core.App, userID string, r *core.Record) {
	t.Helper()
	if err := app.SaveWithContext(history.WithEditor(context.Background(), userID), r); err != nil {
		t.Fatal(err)
	}
}

func versions(t *testing.T, app core.App, k history.Kind, parentID string) []*core.Record {
	t.Helper()
	records, err := app.FindRecordsByFilter(k.Versions, k.Parent+" = {:id}", "number", 0, 0, dbx.Params{"id": parentID})
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func changedOf(t *testing.T, v *core.Record) []string {
	t.Helper()
	var changed []string
	if err := v.UnmarshalJSONField("changed", &changed); err != nil {
		t.Fatal(err)
	}
	return changed
}

func snapshotOf(t *testing.T, v *core.Record) map[string]any {
	t.Helper()
	var snapshot map[string]any
	if err := v.UnmarshalJSONField("snapshot", &snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func wantVersion(t *testing.T, v *core.Record, number int, action, source, editor string, changed ...string) {
	t.Helper()
	if v.GetInt("number") != number || v.GetString("action") != action || v.GetString("source") != source || v.GetString("editor") != editor {
		t.Errorf("version %d %s %s %q, want %d %s %s %q", v.GetInt("number"), v.GetString("action"), v.GetString("source"), v.GetString("editor"),
			number, action, source, editor)
	}
	if got := changedOf(t, v); !slices.Equal(got, changed) {
		t.Errorf("version %d changed %v, want %v", number, got, changed)
	}
}

func TestVersionsOfSaves(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	history.Register(app)
	ana := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse-battery", "name": "Ana"})
	joe := newRecord(t, app, "users", map[string]any{"email": "joe@example.com", "password": "correct-horse-battery", "name": "Joe"})

	c, err := app.FindCollectionByNameOrId("restaurants")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(map[string]any{"name": "Can Solé", "phone": "931 000 000", "address": "Carrer 1", "latitude": 41.4, "longitude": 2.2, "creator": ana.Id})
	saveAs(t, app, ana.Id, r)

	t.Run("a create writes version 1 with the set properties", func(t *testing.T) {
		vs := versions(t, app, history.Restaurants, r.Id)
		if len(vs) != 1 {
			t.Fatalf("%d versions, want 1", len(vs))
		}
		wantVersion(t, vs[0], 1, history.ActionCreate, history.SourceUser, ana.Id, "name", "location", "phone")
		want := map[string]any{
			"name": "Can Solé", "description": nil, "tags": nil,
			"location": map[string]any{"latitude": 41.4, "longitude": 2.2, "address": "Carrer 1"},
			"website":  nil, "phone": "931 000 000", "opening_hours": nil, "price_level": nil,
		}
		if got := snapshotOf(t, vs[0]); !reflect.DeepEqual(got, want) {
			t.Errorf("snapshot %v, want %v", got, want)
		}
	})

	t.Run("an update writes only the changed properties", func(t *testing.T) {
		r.Set("phone", "931 111 111")
		r.Set("tags", []string{newTag(t, app, "tapas")})
		saveAs(t, app, joe.Id, r)
		vs := versions(t, app, history.Restaurants, r.Id)
		if len(vs) != 2 {
			t.Fatalf("%d versions, want 2", len(vs))
		}
		wantVersion(t, vs[1], 2, history.ActionUpdate, history.SourceUser, joe.Id, "tags", "phone")
	})

	t.Run("a save without a change of a property writes nothing", func(t *testing.T) {
		saveAs(t, app, joe.Id, r)
		r.Set("creator", "")
		saveAs(t, app, joe.Id, r)
		if n := len(versions(t, app, history.Restaurants, r.Id)); n != 2 {
			t.Errorf("%d versions, want 2", n)
		}
	})

	t.Run("a save without an editor gets the source admin", func(t *testing.T) {
		r.Set("description", "Small tapas bar")
		if err := app.Save(r); err != nil {
			t.Fatal(err)
		}
		vs := versions(t, app, history.Restaurants, r.Id)
		wantVersion(t, vs[len(vs)-1], 3, history.ActionUpdate, history.SourceAdmin, "", "description")
	})

	t.Run("a failed version write stops the save", func(t *testing.T) {
		r.Set("name", "Can Solé Nou")
		if err := app.SaveWithContext(history.WithEditor(context.Background(), "nouser000000000"), r); err == nil {
			t.Fatal("the save with an unknown editor works")
		}
		stored, err := app.FindRecordById("restaurants", r.Id)
		if err != nil {
			t.Fatal(err)
		}
		if stored.GetString("name") != "Can Solé" {
			t.Errorf("stored name %q, want the old name", stored.GetString("name"))
		}
		r.Set("name", "Can Solé")
	})

	t.Run("the delete of a user clears the editor", func(t *testing.T) {
		if err := app.Delete(joe); err != nil {
			t.Fatal(err)
		}
		vs := versions(t, app, history.Restaurants, r.Id)
		wantVersion(t, vs[1], 2, history.ActionUpdate, history.SourceUser, "", "tags", "phone")
	})

	t.Run("the delete of a parent deletes its versions", func(t *testing.T) {
		if err := app.Delete(r); err != nil {
			t.Fatal(err)
		}
		if n := len(versions(t, app, history.Restaurants, r.Id)); n != 0 {
			t.Errorf("%d versions, want 0", n)
		}
	})
}

func TestVersionsOfRecipes(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	history.Register(app)
	ana := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse-battery", "name": "Ana"})
	c, err := app.FindCollectionByNameOrId("recipes")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(map[string]any{"name": "Dal", "ingredients": []string{"lentils"}, "steps": []string{"Cook."}, "servings": 2, "creator": ana.Id})
	saveAs(t, app, ana.Id, r)
	r.Set("steps", []string{"Rinse.", "Cook."})
	r.Set("servings", 0)
	saveAs(t, app, ana.Id, r)

	vs := versions(t, app, history.Recipes, r.Id)
	if len(vs) != 2 {
		t.Fatalf("%d versions, want 2", len(vs))
	}
	wantVersion(t, vs[0], 1, history.ActionCreate, history.SourceUser, ana.Id, "name", "ingredients", "steps", "servings")
	wantVersion(t, vs[1], 2, history.ActionUpdate, history.SourceUser, ana.Id, "steps", "servings")
	if got := snapshotOf(t, vs[1])["servings"]; got != nil {
		t.Errorf("servings %v, want null", got)
	}
}

func TestParentWithoutVersionGetsBaseline(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	r := newRecord(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "Old", "description": "Before the history"})
	history.Register(app)
	r.Set("description", "After")
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	vs := versions(t, app, history.Restaurants, r.Id)
	if len(vs) != 2 {
		t.Fatalf("%d versions, want 2", len(vs))
	}
	wantVersion(t, vs[0], 1, history.ActionBaseline, history.SourceAdmin, "")
	wantVersion(t, vs[1], 2, history.ActionUpdate, history.SourceAdmin, "", "description")
	if got := snapshotOf(t, vs[0])["description"]; got != "Before the history" {
		t.Errorf("baseline description %v, want the old value", got)
	}
}

func TestEmptyTagsAreNoTags(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	history.Register(app)
	r := newRecord(t, app, "restaurants", map[string]any{"name": "Can Solé", "address": "Carrer 1"})
	for _, tags := range []any{[]string{}, nil, []string{}} {
		r.Set("tags", tags)
		saveAs(t, app, "", r)
	}
	vs := versions(t, app, history.Restaurants, r.Id)
	if len(vs) != 1 {
		t.Fatalf("%d versions, want 1", len(vs))
	}
	if tags := snapshotOf(t, vs[0])["tags"]; tags != nil {
		t.Errorf("snapshot tags %v, want nil", tags)
	}
}

func TestRenameOfTagWritesAdminVersions(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	history.Register(app)
	tapas, cheap := newTag(t, app, "tapas"), newTag(t, app, "cheap")
	r := newRecord(t, app, "restaurants", map[string]any{"name": "Can Solé", "address": "Carrer 1", "tags": []string{tapas, cheap}})
	other := newRecord(t, app, "restaurants", map[string]any{"name": "Bar", "address": "Carrer 2", "tags": []string{cheap}})

	tag, err := app.FindRecordById("tags", tapas)
	if err != nil {
		t.Fatal(err)
	}
	tag.Set("name", "tapes")
	if err := app.Save(tag); err != nil {
		t.Fatal(err)
	}
	vs := versions(t, app, history.Restaurants, r.Id)
	if len(vs) != 2 {
		t.Fatalf("%d versions, want 2", len(vs))
	}
	wantVersion(t, vs[1], 2, history.ActionUpdate, history.SourceAdmin, "", "tags")
	if got := snapshotOf(t, vs[1])["tags"]; !reflect.DeepEqual(got, []any{"tapes", "cheap"}) {
		t.Errorf("snapshot tags %v, want [tapes cheap]", got)
	}
	if n := len(versions(t, app, history.Restaurants, other.Id)); n != 1 {
		t.Errorf("the restaurant without the tag has %d versions, want 1", n)
	}
}
