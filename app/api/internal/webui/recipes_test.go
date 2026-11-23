package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/baumac/reviews-app/app/api/internal/testapp"
	"github.com/pocketbase/pocketbase/tests"
)

func TestRecipeMeta(t *testing.T) {
	t.Parallel()
	app, err := tests.NewTestApp(testapp.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://reviews.example.com/"
	baseURL := publicBaseURL(app.Settings().Meta.AppURL)

	user := save(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana", "approved": true})
	recipe := save(t, app, "recipes", map[string]any{
		"name": "Shakshuka", "description": "Eggs in\na sauce.", "creator": user.Id,
		"ingredients": []string{"6 eggs", "1 onion"}, "steps": []string{"Cook."},
	})
	plain := save(t, app, "recipes", map[string]any{"name": "Bread", "ingredients": []string{"flour"}, "steps": []string{"Bake."}})
	review := save(t, app, "recipe_reviews", map[string]any{
		"recipe": recipe.Id, "creator": user.Id, "date": "2024-11-06", "title": "Quick", "feedback": "More salt.",
		"taste": 9, "ease": 8, "instructions": 8, "value": 8,
	})
	photo := savePhoto(t, app, "recipe_review_photos", review.Id)
	recipePath := "/recipes/" + recipe.Id
	reviewPath := recipePath + "/reviews/" + review.Id

	t.Run("the meta tags have the recipe paths", func(t *testing.T) {
		p, ok := readIndex(fstest.MapFS{"index.html": {Data: []byte(testIndex)}})
		if !ok {
			t.Fatal("no index")
		}
		if meta := p.parentMeta(recipeKind, recipe, baseURL); meta.URL != "https://reviews.example.com"+recipePath || meta.Description != "Eggs in a sauce." {
			t.Errorf("recipe meta %+v", meta)
		}
		meta := p.reviewMeta(recipeKind, recipe, review, photo, baseURL)
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

	t.Run("the recipe pages have their meta", func(t *testing.T) {
		for path, want := range map[string][]string{
			recipePath:             {`<title>Shakshuka · Sponga47 Reviews</title>`},
			"/Recipes/" + plain.Id: {`<meta name="description" content="Reviews of Bread on Sponga47 Reviews." />`},
			reviewPath:             {`<meta property="og:title" content="Quick · Shakshuka" />`},
			"/recipes/" + plain.Id + "/reviews/" + review.Id: {`content="Default"`},
			"/recipes/aaaaaaaaaaaaaaa":                       {`content="Default"`},
			"/recipes/new":                                   {`content="Default"`},
		} {
			body := get(path)
			for _, w := range want {
				if !strings.Contains(body, w) {
					t.Errorf("%s: no %s in\n%s", path, w, body)
				}
			}
		}
	})

	t.Run("a list counts its recipes", func(t *testing.T) {
		recipes := save(t, app, "lists", map[string]any{"creator": user.Id, "name": "Weeknights", "kind": "custom"})
		save(t, app, "list_recipes", map[string]any{"list": recipes.Id, "recipe": recipe.Id})
		save(t, app, "list_recipes", map[string]any{"list": recipes.Id, "recipe": plain.Id})
		mixed := save(t, app, "lists", map[string]any{"creator": user.Id, "name": "Mixed", "kind": "custom"})
		save(t, app, "list_recipes", map[string]any{"list": mixed.Id, "recipe": recipe.Id})
		restaurant := save(t, app, "restaurants", map[string]any{"address": "Carrer 1", "name": "Luigi's", "creator": user.Id})
		save(t, app, "list_items", map[string]any{"list": mixed.Id, "restaurant": restaurant.Id})

		for path, want := range map[string][]string{
			"/lists/" + recipes.Id: {
				`<meta name="description" content="A list of 2 recipes by Ana on Sponga47 Reviews." />`,
			},
			"/lists/" + mixed.Id: {
				`<meta name="description" content="A list of 1 restaurant and 1 recipe by Ana on Sponga47 Reviews." />`,
			},
		} {
			body := get(path)
			for _, w := range want {
				if !strings.Contains(body, w) {
					t.Errorf("%s: no %s in\n%s", path, w, body)
				}
			}
		}
	})

}
