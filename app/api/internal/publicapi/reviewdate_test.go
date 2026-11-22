package publicapi_test

import (
	"net/http"
	"slices"
	"testing"
	"time"
)

func restaurantReviewBody(date, title string) string {
	return `{"date":"` + date + `","space":7,"service":9,"food":8,"price":6,"title":"` + title + `"}`
}

func recipeReviewOn(date, title string) string {
	return `{"date":"` + date + `","taste":9,"ease":8,"instructions":8,"value":8,"title":"` + title + `"}`
}

func TestManyReviewsOfOneUser(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	recipe := e.createRecipe(ana.Token, `{"name":"Dal"}`)["id"].(string)

	march := e.createReview(ana.Token, restaurant, restaurantReviewBody("2024-03-01", "March"))
	e.createReview(ana.Token, restaurant, restaurantReviewBody("2024-11-06", "November"))
	e.createReview(ana.Token, restaurant, restaurantReviewBody("2024-01-15", "January"))
	e.createRecipeReview(ana.Token, recipe, recipeReviewOn("2024-05-01", "First"))
	e.createRecipeReview(ana.Token, recipe, recipeReviewOn("2024-06-01", "Second"))
	if march["date"] != "2024-03-01" {
		t.Errorf("date %v, want 2024-03-01", march["date"])
	}

	reviews := "/v1/restaurants/" + restaurant + "/reviews"
	for path, want := range map[string]string{
		reviews:                                                "November,March,January",
		reviews + "?sort=date":                                 "January,March,November",
		reviews + "?creator[eq]=" + ana.User.ID:                "November,March,January",
		"/v1/recipes/" + recipe + "/reviews":                   "Second,First",
		"/v1/recipes/" + recipe + "/reviews?sort=date":         "First,Second",
		"/v1/reviewers/" + ana.User.ID + "/restaurant-reviews": "November,March,January",
	} {
		if got, _ := e.names(path); got != want {
			t.Errorf("%s: %s, want %s", path, got, want)
		}
	}

	if got := e.expect(request{method: "GET", path: "/v1/restaurants/" + restaurant}, http.StatusOK).object(t); got["review_count"] != float64(3) {
		t.Errorf("review_count %v, want 3", got["review_count"])
	}
	joe := e.signUp("joe@example.com", "Joe")
	for _, c := range []struct {
		path, token string
		want        float64
	}{
		{"/v1/restaurants/" + restaurant, ana.Token, 3},
		{"/v1/restaurants/" + restaurant, "", 0},
		{"/v1/restaurants/" + restaurant, joe.Token, 0},
		{"/v1/recipes/" + recipe, ana.Token, 2},
		{"/v1/recipes/" + recipe, "", 0},
	} {
		got := e.expect(request{method: "GET", path: c.path, token: c.token}, http.StatusOK).object(t)
		if got["own_review_count"] != c.want {
			t.Errorf("%s with token %t: own_review_count %v, want %v", c.path, c.token != "", got["own_review_count"], c.want)
		}
	}
	var page struct {
		Items []struct {
			ID             string  `json:"id"`
			OwnReviewCount float64 `json:"own_review_count"`
		} `json:"items"`
	}
	e.expect(request{method: "GET", path: "/v1/recipes", token: ana.Token}, http.StatusOK).decode(t, &page)
	if len(page.Items) != 1 || page.Items[0].OwnReviewCount != 2 {
		t.Errorf("recipe list %+v, want one recipe with 2 own reviews", page.Items)
	}
	reviewer := e.expect(request{method: "GET", path: "/v1/reviewers/" + ana.User.ID}, http.StatusOK).object(t)
	if reviewer["restaurant_review_count"] != float64(3) || reviewer["recipe_review_count"] != float64(2) {
		t.Errorf("reviewer %v, want 3 restaurant reviews and 2 recipe reviews", reviewer)
	}

	path := reviews + "/" + march["id"].(string)
	got := e.expect(request{method: "PATCH", path: path, body: `{"date":"2024-12-24"}`, token: ana.Token}, http.StatusOK).object(t)
	if got["date"] != "2024-12-24" {
		t.Errorf("patched date %v, want 2024-12-24", got["date"])
	}
	if got, _ := e.names(reviews); got != "March,November,January" {
		t.Errorf("after the patch: %s", got)
	}
}

