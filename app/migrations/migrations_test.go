package migrations

import (
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
