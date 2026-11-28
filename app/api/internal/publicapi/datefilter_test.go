package publicapi_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestDateFilter(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	dates := []string{"2025-11-22", "2025-11-23", "2025-11-29", "2025-11-30"}

	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	recipe := e.createRecipe(ana.Token, `{"name":"Shakshuka"}`)["id"].(string)
	pantryItem := e.createPantryItem(ana.Token, `{"name":"Rioja Reserva"}`)["id"].(string)
	for _, d := range dates {
		e.createReview(ana.Token, restaurant, fmt.Sprintf(`{"date":%q,"space":7,"service":7,"food":7,"price":7,"title":%q}`, d, d))
		e.createRecipeReview(ana.Token, recipe, fmt.Sprintf(`{"date":%q,"taste":7,"ease":7,"instructions":7,"value":7,"title":%q}`, d, d))
		e.createPantryItemReview(ana.Token, pantryItem, fmt.Sprintf(`{"date":%q,"taste":7,"aroma":7,"texture":7,"value":7,"title":%q}`, d, d))
	}

	lists := []string{
		"/v1/restaurant-reviews", "/v1/restaurants/" + restaurant + "/reviews", "/v1/reviewers/" + ana.User.ID + "/restaurant-reviews",
		"/v1/recipe-reviews", "/v1/recipes/" + recipe + "/reviews", "/v1/reviewers/" + ana.User.ID + "/recipe-reviews",
		"/v1/pantry-item-reviews", "/v1/pantry/" + pantryItem + "/reviews", "/v1/reviewers/" + ana.User.ID + "/pantry-item-reviews",
	}
	cases := map[string]string{
		"date[lt]=2025-11-29":                                 "2025-11-23,2025-11-22",
		"date[lte]=2025-11-29":                                "2025-11-29,2025-11-23,2025-11-22",
		"date[eq]=2025-11-23":                                 "2025-11-23",
		"date[gte]=2025-11-23":                                "2025-11-30,2025-11-29,2025-11-23",
		"date[gt]=2025-11-23":                                 "2025-11-30,2025-11-29",
		"date[gte]=2025-11-23&date[lte]=2025-11-29":           "2025-11-29,2025-11-23",
		"date[gte]=2025-11-23&date[lte]=2025-11-29&sort=date": "2025-11-23,2025-11-29",
		"date[gt]=2999-01-01":                                 "",
	}
	for _, path := range lists {
		for query, want := range cases {
			if got, _ := e.names(path + "?" + query); got != want {
				t.Errorf("%s?%s: %q, want %q", path, query, got, want)
			}
		}
		for _, query := range []string{"date[gte]=2026-02-30", "date[eq]=2025-1-1", "date[foo]=2025-11-23", "date=2025-11-23"} {
			e.expect(request{method: "GET", path: path + "?" + query}, http.StatusBadRequest)
		}
	}

	res := e.expect(request{method: "GET", path: "/v1/restaurant-reviews?date[gte]=2026-02-30"}, http.StatusBadRequest)
	if got := res.object(t)["detail"]; got != "The query parameter date[gte] is not valid." {
		t.Errorf("detail %q", got)
	}
}
