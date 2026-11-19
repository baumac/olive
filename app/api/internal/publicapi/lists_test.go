package publicapi_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type listPage struct {
	Items []struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		Kind            string `json:"kind"`
		RestaurantCount int    `json:"restaurant_count"`
		RecipeCount     int    `json:"recipe_count"`
		HasRestaurant   *bool  `json:"has_restaurant"`
		HasRecipe       *bool  `json:"has_recipe"`
		Creator         struct {
			ID string `json:"id"`
		} `json:"creator"`
	} `json:"items"`
}

func (e *env) lists(userID, query string) listPage {
	e.t.Helper()
	var p listPage
	e.expect(request{method: "GET", path: "/v1/reviewers/" + userID + "/lists" + query}, http.StatusOK).decode(e.t, &p)
	return p
}

func (e *env) listsAs(userID, token string) listPage {
	e.t.Helper()
	var p listPage
	e.expect(request{method: "GET", path: "/v1/reviewers/" + userID + "/lists", token: token}, http.StatusOK).decode(e.t, &p)
	return p
}

func (e *env) listRestaurants(path, token string) (string, *string) {
	e.t.Helper()
	var p struct {
		Items []struct {
			Name     string `json:"name"`
			Favorite bool   `json:"favorite"`
		} `json:"items"`
		NextCursor *string `json:"next_cursor"`
	}
	e.expect(request{method: "GET", path: path, token: token}, http.StatusOK).decode(e.t, &p)
	var out []string
	for _, it := range p.Items {
		name := it.Name
		if it.Favorite {
			name += "*"
		}
		out = append(out, name)
	}
	return strings.Join(out, ","), p.NextCursor
}

