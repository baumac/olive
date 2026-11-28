package publicapi_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type tagPage struct {
	Items []struct {
		Name            string `json:"name"`
		RestaurantCount int    `json:"restaurant_count"`
		RecipeCount     int    `json:"recipe_count"`
		PantryItemCount int    `json:"pantry_item_count"`
	} `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

func (e *env) tags(query string) tagPage {
	e.t.Helper()
	var page tagPage
	e.expect(request{method: http.MethodGet, path: "/v1/tags" + query}, http.StatusOK).decode(e.t, &page)
	return page
}

func (p tagPage) names() string {
	names := make([]string, len(p.Items))
	for i, item := range p.Items {
		names[i] = item.Name
	}
	return strings.Join(names, ",")
}

func (e *env) tagCount() int64 {
	e.t.Helper()
	n, err := e.app.CountRecords("tags")
	if err != nil {
		e.t.Fatal(err)
	}
	return n
}

func TestTagsOfSaves(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	place := `"location":{"latitude":41.4,"longitude":2.2,"address":"Carrer 1","place_id":"osm:N1"}`
	restaurant := e.createRestaurant(ana.Token, `{"name":"Forno","tags":["pizza","wood-fired"],`+place+`}`)
	recipe := e.createRecipe(ana.Token, `{"name":"Margherita","tags":["pizza"]}`)
	if tags := restaurant["tags"].([]any); len(tags) != 2 || tags[0] != "pizza" || tags[1] != "wood-fired" {
		t.Errorf("restaurant tags %v", tags)
	}
	restaurantPath, recipePath := "/v1/restaurants/"+restaurant["id"].(string), "/v1/recipes/"+recipe["id"].(string)

	t.Run("a save makes each new tag one time", func(t *testing.T) {
		page := e.tags("")
		if page.names() != "pizza,wood-fired" || page.Items[0].RestaurantCount != 1 || page.Items[0].RecipeCount != 1 {
			t.Errorf("tags %+v", page.Items)
		}
		if n := e.tagCount(); n != 2 {
			t.Errorf("%d tag records, want 2", n)
		}
	})

	t.Run("a repeat is one tag in the place of the first copy", func(t *testing.T) {
		got := e.createRecipe(ana.Token, `{"name":"Repeats","tags":["Pizza","pasta","pizza","PASTA","pizza"]}`)
		if fmt.Sprint(got["tags"]) != "[pizza pasta]" {
			t.Errorf("tags %v", got["tags"])
		}
		texts := make([]string, 21)
		for i := range 20 {
			texts[i] = fmt.Sprintf(`"T%d"`, i)
		}
		texts[20] = `"t0"`
		patched := e.expect(request{method: http.MethodPatch, path: "/v1/recipes/" + got["id"].(string), token: ana.Token,
			body: `{"tags":[` + strings.Join(texts, ",") + `]}`}, http.StatusOK).object(t)
		if tags := patched["tags"].([]any); len(tags) != 20 || tags[0] != "t0" || tags[19] != "t19" {
			t.Errorf("tags %v", tags)
		}
		e.expect(request{method: http.MethodDelete, path: "/v1/recipes/" + got["id"].(string), token: ana.Token}, http.StatusNoContent)
	})

	t.Run("a conflict makes no tag", func(t *testing.T) {
		e.expect(request{method: http.MethodPost, path: "/v1/restaurants", token: ana.Token,
			body: `{"name":"Forno 2","tags":["second"],` + place + `}`}, http.StatusConflict)
		other := e.createRestaurant(ana.Token, `{"name":"Other"}`)["id"].(string)
		e.expect(request{method: http.MethodPatch, path: "/v1/restaurants/" + other, token: ana.Token,
			body: `{"tags":["second"],` + place + `}`}, http.StatusConflict)
		if n := e.tagCount(); n != 2 {
			t.Errorf("%d tag records, want 2", n)
		}
	})

	t.Run("a save deletes the tags that it leaves without a parent", func(t *testing.T) {
		e.expect(request{method: http.MethodPatch, path: restaurantPath, token: ana.Token, body: `{"tags":["pizza"]}`}, http.StatusOK)
		e.expect(request{method: http.MethodGet, path: "/v1/tags/wood-fired"}, http.StatusNotFound)
		if n := e.tagCount(); n != 1 {
			t.Errorf("%d tag records, want 1", n)
		}
	})

	t.Run("a delete keeps a tag of another parent", func(t *testing.T) {
		e.expect(request{method: http.MethodDelete, path: recipePath, token: ana.Token}, http.StatusNoContent)
		e.expect(request{method: http.MethodGet, path: "/v1/tags/pizza"}, http.StatusOK)
		e.expect(request{method: http.MethodDelete, path: restaurantPath, token: ana.Token}, http.StatusNoContent)
		if n := e.tagCount(); n != 0 {
			t.Errorf("%d tag records, want 0", n)
		}
	})
}

func TestListTags(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	e.createRestaurant(ana.Token, `{"name":"A","tags":["café","wood-fired","ラーメン"]}`)
	e.createRestaurant(ana.Token, `{"name":"B","tags":["cafe-bar","wood-fired"]}`)
	e.createRecipe(ana.Token, `{"name":"C","tags":["wood-fired","café"]}`)
	e.createRecipe(ana.Token, `{"name":"D","tags":["wood-fired"]}`)

	for name, tc := range map[string]struct{ query, want string }{
		"the most used tags come first":     {"", "wood-fired,café,cafe-bar,ラーメン"},
		"the count of one kind":             {"?kind=recipes", "wood-fired,café"},
		"the count of restaurants":          {"?kind=restaurants", "wood-fired,cafe-bar,café,ラーメン"},
		"the fewest first":                  {"?sort=count", "cafe-bar,ラーメン,café,wood-fired"},
		"by name":                           {"?sort=name", "cafe-bar,café,wood-fired,ラーメン"},
		"by name in reverse":                {"?sort=-name", "ラーメン,wood-fired,café,cafe-bar"},
		"a search without case and accents": {"?search=Cafe", "café,cafe-bar"},
		"a search of a word":                {"?search=fired", "wood-fired"},
		"a search with a space":             {"?search=" + url.QueryEscape("wood f"), "wood-fired"},
		"a search of another script":        {"?search=" + url.QueryEscape("ラー"), "ラーメン"},
		"a search of the middle of a word":  {"?search=ood", ""},
		"a search with a LIKE wildcard":     {"?search=%25", ""},
		"a search with punctuation":         {"?search=Wood_F", "wood-fired"},
		"a search without a letter":         {"?search=%21%21", ""},
		"a search that ends in a space":     {"?search=" + url.QueryEscape("wood "), "wood-fired"},
		"a search that ends after a word":   {"?search=cafe-", "café,cafe-bar"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := e.tags(tc.query).names(); got != tc.want {
				t.Errorf("%s: %s, want %s", tc.query, got, tc.want)
			}
		})
	}

	t.Run("the exact name comes first", func(t *testing.T) {
		if got := e.tags("?search=caf%C3%A9&sort=name").names(); got != "café,cafe-bar" {
			t.Errorf("café first: %s", got)
		}
		// The restaurant cafe-bar has no recipe. The exact tag still comes.
		if got := e.tags("?search=cafe-bar&kind=recipes").names(); got != "cafe-bar" {
			t.Errorf("exact tag of the other kind: %s", got)
		}
		if got := e.tags("?search=cafe&kind=recipes").names(); got != "café" {
			t.Errorf("a prefix of the other kind: %s", got)
		}
		page := e.tags("?search=wood-fired&limit=1")
		if page.names() != "wood-fired" || page.NextCursor != nil {
			t.Errorf("exact page %+v", page)
		}
	})

	t.Run("the counts", func(t *testing.T) {
		page := e.tags("?search=wood")
		if len(page.Items) != 1 || page.Items[0].RestaurantCount != 2 || page.Items[0].RecipeCount != 2 {
			t.Errorf("tags %+v", page.Items)
		}
	})

	t.Run("the pages", func(t *testing.T) {
		var names []string
		query := "?limit=1"
		for range 5 {
			page := e.tags(query)
			names = append(names, page.names())
			if page.NextCursor == nil {
				break
			}
			query = "?limit=1&cursor=" + *page.NextCursor
		}
		if got := strings.Join(names, ","); got != "wood-fired,café,cafe-bar,ラーメン" {
			t.Errorf("pages %s", got)
		}
		cursor := *e.tags("?limit=1").NextCursor
		e.expect(request{method: http.MethodGet, path: "/v1/tags?kind=recipes&limit=1&cursor=" + cursor}, http.StatusBadRequest)
	})

	for _, query := range []string{"?sort=created", "?kind=wines", "?limit=0", "?search=", "?search=%21&cursor=bad"} {
		t.Run(query+" gets 400", func(t *testing.T) {
			e.expect(request{method: http.MethodGet, path: "/v1/tags" + query}, http.StatusBadRequest)
		})
	}
}

func TestListTagsKey(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	e.createRestaurant(ana.Token, `{"name":"A","tags":["café","cafe-bar","cafés"]}`)
	e.createRestaurant(ana.Token, `{"name":"B","tags":["café","cafe-bar"]}`)
	e.createRecipe(ana.Token, `{"name":"C","tags":["cafe"]}`)

	for query, want := range map[string]string{
		"?search=cafe":                   "cafe,café,cafe-bar,cafés",
		"?search=Caf%C3%A9":              "café,cafe,cafe-bar,cafés",
		"?search=cafe&kind=recipes":      "cafe,café",
		"?search=cafe&kind=restaurants":  "cafe,café,cafe-bar,cafés",
		"?search=cafe&sort=-name":        "cafe,café,cafés,cafe-bar",
		"?search=cafes&kind=restaurants": "cafés",
	} {
		if got := e.tags(query).names(); got != want {
			t.Errorf("%s: %s, want %s", query, got, want)
		}
	}
	var names []string
	query := "?search=cafe&limit=1"
	for range 5 {
		page := e.tags(query)
		names = append(names, page.names())
		if page.NextCursor == nil {
			break
		}
		query = "?search=cafe&limit=1&cursor=" + *page.NextCursor
	}
	if got := strings.Join(names, ","); got != "cafe,café,cafe-bar,cafés" {
		t.Errorf("pages %s", got)
	}
}

func TestListTagsGreek(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	e.createRestaurant(ana.Token, `{"name":"A","tags":["οδοσημανση","οδος"]}`)
	for search, want := range map[string]string{"οδοσ": "οδος,οδοσημανση", "ΟΔΟΣ": "οδος,οδοσημανση", "οδος": "οδος", "οδοσ%20": "οδος"} {
		if got := e.tags("?search=" + search).names(); got != want {
			t.Errorf("%s: %s, want %s", search, got, want)
		}
	}
}

func TestGetTag(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	e.createRestaurant(ana.Token, `{"name":"A","tags":["café"]}`)
	e.createRecipe(ana.Token, `{"name":"B","tags":["café"]}`)

	for _, path := range []string{"/v1/tags/caf%C3%A9", "/v1/tags/cafe%CC%81", "/v1/tags/Caf%C3%A9", "/v1/tags/%20CAF%C3%89!"} {
		got := e.expect(request{method: http.MethodGet, path: path}, http.StatusOK).object(t)
		if got["name"] != "café" || got["restaurant_count"] != 1.0 || got["recipe_count"] != 1.0 {
			t.Errorf("%s: %v", path, got)
		}
	}
	e.expect(request{method: http.MethodGet, path: "/v1/tags/%21%21%21"}, http.StatusBadRequest)
	e.expect(request{method: http.MethodGet, path: "/v1/tags/cafe"}, http.StatusNotFound)
	e.expect(request{method: http.MethodGet, path: "/v1/tags/pasta"}, http.StatusNotFound)
	restaurants := e.expect(request{method: http.MethodGet, path: "/v1/restaurants?tags[has]=cafe%CC%81"}, http.StatusOK).object(t)
	if n := len(restaurants["items"].([]any)); n != 1 {
		t.Errorf("%d restaurants with the tag, want 1", n)
	}
}

func TestDashboardTags(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	root := e.superuserToken("root@example.com")
	records := "/api/collections/tags/records"
	id := e.createRestaurant(ana.Token, `{"name":"A","tags":["woodfired","pizza"]}`)["id"].(string)
	path := "/v1/restaurants/" + id

	for name, body := range map[string]string{
		"a name without a letter":        `{"name":"!!!"}`,
		"a name that exists":             `{"name":"pizza"}`,
		"a text with the name of a tag":  `{"name":" Pizza️ "}`,
		"a name of more than 63 letters": `{"name":"` + strings.Repeat("a", 64) + `"}`,
	} {
		t.Run("the dashboard refuses "+name, func(t *testing.T) {
			res := e.dashboard(request{method: http.MethodPost, path: records, token: root, body: body}, http.StatusBadRequest)
			if data, _ := res.object(t)["data"].(map[string]any); data["name"] == nil {
				t.Errorf("errors %s, want an error of name", res.body)
			}
		})
	}

	t.Run("the dashboard changes a name to normalization form C", func(t *testing.T) {
		got := e.dashboard(request{method: http.MethodPost, path: records, token: root, body: `{"name":"café"}`}, http.StatusOK).object(t)
		if got["name"] != "café" || got["name_key"] != "cafe" {
			t.Errorf("tag %v", got)
		}
	})

	t.Run("the dashboard changes free text to a tag", func(t *testing.T) {
		got := e.dashboard(request{method: http.MethodPost, path: records, token: root, body: `{"name":"Dim Sum"}`}, http.StatusOK).object(t)
		if got["name"] != "dim-sum" || got["name_key"] != "dim-sum" {
			t.Errorf("tag %v", got)
		}
	})

	t.Run("a rename changes the restaurant", func(t *testing.T) {
		tag, err := e.app.FindFirstRecordByData("tags", "name", "woodfired")
		if err != nil {
			t.Fatal(err)
		}
		e.dashboard(request{method: http.MethodPatch, path: records + "/" + tag.Id, token: root, body: `{"name":"wood-fired"}`}, http.StatusOK)
		got := e.expect(request{method: http.MethodGet, path: path}, http.StatusOK).object(t)
		if fmt.Sprint(got["tags"]) != "[wood-fired pizza]" {
			t.Errorf("tags %v", got["tags"])
		}
		search := e.expect(request{method: http.MethodGet, path: "/v1/restaurants?search=wood"}, http.StatusOK).object(t)
		if n := len(search["items"].([]any)); n != 1 {
			t.Errorf("%d restaurants for the new name, want 1", n)
		}
		vs := e.versions(path+"/versions", ana.Token).Items
		if len(vs) != 2 || vs[0].Source != "admin" {
			t.Errorf("versions %+v", vs)
		}
	})

	t.Run("a delete removes the tag from the restaurant", func(t *testing.T) {
		tag, err := e.app.FindFirstRecordByData("tags", "name", "pizza")
		if err != nil {
			t.Fatal(err)
		}
		e.dashboard(request{method: http.MethodDelete, path: records + "/" + tag.Id, token: root}, http.StatusNoContent)
		got := e.expect(request{method: http.MethodGet, path: path}, http.StatusOK).object(t)
		if fmt.Sprint(got["tags"]) != "[wood-fired]" {
			t.Errorf("tags %v", got["tags"])
		}
		if n := len(e.versions(path+"/versions", ana.Token).Items); n != 3 {
			t.Errorf("%d versions, want 3", n)
		}
	})
}
