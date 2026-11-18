package publicapi_test

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const recipeReviewBody = `{"taste":9,"ease":8,"instructions":8,"value":8,"title":"Good"}`

// createRecipe adds one ingredient and one step to a body without them.
func (e *env) createRecipe(token, body string) map[string]any {
	e.t.Helper()
	if !strings.Contains(body, `"ingredients"`) {
		body = strings.TrimSuffix(body, "}") + `,"ingredients":["1 egg"]}`
	}
	if !strings.Contains(body, `"steps"`) {
		body = strings.TrimSuffix(body, "}") + `,"steps":["Cook it."]}`
	}
	return e.expect(request{method: "POST", path: "/v1/recipes", body: body, token: token}, http.StatusCreated).object(e.t)
}

func (e *env) createRecipeReview(token, recipeID, body string) map[string]any {
	e.t.Helper()
	return e.expect(request{method: "POST", path: "/v1/recipes/" + recipeID + "/reviews", body: body, token: token}, http.StatusCreated).object(e.t)
}

// names gets the names of a page of recipes, or the titles of a page of
// reviews, joined with commas.
func (e *env) names(path string) (string, *string) {
	e.t.Helper()
	return e.namesAs(path, "")
}

func (e *env) namesAs(path, token string) (string, *string) {
	e.t.Helper()
	var p struct {
		Items []struct {
			Name  string `json:"name"`
			Title string `json:"title"`
		} `json:"items"`
		NextCursor *string `json:"next_cursor"`
	}
	e.expect(request{method: "GET", path: path, token: token}, http.StatusOK).decode(e.t, &p)
	var out []string
	for _, it := range p.Items {
		out = append(out, it.Name+it.Title)
	}
	return strings.Join(out, ","), p.NextCursor
}

func TestCreateRecipe(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	pending := e.signUpPending("pat@example.com", "Pat")

	res := e.expect(request{method: "POST", path: "/v1/recipes", token: ana.Token, body: `{"name":"  Shakshuka ",
		"ingredients":["  6 eggs ","1 onion"],"steps":[" Cook the onion. ","Add the eggs."],"tags":["breakfast"],
		"source":"https://Recipes.Example/shakshuka/#top","total_time":30,"servings":3}`}, http.StatusCreated)
	got := res.object(t)
	id := got["id"].(string)

	t.Run("the response has the trimmed values", func(t *testing.T) {
		if loc := res.header.Get("Location"); loc != "/v1/recipes/"+id {
			t.Errorf("location %q", loc)
		}
		ingredients := got["ingredients"].([]any)
		steps := got["steps"].([]any)
		if got["name"] != "Shakshuka" || ingredients[0] != "6 eggs" || steps[0] != "Cook the onion." {
			t.Errorf("recipe %v, want the trimmed name, ingredients, and steps", got)
		}
		if got["total_time"] != 30.0 || got["servings"] != 3.0 || got["review_count"] != 0.0 || got["rating"] != nil || got["cover"] != nil {
			t.Errorf("recipe %v", got)
		}
		if got["creator"].(map[string]any)["id"] != ana.User.ID {
			t.Errorf("creator %v, want Ana", got["creator"])
		}
	})

	t.Run("the source key has no case, fragment, or final slash", func(t *testing.T) {
		r, err := e.app.FindRecordById("recipes", id)
		if err != nil {
			t.Fatal(err)
		}
		if key := r.GetString("source_key"); key != "https://recipes.example/shakshuka" {
			t.Errorf("source_key %q", key)
		}
	})

	t.Run("a second recipe with the source gets 409", func(t *testing.T) {
		for _, source := range []string{"https://recipes.example/shakshuka", "https://RECIPES.example/shakshuka/"} {
			p := e.expectCode(request{method: "POST", path: "/v1/recipes", token: ana.Token,
				body: `{"name":"Copy","ingredients":["x"],"steps":["y"],"source":"` + source + `"}`}, http.StatusConflict, "source_taken").object(t)
			if p["recipe"] != id || p["property"] != "source" {
				t.Errorf("%s: problem %v, want the first recipe and the source property", source, p)
			}
		}
		other := e.createRecipe(ana.Token, `{"name":"Other","source":"https://recipes.example/Shakshuka"}`)
		if other["source"] != "https://recipes.example/Shakshuka" {
			t.Errorf("a path with other case is another page: %v", other)
		}
	})

	for name, body := range map[string]string{
		"no ingredients":              `{"name":"N","steps":["s"]}`,
		"empty ingredients":           `{"name":"N","ingredients":[],"steps":["s"]}`,
		"an ingredient with a break":  `{"name":"N","ingredients":["a\nb"],"steps":["s"]}`,
		"a step of whitespace":        `{"name":"N","ingredients":["a"],"steps":["\n\t "]}`,
		"no steps":                    `{"name":"N","ingredients":["a"]}`,
		"a total time of 0":           `{"name":"N","ingredients":["a"],"steps":["s"],"total_time":0}`,
		"too many servings":           `{"name":"N","ingredients":["a"],"steps":["s"],"servings":101}`,
		"a source that is not a link": `{"name":"N","ingredients":["a"],"steps":["s"],"source":"recipes.example"}`,
	} {
		t.Run(name+" gets 422", func(t *testing.T) {
			e.expect(request{method: "POST", path: "/v1/recipes", token: ana.Token, body: body}, http.StatusUnprocessableEntity)
		})
	}

	t.Run("a step keeps its line breaks", func(t *testing.T) {
		got := e.createRecipe(ana.Token, `{"name":"Lines","steps":["Mix.\nBake."]}`)
		if got["steps"].([]any)[0] != "Mix.\nBake." {
			t.Errorf("steps %v", got["steps"])
		}
	})

	t.Run("a user without approval gets 403", func(t *testing.T) {
		e.expectCode(request{method: "POST", path: "/v1/recipes", token: pending.Token, body: `{"name":"N","ingredients":["a"],"steps":["s"]}`},
			http.StatusForbidden, "not_approved")
	})

	t.Run("an unknown recipe gets 404", func(t *testing.T) {
		e.expect(request{method: "GET", path: "/v1/recipes/aaaaaaaaaaaaaaa"}, http.StatusNotFound)
	})
}

