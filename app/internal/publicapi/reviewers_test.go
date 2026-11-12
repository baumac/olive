package publicapi_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestReviewOverall(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	sam := e.signUp("sam@example.com", "Sam")
	id := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	reviews := "/v1/restaurants/" + id + "/reviews"

	low := e.createReview(ana.Token, id, `{"space":4,"service":5,"food":6,"price":7,"cleanliness":3,"title":"Low"}`)
	e.createReview(joe.Token, id, `{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"High"}`)
	e.createReview(sam.Token, id, `{"space":7,"service":7,"food":7,"price":7,"cleanliness":7,"title":"Mid"}`)

	if low["overall"] != 5.0 {
		t.Errorf("create: overall %v, want 5", low["overall"])
	}
	lowPath := reviews + "/" + low["id"].(string)
	patched := e.expect(request{method: "PATCH", path: lowPath, body: `{"food":7}`, token: ana.Token}, http.StatusOK).object(t)
	if patched["overall"] != 5.2 {
		t.Errorf("patch: overall %v, want 5.2", patched["overall"])
	}
	if got := e.expect(request{method: "GET", path: lowPath}, http.StatusOK).object(t)["overall"]; got != 5.2 {
		t.Errorf("get: overall %v, want 5.2", got)
	}

	restaurant := e.expect(request{method: "GET", path: "/v1/restaurants/" + id}, http.StatusOK).object(t)
	if got := restaurant["rating"].(map[string]any)["overall"]; got != 6.6 {
		t.Errorf("rating.overall %v, want 6.6, the average of 5.2, 7.6, and 7", got)
	}

	list := func(path string) string {
		t.Helper()
		var p struct {
			Items []struct {
				Title   string  `json:"title"`
				Overall float64 `json:"overall"`
			} `json:"items"`
		}
		e.expect(request{method: "GET", path: path}, http.StatusOK).decode(t, &p)
		var out []string
		for _, it := range p.Items {
			out = append(out, it.Title)
		}
		return strings.Join(out, ",")
	}
	joeReviews := "/v1/reviewers/" + joe.User.ID + "/reviews"
	for query, want := range map[string]struct{ all, joe string }{
		"sort=overall":                 {"Low,Mid,High", "High"},
		"sort=-overall":                {"High,Mid,Low", "High"},
		"overall[eq]=7.6":              {"High", "High"},
		"overall[gte]=7&sort=-overall": {"High,Mid", "High"},
		"overall[lt]=7":                {"Low", ""},
	} {
		for path, want := range map[string]string{reviews: want.all, "/v1/reviews": want.all, joeReviews: want.joe} {
			if got := list(path + "?" + query); got != want {
				t.Errorf("%s?%s: %q, want %q", path, query, got, want)
			}
		}
	}

	first := e.expect(request{method: "GET", path: reviews + "?sort=-overall&limit=2"}, http.StatusOK).object(t)
	cursor := url.QueryEscape(first["next_cursor"].(string))
	if got := list(reviews + "?sort=-overall&limit=2&cursor=" + cursor); got != "Low" {
		t.Errorf("second page by overall: %q", got)
	}

	for _, query := range []string{"overall[gte]=0.5", "overall[gte]=x", "overall[gte]=NaN", "sort=-overalls"} {
		e.expect(request{method: "GET", path: reviews + "?" + query}, http.StatusBadRequest)
	}
	e.expect(request{method: "PATCH", path: lowPath, body: `{"overall":9}`, token: ana.Token}, http.StatusBadRequest)
	e.expect(request{method: "POST", path: "/v1/restaurants/" + e.createRestaurant(ana.Token, `{"name":"Other"}`)["id"].(string) + "/reviews",
		body: `{"overall":9,"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"X"}`, token: ana.Token}, http.StatusBadRequest)
}

