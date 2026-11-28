package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/baumac/olive/app/api/internal/testapp"
	"github.com/pocketbase/pocketbase/tests"
)

func TestPantryItemMeta(t *testing.T) {
	t.Parallel()
	app, err := tests.NewTestApp(testapp.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://reviews.example.com/"
	baseURL := publicBaseURL(app.Settings().Meta.AppURL)

	user := save(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana", "approved": true})
	aged := save(t, app, "tags", map[string]any{"name": "aged"})
	item := save(t, app, "pantry_items", map[string]any{
		"name": "Comté", "category": "cheese", "description": "Nutty and\nfirm.", "creator": user.Id, "tags": []string{aged.Id},
	})
	plain := save(t, app, "pantry_items", map[string]any{"name": "Pils", "category": "beer"})
	review := save(t, app, "pantry_item_reviews", map[string]any{
		"pantry_item": item.Id, "creator": user.Id, "date": "2024-11-06", "title": "Sharp",
		"taste": 9, "aroma": 8, "texture": 8, "value": 8,
	})
	plainReview := save(t, app, "pantry_item_reviews", map[string]any{
		"pantry_item": plain.Id, "creator": user.Id, "date": "2024-11-06", "title": "Cold",
		"taste": 7, "aroma": 7, "texture": 7, "value": 7,
	})
	photo := savePhoto(t, app, "pantry_item_review_photos", review.Id)
	itemPath := "/pantry/" + item.Id
	reviewPath := itemPath + "/reviews/" + review.Id

	t.Run("the meta tags have the pantry paths", func(t *testing.T) {
		p, ok := readIndex(fstest.MapFS{"index.html": {Data: []byte(testIndex)}})
		if !ok {
			t.Fatal("no index")
		}
		if meta := p.parentMeta(pantryItemKind, item, baseURL); meta.URL != "https://reviews.example.com"+itemPath || meta.Description != "Nutty and firm." {
			t.Errorf("pantry item meta %+v", meta)
		}
		meta := p.reviewMeta(pantryItemKind, item, review, photo, baseURL)
		if meta.URL != "https://reviews.example.com"+reviewPath || meta.Image != "https://reviews.example.com/v1"+reviewPath+"/photos/"+photo.Id+"/full" {
			t.Errorf("review meta %+v", meta)
		}
	})

	h := newHandler(t, app, fstest.MapFS{"index.html": {Data: []byte(testIndex)}})
	get := func(path string) string {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w.Body.String()
	}

	t.Run("the pantry pages have their meta", func(t *testing.T) {
		for path, want := range map[string][]string{
			itemPath:              {`<title>Comté · Olive</title>`},
			"/Pantry/" + plain.Id: {`<meta name="description" content="Tastings of Pils on Olive." />`},
			reviewPath:            {`<meta property="og:title" content="Sharp · Comté" />`},
			"/pantry/" + plain.Id + "/reviews/" + plainReview.Id: {`<meta name="description" content="A tasting of Pils on Olive." />`},
			"/pantry/" + plain.Id + "/reviews/" + review.Id:      {`content="Default"`},
			"/pantry_items/" + item.Id:                           {`content="Default"`},
			"/pantry/aaaaaaaaaaaaaaa":                            {`content="Default"`},
			"/pantry/new":                                        {`content="Default"`},
		} {
			body := get(path)
			for _, w := range want {
				if !strings.Contains(body, w) {
					t.Errorf("%s: no %s in\n%s", path, w, body)
				}
			}
		}
	})

	t.Run("a list and a tag count the pantry items", func(t *testing.T) {
		recipe := save(t, app, "recipes", map[string]any{"name": "Fondue", "ingredients": []string{"cheese"}, "steps": []string{"Melt."}, "tags": []string{aged.Id}})
		restaurant := save(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "Fromagerie", "tags": []string{aged.Id}})
		pantry := save(t, app, "lists", map[string]any{"creator": user.Id, "name": "Cheeses", "kind": "custom"})
		save(t, app, "list_pantry_items", map[string]any{"list": pantry.Id, "pantry_item": item.Id})
		save(t, app, "list_pantry_items", map[string]any{"list": pantry.Id, "pantry_item": plain.Id})
		mixed := save(t, app, "lists", map[string]any{"creator": user.Id, "name": "Mixed", "kind": "custom"})
		save(t, app, "list_pantry_items", map[string]any{"list": mixed.Id, "pantry_item": item.Id})
		save(t, app, "list_recipes", map[string]any{"list": mixed.Id, "recipe": recipe.Id})
		save(t, app, "list_items", map[string]any{"list": mixed.Id, "restaurant": restaurant.Id})
		other := save(t, app, "lists", map[string]any{"creator": user.Id, "name": "Other", "kind": "custom"})
		save(t, app, "list_pantry_items", map[string]any{"list": other.Id, "pantry_item": item.Id})
		save(t, app, "list_items", map[string]any{"list": other.Id, "restaurant": restaurant.Id})

		for path, want := range map[string]string{
			"/lists/" + pantry.Id: `<meta name="description" content="A list of 2 pantry items by Ana on Olive." />`,
			"/lists/" + mixed.Id:  `<meta name="description" content="A list of 1 restaurant, 1 recipe, and 1 pantry item by Ana on Olive." />`,
			"/lists/" + other.Id:  `<meta name="description" content="A list of 1 restaurant and 1 pantry item by Ana on Olive." />`,
			"/tags/aged":          `<meta name="description" content="1 restaurant, 1 recipe, and 1 pantry item with the tag aged on Olive." />`,
		} {
			if body := get(path); !strings.Contains(body, want) {
				t.Errorf("%s: no %s in\n%s", path, want, body)
			}
		}
	})

	t.Run("a tag of only pantry items has a page", func(t *testing.T) {
		save(t, app, "tags", map[string]any{"name": "unused"})
		lager := save(t, app, "tags", map[string]any{"name": "lager"})
		plain.Set("tags", []string{lager.Id})
		if err := app.Save(plain); err != nil {
			t.Fatal(err)
		}
		if body := get("/tags/lager"); !strings.Contains(body, `content="1 pantry item with the tag lager on Olive."`) {
			t.Errorf("no tag meta in\n%s", body)
		}
		if body := get("/tags/unused"); !strings.Contains(body, `content="Default"`) {
			t.Errorf("a tag without a subject has meta:\n%s", body)
		}
	})
}