func TestReviewDateChecks(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	recipe := e.createRecipe(ana.Token, `{"name":"Dal"}`)["id"].(string)
	reviews := "/v1/restaurants/" + restaurant + "/reviews"
	recipeReviews := "/v1/recipes/" + recipe + "/reviews"
	inTwoDays := time.Now().UTC().AddDate(0, 0, 2).Format(time.DateOnly)

	for _, date := range []string{"2024-02-30", "2023-02-29", "2024-13-01", "1899-12-31", "24-11-06", "2024-11-6", "2024/11/06", "", inTwoDays} {
		t.Run("a create with "+date+" gets 422", func(t *testing.T) {
			for _, path := range []string{reviews, recipeReviews} {
				body := restaurantReviewBody(date, "X")
				if path == recipeReviews {
					body = recipeReviewOn(date, "X")
				}
				res := e.expect(request{method: "POST", path: path, body: body, token: ana.Token}, http.StatusUnprocessableEntity)
				if got := errorProperties(t, res); !slices.Equal(got, []string{"date"}) {
					t.Errorf("%s: errors %v, want date: %s", path, got, res.body)
				}
			}
		})
	}

	t.Run("a create without a date gets 422", func(t *testing.T) {
		res := e.expect(request{method: "POST", path: reviews, token: ana.Token,
			body: `{"space":7,"service":9,"food":8,"price":6,"title":"X"}`}, http.StatusUnprocessableEntity)
		if got := errorProperties(t, res); !slices.Equal(got, []string{"date"}) {
			t.Errorf("errors %v, want date", got)
		}
	})

	t.Run("the first date and today in UTC+14 are valid", func(t *testing.T) {
		latest := time.Now().In(time.FixedZone("UTC+14", 14*60*60)).Format(time.DateOnly)
		for _, date := range []string{"1900-01-01", "2024-02-29", latest} {
			e.createReview(ana.Token, restaurant, restaurantReviewBody(date, "X"))
			e.createRecipeReview(ana.Token, recipe, recipeReviewOn(date, "X"))
		}
	})

	t.Run("a patch with an invalid date gets 422", func(t *testing.T) {
		id := e.createReview(ana.Token, restaurant, restaurantReviewBody("2024-11-06", "X"))["id"].(string)
		e.expect(request{method: "PATCH", path: reviews + "/" + id, body: `{"date":"2024-02-30"}`, token: ana.Token}, http.StatusUnprocessableEntity)
		e.expect(request{method: "PATCH", path: reviews + "/" + id, body: `{"date":null}`, token: ana.Token}, http.StatusUnprocessableEntity)
	})
}

func TestDashboardReviewDate(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	root := e.superuserToken("root@example.com")
	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	recipe := e.createRecipe(ana.Token, `{"name":"Dal"}`)["id"].(string)
	review := e.createReview(ana.Token, restaurant, restaurantReviewBody("2024-11-06", "X"))["id"].(string)
	recipeReview := e.createRecipeReview(ana.Token, recipe, recipeReviewOn("2024-11-06", "X"))["id"].(string)

	for collection, id := range map[string]string{"reviews": review, "recipe_reviews": recipeReview} {
		records := "/api/collections/" + collection + "/records/"
		t.Run("the dashboard refuses a date that does not exist in "+collection, func(t *testing.T) {
			res := e.dashboard(request{method: http.MethodPatch, path: records + id, token: root, body: `{"date":"2024-02-30"}`}, http.StatusBadRequest)
			if data, _ := res.object(t)["data"].(map[string]any); data["date"] == nil {
				t.Errorf("errors %s, want an error of date", res.body)
			}
		})
		t.Run("the dashboard changes the date in "+collection, func(t *testing.T) {
			got := e.dashboard(request{method: http.MethodPatch, path: records + id, token: root, body: `{"date":"2024-01-02"}`}, http.StatusOK).object(t)
			if got["date"] != "2024-01-02" {
				t.Errorf("date %v, want 2024-01-02", got["date"])
			}
		})
	}
}

func TestReviewsOfOneDayPutTheLastWrittenFirst(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	for _, title := range []string{"Lunch", "Dinner"} {
		e.createReview(ana.Token, restaurant, restaurantReviewBody("2024-11-06", title))
		time.Sleep(2 * time.Millisecond)
	}
	reviews := "/v1/restaurants/" + restaurant + "/reviews"
	for path, want := range map[string]string{reviews: "Dinner,Lunch", reviews + "?sort=date": "Lunch,Dinner"} {
		if got, _ := e.names(path); got != want {
			t.Errorf("%s: %s, want %s", path, got, want)
		}
	}
}