func TestFavorites(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	luigi := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)
	if luigi["favorite"] != false {
		t.Errorf("create: favorite %v", luigi["favorite"])
	}
	sole := e.createRestaurant(joe.Token, `{"name":"Can Sole"}`)["id"].(string)
	luigiID := luigi["id"].(string)

	lists := e.lists(ana.User.ID, "")
	if len(lists.Items) != 1 || lists.Items[0].Kind != "favorites" || lists.Items[0].Name != "Favorites" ||
		lists.Items[0].RestaurantCount != 0 || lists.Items[0].Creator.ID != ana.User.ID || lists.Items[0].HasRestaurant != nil {
		t.Fatalf("lists of a new user: %+v", lists.Items)
	}
	favorites := "/v1/lists/" + lists.Items[0].ID

	put := "/v1/lists/favorites/restaurants/"
	e.expect(request{method: "PUT", path: put + luigiID}, http.StatusUnauthorized)
	e.expect(request{method: "PUT", path: put + luigiID, token: ana.Token}, http.StatusNoContent)
	e.expect(request{method: "PUT", path: put + luigiID, token: ana.Token}, http.StatusNoContent)
	e.expect(request{method: "PUT", path: put + "aaaaaaaaaaaaaaa", token: ana.Token}, http.StatusNotFound)
	e.expect(request{method: "PUT", path: put + "BAD", token: ana.Token}, http.StatusBadRequest)
	e.expect(request{method: "PUT", path: "/v1/lists/favourites/restaurants/" + luigiID, token: ana.Token}, http.StatusBadRequest)
	e.expect(request{method: "POST", path: put + luigiID, token: ana.Token}, http.StatusMethodNotAllowed)

	for token, want := range map[string]string{ana.Token: "Luigi's*,Can Sole", joe.Token: "Luigi's,Can Sole", "": "Luigi's,Can Sole"} {
		if got, _ := e.listRestaurants("/v1/restaurants?sort=-name", token); got != want {
			t.Errorf("restaurants with the token %q: %q, want %q", token, got, want)
		}
	}
	if got := e.expect(request{method: "GET", path: "/v1/restaurants/" + luigiID, token: ana.Token}, http.StatusOK).object(t)["favorite"]; got != true {
		t.Errorf("get: favorite %v", got)
	}
	patched := e.expect(request{method: "PATCH", path: "/v1/restaurants/" + luigiID, body: `{"description":"Good"}`, token: ana.Token}, http.StatusOK).object(t)
	if patched["favorite"] != true {
		t.Errorf("patch: favorite %v", patched["favorite"])
	}
	if got, _ := e.listRestaurants(favorites+"/restaurants", ""); got != "Luigi's" {
		t.Errorf("favorites of Ana without a token: %q", got)
	}
	if got, _ := e.listRestaurants(favorites+"/restaurants", ana.Token); got != "Luigi's*" {
		t.Errorf("favorites of Ana with her token: %q", got)
	}

	t.Run("the favorites list is fixed", func(t *testing.T) {
		e.expectCode(request{method: "DELETE", path: favorites, token: ana.Token}, http.StatusForbidden, "favorites_list")
		e.expectCode(request{method: "PATCH", path: favorites, body: `{"name":"Best"}`, token: ana.Token}, http.StatusForbidden, "favorites_list")
		e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"favorites"}`, token: ana.Token}, http.StatusConflict)
		if got := e.lists(ana.User.ID, "").Items; len(got) != 1 || got[0].Name != "Favorites" {
			t.Errorf("lists: %+v", got)
		}
	})

	t.Run("only the creator changes a list", func(t *testing.T) {
		id := lists.Items[0].ID
		e.expectCode(request{method: "PUT", path: "/v1/lists/" + id + "/restaurants/" + sole, token: joe.Token}, http.StatusForbidden, "not_creator")
		e.expect(request{method: "DELETE", path: "/v1/lists/" + id + "/restaurants/" + luigiID, token: joe.Token}, http.StatusForbidden)
		e.expect(request{method: "PUT", path: "/v1/lists/" + id + "/restaurants/" + sole, token: ana.Token}, http.StatusNoContent)
		if got, _ := e.listRestaurants(favorites+"/restaurants", joe.Token); got != "Can Sole,Luigi's" {
			t.Errorf("favorites of Ana: %q", got)
		}
	})

	t.Run("remove", func(t *testing.T) {
		del := request{method: "DELETE", path: put + luigiID, token: ana.Token}
		e.expect(del, http.StatusNoContent)
		e.expect(del, http.StatusNoContent)
		if got, _ := e.listRestaurants("/v1/restaurants?sort=name", ana.Token); got != "Can Sole*,Luigi's" {
			t.Errorf("restaurants after the remove: %q", got)
		}
	})

	t.Run("a deleted restaurant leaves the list", func(t *testing.T) {
		e.expect(request{method: "DELETE", path: "/v1/restaurants/" + sole, token: joe.Token}, http.StatusNoContent)
		if got := e.lists(ana.User.ID, "").Items[0].RestaurantCount; got != 0 {
			t.Errorf("restaurant_count %d, want 0", got)
		}
	})
}

func TestLists(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	var restaurants []string
	for _, name := range []string{"A", "B", "C"} {
		restaurants = append(restaurants, e.createRestaurant(ana.Token, fmt.Sprintf(`{"name":%q}`, name))["id"].(string))
	}

	e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Date night"}`}, http.StatusUnauthorized)
	res := e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Date night"}`, token: ana.Token}, http.StatusCreated)
	list := res.object(t)
	id := list["id"].(string)
	path := "/v1/lists/" + id
	if res.header.Get("Location") != path || list["kind"] != "custom" || list["restaurant_count"] != 0.0 ||
		list["creator"].(map[string]any)["name"] != "Ana" {
		t.Errorf("create: %v, Location %q", list, res.header.Get("Location"))
	}
	if _, ok := list["has_restaurant"]; ok {
		t.Errorf("create: has_restaurant in %v", list)
	}
	e.expectCode(request{method: "POST", path: "/v1/lists", body: `{"name":"DATE NIGHT"}`, token: ana.Token}, http.StatusConflict, "list_name_taken")
	e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Date night"}`, token: joe.Token}, http.StatusCreated)
	e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":" "}`, token: ana.Token}, http.StatusUnprocessableEntity)
	e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"X","kind":"favorites"}`, token: ana.Token}, http.StatusBadRequest)

	for i, r := range restaurants {
		e.expect(request{method: "PUT", path: path + "/restaurants/" + r, token: ana.Token}, http.StatusNoContent)
		if i < len(restaurants)-1 {
			time.Sleep(2 * time.Millisecond)
		}
	}
	got := e.expect(request{method: "GET", path: path}, http.StatusOK).object(t)
	if got["restaurant_count"] != 3.0 || got["name"] != "Date night" {
		t.Errorf("get: %v", got)
	}

	t.Run("pages", func(t *testing.T) {
		first, cursor := e.listRestaurants(path+"/restaurants?limit=2", "")
		if first != "C,B" || cursor == nil {
			t.Fatalf("first page %q, cursor %v", first, cursor)
		}
		second, next := e.listRestaurants(path+"/restaurants?limit=2&cursor="+url.QueryEscape(*cursor), "")
		if second != "A" || next != nil {
			t.Errorf("second page %q, cursor %v", second, next)
		}
		e.expect(request{method: "GET", path: "/v1/lists/aaaaaaaaaaaaaaa/restaurants"}, http.StatusNotFound)
		e.expect(request{method: "GET", path: path + "/restaurants?cursor=x"}, http.StatusBadRequest)
		e.expect(request{method: "GET", path: path + "/restaurants?sort=name"}, http.StatusBadRequest)
	})

	t.Run("the lists of a user", func(t *testing.T) {
		time.Sleep(2 * time.Millisecond)
		e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Lunch"}`, token: ana.Token}, http.StatusCreated)
		lists := e.lists(ana.User.ID, "?restaurant[eq]="+restaurants[0])
		var names []string
		for _, l := range lists.Items {
			if l.HasRestaurant == nil {
				t.Fatalf("%s has no has_restaurant", l.Name)
			}
			names = append(names, fmt.Sprintf("%s:%d:%v", l.Name, l.RestaurantCount, *l.HasRestaurant))
		}
		if got := strings.Join(names, ","); got != "Favorites:0:false,Lunch:0:false,Date night:3:true" {
			t.Errorf("lists: %s", got)
		}
		e.expect(request{method: "GET", path: "/v1/reviewers/aaaaaaaaaaaaaaa/lists"}, http.StatusNotFound)
		profile := e.expect(request{method: "GET", path: "/v1/reviewers/" + ana.User.ID}, http.StatusOK).object(t)
		if profile["list_count"] != 3.0 {
			t.Errorf("list_count %v, want 3", profile["list_count"])
		}
	})

	t.Run("rename", func(t *testing.T) {
		e.expectCode(request{method: "PATCH", path: path, body: `{"name":"Dinner"}`, token: joe.Token}, http.StatusForbidden, "not_creator")
		e.expect(request{method: "PATCH", path: path, body: `{"name":"lunch"}`, token: ana.Token}, http.StatusConflict)
		e.expect(request{method: "PATCH", path: path, body: `{"name":"favorites"}`, token: ana.Token}, http.StatusConflict)
		e.expect(request{method: "PATCH", path: "/v1/lists/aaaaaaaaaaaaaaa", body: `{"name":"Dinner"}`, token: ana.Token}, http.StatusNotFound)
		renamed := e.expect(request{method: "PATCH", path: path, body: `{"name":"DATE night"}`, token: ana.Token}, http.StatusOK).object(t)
		if renamed["name"] != "DATE night" || renamed["restaurant_count"] != 3.0 {
			t.Errorf("rename: %v", renamed)
		}
		same := e.expect(request{method: "PATCH", path: path, body: `{}`, token: ana.Token}, http.StatusOK).object(t)
		if same["name"] != "DATE night" {
			t.Errorf("empty patch: %v", same)
		}
	})

	t.Run("delete", func(t *testing.T) {
		e.expect(request{method: "DELETE", path: path, token: joe.Token}, http.StatusForbidden)
		e.expect(request{method: "DELETE", path: path, token: ana.Token}, http.StatusNoContent)
		e.expect(request{method: "DELETE", path: path, token: ana.Token}, http.StatusNotFound)
		e.expect(request{method: "GET", path: path}, http.StatusNotFound)
		e.expect(request{method: "GET", path: "/v1/restaurants/" + restaurants[0]}, http.StatusOK)
	})
}

