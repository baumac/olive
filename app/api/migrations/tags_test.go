package migrations

import (
	"database/sql"
	"fmt"
	"slices"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestTagsMigration(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	if err := tagsDown(app); err != nil {
		t.Fatal(err)
	}
	if _, err := app.FindCollectionByNameOrId("tags"); err == nil {
		t.Fatal("down: the tags collection stays")
	}
	const updated = "2024-11-04 01:26:55.002Z"
	for _, row := range []dbx.Params{
		{"id": "restaurant00001", "name": "A", "address": "Carrer 1", "tags": `["café", "pasta", "café", 3]`, "updated": updated},
		{"id": "restaurant00002", "name": "B", "address": "Carrer 2", "updated": updated},
		{"id": "restaurant00003", "name": "C", "address": "Carrer 3", "tags": `["pasta", "café"]`, "updated": updated},
	} {
		if _, err := app.DB().Insert("restaurants", row).Execute(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := app.DB().Insert("recipes", dbx.Params{
		"id": "recipe000000001", "name": "Dal", "ingredients": `["lentils"]`, "steps": `["Cook."]`, "tags": `["pasta", "vegan"]`,
	}).Execute(); err != nil {
		t.Fatal(err)
	}

	if err := tagsUp(app); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	keys := map[string]string{}
	tags, err := app.FindAllRecords("tags")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range tags {
		ids[r.GetString("name")] = r.Id
		keys[r.GetString("name")] = r.GetString("name_key")
	}
	if len(ids) != 3 || keys["café"] != "cafe" || keys["vegan"] != "vegan" {
		t.Fatalf("tags %v, keys %v", ids, keys)
	}
	a, err := app.FindRecordById("restaurants", "restaurant00001")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := a.GetStringSlice("tags"), []string{ids["café"], ids["pasta"]}; !slices.Equal(got, want) {
		t.Errorf("restaurant tags %v, want %v", got, want)
	}
	if a.GetString("updated") != updated {
		t.Errorf("updated %q, want the old time", a.GetString("updated"))
	}
	if got := storedTags(t, app, a.Id); !slices.Equal(got, sorted(ids["café"], ids["pasta"])) {
		t.Errorf("restaurant_tags %v", got)
	}
	b, err := app.FindRecordById("restaurants", "restaurant00002")
	if err != nil {
		t.Fatal(err)
	}
	if got := b.GetStringSlice("tags"); len(got) != 0 {
		t.Errorf("restaurant without tags has %v", got)
	}
	dal, err := app.FindRecordById("recipes", "recipe000000001")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := dal.GetStringSlice("tags"), []string{ids["pasta"], ids["vegan"]}; !slices.Equal(got, want) {
		t.Errorf("recipe tags %v, want %v", got, want)
	}
	restaurants, err := app.FindCollectionByNameOrId("restaurants")
	if err != nil {
		t.Fatal(err)
	}
	f, ok := restaurants.Fields.GetByName("tags").(*core.RelationField)
	if !ok || f.MaxSelect != maxTags || f.CollectionId != tags[0].Collection().Id {
		t.Errorf("tags field %+v", restaurants.Fields.GetByName("tags"))
	}
	if i := slices.IndexFunc(restaurants.Fields, func(f core.Field) bool { return f.GetName() == "tags" }); i != 3 {
		t.Errorf("tags field at %d, want 3", i)
	}

	if err := tagsDown(app); err != nil {
		t.Fatal(err)
	}
	var back []struct {
		ID   string         `db:"id"`
		Tags sql.NullString `db:"tags"`
	}
	if err := app.DB().Select("id", "tags").From("restaurants").OrderBy("id").All(&back); err != nil {
		t.Fatal(err)
	}
	if len(back) != 3 || back[0].Tags.String != `["café","pasta"]` || back[1].Tags.Valid || back[2].Tags.String != `["pasta","café"]` {
		t.Errorf("down: %+v", back)
	}
	if got := storedTags(t, app, "restaurant00001"); !slices.Equal(got, []string{"café", "pasta"}) {
		t.Errorf("down: restaurant_tags %v", got)
	}
}

func TestTagNames(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	for _, name := range []string{"Café", "pasta ", "", "dim sum"} {
		c, err := app.FindCollectionByNameOrId("tags")
		if err != nil {
			t.Fatal(err)
		}
		r := core.NewRecord(c)
		r.Set("name", name)
		if err := app.Save(r); err == nil {
			t.Errorf("the tag %q is saved", name)
		}
	}
	newTag(t, app, "ラーメン")
	c, err := app.FindCollectionByNameOrId("tags")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Set("name", "ラーメン")
	if err := app.Save(r); err == nil {
		t.Error("a second tag with one name is saved")
	}
}

func TestTagNamesMigration(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	for _, row := range []dbx.Params{
		{"id": "tag000000000001", "name": "ｐａｓｔａ", "created": "2024-01-01 00:00:00.000Z"},
		{"id": "tag000000000002", "name": "pasta", "created": "2024-02-01 00:00:00.000Z"},
		{"id": "tag000000000003", "name": "οδοσ", "created": "2024-03-01 00:00:00.000Z"},
		{"id": "tag000000000004", "name": "pizza", "created": "2024-01-01 00:00:00.000Z"},
	} {
		row["name_key"] = tagKey(row["name"].(string))
		if _, err := app.DB().Insert("tags", row).Execute(); err != nil {
			t.Fatal(err)
		}
	}
	const updated = "2024-11-04 01:26:55.002Z"
	inserts := []struct {
		table string
		row   dbx.Params
	}{
		{"restaurants", dbx.Params{"id": "restaurant00001", "name": "A", "address": "Carrer 1", "updated": updated,
			"tags": `["tag000000000002","tag000000000001","tag000000000004"]`}},
		{"recipes", dbx.Params{"id": "recipe000000001", "name": "Dal", "ingredients": `["lentils"]`, "steps": `["Cook."]`,
			"tags": `["tag000000000003"]`}},
		{"restaurants_fts", dbx.Params{"id": "restaurant00001", "name": "A", "tags": "pasta ｐａｓｔａ pizza"}},
		{"recipes_fts", dbx.Params{"id": "recipe000000001", "name": "Dal", "tags": "οδοσ"}},
	}
	for _, in := range inserts {
		if _, err := app.DB().Insert(in.table, in.row).Execute(); err != nil {
			t.Fatal(err)
		}
	}

	for range 2 {
		if err := tagNamesUp(app); err != nil {
			t.Fatal(err)
		}
	}
	var tags []struct {
		ID      string `db:"id"`
		Name    string `db:"name"`
		NameKey string `db:"name_key"`
	}
	if err := app.DB().Select("id", "name", "name_key").From("tags").OrderBy("id").All(&tags); err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(tags); got != "[{tag000000000001 pasta pasta} {tag000000000003 οδος οδος} {tag000000000004 pizza pizza}]" {
		t.Errorf("tags %s", got)
	}
	a, err := app.FindRecordById("restaurants", "restaurant00001")
	if err != nil {
		t.Fatal(err)
	}
	if got := a.GetStringSlice("tags"); !slices.Equal(got, []string{"tag000000000001", "tag000000000004"}) {
		t.Errorf("restaurant tags %v", got)
	}
	if a.GetString("updated") != updated {
		t.Errorf("updated %q, want the old time", a.GetString("updated"))
	}
	if got := storedTags(t, app, a.Id); !slices.Equal(got, []string{"tag000000000001", "tag000000000004"}) {
		t.Errorf("restaurant_tags %v", got)
	}
	for table, want := range map[string]string{"restaurants_fts": "pasta pizza", "recipes_fts": "οδος"} {
		var text string
		if err := app.DB().Select("tags").From(table).Row(&text); err != nil {
			t.Fatal(err)
		}
		if text != want {
			t.Errorf("%s tags %q, want %q", table, text, want)
		}
	}
}

func TestTagNameOf(t *testing.T) {
	t.Parallel()
	for text, want := range map[string]string{"ｐａｓｔａ": "pasta", "οδοσ": "οδος", "ﬁsh": "fish", "wood-fired": "wood-fired", "café": "café"} {
		if got := tagNameOf(text); got != want {
			t.Errorf("tagNameOf(%q) = %q, want %q", text, got, want)
		}
	}
}
