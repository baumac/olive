package publicapi_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestRecipeLists(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	dal := e.createRecipe(ana.Token, `{"name":"Dal"}`)["id"].(string)
	soup := e.createRecipe(joe.Token, `{"name":"Soup"}`)["id"].(string)
	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	list := e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Weeknights"}`, token: ana.Token}, http.StatusCreated).object(t)["id"].(string)
	favorites := e.lists(ana.User.ID, "").Items[0].ID

	e.expect(request{method: "PUT", path: "/v1/lists/favorites/recipes/" + dal, token: ana.Token}, http.StatusNoContent)
	e.expect(request{method: "PUT", path: "/v1/lists/favorites/restaurants/" + restaurant, token: ana.Token}, http.StatusNoContent)
	for range 2 {
		e.expect(request{method: "PUT", path: "/v1/lists/" + list + "/recipes/" + soup, token: ana.Token}, http.StatusNoContent)
	}

	t.Run("a recipe tells if it is a favorite of the user", func(t *testing.T) {
		for token, want := range map[string]bool{ana.Token: true, joe.Token: false, "": false} {
			got := e.expect(request{method: "GET", path: "/v1/recipes/" + dal, token: token}, http.StatusOK).object(t)
			if got["favorite"] != want {
				t.Errorf("favorite %v, want %v", got["favorite"], want)
			}
		}
		var page struct {
			Items []struct {
				Name     string `json:"name"`
				Favorite bool   `json:"favorite"`
			} `json:"items"`
		}
		e.expect(request{method: "GET", path: "/v1/recipes?sort=name", token: ana.Token}, http.StatusOK).decode(t, &page)
		if len(page.Items) != 2 || !page.Items[0].Favorite || page.Items[1].Favorite {
			t.Errorf("recipes %+v, want only Dal as a favorite", page.Items)
		}
	})

	t.Run("a list counts its restaurants and its recipes", func(t *testing.T) {
		lists := e.lists(ana.User.ID, "?restaurant[eq]="+restaurant+"&recipe[eq]="+dal).Items
		if lists[0].ID != favorites || lists[0].RestaurantCount != 1 || lists[0].RecipeCount != 1 || !*lists[0].HasRestaurant || !*lists[0].HasRecipe {
			t.Errorf("favorites %+v", lists[0])
		}
		if lists[1].ID != list || lists[1].RestaurantCount != 0 || lists[1].RecipeCount != 1 || *lists[1].HasRestaurant || *lists[1].HasRecipe {
			t.Errorf("list %+v", lists[1])
		}
		if plain := e.lists(ana.User.ID, "").Items[0]; plain.HasRecipe != nil {
			t.Errorf("has_recipe %v without recipe[eq]", *plain.HasRecipe)
		}
		got := e.expect(request{method: "GET", path: "/v1/lists/" + list}, http.StatusOK).object(t)
		if got["recipe_count"] != 1.0 || got["restaurant_count"] != 0.0 {
			t.Errorf("list %v", got)
		}
		item := e.expect(request{method: "GET", path: "/v1/lists/" + list + "/recipes"}, http.StatusOK).object(t)["items"].([]any)[0].(map[string]any)
		_, ingredients := item["ingredients"]
		_, steps := item["steps"]
		if ingredients || steps {
			t.Errorf("item %v, want a summary without the ingredients and the steps", item)
		}
	})

	t.Run("the recipes of a list come with the last one first", func(t *testing.T) {
		e.expect(request{method: "PUT", path: "/v1/lists/" + list + "/recipes/" + dal, token: ana.Token}, http.StatusNoContent)
		first, next := e.names("/v1/lists/" + list + "/recipes?limit=1")
		if first != "Dal" || next == nil {
			t.Fatalf("first page %q, next %v", first, next)
		}
		second, _ := e.names("/v1/lists/" + list + "/recipes?limit=1&cursor=" + url.QueryEscape(*next))
		if second != "Soup" {
			t.Errorf("second page %q", second)
		}
		e.expect(request{method: "GET", path: "/v1/lists/aaaaaaaaaaaaaaa/recipes"}, http.StatusNotFound)
		e.expect(request{method: "GET", path: "/v1/lists/" + list + "/recipes?cursor=bad"}, http.StatusBadRequest)
	})

	t.Run("listRecipes selects the recipes of a list", func(t *testing.T) {
		for query, want := range map[string]string{
			"list[eq]=" + list + "&sort=name":  "Dal,Soup",
			"list[eq]=" + favorites:            "Dal",
			"list[eq]=aaaaaaaaaaaaaaa":         "",
			"list[eq]=" + list + "&search=sou": "Soup",
		} {
			if got, _ := e.names("/v1/recipes?" + query); got != want {
				t.Errorf("%s: names %q, want %q", query, got, want)
			}
		}
	})

	t.Run("only the creator changes the recipes of a list", func(t *testing.T) {
		e.expectCode(request{method: "PUT", path: "/v1/lists/" + list + "/recipes/" + dal, token: joe.Token}, http.StatusForbidden, "not_creator")
		e.expectCode(request{method: "DELETE", path: "/v1/lists/" + list + "/recipes/" + dal, token: joe.Token}, http.StatusForbidden, "not_creator")
		e.expect(request{method: "PUT", path: "/v1/lists/" + list + "/recipes/aaaaaaaaaaaaaaa", token: ana.Token}, http.StatusNotFound)
		e.expect(request{method: "PUT", path: "/v1/lists/aaaaaaaaaaaaaaa/recipes/" + dal, token: ana.Token}, http.StatusNotFound)
		e.expect(request{method: "DELETE", path: "/v1/lists/aaaaaaaaaaaaaaa/recipes/" + dal, token: ana.Token}, http.StatusNotFound)
	})

	t.Run("a remove changes the list once", func(t *testing.T) {
		for range 2 {
			e.expect(request{method: "DELETE", path: "/v1/lists/" + list + "/recipes/" + soup, token: ana.Token}, http.StatusNoContent)
		}
		if got, _ := e.names("/v1/lists/" + list + "/recipes"); got != "Dal" {
			t.Errorf("recipes %q", got)
		}
	})

	t.Run("the favorites ref removes a recipe", func(t *testing.T) {
		e.expect(request{method: "PUT", path: "/v1/lists/favorites/recipes/" + soup, token: ana.Token}, http.StatusNoContent)
		e.expect(request{method: "DELETE", path: "/v1/lists/favorites/recipes/" + soup, token: ana.Token}, http.StatusNoContent)
		if got, _ := e.names("/v1/lists/" + favorites + "/recipes"); got != "Dal" {
			t.Errorf("favorites %q, want only Dal", got)
		}
	})

	t.Run("the list of a user without approval is only for that user", func(t *testing.T) {
		pat := e.signUpPending("pat@example.com", "Pat")
		patFavorites := e.listsAs(pat.User.ID, pat.Token).Items[0].ID
		items, err := e.app.FindCollectionByNameOrId("list_recipes")
		if err != nil {
			t.Fatal(err)
		}
		item := core.NewRecord(items)
		item.Load(map[string]any{"list": patFavorites, "recipe": soup})
		if err := e.app.Save(item); err != nil {
			t.Fatal(err)
		}
		path := "/v1/lists/" + patFavorites + "/recipes"
		e.expect(request{method: "GET", path: path, token: joe.Token}, http.StatusNotFound)
		e.expect(request{method: "GET", path: path}, http.StatusNotFound)
		if got, _ := e.namesAs(path, pat.Token); got != "Soup" {
			t.Errorf("recipes for Pat %q", got)
		}
		e.expect(request{method: "GET", path: "/v1/recipes?list[eq]=" + patFavorites, token: joe.Token}, http.StatusOK)
	})

	t.Run("a delete of a recipe removes it from the lists", func(t *testing.T) {
		e.expect(request{method: "DELETE", path: "/v1/recipes/" + dal, token: ana.Token}, http.StatusNoContent)
		lists := e.lists(ana.User.ID, "").Items
		if lists[0].RecipeCount != 0 || lists[1].RecipeCount != 0 {
			t.Errorf("lists %+v, want no recipes", lists)
		}
	})
}

func TestListRecipeLimit(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	recipe := e.createRecipe(ana.Token, `{"name":"Dal"}`)["id"].(string)
	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	list := e.lists(ana.User.ID, "").Items[0].ID
	// The rows have no recipe record. The limit check only counts them.
	_, err := e.app.DB().NewQuery(`
		WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 500)
		INSERT INTO list_recipes (list, recipe, created) SELECT {:list}, printf('x%014d', i), '2024-11-04 01:26:55.002Z' FROM n`).
		Bind(map[string]any{"list": list}).Execute()
	if err != nil {
		t.Fatal(err)
	}
	e.expectCode(request{method: "PUT", path: "/v1/lists/favorites/recipes/" + recipe, token: ana.Token}, http.StatusConflict, "list_recipe_limit")
	// The restaurants of the list have their own limit.
	e.expect(request{method: "PUT", path: "/v1/lists/favorites/restaurants/" + restaurant, token: ana.Token}, http.StatusNoContent)
}
