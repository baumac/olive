package publicapi_test

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
)

const pantryItemReviewBody = `{"date":"2024-11-06","taste":9,"aroma":8,"texture":8,"value":8,"title":"Good"}`

// createPantryItem adds the category wine to a body without a category.
func (e *env) createPantryItem(token, body string) map[string]any {
	e.t.Helper()
	if !strings.Contains(body, `"category"`) {
		body = strings.TrimSuffix(body, "}") + `,"category":"wine"}`
	}
	return e.expect(request{method: "POST", path: "/v1/pantry", body: body, token: token}, http.StatusCreated).object(e.t)
}

func (e *env) createPantryItemReview(token, itemID, body string) map[string]any {
	e.t.Helper()
	return e.expect(request{method: "POST", path: "/v1/pantry/" + itemID + "/reviews", body: body, token: token}, http.StatusCreated).object(e.t)
}

func TestCreatePantryItem(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	pending := e.signUpPending("pat@example.com", "Pat")

	res := e.expect(request{method: "POST", path: "/v1/pantry", token: ana.Token, body: `{"name":"  Wehlener Sonnenuhr ","category":"wine",
		"producer":" Weingut Example ","origin":"Mosel, Germany","style":"Riesling","description":"Off-dry","tags":["White","off-dry"]}`},
		http.StatusCreated)
	got := res.object(t)
	id := got["id"].(string)

	t.Run("the response has the trimmed values", func(t *testing.T) {
		if loc := res.header.Get("Location"); loc != "/v1/pantry/"+id {
			t.Errorf("location %q", loc)
		}
		want := map[string]any{
			"name": "Wehlener Sonnenuhr", "category": "wine", "producer": "Weingut Example", "origin": "Mosel, Germany",
			"style": "Riesling", "description": "Off-dry", "review_count": 0.0, "rating": nil, "cover": nil, "own_review_count": 0.0,
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s %v, want %v", k, got[k], v)
			}
		}
		if fmt.Sprint(got["tags"]) != "[white off-dry]" || got["creator"].(map[string]any)["id"] != ana.User.ID {
			t.Errorf("pantry item %v", got)
		}
		if same := e.expect(request{method: "GET", path: "/v1/pantry/" + id}, http.StatusOK).object(t); same["name"] != got["name"] {
			t.Errorf("get %v", same)
		}
	})

	t.Run("two pantry items can have one name", func(t *testing.T) {
		other := e.createPantryItem(ana.Token, `{"name":"Wehlener Sonnenuhr","producer":"Weingut Example"}`)
		if other["id"] == id {
			t.Errorf("one record for two creates")
		}
	})

	t.Run("a pantry item without optional details has no such properties", func(t *testing.T) {
		got := e.createPantryItem(ana.Token, `{"name":"Gouda","category":"cheese"}`)
		for _, name := range []string{"producer", "origin", "style", "description", "tags"} {
			if _, ok := got[name]; ok {
				t.Errorf("%s %v, want none", name, got[name])
			}
		}
	})

	for name, body := range map[string]string{
		"no category":         `{"name":"N"}`,
		"an unknown category": `{"name":"N","category":"cider"}`,
		"a null category":     `{"name":"N","category":null}`,
		"no name":             `{"category":"beer"}`,
		"an empty producer":   `{"name":"N","category":"beer","producer":"  "}`,
		"a long style":        `{"name":"N","category":"beer","style":"` + strings.Repeat("a", 256) + `"}`,
	} {
		t.Run(name+" gets 422", func(t *testing.T) {
			e.expect(request{method: "POST", path: "/v1/pantry", token: ana.Token, body: body}, http.StatusUnprocessableEntity)
		})
	}

	t.Run("a vintage gets 400", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/pantry", token: ana.Token, body: `{"name":"N","category":"wine","vintage":2019}`}, http.StatusBadRequest)
	})

	t.Run("a user without approval gets 403", func(t *testing.T) {
		e.expectCode(request{method: "POST", path: "/v1/pantry", token: pending.Token, body: `{"name":"N","category":"wine"}`},
			http.StatusForbidden, "not_approved")
	})

	t.Run("an unknown pantry item gets 404", func(t *testing.T) {
		e.expect(request{method: "GET", path: "/v1/pantry/aaaaaaaaaaaaaaa"}, http.StatusNotFound)
	})
}

