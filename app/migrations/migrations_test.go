package migrations

import (
	"slices"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
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

	downTo(t, app, "1794096000_create_collections.go")
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

func TestSearchIndexFillsExistingRecords(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	redo := downTo(t, app, "1794268805_add_search_index.go")

	user := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	restaurant := newRecord(t, app, "restaurants", map[string]any{"name": "Café Olé", "tags": []string{"date-night"}})
	newRecord(t, app, "reviews", map[string]any{
		"restaurant": restaurant.Id, "creator": user.Id, "space": 3, "service": 9, "food": 8, "price": 6, "cleanliness": 4,
		"title": "Good", "feedback": "Creamy carbonara",
	})

	redo()
	for table, match := range map[string]string{"restaurants_fts": "ole night", "reviews_fts": "carbonara ole"} {
		var n int
		if err := app.DB().NewQuery("SELECT COUNT(*) FROM " + table + " WHERE " + table + " MATCH {:q}").Bind(map[string]any{"q": match}).Row(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("%s MATCH %q: %d rows, want 1", table, match, n)
		}
	}
}

func TestRemoveReviewTags(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	redo := downTo(t, app, "1794268807_remove_review_tags.go")

	user := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	restaurant := newRecord(t, app, "restaurants", map[string]any{"name": "Café Olé", "tags": []string{"italian"}})
	newRecord(t, app, "reviews", map[string]any{
		"restaurant": restaurant.Id, "creator": user.Id, "space": 3, "service": 9, "food": 8, "price": 6, "cleanliness": 4,
		"title": "Good", "feedback": "Creamy carbonara", "tags": []string{"date-night"},
	})

	redo()
	reviews, err := app.FindCollectionByNameOrId("reviews")
	if err != nil {
		t.Fatal(err)
	}
	if reviews.Fields.GetByName("tags") != nil {
		t.Error("reviews still has tags")
	}
	count := func(table, match string) int {
		t.Helper()
		var n int
		if err := app.DB().NewQuery("SELECT COUNT(*) FROM " + table + " WHERE " + table + " MATCH {:q}").Bind(map[string]any{"q": match}).Row(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	for match, want := range map[string]int{"carbonara ole": 1, "good": 1, "night": 0} {
		if n := count("reviews_fts", match); n != want {
			t.Errorf("reviews_fts MATCH %q: %d rows, want %d", match, n, want)
		}
	}
	downTo(t, app, "1794268807_remove_review_tags.go")
	if reviews, err = app.FindCollectionByNameOrId("reviews"); err != nil {
		t.Fatal(err)
	}
	if reviews.Fields.GetByName("tags") == nil {
		t.Error("down did not add tags")
	}
	if n := count("reviews_fts", "carbonara"); n != 1 {
		t.Errorf("down: reviews_fts MATCH carbonara: %d rows, want 1", n)
	}
}

func TestReviewPhotos(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	photos, err := app.FindCollectionByNameOrId("review_photos")
	if err != nil {
		t.Fatal(err)
	}
	if photos.ListRule != nil || photos.ViewRule != nil || photos.CreateRule != nil || photos.UpdateRule != nil || photos.DeleteRule != nil {
		t.Error("review_photos has API rules")
	}
	for _, name := range []string{"full", "thumb"} {
		f, ok := photos.Fields.GetByName(name).(*core.FileField)
		if !ok || !f.Protected || f.MaxSelect != 1 || !slices.Equal(f.MimeTypes, []string{"image/jpeg"}) {
			t.Errorf("%s: %+v", name, f)
		}
	}
	if review, ok := photos.Fields.GetByName("review").(*core.RelationField); !ok || !review.Required || !review.CascadeDelete {
		t.Errorf("review: %+v", review)
	}

	redo := downTo(t, app, "1794268806_add_review_photos.go")
	if _, err := app.FindCollectionByNameOrId("review_photos"); err == nil {
		t.Error("down kept review_photos")
	}
	redo()
}

func TestRestaurantIndexFillsExistingRecords(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	redo := downTo(t, app, "1794268808_add_restaurant_index.go")

	user := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	rated := newRecord(t, app, "restaurants", map[string]any{"name": "Rated", "tags": []any{"italian", "italian", 3, "pasta"}})
	unrated := newRecord(t, app, "restaurants", map[string]any{"name": "Unrated", "tags": "italian"})
	newRecord(t, app, "reviews", map[string]any{
		"restaurant": rated.Id, "creator": user.Id, "space": 3, "service": 9, "food": 8, "price": 6, "cleanliness": 4, "title": "Good",
	})

	redo()
	var overall []struct {
		Restaurant string   `db:"restaurant"`
		Overall    *float64 `db:"overall"`
	}
	if err := app.DB().NewQuery("SELECT restaurant, overall FROM restaurant_overall").All(&overall); err != nil {
		t.Fatal(err)
	}
	got := map[string]*float64{}
	for _, row := range overall {
		got[row.Restaurant] = row.Overall
	}
	if len(got) != 2 || got[rated.Id] == nil || *got[rated.Id] != 6 || got[unrated.Id] != nil {
		t.Errorf("restaurant_overall: %v", overall)
	}
	var tags []string
	if err := app.DB().NewQuery("SELECT tag || ' ' || (restaurant = {:id}) FROM restaurant_tags ORDER BY tag").Bind(map[string]any{"id": rated.Id}).Column(&tags); err != nil {
		t.Fatal(err)
	}
	if strings.Join(tags, ",") != "italian 1,pasta 1" {
		t.Errorf("restaurant_tags: %v", tags)
	}
}

func TestRestaurantLocation(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	redo := downTo(t, app, "1794268809_add_restaurant_location.go")

	restaurants, err := app.FindCollectionByNameOrId("restaurants")
	if err != nil {
		t.Fatal(err)
	}
	if restaurants.Fields.GetByName("address") != nil {
		t.Fatal("down keeps the address field")
	}
	old := newRecord(t, app, "restaurants", map[string]any{"name": "Café Olé", "tags": []string{"date-night"}})
	if _, err := app.DB().Insert("restaurants_fts", map[string]any{"id": old.Id, "name": "Café Olé", "description": "", "tags": "date-night"}).Execute(); err != nil {
		t.Fatal(err)
	}

	redo()
	old, err = app.FindRecordById("restaurants", old.Id)
	if err != nil {
		t.Fatal(err)
	}
	if old.GetString("address") != "" || old.GetString("place_id") != "" {
		t.Errorf("the old restaurant has a location: %v", old.FieldsData())
	}
	count := func(match string) int {
		t.Helper()
		var n int
		if err := app.DB().NewQuery("SELECT COUNT(*) FROM restaurants_fts WHERE restaurants_fts MATCH {:q}").Bind(map[string]any{"q": match}).Row(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := count("ole night"); n != 1 {
		t.Errorf("the old restaurant has %d index rows", n)
	}

	// Two restaurants without a place ID do not break the unique index.
	newRecord(t, app, "restaurants", map[string]any{"name": "A", "latitude": 41.3766, "longitude": 2.1893, "address": "Carrer Nou 1", "place_id": "osm:N1"})
	newRecord(t, app, "restaurants", map[string]any{"name": "B", "address": "Carrer Nou 2"})
	c, err := app.FindCollectionByNameOrId("restaurants")
	if err != nil {
		t.Fatal(err)
	}
	same := core.NewRecord(c)
	same.Load(map[string]any{"name": "C", "address": "Carrer Nou 3", "place_id": "osm:N1"})
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
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	redo := downTo(t, app, "1794268811_add_review_overall.go")

	user := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	restaurant := newRecord(t, app, "restaurants", map[string]any{"name": "Luigi's"})
	review := newRecord(t, app, "reviews", map[string]any{
		"restaurant": restaurant.Id, "creator": user.Id, "space": 3, "service": 9, "food": 8, "price": 6, "cleanliness": 7, "title": "Good",
	})

	hasCreatorIndex := func() bool {
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			t.Fatal(err)
		}
		return reviews.GetIndex("idx_reviews_creator") != ""
	}
	if !hasCreatorIndex() {
		t.Error("down: reviews has no idx_reviews_creator")
	}

	redo()
	if hasCreatorIndex() {
		t.Error("up: reviews still has idx_reviews_creator")
	}
	if got := reviewOverall(t, app, review.Id); got != 6.6 {
		t.Errorf("overall is %v, want 6.6", got)
	}
	review, err = app.FindRecordById("reviews", review.Id)
	if err != nil {
		t.Fatal(err)
	}
	review.Set("food", 10)
	if err := app.Save(review); err != nil {
		t.Fatal(err)
	}
	if got := reviewOverall(t, app, review.Id); got != 7 {
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
	if got := reviewOverall(t, app, review.Id); got != 7 {
		t.Errorf("overall after a field change is %v, want 7", got)
	}
}

func TestListsMigration(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	redo := downTo(t, app, "1794268813_add_lists.go")
	ana := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	restaurant := newRecord(t, app, "restaurants", map[string]any{"name": "Luigi's"})
	redo()

	favorites, err := app.FindAllRecords("lists", dbx.HashExp{"creator": ana.Id})
	if err != nil {
		t.Fatal(err)
	}
	if len(favorites) != 1 || favorites[0].GetString("kind") != "favorites" || favorites[0].GetString("name") != "Favorites" ||
		len(favorites[0].Id) != 15 || favorites[0].GetDateTime("created").IsZero() {
		t.Fatalf("lists of an existing user: %v", favorites)
	}

	second := map[string]any{"creator": ana.Id, "name": "Other", "kind": "favorites"}
	lists, _ := app.FindCollectionByNameOrId("lists")
	r := core.NewRecord(lists)
	r.Load(second)
	if err := app.Save(r); err == nil {
		t.Error("a second favorites list was saved")
	}
	r = core.NewRecord(lists)
	r.Load(map[string]any{"creator": ana.Id, "name": "FAVORITES", "kind": "custom"})
	if err := app.Save(r); err == nil {
		t.Error("a list with the name of another list was saved")
	}

	custom := newRecord(t, app, "lists", map[string]any{"creator": ana.Id, "name": "Date night", "kind": "custom"})
	newRecord(t, app, "list_items", map[string]any{"list": custom.Id, "restaurant": restaurant.Id})
	newRecord(t, app, "list_items", map[string]any{"list": favorites[0].Id, "restaurant": restaurant.Id})
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
	if n := count("list_items"); n != 0 {
		t.Errorf("%d items after the restaurant delete, want 0", n)
	}
	if n := count("lists"); n != 2 {
		t.Errorf("%d lists after the restaurant delete, want 2", n)
	}
	if err := app.Delete(ana); err != nil {
		t.Fatal(err)
	}
	if n := count("lists"); n != 0 {
		t.Errorf("%d lists after the user delete, want 0", n)
	}
}

func TestUserApprovalMigration(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	redo := downTo(t, app, "1794268814_add_user_approval.go")
	old := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	redo()

	old, err = app.FindRecordById("users", old.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !old.GetBool("approved") {
		t.Error("a user from before the migration is not approved")
	}
	later := newRecord(t, app, "users", map[string]any{"email": "joe@example.com", "password": "correct-horse", "name": "Joe"})
	if later.GetBool("approved") {
		t.Error("a new user is approved")
	}
}
