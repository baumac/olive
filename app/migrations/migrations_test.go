package migrations

import (
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/dbutils"
)

func TestDownRestoresUsers(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	if idx, ok := dbutils.FindSingleColumnUniqueIndex(users.Indexes, "email"); !ok || idx.Columns[0].Collate != "NOCASE" {
		t.Errorf("up: email index %+v", idx)
	}

	if err := down(app); err != nil {
		t.Fatal(err)
	}
	if _, err := app.FindCollectionByNameOrId("restaurants"); err == nil {
		t.Error("restaurants still exists")
	}
	users, err = app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	if users.AuthRule == nil || users.ListRule == nil {
		t.Errorf("rules are not restored: auth %v, list %v", users.AuthRule, users.ListRule)
	}
	if idx, ok := dbutils.FindSingleColumnUniqueIndex(users.Indexes, "email"); !ok || idx.Columns[0].Collate != "" {
		t.Errorf("down: email index %+v", idx)
	}
}

func TestRateLimitsAreOn(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if !app.Settings().RateLimits.Enabled {
		t.Error("the PocketBase rate limits are off")
	}
	if _, ok := app.Settings().RateLimits.FindRateLimitRule([]string{"POST /csp-report"}); !ok {
		t.Error("no rate limit rule for the CSP reports")
	}
}

func TestUsersHaveNoFileFields(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range users.Fields {
		if f.Type() == core.FieldTypeFile {
			t.Errorf("users has the file field %s", f.GetName())
		}
	}
}

func TestSingleReviewKindKeepsBasicReviews(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	redo := downTo(t, app, "1794268803_single_review_kind.go")

	user := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	restaurant := newRecord(t, app, "restaurants", map[string]any{"name": "Luigi's"})
	review := newRecord(t, app, "reviews", map[string]any{
		"restaurant": restaurant.Id, "creator": user.Id, "kind": "basic", "rating": 4, "title": "Good",
	})

	redo()
	got, err := app.FindRecordById("reviews", review.Id)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"space", "service", "food", "price", "cleanliness"} {
		if v := got.GetInt(name); v != 8 {
			t.Errorf("%s is %d, want 8", name, v)
		}
	}
	reviews, err := app.FindCollectionByNameOrId("reviews")
	if err != nil {
		t.Fatal(err)
	}
	if reviews.Fields.GetByName("kind") != nil || reviews.Fields.GetByName("rating") != nil {
		t.Error("reviews still has kind or rating")
	}
}

func TestCleanlinessCopiesSpace(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	redo := downTo(t, app, "1794268804_add_review_cleanliness.go")

	user := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	restaurant := newRecord(t, app, "restaurants", map[string]any{"name": "Luigi's"})
	review := newRecord(t, app, "reviews", map[string]any{
		"restaurant": restaurant.Id, "creator": user.Id, "space": 3, "service": 9, "food": 8, "price": 6, "title": "Good",
	})

	redo()
	got, err := app.FindRecordById("reviews", review.Id)
	if err != nil {
		t.Fatal(err)
	}
	if v := got.GetInt("cleanliness"); v != 3 {
		t.Errorf("cleanliness is %d, want 3", v)
	}
}

// downTo reverts the app migrations from the last one back to file. The
// returned function applies them again in order.
func downTo(t *testing.T, app core.App, file string) func() {
	t.Helper()
	items := core.AppMigrations.Items()
	i := slices.IndexFunc(items, func(m *core.Migration) bool { return m.File == file })
	if i < 0 {
		t.Fatalf("no migration %s", file)
	}
	for j := len(items) - 1; j >= i; j-- {
		if err := items[j].Down(app); err != nil {
			t.Fatalf("down %s: %v", items[j].File, err)
		}
	}
	return func() {
		t.Helper()
		for _, m := range items[i:] {
			if err := m.Up(app); err != nil {
				t.Fatalf("up %s: %v", m.File, err)
			}
		}
	}
}

func newRecord(t *testing.T, app core.App, collection string, data map[string]any) *core.Record {
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