func TestListPantryItems(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	riesling := e.createPantryItem(ana.Token, `{"name":"Kabinett","producer":"Weingut Qwerty","style":"Riesling","origin":"Mosel","tags":["white","off-dry"]}`)["id"].(string)
	e.createPantryItem(ana.Token, `{"name":"Pils","category":"beer","producer":"Brauerei Zyx","origin":"Bavaria","tags":["lager"]}`)
	e.createPantryItem(ana.Token, `{"name":"Roquefort","category":"cheese","style":"Blue","description":"Sheep milk","tags":["raw-milk","white"]}`)
	e.createPantryItem(joe.Token, `{"name":"Dark 70","category":"chocolate","style":"70 percent cocoa"}`)
	e.createPantryItemReview(joe.Token, riesling, pantryItemReviewBody)

	for _, tc := range []struct{ query, want string }{
		{"sort=name", "Dark 70,Kabinett,Pils,Roquefort"},
		{"category[eq]=beer", "Pils"},
		{"category[in]=cheese&category[in]=chocolate&sort=name", "Dark 70,Roquefort"},
		{"search=qwerty", "Kabinett"},
		{"search=riesling", "Kabinett"},
		{"search=bavaria", "Pils"},
		{"search=sheep", "Roquefort"},
		{"search=cocoa", "Dark 70"},
		{"search=white&sort=name", "Kabinett,Roquefort"},
		{"tags[has]=white&sort=name", "Kabinett,Roquefort"},
		{"tags[has_any]=lager&tags[has_any]=raw-milk&sort=name", "Pils,Roquefort"},
		{"tags[has_all]=white&tags[has_all]=raw-milk", "Roquefort"},
		{"tags[has]=White&sort=name", "Kabinett,Roquefort"},
		{"tags[has]=lager&category[eq]=wine", ""},
		{"sort=-overall&limit=1", "Kabinett"},
		{"overall[gte]=5", "Kabinett"},
		{"creator[eq]=" + joe.User.ID, "Dark 70"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			if got, _ := e.names("/v1/pantry?" + tc.query); got != tc.want {
				t.Errorf("names %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("a name match comes before a producer match", func(t *testing.T) {
		e.createPantryItem(ana.Token, `{"name":"Qwerty Spätlese"}`)
		if got, _ := e.names("/v1/pantry?search=qwerty"); got != "Qwerty Spätlese,Kabinett" {
			t.Errorf("names %q", got)
		}
	})

	t.Run("the pages follow the cursor", func(t *testing.T) {
		var got []string
		path := "/v1/pantry?limit=2&sort=name&category[in]=wine&category[in]=beer"
		for range 4 {
			names, next := e.names(path)
			got = append(got, names)
			if next == nil {
				break
			}
			path = "/v1/pantry?limit=2&sort=name&category[in]=wine&category[in]=beer&cursor=" + url.QueryEscape(*next)
		}
		if strings.Join(got, ",") != "Kabinett,Pils,Qwerty Spätlese" {
			t.Errorf("pages %v", got)
		}
	})

	for _, query := range []string{"sort=total_time", "category[eq]=cider", "category[in]=wine&category[in]=wine&category[in]=wine&category[in]=wine&category[in]=wine"} {
		t.Run(query+" gets 400", func(t *testing.T) {
			e.expect(request{method: "GET", path: "/v1/pantry?" + query}, http.StatusBadRequest)
		})
	}
}

func TestUpdatePantryItem(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	id := e.createPantryItem(ana.Token, `{"name":"Kabinett","producer":"Weingut A","origin":"Mosel"}`)["id"].(string)
	path := "/v1/pantry/" + id

	t.Run("each user changes each property of a pantry item", func(t *testing.T) {
		got := e.expect(request{method: "PATCH", path: path, token: joe.Token,
			body: `{"name":"Spätlese","category":"beer","producer":"B","origin":"Rheingau","style":"Riesling","description":"Sweet","tags":["sweet"]}`},
			http.StatusOK).object(t)
		if got["name"] != "Spätlese" || got["category"] != "beer" || got["producer"] != "B" || got["style"] != "Riesling" || fmt.Sprint(got["tags"]) != "[sweet]" {
			t.Errorf("pantry item %v", got)
		}
	})

	t.Run("null clears an optional property", func(t *testing.T) {
		got := e.expect(request{method: "PATCH", path: path, token: ana.Token,
			body: `{"category":"wine","producer":null,"origin":null,"style":null,"description":null,"tags":null}`}, http.StatusOK).object(t)
		for _, name := range []string{"producer", "origin", "style", "description", "tags"} {
			if _, ok := got[name]; ok {
				t.Errorf("%s %v, want none", name, got[name])
			}
		}
		if got["category"] != "wine" {
			t.Errorf("category %v", got["category"])
		}
	})

	t.Run("the category cannot be null", func(t *testing.T) {
		e.expect(request{method: "PATCH", path: path, token: ana.Token, body: `{"category":null}`}, http.StatusUnprocessableEntity)
	})

	t.Run("an empty patch keeps the updated time", func(t *testing.T) {
		before := e.expect(request{method: "GET", path: path}, http.StatusOK).object(t)
		after := e.expect(request{method: "PATCH", path: path, token: ana.Token, body: `{"name":"Spätlese","category":"wine"}`}, http.StatusOK).object(t)
		if after["updated"] != before["updated"] {
			t.Errorf("updated %v, want %v", after["updated"], before["updated"])
		}
	})

	t.Run("a rename changes the name of the reviews", func(t *testing.T) {
		e.createPantryItemReview(ana.Token, id, pantryItemReviewBody)
		e.expect(request{method: "PATCH", path: path, token: joe.Token, body: `{"name":"Auslese"}`}, http.StatusOK)
		titles, _ := e.names("/v1/pantry-item-reviews?search=auslese")
		if titles != "Good" {
			t.Errorf("reviews %q, want the review with the new name", titles)
		}
		review := e.expect(request{method: "GET", path: "/v1/pantry-item-reviews"}, http.StatusOK).object(t)["items"].([]any)[0].(map[string]any)
		if review["pantry_item_name"] != "Auslese" {
			t.Errorf("review %v", review)
		}
	})

	t.Run("an unknown pantry item gets 404", func(t *testing.T) {
		e.expect(request{method: "PATCH", path: "/v1/pantry/aaaaaaaaaaaaaaa", token: ana.Token, body: `{}`}, http.StatusNotFound)
	})
}

func TestDeletePantryItem(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	id := e.createPantryItem(ana.Token, `{"name":"Gouda","category":"cheese"}`)["id"].(string)
	path := "/v1/pantry/" + id
	own := e.createPantryItemReview(ana.Token, id, pantryItemReviewBody)["id"].(string)

	e.expectCode(request{method: "DELETE", path: path, token: joe.Token}, http.StatusForbidden, "not_creator")
	joeReview := e.createPantryItemReview(joe.Token, id, pantryItemReviewBody)["id"].(string)
	e.expectCode(request{method: "DELETE", path: path, token: ana.Token}, http.StatusConflict, "other_reviews")
	e.expect(request{method: "DELETE", path: path + "/reviews/" + joeReview, token: joe.Token}, http.StatusNoContent)
	e.expect(request{method: "DELETE", path: path, token: ana.Token}, http.StatusNoContent)
	e.expect(request{method: "GET", path: path}, http.StatusNotFound)
	if _, err := e.app.FindRecordById("pantry_item_reviews", own); err == nil {
		t.Error("the review of the creator stays")
	}
	e.expect(request{method: "DELETE", path: path, token: ana.Token}, http.StatusNotFound)
}

func TestPantryItemReviews(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	id := e.createPantryItem(ana.Token, `{"name":"Kabinett"}`)["id"].(string)
	other := e.createPantryItem(ana.Token, `{"name":"Pils","category":"beer"}`)["id"].(string)
	reviews := "/v1/pantry/" + id + "/reviews"

	res := e.expect(request{method: "POST", path: reviews, token: ana.Token,
		body: `{"date":"2024-11-06","taste":9,"aroma":8,"texture":8,"value":8,"title":" Fresh white ","feedback":"Green apple"}`}, http.StatusCreated)
	review := res.object(t)
	reviewID := review["id"].(string)
	e.createPantryItemReview(joe.Token, id, `{"date":"2024-11-06","taste":5,"aroma":5,"texture":5,"value":5,"title":"Fine"}`)

	t.Run("a review has the average of its four scores", func(t *testing.T) {
		if loc := res.header.Get("Location"); loc != reviews+"/"+reviewID {
			t.Errorf("location %q", loc)
		}
		if review["overall"] != 8.25 || review["pantry_item"] != id || review["pantry_item_name"] != "Kabinett" || review["title"] != "Fresh white" {
			t.Errorf("review %v", review)
		}
	})

	t.Run("a review of an unknown pantry item gets 404", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/pantry/aaaaaaaaaaaaaaa/reviews", token: ana.Token, body: pantryItemReviewBody}, http.StatusNotFound)
		e.expect(request{method: "GET", path: "/v1/pantry/aaaaaaaaaaaaaaa/reviews"}, http.StatusNotFound)
		e.expect(request{method: "GET", path: "/v1/pantry/" + other + "/reviews/" + reviewID}, http.StatusNotFound)
		e.expect(request{method: "GET", path: "/v1/recipes/" + id + "/reviews/" + reviewID}, http.StatusNotFound)
	})

	t.Run("the scores of another kind get 400", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/pantry/" + other + "/reviews", token: ana.Token,
			body: `{"date":"2024-11-06","taste":11,"aroma":8,"texture":8,"value":8,"title":"T"}`}, http.StatusUnprocessableEntity)
		e.expect(request{method: "POST", path: "/v1/pantry/" + other + "/reviews", token: ana.Token,
			body: `{"date":"2024-11-06","taste":8,"ease":8,"texture":8,"value":8,"title":"T"}`}, http.StatusBadRequest)
	})

	for _, tc := range []struct{ path, want string }{
		{reviews + "?sort=-aroma", "Fresh white,Fine"},
		{reviews + "?sort=texture", "Fine,Fresh white"},
		{reviews + "?aroma[gte]=8", "Fresh white"},
		{reviews + "?texture[lt]=6", "Fine"},
		{reviews + "?search=apple", "Fresh white"},
		{reviews + "?search=kabinett&sort=title", "Fine,Fresh white"},
		{"/v1/pantry-item-reviews?pantry_item[eq]=" + id + "&sort=title", "Fine,Fresh white"},
		{"/v1/pantry-item-reviews?pantry_item[in]=" + other, ""},
		{"/v1/pantry-item-reviews?creator[eq]=" + joe.User.ID, "Fine"},
		{"/v1/reviewers/" + ana.User.ID + "/pantry-item-reviews", "Fresh white"},
		{"/v1/reviewers/" + ana.User.ID + "/pantry-item-reviews?pantry_item[eq]=" + other, ""},
	} {
		t.Run(tc.path, func(t *testing.T) {
			if got, _ := e.names(tc.path); got != tc.want {
				t.Errorf("titles %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("a pantry item has the rating of its reviews", func(t *testing.T) {
		got := e.expect(request{method: "GET", path: "/v1/pantry/" + id, token: ana.Token}, http.StatusOK).object(t)
		want := map[string]any{"overall": 6.6, "taste": 7.0, "aroma": 6.5, "texture": 6.5, "value": 6.5}
		if !reflect.DeepEqual(got["rating"], want) || got["review_count"] != 2.0 || got["own_review_count"] != 1.0 {
			t.Errorf("rating %v, review_count %v, own_review_count %v", got["rating"], got["review_count"], got["own_review_count"])
		}
	})

	t.Run("the reviews of an unknown user get 404", func(t *testing.T) {
		e.expect(request{method: "GET", path: "/v1/reviewers/aaaaaaaaaaaaaaa/pantry-item-reviews"}, http.StatusNotFound)
		e.expect(request{method: "GET", path: "/v1/reviewers/aaaaaaaaaaaaaaa/pantry-item-reviews?sort=bad"}, http.StatusBadRequest)
	})

	t.Run("the reviewer profile counts the pantry items and their reviews", func(t *testing.T) {
		got := e.expect(request{method: "GET", path: "/v1/reviewers/" + ana.User.ID}, http.StatusOK).object(t)
		if got["pantry_item_review_count"] != 1.0 || got["pantry_item_average_overall"] != 8.3 || got["pantry_item_count"] != 2.0 ||
			got["recipe_review_count"] != 0.0 || got["recipe_count"] != 0.0 {
			t.Errorf("profile %v", got)
		}
	})

	t.Run("only the creator changes a review", func(t *testing.T) {
		path := reviews + "/" + reviewID
		e.expectCode(request{method: "PATCH", path: path, token: joe.Token, body: `{"aroma":1}`}, http.StatusForbidden, "not_creator")
		got := e.expect(request{method: "PATCH", path: path, token: ana.Token, body: `{"texture":6,"feedback":null}`}, http.StatusOK).object(t)
		if got["overall"] != 7.75 || got["feedback"] != nil || got["texture"] != 6.0 {
			t.Errorf("review %v, want the overall 7.75 and no feedback", got)
		}
	})

	t.Run("only the creator deletes a review", func(t *testing.T) {
		path := reviews + "/" + reviewID
		e.expectCode(request{method: "DELETE", path: path, token: joe.Token}, http.StatusForbidden, "not_creator")
		e.expect(request{method: "DELETE", path: path, token: ana.Token}, http.StatusNoContent)
		e.expect(request{method: "GET", path: path}, http.StatusNotFound)
	})
}

func TestPantryItemOverallColumnEqualsResponse(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	for i, body := range []string{
		`{"date":"2024-11-06","taste":1,"aroma":1,"texture":1,"value":2,"title":"T"}`,
		`{"date":"2024-11-06","taste":7,"aroma":9,"texture":8,"value":6,"title":"T"}`,
		`{"date":"2024-11-06","taste":10,"aroma":10,"texture":10,"value":9,"title":"T"}`,
	} {
		item := e.createPantryItem(ana.Token, fmt.Sprintf(`{"name":"P%d"}`, i))["id"].(string)
		review := e.createPantryItemReview(ana.Token, item, body)
		var stored float64
		if err := e.app.DB().Select("overall").From("pantry_item_reviews").Where(dbx.HashExp{"id": review["id"]}).Row(&stored); err != nil {
			t.Fatal(err)
		}
		if stored != review["overall"] {
			t.Errorf("%s: column is %v, response has %v", body, stored, review["overall"])
		}
	}
}

func TestPantryItemReviewPhotos(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	item := e.createPantryItem(ana.Token, `{"name":"Kabinett"}`)["id"].(string)
	review := e.createPantryItemReview(ana.Token, item, pantryItemReviewBody)["id"].(string)
	reviewPath := "/v1/pantry/" + item + "/reviews/" + review
	jpg := string(testJPEG(t, 40, 20, 0))

	res := e.expect(request{method: "POST", path: reviewPath + "/photos", body: jpg, contentType: "image/jpeg", token: ana.Token}, http.StatusCreated)
	photo := res.object(t)
	photoID := photo["id"].(string)
	photoPath := reviewPath + "/photos/" + photoID

	t.Run("the photo has the paths of the pantry item", func(t *testing.T) {
		if loc := res.header.Get("Location"); loc != photoPath {
			t.Errorf("location %q", loc)
		}
		if urlOf(photo, "full") != photoPath+"/full" || urlOf(photo, "thumb") != photoPath+"/thumb" {
			t.Errorf("urls %v", photo["urls"])
		}
		file := e.expect(request{method: "GET", path: photoPath + "/thumb"}, http.StatusOK)
		if file.header.Get("Content-Type") != "image/jpeg" || len(file.body) == 0 {
			t.Errorf("file %v", file.header)
		}
		e.expect(request{method: "GET", path: "/v1/recipes/" + item + "/reviews/" + review + "/photos/" + photoID + "/full"}, http.StatusNotFound)
	})

	t.Run("the pantry item has the photo as its cover and in its gallery", func(t *testing.T) {
		got := e.expect(request{method: "GET", path: "/v1/pantry/" + item}, http.StatusOK).object(t)
		if cover, ok := got["cover"].(map[string]any); !ok || cover["id"] != photoID || cover["review"] != review {
			t.Errorf("cover %v", got["cover"])
		}
		var page struct {
			Items []map[string]any `json:"items"`
		}
		e.expect(request{method: "GET", path: "/v1/pantry/" + item + "/photos"}, http.StatusOK).decode(t, &page)
		if len(page.Items) != 1 || page.Items[0]["id"] != photoID {
			t.Errorf("gallery %v", page.Items)
		}
		e.expect(request{method: "GET", path: "/v1/pantry/aaaaaaaaaaaaaaa/photos"}, http.StatusNotFound)
	})

	t.Run("only the creator of the review changes its photos", func(t *testing.T) {
		e.expectCode(request{method: "POST", path: reviewPath + "/photos", body: jpg, contentType: "image/jpeg", token: joe.Token}, http.StatusForbidden, "not_creator")
		got := e.expect(request{method: "PATCH", path: photoPath, token: ana.Token, body: `{"alt":" A glass "}`}, http.StatusOK).object(t)
		if got["alt"] != "A glass" {
			t.Errorf("alt %v", got["alt"])
		}
		e.expectCode(request{method: "DELETE", path: photoPath, token: joe.Token}, http.StatusForbidden, "not_creator")
		e.expect(request{method: "DELETE", path: photoPath, token: ana.Token}, http.StatusNoContent)
		e.expect(request{method: "GET", path: photoPath + "/full"}, http.StatusNotFound)
	})

	t.Run("a delete of the pantry item deletes the photos", func(t *testing.T) {
		e.expect(request{method: "POST", path: reviewPath + "/photos", body: jpg, contentType: "image/jpeg", token: ana.Token}, http.StatusCreated)
		e.expect(request{method: "DELETE", path: "/v1/pantry/" + item, token: ana.Token}, http.StatusNoContent)
		n, err := e.app.CountRecords("pantry_item_review_photos")
		if err != nil || n != 0 {
			t.Errorf("%d photos, error %v", n, err)
		}
	})
}

func TestPantryItemLists(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	gouda := e.createPantryItem(ana.Token, `{"name":"Gouda","category":"cheese"}`)["id"].(string)
	pils := e.createPantryItem(joe.Token, `{"name":"Pils","category":"beer"}`)["id"].(string)
	recipe := e.createRecipe(ana.Token, `{"name":"Dal"}`)["id"].(string)
	list := e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Picnic"}`, token: ana.Token}, http.StatusCreated).object(t)["id"].(string)
	favorites := e.lists(ana.User.ID, "").Items[0].ID

	e.expect(request{method: "PUT", path: "/v1/lists/favorites/pantry/" + gouda, token: ana.Token}, http.StatusNoContent)
	e.expect(request{method: "PUT", path: "/v1/lists/favorites/recipes/" + recipe, token: ana.Token}, http.StatusNoContent)
	for range 2 {
		e.expect(request{method: "PUT", path: "/v1/lists/" + list + "/pantry/" + pils, token: ana.Token}, http.StatusNoContent)
	}

	t.Run("a pantry item tells if it is a favorite of the user", func(t *testing.T) {
		for token, want := range map[string]bool{ana.Token: true, joe.Token: false, "": false} {
			got := e.expect(request{method: "GET", path: "/v1/pantry/" + gouda, token: token}, http.StatusOK).object(t)
			if got["favorite"] != want {
				t.Errorf("favorite %v, want %v", got["favorite"], want)
			}
		}
	})

	t.Run("a list counts its pantry items", func(t *testing.T) {
		lists := e.lists(ana.User.ID, "?pantry_item[eq]="+gouda).Items
		if lists[0].ID != favorites || lists[0].PantryItemCount != 1 || lists[0].RecipeCount != 1 || !*lists[0].HasPantryItem || lists[0].HasRecipe != nil {
			t.Errorf("favorites %+v", lists[0])
		}
		if lists[1].ID != list || lists[1].PantryItemCount != 1 || *lists[1].HasPantryItem {
			t.Errorf("list %+v", lists[1])
		}
		got := e.expect(request{method: "GET", path: "/v1/lists/" + list}, http.StatusOK).object(t)
		if got["pantry_item_count"] != 1.0 || got["recipe_count"] != 0.0 {
			t.Errorf("list %v", got)
		}
	})

	t.Run("the pantry items of a list come with the last one first", func(t *testing.T) {
		e.expect(request{method: "PUT", path: "/v1/lists/" + list + "/pantry/" + gouda, token: ana.Token}, http.StatusNoContent)
		first, next := e.names("/v1/lists/" + list + "/pantry?limit=1")
		if first != "Gouda" || next == nil {
			t.Fatalf("first page %q, next %v", first, next)
		}
		second, _ := e.names("/v1/lists/" + list + "/pantry?limit=1&cursor=" + url.QueryEscape(*next))
		if second != "Pils" {
			t.Errorf("second page %q", second)
		}
		e.expect(request{method: "GET", path: "/v1/lists/aaaaaaaaaaaaaaa/pantry"}, http.StatusNotFound)
	})

	t.Run("listPantryItems selects the pantry items of a list", func(t *testing.T) {
		for query, want := range map[string]string{
			"list[eq]=" + list + "&sort=name":            "Gouda,Pils",
			"list[eq]=" + favorites:                      "Gouda",
			"list[eq]=" + list + "&category[eq]=beer":    "Pils",
			"list[eq]=" + favorites + "&search=pils":     "",
			"list[eq]=aaaaaaaaaaaaaaa&category[eq]=beer": "",
		} {
			if got, _ := e.names("/v1/pantry?" + query); got != want {
				t.Errorf("%s: names %q, want %q", query, got, want)
			}
		}
	})

	t.Run("only the creator changes the pantry items of a list", func(t *testing.T) {
		res := e.expectCode(request{method: "PUT", path: "/v1/lists/" + list + "/pantry/" + gouda, token: joe.Token}, http.StatusForbidden, "not_creator")
		if detail := res.object(t)["detail"]; detail != "Only the creator of the list can change its pantry items." {
			t.Errorf("detail %v", detail)
		}
		e.expect(request{method: "PUT", path: "/v1/lists/" + list + "/pantry/aaaaaaaaaaaaaaa", token: ana.Token}, http.StatusNotFound)
	})

	t.Run("a remove changes the list once", func(t *testing.T) {
		for range 2 {
			e.expect(request{method: "DELETE", path: "/v1/lists/" + list + "/pantry/" + pils, token: ana.Token}, http.StatusNoContent)
		}
		if got, _ := e.names("/v1/lists/" + list + "/pantry"); got != "Gouda" {
			t.Errorf("pantry items %q", got)
		}
	})

	t.Run("a delete of a pantry item removes it from the lists", func(t *testing.T) {
		e.expect(request{method: "DELETE", path: "/v1/pantry/" + gouda, token: ana.Token}, http.StatusNoContent)
		for _, l := range e.lists(ana.User.ID, "").Items {
			if l.PantryItemCount != 0 {
				t.Errorf("list %+v, want no pantry items", l)
			}
		}
	})
}

func TestListPantryItemLimit(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	item := e.createPantryItem(ana.Token, `{"name":"Gouda","category":"cheese"}`)["id"].(string)
	recipe := e.createRecipe(ana.Token, `{"name":"Dal"}`)["id"].(string)
	list := e.lists(ana.User.ID, "").Items[0].ID
	// The rows have no pantry item record. The limit check only counts them.
	_, err := e.app.DB().NewQuery(`
		WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 500)
		INSERT INTO list_pantry_items (list, pantry_item, created) SELECT {:list}, printf('x%014d', i), '2024-11-04 01:26:55.002Z' FROM n`).
		Bind(map[string]any{"list": list}).Execute()
	if err != nil {
		t.Fatal(err)
	}
	res := e.expectCode(request{method: "PUT", path: "/v1/lists/favorites/pantry/" + item, token: ana.Token}, http.StatusConflict, "list_pantry_item_limit")
	if detail := res.object(t)["detail"]; detail != "The list has the maximum number of pantry items." {
		t.Errorf("detail %v", detail)
	}
	e.expect(request{method: "PUT", path: "/v1/lists/favorites/recipes/" + recipe, token: ana.Token}, http.StatusNoContent)
}

func TestPantryItemVersions(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	id := e.createPantryItem(ana.Token, `{"name":"Kabinett","origin":"Mosel"}`)["id"].(string)
	path := "/v1/pantry/" + id
	e.expect(request{method: http.MethodPatch, path: path, token: joe.Token, body: `{"style":"Riesling","origin":null,"category":"beer"}`}, http.StatusOK)

	page := e.versions(path+"/versions", ana.Token)
	if len(page.Items) != 2 {
		t.Fatalf("%d versions, want 2", len(page.Items))
	}
	joe2, ana1 := page.Items[0], page.Items[1]
	if want := map[string]any{"category": "wine", "origin": "Mosel", "style": nil}; !reflect.DeepEqual(joe2.Before, want) {
		t.Errorf("version 2 before %v, want %v", joe2.Before, want)
	}
	if want := map[string]any{"category": "beer", "origin": nil, "style": "Riesling"}; !reflect.DeepEqual(joe2.After, want) || joe2.Editor.Name != "Joe" {
		t.Errorf("version 2 %+v", joe2)
	}
	if ana1.Action != "create" || ana1.After["name"] != "Kabinett" || ana1.After["category"] != "wine" {
		t.Errorf("version 1 %+v", ana1)
	}
	e.expect(request{method: http.MethodGet, path: path + "/versions"}, http.StatusUnauthorized)
	e.expect(request{method: http.MethodGet, path: "/v1/pantry/aaaaaaaaaaaaaaa/versions", token: ana.Token}, http.StatusNotFound)
}

func TestPantryItemTags(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	root := e.superuserToken("root@example.com")
	e.createRestaurant(ana.Token, `{"name":"Fromagerie","tags":["raw-milk","cheese"]}`)
	e.createRecipe(ana.Token, `{"name":"Fondue","tags":["cheese"]}`)
	gouda := e.createPantryItem(ana.Token, `{"name":"Gouda","category":"cheese","tags":["Raw Milk","aged"]}`)["id"].(string)
	brie := e.createPantryItem(ana.Token, `{"name":"Brie","category":"cheese","tags":["aged","soft"]}`)["id"].(string)

	t.Run("a pantry item uses the existing tag", func(t *testing.T) {
		got := e.expect(request{method: http.MethodGet, path: "/v1/tags/raw-milk"}, http.StatusOK).object(t)
		if got["restaurant_count"] != 1.0 || got["recipe_count"] != 0.0 || got["pantry_item_count"] != 1.0 {
			t.Errorf("tag %v", got)
		}
		if n := e.tagCount(); n != 4 {
			t.Errorf("%d tag records, want 4", n)
		}
	})

	for name, tc := range map[string]struct{ query, want string }{
		"all kinds count":          {"", "aged,cheese,raw-milk,soft"},
		"the count of the pantry":  {"?kind=pantry", "aged,raw-milk,soft"},
		"the count of the recipes": {"?kind=recipes", "cheese"},
		"the restaurants":          {"?kind=restaurants", "cheese,raw-milk"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := e.tags(tc.query).names(); got != tc.want {
				t.Errorf("%s: %s, want %s", tc.query, got, tc.want)
			}
		})
	}

	t.Run("a tag of only pantry items exists", func(t *testing.T) {
		got := e.expect(request{method: http.MethodGet, path: "/v1/tags/aged"}, http.StatusOK).object(t)
		if got["pantry_item_count"] != 2.0 || got["restaurant_count"] != 0.0 {
			t.Errorf("tag %v", got)
		}
		page := e.tags("?search=aged")
		if len(page.Items) != 1 || page.Items[0].PantryItemCount != 2 {
			t.Errorf("tags %+v", page.Items)
		}
	})

	t.Run("a delete of a pantry item deletes its unused tags", func(t *testing.T) {
		e.expect(request{method: http.MethodDelete, path: "/v1/pantry/" + brie, token: ana.Token}, http.StatusNoContent)
		e.expect(request{method: http.MethodGet, path: "/v1/tags/soft"}, http.StatusNotFound)
		e.expect(request{method: http.MethodGet, path: "/v1/tags/aged"}, http.StatusOK)
		e.expect(request{method: http.MethodPatch, path: "/v1/pantry/" + gouda, token: ana.Token, body: `{"tags":["raw-milk"]}`}, http.StatusOK)
		e.expect(request{method: http.MethodGet, path: "/v1/tags/aged"}, http.StatusNotFound)
		if n := e.tagCount(); n != 2 {
			t.Errorf("%d tag records, want 2", n)
		}
	})

	t.Run("a rename of a tag changes the pantry item", func(t *testing.T) {
		tag, err := e.app.FindFirstRecordByData("tags", "name", "raw-milk")
		if err != nil {
			t.Fatal(err)
		}
		e.dashboard(request{method: http.MethodPatch, path: "/api/collections/tags/records/" + tag.Id, token: root, body: `{"name":"unpasteurized"}`}, http.StatusOK)
		got := e.expect(request{method: http.MethodGet, path: "/v1/pantry/" + gouda}, http.StatusOK).object(t)
		if fmt.Sprint(got["tags"]) != "[unpasteurized]" {
			t.Errorf("tags %v", got["tags"])
		}
		if names, _ := e.names("/v1/pantry?search=unpasteurized"); names != "Gouda" {
			t.Errorf("search %q, want the pantry item with the new tag name", names)
		}
		if names, _ := e.names("/v1/pantry?tags[has]=unpasteurized"); names != "Gouda" {
			t.Errorf("filter %q", names)
		}
		vs := e.versions("/v1/pantry/"+gouda+"/versions", ana.Token).Items
		if len(vs) != 3 || vs[0].Source != "admin" || fmt.Sprint(vs[0].After["tags"]) != "[unpasteurized]" {
			t.Errorf("versions %+v", vs)
		}
	})
}

func TestDashboardPantryItemEdit(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	root := e.superuserToken("root@example.com")
	id := e.createPantryItem(ana.Token, `{"name":"Gouda","category":"cheese"}`)["id"].(string)
	tag := e.createPantryItem(ana.Token, `{"name":"Brie","category":"cheese","tags":["soft"]}`)["tags"].([]any)[0].(string)
	records := "/api/collections/pantry_items/records/"

	t.Run("a superuser changes a pantry item with the checks of the API", func(t *testing.T) {
		e.dashboard(request{method: http.MethodPatch, path: records + id, token: root, body: `{"name":"  Old Gouda ","producer":" Dairy "}`}, http.StatusOK)
		got := e.expect(request{method: "GET", path: "/v1/pantry/" + id}, http.StatusOK).object(t)
		if got["name"] != "Old Gouda" || got["producer"] != "Dairy" {
			t.Errorf("pantry item %v, want the trimmed values", got)
		}
	})

	for name, body := range map[string]string{
		"an unknown category": `{"category":"cider"}`,
		"an empty name":       `{"name":" "}`,
		"a style of spaces":   `{"style":"   "}`,
		"an invisible origin": `{"origin":"Mo` + "​" + `sel"}`,
	} {
		t.Run(name+" gets 400", func(t *testing.T) {
			e.dashboard(request{method: http.MethodPatch, path: records + id, token: root, body: body}, http.StatusBadRequest)
		})
	}

	t.Run("a pantry item needs a name and a category", func(t *testing.T) {
		for _, body := range []string{`{"name":"N"}`, `{"category":"wine"}`} {
			e.dashboard(request{method: http.MethodPost, path: "/api/collections/pantry_items/records", token: root, body: body}, http.StatusBadRequest)
		}
	})

	t.Run("the dashboard sets the tags with their IDs", func(t *testing.T) {
		record, err := e.app.FindFirstRecordByData("tags", "name", tag)
		if err != nil {
			t.Fatal(err)
		}
		e.dashboard(request{method: http.MethodPatch, path: records + id, token: root, body: `{"tags":["` + record.Id + `"]}`}, http.StatusOK)
		e.dashboard(request{method: http.MethodPatch, path: records + id, token: root, body: `{"tags":["aaaaaaaaaaaaaaa"]}`}, http.StatusBadRequest)
		got := e.expect(request{method: "GET", path: "/v1/pantry/" + id}, http.StatusOK).object(t)
		if fmt.Sprint(got["tags"]) != "[soft]" {
			t.Errorf("tags %v", got["tags"])
		}
		vs := e.versions("/v1/pantry/"+id+"/versions", ana.Token).Items
		if vs[0].Source != "admin" {
			t.Errorf("versions %+v", vs)
		}
	})
}

func TestPantryItemProblems(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	item := e.createPantryItem(ana.Token, `{"name":"Kabinett"}`)["id"].(string)
	review := e.createPantryItemReview(ana.Token, item, pantryItemReviewBody)["id"].(string)
	reviewPath := "/v1/pantry/" + item + "/reviews/" + review
	jpg := string(testJPEG(t, 40, 20, 0))
	photo := e.expect(request{method: "POST", path: reviewPath + "/photos", body: jpg, contentType: "image/jpeg", token: ana.Token}, http.StatusCreated).object(t)["id"].(string)
	list := e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Cellar"}`, token: ana.Token}, http.StatusCreated).object(t)["id"].(string)
	unknown := "aaaaaaaaaaaaaaa"

	t.Run("a review has its photos", func(t *testing.T) {
		got := e.expect(request{method: "GET", path: reviewPath}, http.StatusOK).object(t)
		if photos := got["photos"].([]any); len(photos) != 1 || photos[0].(map[string]any)["id"] != photo {
			t.Errorf("photos %v", got["photos"])
		}
	})

	for _, tc := range []struct {
		name   string
		req    request
		status int
	}{
		{"a bad sort of all reviews", request{method: "GET", path: "/v1/pantry-item-reviews?sort=bad"}, http.StatusBadRequest},
		{"a bad cursor of all reviews", request{method: "GET", path: "/v1/pantry-item-reviews?cursor=bad"}, http.StatusBadRequest},
		{"a bad cursor of the reviews", request{method: "GET", path: "/v1/pantry/" + item + "/reviews?cursor=bad"}, http.StatusBadRequest},
		{"a bad cursor of the pantry", request{method: "GET", path: "/v1/pantry?cursor=bad"}, http.StatusBadRequest},
		{"a bad cursor of the gallery", request{method: "GET", path: "/v1/pantry/" + item + "/photos?cursor=bad"}, http.StatusBadRequest},
		{"a bad cursor of the versions", request{method: "GET", path: "/v1/pantry/" + item + "/versions?cursor=bad", token: ana.Token}, http.StatusBadRequest},
		{"a bad cursor of a list", request{method: "GET", path: "/v1/lists/" + list + "/pantry?cursor=bad"}, http.StatusBadRequest},
		{"an unknown review", request{method: "GET", path: "/v1/pantry/" + item + "/reviews/" + unknown}, http.StatusNotFound},
		{"an update of an unknown review", request{method: "PATCH", path: "/v1/pantry/" + item + "/reviews/" + unknown, token: ana.Token, body: `{}`}, http.StatusNotFound},
		{"a delete of an unknown review", request{method: "DELETE", path: "/v1/pantry/" + item + "/reviews/" + unknown, token: ana.Token}, http.StatusNotFound},
		{"a delete of an unknown pantry item", request{method: "DELETE", path: "/v1/pantry/" + unknown, token: ana.Token}, http.StatusNotFound},
		{"a photo of an unknown review", request{method: "POST", path: "/v1/pantry/" + item + "/reviews/" + unknown + "/photos", body: jpg, contentType: "image/jpeg", token: ana.Token}, http.StatusNotFound},
		{"a photo of another type", request{method: "POST", path: reviewPath + "/photos", body: "text", contentType: "text/plain", token: ana.Token}, http.StatusUnsupportedMediaType},
		{"a photo that is not an image", request{method: "POST", path: reviewPath + "/photos", body: "not a jpeg", contentType: "image/jpeg", token: ana.Token}, http.StatusUnprocessableEntity},
		{"an update of an unknown photo", request{method: "PATCH", path: reviewPath + "/photos/" + unknown, token: ana.Token, body: `{}`}, http.StatusNotFound},
		{"a delete of an unknown photo", request{method: "DELETE", path: reviewPath + "/photos/" + unknown, token: ana.Token}, http.StatusNotFound},
		{"the file of an unknown photo", request{method: "GET", path: reviewPath + "/photos/" + unknown + "/full"}, http.StatusNotFound},
		{"the gallery of an unknown pantry item", request{method: "GET", path: "/v1/pantry/" + unknown + "/photos"}, http.StatusNotFound},
		{"the versions of an unknown pantry item", request{method: "GET", path: "/v1/pantry/" + unknown + "/versions", token: ana.Token}, http.StatusNotFound},
		{"an unknown list", request{method: "GET", path: "/v1/lists/" + unknown + "/pantry"}, http.StatusNotFound},
		{"an add to the list of another user", request{method: "PUT", path: "/v1/lists/" + list + "/pantry/" + item, token: joe.Token}, http.StatusForbidden},
		{"an add to an unknown list", request{method: "PUT", path: "/v1/lists/" + unknown + "/pantry/" + item, token: ana.Token}, http.StatusNotFound},
		{"a remove from the list of another user", request{method: "DELETE", path: "/v1/lists/" + list + "/pantry/" + item, token: joe.Token}, http.StatusForbidden},
		{"a remove from an unknown list", request{method: "DELETE", path: "/v1/lists/" + unknown + "/pantry/" + item, token: ana.Token}, http.StatusNotFound},
		{"an update of the photo of another user", request{method: "PATCH", path: reviewPath + "/photos/" + photo, token: joe.Token, body: `{"alt":"x"}`}, http.StatusForbidden},
		{"a delete of the review of another user", request{method: "DELETE", path: reviewPath, token: joe.Token}, http.StatusForbidden},
		{"a review of an unknown pantry item", request{method: "POST", path: "/v1/pantry/" + unknown + "/reviews", token: ana.Token, body: pantryItemReviewBody}, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e.expect(tc.req, tc.status)
		})
	}

	t.Run("a sixth photo gets 409", func(t *testing.T) {
		for range 4 {
			e.expect(request{method: "POST", path: reviewPath + "/photos", body: jpg, contentType: "image/jpeg", token: ana.Token}, http.StatusCreated)
		}
		e.expectCode(request{method: "POST", path: reviewPath + "/photos", body: jpg, contentType: "image/jpeg", token: ana.Token}, http.StatusConflict, "photo_limit")
	})
}
