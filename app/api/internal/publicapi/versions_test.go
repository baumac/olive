package publicapi_test

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

type versionPage struct {
	Items []struct {
		ID     string `json:"id"`
		Number int    `json:"number"`
		Action string `json:"action"`
		Source string `json:"source"`
		Editor *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"editor"`
		Before  map[string]any `json:"before"`
		After   map[string]any `json:"after"`
		Created string         `json:"created"`
	} `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

func (e *env) versions(path, token string) versionPage {
	e.t.Helper()
	var page versionPage
	e.expect(request{method: http.MethodGet, path: path, token: token}, http.StatusOK).decode(e.t, &page)
	return page
}

func TestRestaurantVersions(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	eva := e.signUp("eva@example.com", "Eva")
	id := e.createRestaurant(ana.Token, `{"name":"Can Solé","description":"Small tapas bar","phone":"931 000 000"}`)["id"].(string)
	path := "/v1/restaurants/" + id
	e.expect(request{method: http.MethodPatch, path: path, token: joe.Token, body: `{"phone":"931 111 111"}`}, http.StatusOK)
	e.expect(request{method: http.MethodPatch, path: path, token: eva.Token, body: `{"description":null,"tags":["tapas"]}`}, http.StatusOK)
	e.expect(request{method: http.MethodPatch, path: path, token: eva.Token, body: `{"tags":["tapas"]}`}, http.StatusOK)

	t.Run("the newest version comes first with the changed properties", func(t *testing.T) {
		page := e.versions(path+"/versions", joe.Token)
		if len(page.Items) != 3 || page.NextCursor != nil {
			t.Fatalf("%d versions, want 3 on one page", len(page.Items))
		}
		eva3, joe2, ana1 := page.Items[0], page.Items[1], page.Items[2]
		if eva3.Number != 3 || eva3.Action != "update" || eva3.Source != "user" || eva3.Editor == nil || eva3.Editor.Name != "Eva" {
			t.Errorf("version 3: %+v", eva3)
		}
		if want := map[string]any{"description": "Small tapas bar", "tags": nil}; !reflect.DeepEqual(eva3.Before, want) {
			t.Errorf("version 3 before %v, want %v", eva3.Before, want)
		}
		if want := map[string]any{"description": nil, "tags": []any{"tapas"}}; !reflect.DeepEqual(eva3.After, want) {
			t.Errorf("version 3 after %v, want %v", eva3.After, want)
		}
		if want := map[string]any{"phone": "931 000 000"}; joe2.Number != 2 || !reflect.DeepEqual(joe2.Before, want) {
			t.Errorf("version 2: %+v", joe2)
		}
		if ana1.Number != 1 || ana1.Action != "create" || len(ana1.Before) != 0 || ana1.After["name"] != "Can Solé" || ana1.After["location"] == nil {
			t.Errorf("version 1: %+v", ana1)
		}
	})

	t.Run("a page continues with the cursor", func(t *testing.T) {
		first := e.versions(path+"/versions?limit=2", ana.Token)
		if len(first.Items) != 2 || first.NextCursor == nil {
			t.Fatalf("first page: %d versions, cursor %v", len(first.Items), first.NextCursor)
		}
		second := e.versions(path+"/versions?limit=2&cursor="+url.QueryEscape(*first.NextCursor), ana.Token)
		if len(second.Items) != 1 || second.Items[0].Number != 1 || second.NextCursor != nil {
			t.Fatalf("second page: %+v", second)
		}
		e.expect(request{method: http.MethodGet, path: "/v1/restaurants/" + e.createRestaurant(ana.Token, `{"name":"Other"}`)["id"].(string) +
			"/versions?cursor=" + url.QueryEscape(*first.NextCursor), token: ana.Token}, http.StatusBadRequest)
	})

	t.Run("an undo is a new version", func(t *testing.T) {
		e.expect(request{method: http.MethodPatch, path: path, token: joe.Token, body: `{"description":"Small tapas bar"}`}, http.StatusOK)
		v := e.versions(path+"/versions?limit=1", ana.Token).Items[0]
		if v.Number != 4 || v.Editor.Name != "Joe" || !reflect.DeepEqual(v.After, map[string]any{"description": "Small tapas bar"}) {
			t.Errorf("version 4: %+v", v)
		}
	})

	t.Run("a dashboard change has the source admin", func(t *testing.T) {
		root := e.superuserToken("root@example.com")
		e.dashboard(request{method: http.MethodPatch, path: "/api/collections/restaurants/records/" + id, token: root, body: `{"website":"https://can.example"}`}, http.StatusOK)
		v := e.versions(path+"/versions?limit=1", ana.Token).Items[0]
		if v.Number != 5 || v.Source != "admin" || v.Editor != nil || !reflect.DeepEqual(v.After, map[string]any{"website": "https://can.example"}) {
			t.Errorf("version 5: %+v", v)
		}
	})

	t.Run("a deleted user has no editor", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/users/me/deletion", token: eva.Token, body: `{"password":"correct-horse-battery"}`}, http.StatusNoContent)
		page := e.versions(path+"/versions", ana.Token)
		v := page.Items[len(page.Items)-3]
		if v.Number != 3 || v.Source != "user" || v.Editor != nil {
			t.Errorf("version 3 after the delete of Eva: %+v", v)
		}
	})

	t.Run("only approved users can read the history", func(t *testing.T) {
		e.expect(request{method: http.MethodGet, path: path + "/versions"}, http.StatusUnauthorized)
		pending := e.signUpPending("tom@example.com", "Tom")
		e.expectCode(request{method: http.MethodGet, path: path + "/versions", token: pending.Token}, http.StatusForbidden, "not_approved")
		e.expect(request{method: http.MethodGet, path: "/v1/restaurants/aaaaaaaaaaaaaaa/versions", token: ana.Token}, http.StatusNotFound)
	})
}

func TestRestaurantVersionsStartAtTheBaseline(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	id := e.createRestaurant(ana.Token, `{"name":"Old","phone":"931 000 000"}`)["id"].(string)
	// A restaurant from before the migration has only the baseline.
	if _, err := e.app.DB().NewQuery("UPDATE {{restaurant_versions}} SET [[action]] = 'baseline', [[source]] = 'admin', [[editor]] = '', [[changed]] = '[]'").Execute(); err != nil {
		t.Fatal(err)
	}
	path := "/v1/restaurants/" + id
	if page := e.versions(path+"/versions", ana.Token); len(page.Items) != 0 {
		t.Fatalf("%d versions, want no baseline in the list", len(page.Items))
	}
	e.expect(request{method: http.MethodPatch, path: path, token: ana.Token, body: `{"phone":"931 111 111"}`}, http.StatusOK)
	page := e.versions(path+"/versions", ana.Token)
	if len(page.Items) != 1 || page.Items[0].Number != 2 || !reflect.DeepEqual(page.Items[0].Before, map[string]any{"phone": "931 000 000"}) {
		t.Errorf("versions after the baseline: %+v", page.Items)
	}
}

func TestRecipeVersions(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	id := e.createRecipe(ana.Token, `{"name":"Dal","ingredients":["lentils"],"steps":["Cook."],"servings":2}`)["id"].(string)
	path := "/v1/recipes/" + id
	e.expect(request{method: http.MethodPatch, path: path, token: joe.Token, body: `{"steps":[],"servings":null}`}, http.StatusUnprocessableEntity)
	e.expect(request{method: http.MethodPatch, path: path, token: joe.Token, body: `{"steps":["Rinse.","Cook."],"servings":null}`}, http.StatusOK)

	page := e.versions(path+"/versions", ana.Token)
	if len(page.Items) != 2 {
		t.Fatalf("%d versions, want 2", len(page.Items))
	}
	v := page.Items[0]
	if v.Editor == nil || v.Editor.Name != "Joe" {
		t.Errorf("editor %+v, want Joe", v.Editor)
	}
	if want := map[string]any{"steps": []any{"Cook."}, "servings": 2.0}; !reflect.DeepEqual(v.Before, want) {
		t.Errorf("before %v, want %v", v.Before, want)
	}
	if want := map[string]any{"steps": []any{"Rinse.", "Cook."}, "servings": nil}; !reflect.DeepEqual(v.After, want) {
		t.Errorf("after %v, want %v", v.After, want)
	}
	e.expect(request{method: http.MethodGet, path: path + "/versions"}, http.StatusUnauthorized)
}