func TestListRecipes(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	hummus := e.createRecipe(ana.Token, `{"name":"Hummus","ingredients":["2 cans of chickpeas","1 lemon"],"tags":["dip"],"total_time":15}`)["id"].(string)
	e.createRecipe(ana.Token, `{"name":"Shakshuka","ingredients":["6 eggs"],"tags":["breakfast","eggs"],"total_time":30}`)
	e.createRecipe(ana.Token, `{"name":"Bread","ingredients":["flour"]}`)
	e.createRecipeReview(joe.Token, hummus, recipeReviewBody)

	for _, tc := range []struct{ query, want string }{
		{"search=chickpea", "Hummus"},
		{"tags[has]=eggs", "Shakshuka"},
		{"tags[has_any]=dip&tags[has_any]=eggs&sort=name", "Hummus,Shakshuka"},
		{"total_time[lte]=20", "Hummus"},
		{"total_time[gte]=1&sort=-total_time", "Shakshuka,Hummus"},
		{"sort=total_time", "Hummus,Shakshuka,Bread"},
		{"sort=-total_time", "Shakshuka,Hummus,Bread"},
		{"sort=-overall&limit=1", "Hummus"},
		{"overall[gte]=5", "Hummus"},
		{"creator[eq]=" + joe.User.ID, ""},
		{"creator[eq]=" + ana.User.ID + "&sort=name", "Bread,Hummus,Shakshuka"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			if got, _ := e.names("/v1/recipes?" + tc.query); got != tc.want {
				t.Errorf("names %q, want %q", got, tc.want)
			}
		})
	}

	for sort, want := range map[string]string{
		"total_time":  "Hummus,Shakshuka,Bread",
		"-total_time": "Shakshuka,Hummus,Bread",
	} {
		t.Run("the pages of "+sort+" follow the cursor", func(t *testing.T) {
			var got []string
			path := "/v1/recipes?limit=1&sort=" + sort
			for range 4 {
				names, next := e.names(path)
				if names != "" {
					got = append(got, names)
				}
				if next == nil {
					break
				}
				path = "/v1/recipes?limit=1&sort=" + sort + "&cursor=" + url.QueryEscape(*next)
			}
			if strings.Join(got, ",") != want {
				t.Errorf("pages %v, want %s", got, want)
			}
		})
	}

	t.Run("the pages of -overall follow the cursor", func(t *testing.T) {
		var got []string
		path := "/v1/recipes?limit=1&sort=-overall"
		for range 4 {
			names, next := e.names(path)
			got = append(got, names)
			if next == nil {
				break
			}
			path = "/v1/recipes?limit=1&sort=-overall&cursor=" + url.QueryEscape(*next)
		}
		// The recipes without reviews come in the order of their IDs.
		slices.Sort(got[1:])
		if strings.Join(got, ",") != "Hummus,Bread,Shakshuka" {
			t.Errorf("pages %v, want Hummus first", got)
		}
	})

	t.Run("a bad sort gets 400", func(t *testing.T) {
		e.expect(request{method: "GET", path: "/v1/recipes?sort=relevance"}, http.StatusBadRequest)
		e.expect(request{method: "GET", path: "/v1/recipes?sort=distance"}, http.StatusBadRequest)
	})

	t.Run("a recipe has the rating of its reviews", func(t *testing.T) {
		got := e.expect(request{method: "GET", path: "/v1/recipes/" + hummus}, http.StatusOK).object(t)
		rating := got["rating"].(map[string]any)
		if got["review_count"] != 1.0 || rating["overall"] != 8.3 || rating["taste"] != 9.0 || rating["value"] != 8.0 {
			t.Errorf("recipe %v, want the overall 8.3 of 8.25", got)
		}
	})
}