func TestListLimits(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	for i := range 49 {
		e.expect(request{method: "POST", path: "/v1/lists", body: fmt.Sprintf(`{"name":"L%d"}`, i), token: ana.Token}, http.StatusCreated)
	}
	e.expectCode(request{method: "POST", path: "/v1/lists", body: `{"name":"One more"}`, token: ana.Token}, http.StatusConflict, "list_limit")
	if got := len(e.lists(ana.User.ID, "").Items); got != 50 {
		t.Errorf("%d lists, want 50", got)
	}
}

func TestListItemLimit(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	list := e.lists(ana.User.ID, "").Items[0].ID
	// The rows have no restaurant record. The limit check only counts them.
	_, err := e.app.DB().NewQuery(`
		WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 500)
		INSERT INTO list_items (list, restaurant, created) SELECT {:list}, printf('x%014d', i), '2024-11-04 01:26:55.002Z' FROM n`).
		Bind(map[string]any{"list": list}).Execute()
	if err != nil {
		t.Fatal(err)
	}
	e.expectCode(request{method: "PUT", path: "/v1/lists/favorites/restaurants/" + restaurant, token: ana.Token}, http.StatusConflict, "list_restaurant_limit")
}

func TestListNamesAndUpdated(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)

	list := e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"  Date night  "}`, token: ana.Token}, http.StatusCreated).object(t)
	if list["name"] != "Date night" {
		t.Errorf("create: name %q", list["name"])
	}
	path := "/v1/lists/" + list["id"].(string)
	e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":" Date night"}`, token: ana.Token}, http.StatusConflict)
	other := e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Lunch"}`, token: ana.Token}, http.StatusCreated).object(t)
	e.expect(request{method: "PATCH", path: "/v1/lists/" + other["id"].(string), body: `{"name":"date night "}`, token: ana.Token}, http.StatusConflict)
	renamed := e.expect(request{method: "PATCH", path: path, body: `{"name":" Dinner "}`, token: ana.Token}, http.StatusOK).object(t)
	if renamed["name"] != "Dinner" {
		t.Errorf("rename: name %q", renamed["name"])
	}

	updated := func() string {
		t.Helper()
		return e.expect(request{method: "GET", path: path}, http.StatusOK).object(t)["updated"].(string)
	}
	before := updated()
	put := request{method: "PUT", path: path + "/restaurants/" + restaurant, token: ana.Token}
	time.Sleep(2 * time.Millisecond)
	e.expect(put, http.StatusNoContent)
	added := updated()
	if added <= before {
		t.Errorf("updated %s after an add, want after %s", added, before)
	}
	time.Sleep(2 * time.Millisecond)
	e.expect(put, http.StatusNoContent)
	if got := updated(); got != added {
		t.Errorf("updated %s after an add that changes nothing, want %s", got, added)
	}
	time.Sleep(2 * time.Millisecond)
	e.expect(request{method: "DELETE", path: path + "/restaurants/" + restaurant, token: ana.Token}, http.StatusNoContent)
	if got := updated(); got <= added {
		t.Errorf("updated %s after a remove, want after %s", got, added)
	}
}