func TestReviewers(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	sam := e.signUp("sam@example.com", "Sam")
	luigi := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	sole := e.createRestaurant(joe.Token, `{"name":"Can Sole"}`)["id"].(string)
	e.createReview(ana.Token, luigi, `{"space":4,"service":4,"food":4,"price":4,"cleanliness":8,"title":"Ana at Luigi"}`)
	time.Sleep(2 * time.Millisecond)
	e.createReview(joe.Token, luigi, `{"space":2,"service":2,"food":2,"price":2,"cleanliness":8,"title":"Joe at Luigi"}`)
	time.Sleep(2 * time.Millisecond)
	e.createReview(ana.Token, sole, `{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"Ana at Sole"}`)

	t.Run("the profile has only public fields", func(t *testing.T) {
		got := e.expect(request{method: "GET", path: "/v1/reviewers/" + ana.User.ID}, http.StatusOK).object(t)
		want := map[string]any{"id": ana.User.ID, "name": "Ana", "review_count": 2.0}
		if len(got) != len(want) {
			t.Errorf("profile %v, want only %v", got, want)
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("profile %s is %v, want %v", k, got[k], v)
			}
		}
		empty := e.expect(request{method: "GET", path: "/v1/reviewers/" + sam.User.ID, token: joe.Token}, http.StatusOK).object(t)
		if empty["review_count"] != 0.0 {
			t.Errorf("profile without reviews: %v", empty)
		}
	})

	t.Run("the list has only the reviews of the user", func(t *testing.T) {
		list := func(id, query string) (string, *string) {
			t.Helper()
			var p struct {
				Items []struct {
					Title   string `json:"title"`
					Creator struct {
						ID string `json:"id"`
					} `json:"creator"`
				} `json:"items"`
				NextCursor *string `json:"next_cursor"`
			}
			e.expect(request{method: "GET", path: "/v1/reviewers/" + id + "/reviews?" + query}, http.StatusOK).decode(t, &p)
			var out []string
			for _, it := range p.Items {
				if it.Creator.ID != id {
					t.Errorf("%s: creator %s, want %s", it.Title, it.Creator.ID, id)
				}
				out = append(out, it.Title)
			}
			return strings.Join(out, ","), p.NextCursor
		}
		for query, want := range map[string]string{
			"":                        "Ana at Sole,Ana at Luigi",
			"restaurant[eq]=" + luigi: "Ana at Luigi",
			"restaurant[in]=" + sole:  "Ana at Sole",
			"sort=title":              "Ana at Luigi,Ana at Sole",
			"search=sole":             "Ana at Sole",
			"food[gte]=5":             "Ana at Sole",
		} {
			if got, _ := list(ana.User.ID, query); got != want {
				t.Errorf("%q: %q, want %q", query, got, want)
			}
		}
		if got, _ := list(sam.User.ID, ""); got != "" {
			t.Errorf("user without reviews: %q", got)
		}

		first, next := list(ana.User.ID, "limit=1")
		second, end := list(ana.User.ID, "limit=1&cursor="+url.QueryEscape(*next))
		if first+"|"+second != "Ana at Sole|Ana at Luigi" || end != nil {
			t.Errorf("pages %q|%q, next %v", first, second, end)
		}
		e.expect(request{method: "GET", path: "/v1/reviewers/" + joe.User.ID + "/reviews?cursor=" + url.QueryEscape(*next)}, http.StatusBadRequest)
	})

	t.Run("errors", func(t *testing.T) {
		unknown := "/v1/reviewers/aaaaaaaaaaaaaaa"
		e.expect(request{method: "GET", path: unknown}, http.StatusNotFound)
		e.expect(request{method: "GET", path: unknown + "/reviews"}, http.StatusNotFound)
		e.expect(request{method: "GET", path: unknown + "/reviews?sort=bad"}, http.StatusBadRequest)
		e.expect(request{method: "GET", path: "/v1/reviewers/BAD"}, http.StatusBadRequest)
		e.expect(request{method: "GET", path: "/v1/reviewers/" + ana.User.ID + "/reviews?creator[eq]=" + joe.User.ID}, http.StatusBadRequest)
		e.expect(request{method: "GET", path: "/v1/reviewers/" + ana.User.ID, token: "not-a-token"}, http.StatusUnauthorized)
	})
}
