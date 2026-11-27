package migrations

import (
	"slices"
	"testing"

	"github.com/baumac/olive/app/api/internal/testapp"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/dbutils"
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

func newUser(t *testing.T, app core.App, name string) *core.Record {
	t.Helper()
	return newRecord(t, app, "users", map[string]any{"email": name + "@example.com", "password": "correct-horse", "name": name})
}

func TestDownRestoresTheDefaults(t *testing.T) {
	t.Parallel()
	app := newApp(t)
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
	for _, name := range []string{
		"restaurants", "reviews", "review_photos", "lists", "list_items", "restaurant_versions", "restaurant_overall", "restaurant_tags",
		"restaurants_fts", "reviews_fts", "recipes", "recipe_reviews", "recipe_review_photos", "list_recipes", "recipe_versions",
		"recipe_overall", "recipe_tags", "recipes_fts", "recipe_reviews_fts",
	} {
		if app.HasTable(name) {
			t.Errorf("table %s stays", name)
		}
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
	if users.Fields.GetByName("avatar") == nil || users.Fields.GetByName("approved") != nil {
		t.Error("down: users has no avatar or still has approved")
	}
	if app.Settings().RateLimits.Enabled {
		t.Error("down: the PocketBase rate limits are on")
	}
	if _, ok := app.Settings().RateLimits.FindRateLimitRule([]string{cspReportRule.Label}); ok {
		t.Error("down: the rate limit rule of the CSP reports stays")
	}

	if err := up(app); err != nil {
		t.Fatalf("up after down: %v", err)
	}
}

func TestUpRefusesAnOldDataFolder(t *testing.T) {
	t.Parallel()
	if err := up(newApp(t)); err == nil {
		t.Error("up ran on a data folder that has the collections")
	}
}

func TestRateLimitsAreOn(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	if !app.Settings().RateLimits.Enabled {
		t.Error("the PocketBase rate limits are off")
	}
	if _, ok := app.Settings().RateLimits.FindRateLimitRule([]string{"POST /csp-report"}); !ok {
		t.Error("no rate limit rule for the CSP reports")
	}
}

func TestUsersHaveNoFileFields(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range users.Fields {
		if f.Type() == core.FieldTypeFile {
			t.Errorf("users has the file field %s", f.GetName())
		}
	}
	if newUser(t, app, "ana").GetBool("approved") {
		t.Error("a new user is approved")
	}
}

func TestReviewPhotos(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	for _, k := range subjectKinds {
		photos, err := app.FindCollectionByNameOrId(k.photos)
		if err != nil {
			t.Fatal(err)
		}
		if photos.ListRule != nil || photos.ViewRule != nil || photos.CreateRule != nil || photos.UpdateRule != nil || photos.DeleteRule != nil {
			t.Errorf("%s has API rules", k.photos)
		}
		for _, name := range []string{"full", "thumb"} {
			f, ok := photos.Fields.GetByName(name).(*core.FileField)
			if !ok || !f.Protected || f.MaxSelect != 1 || !slices.Equal(f.MimeTypes, []string{"image/jpeg"}) {
				t.Errorf("%s %s: %+v", k.photos, name, f)
			}
		}
		if review, ok := photos.Fields.GetByName("review").(*core.RelationField); !ok || !review.Required || !review.CascadeDelete {
			t.Errorf("%s review: %+v", k.photos, review)
		}
	}
}

func TestPlaceIDIsUnique(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	// Two restaurants without a place ID do not break the unique index.
	newRecord(t, app, "restaurants", map[string]any{"name": "A", "latitude": 41.3766, "longitude": 2.1893, "address": "Carrer Nou 1", "place_id": "osm:N1"})
	newRecord(t, app, "restaurants", map[string]any{"name": "B", "address": "Carrer Nou 2"})
	newRecord(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "C"})
	c, err := app.FindCollectionByNameOrId("restaurants")
	if err != nil {
		t.Fatal(err)
	}
	same := core.NewRecord(c)
	same.Load(map[string]any{"name": "D", "address": "Carrer Nou 3", "place_id": "osm:N1"})
	if err := app.Save(same); err == nil {
		t.Error("two restaurants have one place ID")
	}
}

func reviewOverall(t *testing.T, app core.App, id string) float64 {
	t.Helper()
	var v float64
	if err := app.DB().Select("overall").From("reviews").Where(dbx.HashExp{"id": id}).Row(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestReviewOverallColumn(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	restaurant := newRecord(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "Luigi's"})
	r := newRecord(t, app, "reviews", review(newUser(t, app, "ana"), restaurant, map[string]any{"space": 3, "service": 9, "food": 8, "price": 6}))
	if got := reviewOverall(t, app, r.Id); got != 6.5 {
		t.Errorf("overall is %v, want 6.5", got)
	}
	r.Set("food", 10)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	if got := reviewOverall(t, app, r.Id); got != 7 {
		t.Errorf("overall after an update is %v, want 7", got)
	}

	// PocketBase changes the table when the fields of the collection change.
	reviews, err := app.FindCollectionByNameOrId("reviews")
	if err != nil {
		t.Fatal(err)
	}
	reviews.Fields.Add(&core.TextField{Name: "note"})
	if err := app.Save(reviews); err != nil {
		t.Fatal(err)
	}
	reviews.Fields.RemoveByName("note")
	if err := app.Save(reviews); err != nil {
		t.Fatal(err)
	}
	if got := reviewOverall(t, app, r.Id); got != 7 {
		t.Errorf("overall after a field change is %v, want 7", got)
	}
}

func TestReviewDate(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	ana := newUser(t, app, "ana")
	luigi := newRecord(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "Luigi's"})
	dal := newRecord(t, app, "recipes", map[string]any{"name": "Dal", "ingredients": []string{"lentils"}, "steps": []string{"Boil."}})
	for range 2 {
		newRecord(t, app, "reviews", review(ana, luigi, scores(6)))
		newRecord(t, app, "recipe_reviews", recipeReview(ana, dal, 6))
	}

	for collection, data := range map[string]map[string]any{
		"reviews":        review(ana, luigi, scores(6)),
		"recipe_reviews": recipeReview(ana, dal, 6),
	} {
		c, err := app.FindCollectionByNameOrId(collection)
		if err != nil {
			t.Fatal(err)
		}
		data["date"] = "6 Nov 2024"
		r := core.NewRecord(c)
		r.Load(data)
		if err := app.Save(r); err == nil {
			t.Errorf("%s: a date that does not match the pattern was saved", collection)
		}
	}
}

func TestLists(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	ana := newUser(t, app, "ana")
	restaurant := newRecord(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "Luigi's"})
	recipe := newRecord(t, app, "recipes", map[string]any{"name": "Dal", "ingredients": []string{"lentils"}, "steps": []string{"Boil."}})
	favorites := newRecord(t, app, "lists", map[string]any{"creator": ana.Id, "name": "Favorites", "kind": "favorites"})

	lists, err := app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(lists)
	r.Load(map[string]any{"creator": ana.Id, "name": "Other", "kind": "favorites"})
	if err := app.Save(r); err == nil {
		t.Error("a second favorites list was saved")
	}
	r = core.NewRecord(lists)
	r.Load(map[string]any{"creator": ana.Id, "name": "FAVORITES", "kind": "custom"})
	if err := app.Save(r); err == nil {
		t.Error("a list with the name of another list was saved")
	}

	custom := newRecord(t, app, "lists", map[string]any{"creator": ana.Id, "name": "Date night", "kind": "custom"})
	for _, list := range []*core.Record{custom, favorites} {
		newRecord(t, app, "list_items", map[string]any{"list": list.Id, "restaurant": restaurant.Id})
		newRecord(t, app, "list_recipes", map[string]any{"list": list.Id, "recipe": recipe.Id})
	}
	count := func(collection string) int64 {
		t.Helper()
		n, err := app.CountRecords(collection)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	if err := app.Delete(restaurant); err != nil {
		t.Fatal(err)
	}
	if err := app.Delete(recipe); err != nil {
		t.Fatal(err)
	}
	for _, collection := range []string{"list_items", "list_recipes"} {
		if n := count(collection); n != 0 {
			t.Errorf("%d %s after the delete of the subjects, want 0", n, collection)
		}
	}
	if n := count("lists"); n != 2 {
		t.Errorf("%d lists after the delete of the subjects, want 2", n)
	}
	if err := app.Delete(ana); err != nil {
		t.Fatal(err)
	}
	if n := count("lists"); n != 0 {
		t.Errorf("%d lists after the user delete, want 0", n)
	}
}

func TestRestaurantDetails(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	r := newRecord(t, app, "restaurants", map[string]any{"name": "Café Olé", "address": "Carrer Nou 1"})
	r.Set("price_level", 5)
	if err := app.Save(r); err == nil {
		t.Error("a price level of 5 is saved")
	}
	r.Set("price_level", 4)
	if err := app.Save(r); err != nil {
		t.Errorf("a price level of 4: %v", err)
	}

	for field, values := range map[string][2]string{
		"website":       {"https://cansole.example/menu", "see menu"},
		"phone":         {"+34 932 21 50 12", "call us"},
		"opening_hours": {"Mo-Fr 12:00-15:00\nSa 12:00-23:00", "Mo​"},
	} {
		r.Set(field, values[0])
		if err := app.Save(r); err != nil {
			t.Errorf("%s %q: %v", field, values[0], err)
		}
		r.Set(field, values[1])
		if err := app.Save(r); err == nil {
			t.Errorf("%s %q is saved", field, values[1])
		}
		r.Set(field, "")
	}
}

func newTag(t *testing.T, app core.App, name string) string {
	t.Helper()
	return newRecord(t, app, "tags", map[string]any{"name": name}).Id
}

func sorted(values ...string) []string {
	slices.Sort(values)
	return values
}