func TestFavoritesListOfOtherUsers(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)

	t.Run("a dashboard user gets the list", func(t *testing.T) {
		users, err := e.app.FindCollectionByNameOrId("users")
		if err != nil {
			t.Fatal(err)
		}
		joe := core.NewRecord(users)
		joe.SetEmail("joe@example.com")
		joe.SetPassword("correct-horse-battery")
		joe.Set("name", "Joe")
		joe.Set("approved", true)
		if err := e.app.Save(joe); err != nil {
			t.Fatal(err)
		}
		if got := e.lists(joe.Id, "").Items; len(got) != 1 || got[0].Kind != "favorites" {
			t.Errorf("lists of a dashboard user: %+v", got)
		}
	})

	t.Run("a user without the list gets it at the first favorite", func(t *testing.T) {
		if _, err := e.app.DB().NewQuery("DELETE FROM lists WHERE creator = {:id}").Bind(map[string]any{"id": ana.User.ID}).Execute(); err != nil {
			t.Fatal(err)
		}
		if got := e.lists(ana.User.ID, "").Items; len(got) != 0 {
			t.Fatalf("lists after the SQL delete: %+v", got)
		}
		e.expect(request{method: "PUT", path: "/v1/lists/favorites/restaurants/" + restaurant, token: ana.Token}, http.StatusNoContent)
		got := e.lists(ana.User.ID, "").Items
		if len(got) != 1 || got[0].Kind != "favorites" || got[0].RestaurantCount != 1 {
			t.Errorf("lists after the favorite: %+v", got)
		}
	})
}

