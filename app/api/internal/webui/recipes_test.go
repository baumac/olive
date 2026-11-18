package webui

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/baumac/reviews-app/app/api/internal/testapp"
	"github.com/pocketbase/pocketbase/tests"
)

func TestRecipeMeta(t *testing.T) {
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
		"source": "https://recipes.example/shakshuka", "total_time": 30, "servings": 3,
	})
	plain := save(t, app, "recipes", map[string]any{"name": "Bread", "ingredients": []string{"flour"}, "steps": []string{"Bake."}})
	review := save(t, app, "recipe_reviews", map[string]any{
		"recipe": recipe.Id, "creator": user.Id, "title": "Quick", "feedback": "More salt.",
		"taste": 9, "ease": 8, "instructions": 8, "value": 8,
	})
	photo := savePhoto(t, app, "recipe_review_photos", review.Id)
	recipePath := "/recipes/" + recipe.Id
	reviewPath := recipePath + "/reviews/" + review.Id

	script := func(d jsonLD, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return string(d.script())
	}

	t.Run("a recipe has the schema.org properties of a recipe", func(t *testing.T) {
		want := `{"@context":"https://schema.org","@type":"Recipe",` +
			`"aggregateRating":{"@type":"AggregateRating","bestRating":10,"ratingValue":8.3,"reviewCount":1,"worstRating":1},` +
			`"description":"Eggs in a sauce.","isBasedOn":"https://recipes.example/shakshuka","name":"Shakshuka",` +
			`"recipeIngredient":["6 eggs","1 onion"],"recipeInstructions":[{"@type":"HowToStep","text":"Cook."}],` +
			`"recipeYield":"3","totalTime":"PT30M","url":"https://reviews.example.com` + recipePath + `"}`
		if got := script(recipeData(app, recipe, baseURL)); got != want {
			t.Errorf("JSON-LD\n%s\nwant\n%s", got, want)
		}
		want = `{"@context":"https://schema.org","@type":"Recipe","name":"Bread",` +
			`"recipeIngredient":["flour"],"recipeInstructions":[{"@type":"HowToStep","text":"Bake."}],` +
			`"url":"https://reviews.example.com/recipes/` + plain.Id + `"}`
		if got := script(recipeData(app, plain, baseURL)); got != want {
			t.Errorf("JSON-LD without reviews\n%s\nwant\n%s", got, want)
		}
	})

	t.Run("a recipe review names the recipe", func(t *testing.T) {
		var got map[string]any
		if err := json.Unmarshal([]byte(reviewData(app, recipeKind, recipe, review, baseURL).script()), &got); err != nil {
			t.Fatal(err)
		}
		item := got["itemReviewed"].(map[string]any)
		rating := got["reviewRating"].(map[string]any)
		if item["@type"] != "Recipe" || item["url"] != "https://reviews.example.com"+recipePath || rating["ratingValue"] != 8.3 ||
			got["url"] != "https://reviews.example.com"+reviewPath {
			t.Errorf("JSON-LD %v", got)
		}
	})

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

	t.Run("the recipe pages have their meta and are in the sitemap", func(t *testing.T) {
		for path, want := range map[string][]string{
			recipePath:             {`<title>Shakshuka · Sponga47 Reviews</title>`, `"@type":"Recipe"`, `"recipeYield":"3"`},
			"/Recipes/" + plain.Id: {`<meta name="description" content="Reviews of Bread on Sponga47 Reviews." />`},
			reviewPath:             {`<meta property="og:title" content="Quick · Shakshuka" />`, `"itemReviewed":{"@type":"Recipe"`},
			"/recipes/" + plain.Id + "/reviews/" + review.Id: {`content="Default"`},
			"/recipes/aaaaaaaaaaaaaaa":                       {`content="Default"`},
			"/recipes/new":                                   {`content="Default"`},
			"/sitemap.xml": {
				"<loc>https://reviews.example.com/recipes</loc>",
				"<loc>https://reviews.example.com" + recipePath + "</loc>",
				"<loc>https://reviews.example.com" + reviewPath + "</loc>",
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

	t.Run("a list counts its recipes", func(t *testing.T) {
		recipes := save(t, app, "lists", map[string]any{"creator": user.Id, "name": "Weeknights", "kind": "custom"})
		save(t, app, "list_recipes", map[string]any{"list": recipes.Id, "recipe": recipe.Id})
		save(t, app, "list_recipes", map[string]any{"list": recipes.Id, "recipe": plain.Id})
		mixed := save(t, app, "lists", map[string]any{"creator": user.Id, "name": "Mixed", "kind": "custom"})
		save(t, app, "list_recipes", map[string]any{"list": mixed.Id, "recipe": recipe.Id})
		restaurant := save(t, app, "restaurants", map[string]any{"name": "Luigi's", "creator": user.Id})
		save(t, app, "list_items", map[string]any{"list": mixed.Id, "restaurant": restaurant.Id})

		for path, want := range map[string][]string{
			"/lists/" + recipes.Id: {
				`<meta name="description" content="A list of 2 recipes by Ana on Sponga47 Reviews." />`,
				`"numberOfItems":2`,
			},
			"/lists/" + mixed.Id: {
				`<meta name="description" content="A list of 1 restaurant and 1 recipe by Ana on Sponga47 Reviews." />`,
				`"numberOfItems":2`,
			},
			"/sitemap.xml": {"/lists/" + recipes.Id, "/lists/" + mixed.Id},
		} {
			body := get(path)
			for _, w := range want {
				if !strings.Contains(body, w) {
					t.Errorf("%s: no %s in\n%s", path, w, body)
				}
			}
		}
	})

	t.Run("the reviewer counts the recipe reviews", func(t *testing.T) {
		if body := get("/reviewers/" + user.Id); !strings.Contains(body, `"userInteractionCount":1`) {
			t.Errorf("profile page:\n%s", body)
		}
	})
}
