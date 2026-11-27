package migrations

import (
	"database/sql"
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