func TestListFilter(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	place := func(name string, lat float64) string {
		body := fmt.Sprintf(`{"name":%q,"location":{"latitude":%v,"longitude":2,"address":"Street"}}`, name, lat)
		return e.createRestaurant(ana.Token, body)["id"].(string)
	}
	far, near, other := place("Far", 41.5), place("Near", 41.1), place("Other", 41.0)
	noLocation := e.createRestaurant(ana.Token, `{"name":"Nowhere","tags":["pizza"]}`)["id"].(string)
	if _, err := e.app.DB().NewQuery("UPDATE restaurants SET address = '' WHERE id = {:id}").
		Bind(map[string]any{"id": noLocation}).Execute(); err != nil {
		t.Fatal(err)
	}
	favorites := e.lists(ana.User.ID, "").Items[0].ID
	date := e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Date night"}`, token: ana.Token}, http.StatusCreated).object(t)["id"].(string)
	for _, r := range []string{far, near, noLocation} {
		e.expect(request{method: "PUT", path: "/v1/lists/favorites/restaurants/" + r, token: ana.Token}, http.StatusNoContent)
	}
	e.expect(request{method: "PUT", path: "/v1/lists/" + date + "/restaurants/" + other, token: ana.Token}, http.StatusNoContent)
	e.expect(request{method: "PUT", path: "/v1/lists/favorites/restaurants/" + other, token: joe.Token}, http.StatusNoContent)

	base := "/v1/restaurants?list[eq]=" + favorites
	for name, c := range map[string]struct{ query, token, want string }{
		"name":          {query: "&sort=name", want: "Far,Near,Nowhere"},
		"favorite flag": {query: "&sort=name", token: ana.Token, want: "Far*,Near*,Nowhere*"},
		"distance":      {query: "&sort=distance&near[latitude]=41&near[longitude]=2", want: "Near,Far,Nowhere"},
		"radius":        {query: "&near[latitude]=41&near[longitude]=2&near[radius]=20000", want: "Near"},
		"within":        {query: "&within[west]=1&within[south]=41.4&within[east]=3&within[north]=42", want: "Far"},
		"tag":           {query: "&tags[has]=pizza", want: "Nowhere"},
		"search":        {query: "&search=near", want: "Near"},
	} {
		t.Run(name, func(t *testing.T) {
			path := base + c.query
			if got, _ := e.listRestaurants(path, c.token); got != c.want {
				t.Errorf("%s: %q, want %q", path, got, c.want)
			}
		})
	}

	if got, _ := e.listRestaurants("/v1/restaurants?list[eq]="+date, ""); got != "Other" {
		t.Errorf("restaurants of Date night: %q", got)
	}

	t.Run("pages", func(t *testing.T) {
		first, cursor := e.listRestaurants(base+"&sort=name&limit=2", "")
		if first != "Far,Near" || cursor == nil {
			t.Fatalf("first page %q, cursor %v", first, cursor)
		}
		second, next := e.listRestaurants(base+"&sort=name&limit=2&cursor="+url.QueryEscape(*cursor), "")
		if second != "Nowhere" || next != nil {
			t.Errorf("second page %q, cursor %v", second, next)
		}
		other := "/v1/restaurants?list[eq]=" + date + "&sort=name&limit=2&cursor=" + url.QueryEscape(*cursor)
		e.expect(request{method: "GET", path: other}, http.StatusBadRequest)
	})

	t.Run("an unknown list has no restaurants", func(t *testing.T) {
		if got, next := e.listRestaurants("/v1/restaurants?list[eq]=aaaaaaaaaaaaaaa", ""); got != "" || next != nil {
			t.Errorf("restaurants %q, cursor %v", got, next)
		}
		e.expect(request{method: "GET", path: "/v1/restaurants?list[eq]=favorites"}, http.StatusBadRequest)
	})
}
