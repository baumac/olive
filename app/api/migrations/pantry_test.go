package migrations

import (
	"database/sql"
	"errors"
	"slices"
	"testing"

	reviewsapi "github.com/baumac/olive/specs/api/sdk/server/go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func pantryItemReview(user, item *core.Record, taste int) map[string]any {
	return map[string]any{"pantry_item": item.Id, "creator": user.Id, "title": "T", "date": "2024-11-06", "taste": taste, "aroma": 8, "texture": 8, "value": 8}
}

// storedPantryItemOverall returns false when the pantry item has no row.
func storedPantryItemOverall(t *testing.T, app core.App, item string) (overall sql.NullFloat64, reviewers int, ok bool) {
	t.Helper()
	err := app.DB().NewQuery("SELECT [[overall]], [[reviewers]] FROM {{pantry_item_overall}} WHERE [[pantry_item]] = {:id}").
		Bind(dbx.Params{"id": item}).Row(&overall, &reviewers)
	if errors.Is(err, sql.ErrNoRows) {
		return overall, reviewers, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return overall, reviewers, true
}

func pantryItemTags(t *testing.T, app core.App, item string) []string {
	t.Helper()
	var tags []string
	if err := app.DB().Select("tag").From("pantry_item_tags").Where(dbx.HashExp{"pantry_item": item}).OrderBy("tag").Column(&tags); err != nil {
		t.Fatal(err)
	}
	return tags
}

func TestPantryItemsMigration(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	if err := pantryItemsDown(app); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pantry_items", "pantry_item_reviews", "list_pantry_items", "pantry_item_overall", "pantry_item_tags", "pantry_items_fts"} {
		if app.HasTable(name) {
			t.Errorf("down: table %s stays", name)
		}
	}

	ana := newUser(t, app, "ana")
	cheese := newTag(t, app, "cheese")
	restaurant := newRecord(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "Fromagerie", "tags": []string{cheese}})
	if err := pantryItemsUp(app); err != nil {
		t.Fatal(err)
	}

	t.Run("the data from before the migration stays", func(t *testing.T) {
		stored, err := app.FindRecordById("restaurants", restaurant.Id)
		if err != nil {
			t.Fatal(err)
		}
		if got := stored.GetStringSlice("tags"); !slices.Equal(got, []string{cheese}) {
			t.Errorf("restaurant tags %v", got)
		}
		if got := storedTags(t, app, restaurant.Id); !slices.Equal(got, []string{cheese}) {
			t.Errorf("restaurant_tags %v", got)
		}
	})

	t.Run("the categories are the values of the spec", func(t *testing.T) {
		items, err := app.FindCollectionByNameOrId("pantry_items")
		if err != nil {
			t.Fatal(err)
		}
		doc, err := reviewsapi.GetSwagger()
		if err != nil {
			t.Fatal(err)
		}
		var want []string
		for _, v := range doc.Components.Schemas["PantryItemCategory"].Value.Enum {
			want = append(want, v.(string))
		}
		f, ok := items.Fields.GetByName("category").(*core.SelectField)
		if !ok || !f.Required || f.MaxSelect != 1 || !slices.Equal(f.Values, want) {
			t.Errorf("category %+v, want the values %v", items.Fields.GetByName("category"), want)
		}
	})

	t.Run("the tags are a relation to the tags collection", func(t *testing.T) {
		items, err := app.FindCollectionByNameOrId("pantry_items")
		if err != nil {
			t.Fatal(err)
		}
		tags, err := app.FindCollectionByNameOrId("tags")
		if err != nil {
			t.Fatal(err)
		}
		f, ok := items.Fields.GetByName("tags").(*core.RelationField)
		if !ok || f.CollectionId != tags.Id || f.MaxSelect != maxTags {
			t.Errorf("tags %+v", items.Fields.GetByName("tags"))
		}
		item := newRecord(t, app, "pantry_items", map[string]any{"name": "Gouda", "category": "cheese", "creator": ana.Id, "tags": []string{cheese}})
		if got := pantryItemTags(t, app, item.Id); !slices.Equal(got, []string{cheese}) {
			t.Errorf("pantry_item_tags %v", got)
		}
	})

	t.Run("the search table has the details", func(t *testing.T) {
		var columns []string
		if err := app.DB().NewQuery("SELECT [[name]] FROM pragma_table_info('pantry_items_fts')").Column(&columns); err != nil {
			t.Fatal(err)
		}
		if want := []string{"id", "name", "description", "tags", "producer", "style", "origin"}; !slices.Equal(columns, want) {
			t.Errorf("columns %v, want %v", columns, want)
		}
	})

	t.Run("a pantry item needs a known category", func(t *testing.T) {
		c, err := app.FindCollectionByNameOrId("pantry_items")
		if err != nil {
			t.Fatal(err)
		}
		for _, category := range []string{"", "cider"} {
			r := core.NewRecord(c)
			r.Load(map[string]any{"name": "N", "category": category})
			if err := app.Save(r); err == nil {
				t.Errorf("the category %q is saved", category)
			}
		}
	})
}

func TestPantryItemTriggers(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	ana := newUser(t, app, "ana")
	joe := newUser(t, app, "joe")
	soft, aged := newTag(t, app, "soft"), newTag(t, app, "aged")
	brie := newRecord(t, app, "pantry_items", map[string]any{"name": "Brie", "category": "cheese", "tags": []string{soft, soft, aged}})

	if overall, reviewers, ok := storedPantryItemOverall(t, app, brie.Id); !ok || overall.Valid || reviewers != 0 {
		t.Errorf("new pantry item: overall %v, reviewers %d, row %v", overall, reviewers, ok)
	}
	first := newRecord(t, app, "pantry_item_reviews", pantryItemReview(ana, brie, 9))
	newRecord(t, app, "pantry_item_reviews", pantryItemReview(joe, brie, 6))
	// The overall values are 8.25 and 7.5.
	if overall, reviewers, _ := storedPantryItemOverall(t, app, brie.Id); overall.Float64 != 7.9 || reviewers != 2 {
		t.Errorf("two reviews: overall %v, reviewers %d", overall, reviewers)
	}
	first.Set("aroma", 4)
	if err := app.Save(first); err != nil {
		t.Fatal(err)
	}
	// Ana has 7.25. Joe has 7.5.
	if overall, _, _ := storedPantryItemOverall(t, app, brie.Id); overall.Float64 != 7.4 {
		t.Errorf("changed review: overall %v, want 7.4", overall)
	}

	if got := pantryItemTags(t, app, brie.Id); !slices.Equal(got, sorted(soft, aged)) {
		t.Errorf("tags %v", got)
	}
	brie.Set("tags", []string{aged})
	if err := app.Save(brie); err != nil {
		t.Fatal(err)
	}
	if got := pantryItemTags(t, app, brie.Id); !slices.Equal(got, []string{aged}) {
		t.Errorf("changed tags %v", got)
	}

	if err := app.Delete(joe); err != nil {
		t.Fatal(err)
	}
	if err := app.Delete(first); err != nil {
		t.Fatal(err)
	}
	if overall, reviewers, ok := storedPantryItemOverall(t, app, brie.Id); !ok || overall.Valid || reviewers != 0 {
		t.Errorf("no reviews: overall %v, reviewers %d, row %v", overall, reviewers, ok)
	}
	if err := app.Delete(brie); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := storedPantryItemOverall(t, app, brie.Id); ok {
		t.Error("deleted pantry item still has a row")
	}
	if got := pantryItemTags(t, app, brie.Id); len(got) != 0 {
		t.Errorf("deleted pantry item still has tags %v", got)
	}
}