func TestUpdateRecipe(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	sam := e.signUp("sam@example.com", "Sam")
	id := e.createRecipe(ana.Token, `{"name":"Dal","source":"https://dal.example","servings":2}`)["id"].(string)
	path := "/v1/recipes/" + id

	t.Run("only the creator changes a recipe", func(t *testing.T) {
		e.expectCode(request{method: "PATCH", path: path, token: joe.Token, body: `{"servings":3}`}, http.StatusForbidden, "not_creator")
	})

	t.Run("null clears an optional property", func(t *testing.T) {
		got := e.expect(request{method: "PATCH", path: path, token: ana.Token, body: `{"servings":4,"source":null,"description":"Red lentils"}`}, http.StatusOK).object(t)
		if got["servings"] != 4.0 || got["source"] != nil || got["description"] != "Red lentils" {
			t.Errorf("recipe %v", got)
		}
		got = e.expect(request{method: "PATCH", path: path, token: ana.Token, body: `{"servings":null}`}, http.StatusOK).object(t)
		if _, ok := got["servings"]; ok {
			t.Errorf("servings %v, want none", got["servings"])
		}
	})

	t.Run("an empty patch keeps the updated time", func(t *testing.T) {
		before := e.expect(request{method: "GET", path: path}, http.StatusOK).object(t)
		after := e.expect(request{method: "PATCH", path: path, token: ana.Token, body: `{"name":"Dal","ingredients":["1 egg"]}`}, http.StatusOK).object(t)
		if after["updated"] != before["updated"] {
			t.Errorf("updated %v, want %v", after["updated"], before["updated"])
		}
	})

	t.Run("the creator changes the name, the ingredients, and the steps without reviews of others", func(t *testing.T) {
		e.createRecipeReview(ana.Token, id, recipeReviewBody)
		got := e.expect(request{method: "PATCH", path: path, token: ana.Token,
			body: `{"name":"Tarka dal","ingredients":["1 cup of lentils"],"steps":["Boil.","Fry."]}`}, http.StatusOK).object(t)
		if got["name"] != "Tarka dal" || len(got["steps"].([]any)) != 2 {
			t.Errorf("recipe %v", got)
		}
		titles, _ := e.names("/v1/recipe-reviews?search=tarka")
		if titles != "Good" {
			t.Errorf("reviews %q, want the review with the new recipe name", titles)
		}
	})

	t.Run("a review of another user freezes the name, the ingredients, and the steps", func(t *testing.T) {
		e.createRecipeReview(joe.Token, id, recipeReviewBody)
		for property, body := range map[string]string{
			"name":        `{"name":"Dal makhani"}`,
			"ingredients": `{"ingredients":["2 cups of lentils"]}`,
			"steps":       `{"steps":["Boil."]}`,
		} {
			p := e.expectCode(request{method: "PATCH", path: path, token: ana.Token, body: body}, http.StatusConflict, "other_reviews").object(t)
			if p["property"] != property {
				t.Errorf("%s: property %v", body, p["property"])
			}
		}
		e.expect(request{method: "PATCH", path: path, token: ana.Token, body: `{"tags":["lentils"],"total_time":40}`}, http.StatusOK)
	})

	t.Run("a source of another recipe gets 409", func(t *testing.T) {
		other := e.createRecipe(joe.Token, `{"name":"Other dal","source":"https://other.example/dal"}`)["id"].(string)
		p := e.expectCode(request{method: "PATCH", path: path, token: ana.Token, body: `{"source":"https://OTHER.example/dal#x"}`},
			http.StatusConflict, "source_taken").object(t)
		if p["recipe"] != other {
			t.Errorf("problem %v, want the other recipe", p)
		}
	})

	t.Run("a recipe without a creator belongs to its reviewers", func(t *testing.T) {
		if _, err := e.app.DB().Update("recipes", dbx.Params{"creator": ""}, dbx.HashExp{"id": id}).Execute(); err != nil {
			t.Fatal(err)
		}
		e.expect(request{method: "PATCH", path: path, token: joe.Token, body: `{"description":"Joe's words"}`}, http.StatusOK)
		e.expectCode(request{method: "PATCH", path: path, token: sam.Token, body: `{"description":"Sam's words"}`}, http.StatusForbidden, "not_creator")
	})

	t.Run("an unknown recipe gets 404", func(t *testing.T) {
		e.expect(request{method: "PATCH", path: "/v1/recipes/aaaaaaaaaaaaaaa", token: ana.Token, body: `{}`}, http.StatusNotFound)
	})
}

func TestDeleteRecipe(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	id := e.createRecipe(ana.Token, `{"name":"Soup"}`)["id"].(string)
	path := "/v1/recipes/" + id
	own := e.createRecipeReview(ana.Token, id, recipeReviewBody)["id"].(string)

	e.expectCode(request{method: "DELETE", path: path, token: joe.Token}, http.StatusForbidden, "not_creator")
	joeReview := e.createRecipeReview(joe.Token, id, recipeReviewBody)["id"].(string)
	e.expectCode(request{method: "DELETE", path: path, token: ana.Token}, http.StatusConflict, "other_reviews")
	e.expect(request{method: "DELETE", path: path + "/reviews/" + joeReview, token: joe.Token}, http.StatusNoContent)
	e.expect(request{method: "DELETE", path: path, token: ana.Token}, http.StatusNoContent)
	e.expect(request{method: "GET", path: path}, http.StatusNotFound)
	if _, err := e.app.FindRecordById("recipe_reviews", own); err == nil {
		t.Error("the review of the creator stays")
	}
	e.expect(request{method: "DELETE", path: path, token: ana.Token}, http.StatusNotFound)
}

func TestRecipeReviews(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	id := e.createRecipe(ana.Token, `{"name":"Shakshuka"}`)["id"].(string)
	other := e.createRecipe(ana.Token, `{"name":"Bread"}`)["id"].(string)
	reviews := "/v1/recipes/" + id + "/reviews"

	res := e.expect(request{method: "POST", path: reviews, token: ana.Token,
		body: `{"taste":9,"ease":8,"instructions":8,"value":8,"title":" Quick breakfast ","feedback":"More salt"}`}, http.StatusCreated)
	review := res.object(t)
	reviewID := review["id"].(string)
	e.createRecipeReview(joe.Token, id, `{"taste":5,"ease":5,"instructions":5,"value":5,"title":"Fine"}`)

	t.Run("a review has the average of its four scores", func(t *testing.T) {
		if loc := res.header.Get("Location"); loc != reviews+"/"+reviewID {
			t.Errorf("location %q", loc)
		}
		if review["overall"] != 8.25 || review["recipe"] != id || review["recipe_name"] != "Shakshuka" || review["title"] != "Quick breakfast" {
			t.Errorf("review %v", review)
		}
		if photos := review["photos"].([]any); len(photos) != 0 {
			t.Errorf("photos %v", photos)
		}
	})

	t.Run("a second review of the user gets 409", func(t *testing.T) {
		e.expectCode(request{method: "POST", path: reviews, token: ana.Token, body: recipeReviewBody}, http.StatusConflict, "review_exists")
	})

	t.Run("a review of an unknown recipe gets 404", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/recipes/aaaaaaaaaaaaaaa/reviews", token: ana.Token, body: recipeReviewBody}, http.StatusNotFound)
		e.expect(request{method: "GET", path: "/v1/recipes/aaaaaaaaaaaaaaa/reviews"}, http.StatusNotFound)
		e.expect(request{method: "GET", path: "/v1/recipes/" + other + "/reviews/" + reviewID}, http.StatusNotFound)
	})

	t.Run("a score out of range gets 422", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/recipes/" + other + "/reviews", token: ana.Token,
			body: `{"taste":11,"ease":8,"instructions":8,"value":8,"title":"T"}`}, http.StatusUnprocessableEntity)
		e.expect(request{method: "POST", path: "/v1/recipes/" + other + "/reviews", token: ana.Token,
			body: `{"space":8,"ease":8,"instructions":8,"value":8,"title":"T"}`}, http.StatusBadRequest)
	})

	for _, tc := range []struct{ path, want string }{
		{reviews + "?sort=-taste", "Quick breakfast,Fine"},
		{reviews + "?sort=overall", "Fine,Quick breakfast"},
		{reviews + "?taste[gte]=9", "Quick breakfast"},
		{reviews + "?search=salt", "Quick breakfast"},
		{reviews + "?search=shakshuka&sort=title", "Fine,Quick breakfast"},
		{reviews + "?creator[eq]=" + joe.User.ID, "Fine"},
		{"/v1/recipe-reviews?recipe[eq]=" + id + "&sort=title", "Fine,Quick breakfast"},
		{"/v1/recipe-reviews?recipe[eq]=" + other, ""},
		{"/v1/recipe-reviews?creator[eq]=" + joe.User.ID, "Fine"},
		{"/v1/recipe-reviews?overall[lt]=6", "Fine"},
		{"/v1/reviewers/" + ana.User.ID + "/recipe-reviews", "Quick breakfast"},
		{"/v1/reviewers/" + ana.User.ID + "/recipe-reviews?recipe[eq]=" + other, ""},
	} {
		t.Run(tc.path, func(t *testing.T) {
			if got, _ := e.names(tc.path); got != tc.want {
				t.Errorf("titles %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("the reviews of an unknown user get 404", func(t *testing.T) {
		e.expect(request{method: "GET", path: "/v1/reviewers/aaaaaaaaaaaaaaa/recipe-reviews"}, http.StatusNotFound)
		e.expect(request{method: "GET", path: "/v1/reviewers/aaaaaaaaaaaaaaa/recipe-reviews?sort=bad"}, http.StatusBadRequest)
	})

	t.Run("the reviewer profile counts the recipes and their reviews", func(t *testing.T) {
		got := e.expect(request{method: "GET", path: "/v1/reviewers/" + ana.User.ID}, http.StatusOK).object(t)
		if got["recipe_review_count"] != 1.0 || got["recipe_average_overall"] != 8.3 || got["recipe_count"] != 2.0 || got["review_count"] != 0.0 {
			t.Errorf("profile %v", got)
		}
	})

	t.Run("only the creator changes a review", func(t *testing.T) {
		path := reviews + "/" + reviewID
		e.expectCode(request{method: "PATCH", path: path, token: joe.Token, body: `{"ease":1}`}, http.StatusForbidden, "not_creator")
		got := e.expect(request{method: "PATCH", path: path, token: ana.Token, body: `{"ease":6,"feedback":null}`}, http.StatusOK).object(t)
		if got["overall"] != 7.75 || got["feedback"] != nil {
			t.Errorf("review %v, want the overall 7.75 and no feedback", got)
		}
		got = e.expect(request{method: "GET", path: path}, http.StatusOK).object(t)
		if got["ease"] != 6.0 {
			t.Errorf("review %v", got)
		}
	})

	t.Run("only the creator deletes a review", func(t *testing.T) {
		path := reviews + "/" + reviewID
		e.expectCode(request{method: "DELETE", path: path, token: joe.Token}, http.StatusForbidden, "not_creator")
		e.expect(request{method: "DELETE", path: path, token: ana.Token}, http.StatusNoContent)
		e.expect(request{method: "GET", path: path}, http.StatusNotFound)
	})
}

func TestRecipeOverallColumnEqualsResponse(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	for i, body := range []string{
		`{"taste":1,"ease":1,"instructions":1,"value":2,"title":"T"}`,
		`{"taste":7,"ease":9,"instructions":8,"value":6,"title":"T"}`,
		`{"taste":10,"ease":10,"instructions":10,"value":9,"title":"T"}`,
		`{"taste":3,"ease":3,"instructions":4,"value":4,"title":"T"}`,
	} {
		recipe := e.createRecipe(ana.Token, fmt.Sprintf(`{"name":"R%d"}`, i))["id"].(string)
		review := e.createRecipeReview(ana.Token, recipe, body)
		var stored float64
		if err := e.app.DB().Select("overall").From("recipe_reviews").Where(dbx.HashExp{"id": review["id"]}).Row(&stored); err != nil {
			t.Fatal(err)
		}
		if stored != review["overall"] {
			t.Errorf("%s: column is %v, response has %v", body, stored, review["overall"])
		}
	}
}

func TestRecipeReviewPhotos(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	recipe := e.createRecipe(ana.Token, `{"name":"Shakshuka"}`)["id"].(string)
	review := e.createRecipeReview(ana.Token, recipe, recipeReviewBody)["id"].(string)
	reviewPath := "/v1/recipes/" + recipe + "/reviews/" + review
	jpg := string(testJPEG(t, 40, 20, 0))

	res := e.expect(request{method: "POST", path: reviewPath + "/photos", body: jpg, contentType: "image/jpeg", token: ana.Token}, http.StatusCreated)
	photo := res.object(t)
	photoID := photo["id"].(string)
	photoPath := reviewPath + "/photos/" + photoID

	t.Run("the photo has the paths of the recipe", func(t *testing.T) {
		if loc := res.header.Get("Location"); loc != photoPath {
			t.Errorf("location %q", loc)
		}
		if urlOf(photo, "full") != photoPath+"/full" || urlOf(photo, "thumb") != photoPath+"/thumb" {
			t.Errorf("urls %v", photo["urls"])
		}
		file := e.expect(request{method: "GET", path: photoPath + "/full"}, http.StatusOK)
		if file.header.Get("Content-Type") != "image/jpeg" || len(file.body) == 0 {
			t.Errorf("file %v", file.header)
		}
		e.expect(request{method: "GET", path: "/v1/restaurants/" + recipe + "/reviews/" + review + "/photos/" + photoID + "/full"}, http.StatusNotFound)
	})

	t.Run("the recipe has the photo as its cover and in its gallery", func(t *testing.T) {
		got := e.expect(request{method: "GET", path: "/v1/recipes/" + recipe}, http.StatusOK).object(t)
		if cover, ok := got["cover"].(map[string]any); !ok || cover["id"] != photoID || cover["review"] != review {
			t.Errorf("cover %v", got["cover"])
		}
		var page struct {
			Items []map[string]any `json:"items"`
		}
		e.expect(request{method: "GET", path: "/v1/recipes/" + recipe + "/photos"}, http.StatusOK).decode(t, &page)
		if len(page.Items) != 1 || page.Items[0]["id"] != photoID || page.Items[0]["creator"].(map[string]any)["id"] != ana.User.ID {
			t.Errorf("gallery %v", page.Items)
		}
		e.expect(request{method: "GET", path: "/v1/recipes/aaaaaaaaaaaaaaa/photos"}, http.StatusNotFound)
		e.expect(request{method: "GET", path: "/v1/recipes/" + recipe + "/photos?cursor=bad"}, http.StatusBadRequest)
	})

	t.Run("the review lists its photo", func(t *testing.T) {
		got := e.expect(request{method: "GET", path: reviewPath}, http.StatusOK).object(t)
		if photos := got["photos"].([]any); len(photos) != 1 {
			t.Errorf("photos %v", photos)
		}
	})

	t.Run("only the creator of the review changes its photos", func(t *testing.T) {
		e.expectCode(request{method: "POST", path: reviewPath + "/photos", body: jpg, contentType: "image/jpeg", token: joe.Token}, http.StatusForbidden, "not_creator")
		e.expect(request{method: "POST", path: reviewPath + "/photos", body: "text", contentType: "text/plain", token: ana.Token}, http.StatusUnsupportedMediaType)
		e.expectCode(request{method: "PATCH", path: photoPath, token: joe.Token, body: `{"alt":"x"}`}, http.StatusForbidden, "not_creator")
		got := e.expect(request{method: "PATCH", path: photoPath, token: ana.Token, body: `{"alt":" Eggs in a pan "}`}, http.StatusOK).object(t)
		if got["alt"] != "Eggs in a pan" {
			t.Errorf("alt %v", got["alt"])
		}
		e.expect(request{method: "PATCH", path: reviewPath + "/photos/aaaaaaaaaaaaaaa", token: ana.Token, body: `{}`}, http.StatusNotFound)
		e.expectCode(request{method: "DELETE", path: photoPath, token: joe.Token}, http.StatusForbidden, "not_creator")
		e.expect(request{method: "DELETE", path: photoPath, token: ana.Token}, http.StatusNoContent)
		e.expect(request{method: "DELETE", path: photoPath, token: ana.Token}, http.StatusNotFound)
		e.expect(request{method: "GET", path: photoPath + "/full"}, http.StatusNotFound)
	})

	t.Run("a sixth photo gets 409", func(t *testing.T) {
		for range 5 {
			e.expect(request{method: "POST", path: reviewPath + "/photos", body: jpg, contentType: "image/jpeg", token: ana.Token}, http.StatusCreated)
		}
		e.expectCode(request{method: "POST", path: reviewPath + "/photos", body: jpg, contentType: "image/jpeg", token: ana.Token}, http.StatusConflict, "photo_limit")
	})

	t.Run("a delete of the recipe deletes the photos", func(t *testing.T) {
		e.expect(request{method: "DELETE", path: "/v1/recipes/" + recipe, token: ana.Token}, http.StatusNoContent)
		n, err := e.app.CountRecords("recipe_review_photos")
		if err != nil || n != 0 {
			t.Errorf("%d photos, error %v", n, err)
		}
	})
}

func TestDashboardRecipeEdit(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	root := e.superuserToken("root@example.com")
	id := e.createRecipe(ana.Token, `{"name":"Dal","source":"https://dal.example"}`)["id"].(string)
	other := e.createRecipe(ana.Token, `{"name":"Soup"}`)["id"].(string)
	records := "/api/collections/recipes/records/"

	t.Run("a superuser changes a recipe with the checks of the API", func(t *testing.T) {
		e.dashboard(request{method: http.MethodPatch, path: records + id, token: root,
			body: `{"name":"  Tarka dal ","ingredients":["  1 cup of lentils "],"steps":[" Boil. "],"tags":["dal"]}`}, http.StatusOK)
		got := e.expect(request{method: "GET", path: "/v1/recipes/" + id}, http.StatusOK).object(t)
		if got["name"] != "Tarka dal" || got["ingredients"].([]any)[0] != "1 cup of lentils" || got["steps"].([]any)[0] != "Boil." {
			t.Errorf("recipe %v, want the trimmed values", got)
		}
	})

	for name, body := range map[string]string{
		"ingredients that are not an array": `{"ingredients":"lentils"}`,
		"an empty step":                     `{"steps":[" "]}`,
		"a bad source":                      `{"source":"dal.example"}`,
		"too many servings":                 `{"servings":500}`,
	} {
		t.Run(name+" gets 400", func(t *testing.T) {
			e.dashboard(request{method: http.MethodPatch, path: records + id, token: root, body: body}, http.StatusBadRequest)
		})
	}

	t.Run("a recipe needs its ingredients and its steps", func(t *testing.T) {
		for _, body := range []string{
			`{"name":"N","steps":["s"]}`,
			`{"name":"N","ingredients":[],"steps":["s"]}`,
			`{"name":"N","ingredients":["a"]}`,
			`{"name":"N","ingredients":["a"],"steps":[]}`,
		} {
			e.dashboard(request{method: http.MethodPost, path: "/api/collections/recipes/records", token: root, body: body}, http.StatusBadRequest)
		}
		for _, body := range []string{`{"ingredients":[]}`, `{"steps":[]}`, `{"ingredients":null}`, `{"steps":null}`} {
			e.dashboard(request{method: http.MethodPatch, path: records + id, token: root, body: body}, http.StatusBadRequest)
		}
		for _, body := range []string{`{"ingredients":[]}`, `{"steps":null}`} {
			e.expect(request{method: "PATCH", path: "/v1/recipes/" + id, token: ana.Token, body: body}, http.StatusUnprocessableEntity)
		}
	})

	t.Run("a source of another recipe gets 400", func(t *testing.T) {
		e.dashboard(request{method: http.MethodPatch, path: records + other, token: root, body: `{"source":"https://DAL.example/"}`}, http.StatusBadRequest)
		e.dashboard(request{method: http.MethodPatch, path: records + other, token: root, body: `{"source":"https://soup.example"}`}, http.StatusOK)
		r, err := e.app.FindRecordById("recipes", other)
		if err != nil || r.GetString("source_key") != "https://soup.example" {
			t.Errorf("source_key of %v, error %v", r, err)
		}
	})
}

// TestSourceTakenDuringTheSave adds a recipe with the source between the
// check and the save. The rollback of the save also removes that recipe.
func TestSourceTakenDuringTheSave(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	e.app.OnRecordValidate("recipes").BindFunc(func(ev *core.RecordEvent) error {
		if strings.Contains(ev.Record.GetString("source"), "race") {
			_, err := ev.App.DB().Insert("recipes", dbx.Params{
				"id": "racer0000000001", "name": "Racer", "source": "x", "source_key": ev.Record.GetString("source_key"),
			}).Execute()
			if err != nil {
				return err
			}
		}
		return ev.Next()
	})

	for name, req := range map[string]request{
		"create": {method: "POST", path: "/v1/recipes", body: `{"name":"N","ingredients":["a"],"steps":["s"],"source":"https://race.example"}`},
		"update": {method: "PATCH", path: "/v1/recipes/" + e.createRecipe(ana.Token, `{"name":"Dal"}`)["id"].(string), body: `{"source":"https://race.example/dal"}`},
	} {
		t.Run(name, func(t *testing.T) {
			req.token = ana.Token
			p := e.expectCode(req, http.StatusConflict, "source_taken").object(t)
			if _, ok := p["recipe"]; ok || p["property"] != "source" {
				t.Errorf("problem %v, want no recipe and the source property", p)
			}
		})
	}
}

func TestSaveWithoutValidationSetsTheSourceKey(t *testing.T) {
	e := newEnv(t, testOptions)
	recipes, err := e.app.FindCollectionByNameOrId("recipes")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(recipes)
	r.Load(map[string]any{"name": "Dal", "source": "https://DAL.example/", "ingredients": []string{"a"}, "steps": []string{"s"}})
	if err := e.app.SaveNoValidate(r); err != nil {
		t.Fatal(err)
	}
	stored, err := e.app.FindRecordById("recipes", r.Id)
	if err != nil || stored.GetString("source_key") != "https://dal.example" {
		t.Errorf("source_key of %v, error %v", stored, err)
	}
}
