package publicapi_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/baumac/reviews-app/app/internal/publicapi"
	_ "github.com/baumac/reviews-app/app/migrations"
	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

var testOptions = publicapi.Options{SignInLimit: 1000, SignInWindow: time.Minute}

type env struct {
	t   *testing.T
	app *tests.TestApp
	h   http.Handler
}

// newEnv starts a PocketBase app with an empty database and the migrations
// of this project. Requests go through the PocketBase router.
func newEnv(t *testing.T, opts publicapi.Options) *env {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	publicapi.Register(app, opts)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	se := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(se, func(*core.ServeEvent) error { return nil }); err != nil {
		t.Fatal(err)
	}
	h, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return &env{t: t, app: app, h: h}
}

type response struct {
	status int
	header http.Header
	body   []byte
}

func (r response) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		t.Fatalf("decode %s: %v", r.body, err)
	}
}

func (r response) object(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	r.decode(t, &m)
	return m
}

type request struct {
	method      string
	path        string
	body        string
	contentType string
	token       string
	header      map[string]string
}

func (e *env) do(req request) response {
	e.t.Helper()
	var body *strings.Reader
	if req.body != "" {
		body = strings.NewReader(req.body)
	} else {
		body = strings.NewReader("")
	}
	r := httptest.NewRequest(req.method, req.path, body)
	if req.body != "" {
		ct := req.contentType
		if ct == "" {
			ct = "application/json"
			if req.method == http.MethodPatch {
				ct = "application/merge-patch+json"
			}
		}
		r.Header.Set("Content-Type", ct)
	}
	if req.token != "" {
		r.Header.Set("Authorization", "Bearer "+req.token)
	}
	for k, v := range req.header {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	e.h.ServeHTTP(w, r)
	return response{status: w.Code, header: w.Header(), body: w.Body.Bytes()}
}

func (e *env) expect(req request, status int) response {
	e.t.Helper()
	res := e.do(req)
	if res.status != status {
		e.t.Fatalf("%s %s: status %d, want %d: %s", req.method, req.path, res.status, status, res.body)
	}
	if status >= 400 && res.header.Get("Content-Type") != "application/problem+json" {
		e.t.Fatalf("%s %s: content type %q, want application/problem+json", req.method, req.path, res.header.Get("Content-Type"))
	}
	return res
}

type authToken struct {
	Token   string `json:"token"`
	Expires string `json:"expires"`
	User    struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"user"`
}

func (e *env) signUp(email, name string) authToken {
	e.t.Helper()
	res := e.expect(request{method: "POST", path: "/v1/users",
		body: fmt.Sprintf(`{"email":%q,"password":"correct-horse-battery","name":%q}`, email, name)}, http.StatusCreated)
	var tok authToken
	res.decode(e.t, &tok)
	return tok
}

func (e *env) createRestaurant(token, body string) map[string]any {
	e.t.Helper()
	return e.expect(request{method: "POST", path: "/v1/restaurants", body: body, token: token}, http.StatusCreated).object(e.t)
}

func (e *env) createReview(token, restaurantID, body string) map[string]any {
	e.t.Helper()
	return e.expect(request{method: "POST", path: "/v1/restaurants/" + restaurantID + "/reviews", body: body, token: token}, http.StatusCreated).object(e.t)
}

func errorProperties(t *testing.T, res response) []string {
	t.Helper()
	var p struct {
		Errors []struct{ Property, Message string } `json:"errors"`
	}
	res.decode(t, &p)
	var out []string
	for _, e := range p.Errors {
		out = append(out, e.Property)
	}
	return out
}

func TestUsers(t *testing.T) {
	e := newEnv(t, testOptions)

	ana := e.signUp("ana@example.com", "Ana")
	if ana.Token == "" || ana.User.Email != "ana@example.com" || ana.User.Name != "Ana" || len(ana.User.ID) != 15 {
		t.Fatalf("sign up: %+v", ana)
	}
	if _, err := time.Parse("2006-01-02 15:04:05.000Z", ana.Expires); err != nil {
		t.Errorf("expires %q: %v", ana.Expires, err)
	}

	t.Run("sign up with a used email", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/users",
			body: `{"email":"ana@example.com","password":"another-password","name":"Ana 2"}`}, http.StatusConflict)
		e.expect(request{method: "POST", path: "/v1/users",
			body: `{"email":"ANA@Example.com","password":"another-password","name":"Ana 2"}`}, http.StatusConflict)
	})

	t.Run("sign in ignores ASCII case in the email", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/users/tokens",
			body: `{"email":"Ana@Example.COM","password":"correct-horse-battery"}`}, http.StatusOK)
	})

	t.Run("sign in", func(t *testing.T) {
		res := e.expect(request{method: "POST", path: "/v1/users/tokens",
			body: `{"email":"ana@example.com","password":"correct-horse-battery"}`}, http.StatusOK)
		var tok authToken
		res.decode(t, &tok)
		if tok.User.ID != ana.User.ID {
			t.Errorf("user %q, want %q", tok.User.ID, ana.User.ID)
		}
	})

	t.Run("sign in ignores an invalid token", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/users/tokens", token: "not-a-token",
			body: `{"email":"ana@example.com","password":"correct-horse-battery"}`}, http.StatusOK)
	})

	t.Run("wrong password and unknown email give the same response", func(t *testing.T) {
		wrong := e.expect(request{method: "POST", path: "/v1/users/tokens",
			body: `{"email":"ana@example.com","password":"wrong-password"}`}, http.StatusUnauthorized)
		unknown := e.expect(request{method: "POST", path: "/v1/users/tokens",
			body: `{"email":"nobody@example.com","password":"wrong-password"}`}, http.StatusUnauthorized)
		if string(wrong.body) != string(unknown.body) {
			t.Errorf("bodies differ:\n%s\n%s", wrong.body, unknown.body)
		}
		if got := wrong.header.Get("WWW-Authenticate"); got != "Bearer" {
			t.Errorf("WWW-Authenticate %q", got)
		}
	})

	t.Run("refresh", func(t *testing.T) {
		res := e.expect(request{method: "POST", path: "/v1/users/tokens/refresh", token: ana.Token}, http.StatusOK)
		var tok authToken
		res.decode(t, &tok)
		if tok.Token == "" || tok.User.ID != ana.User.ID {
			t.Errorf("refresh: %+v", tok)
		}
	})

	t.Run("refresh with a token that cannot be refreshed", func(t *testing.T) {
		user, err := e.app.FindRecordById("users", ana.User.ID)
		if err != nil {
			t.Fatal(err)
		}
		static, err := user.NewStaticAuthToken(time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		e.expect(request{method: "GET", path: "/v1/restaurants", token: static}, http.StatusOK)
		e.expect(request{method: "POST", path: "/v1/users/tokens/refresh", token: static}, http.StatusUnauthorized)
	})

	t.Run("refresh without a token", func(t *testing.T) {
		res := e.expect(request{method: "POST", path: "/v1/users/tokens/refresh"}, http.StatusUnauthorized)
		if got := res.header.Get("WWW-Authenticate"); got != "Bearer" {
			t.Errorf("WWW-Authenticate %q", got)
		}
	})

	t.Run("an invalid token on a read operation", func(t *testing.T) {
		res := e.expect(request{method: "GET", path: "/v1/restaurants", token: "not-a-token"}, http.StatusUnauthorized)
		if got := res.header.Get("WWW-Authenticate"); got != `Bearer error="invalid_token"` {
			t.Errorf("WWW-Authenticate %q", got)
		}
	})

	t.Run("the PocketBase account routes of users are closed", func(t *testing.T) {
		for _, path := range []string{
			"/api/collections/users/request-password-reset",
			"/api/collections/_pb_users_auth_/request-password-reset",
			"/api/collections/users/request-email-change",
			"/api/collections/users/confirm-password-reset",
		} {
			for _, email := range []string{"ana@example.com", "nobody@example.com"} {
				res := e.do(request{method: "POST", path: path, body: `{"email":"` + email + `","newEmail":"x@example.com"}`, token: ana.Token})
				if res.status != http.StatusNotFound {
					t.Errorf("%s with %s: status %d", path, email, res.status)
				}
			}
		}
	})

	t.Run("the PocketBase auth endpoint is off", func(t *testing.T) {
		res := e.do(request{method: "POST", path: "/api/collections/users/auth-with-password",
			body: `{"identity":"ana@example.com","password":"correct-horse-battery"}`})
		if res.status == http.StatusOK {
			t.Errorf("PocketBase sign in worked: %s", res.body)
		}
	})
}

func TestSignUpBody(t *testing.T) {
	e := newEnv(t, testOptions)
	tests := []struct {
		name   string
		body   string
		ct     string
		status int
		props  []string
	}{
		{"invalid values", `{"email":"not-an-email","password":"short","name":"   "}`, "", 422, []string{"email", "name", "password"}},
		{"password over 72 bytes", `{"email":"b@example.com","password":"` + strings.Repeat("é", 40) + `","name":"B"}`, "", 422, []string{"password"}},
		{"missing properties", `{}`, "", 422, []string{"email", "name", "password"}},
		{"hidden characters in the name", `{"email":"b@example.com","password":"correct-horse","name":"Ana\u202eb"}`, "", 422, []string{"name"}},
		{"wrong types", `{"email":"b@example.com","password":12345678,"name":null}`, "", 422, []string{"name", "password"}},
		{"unknown property", `{"email":"b@example.com","password":"correct-horse","name":"B","admin":true}`, "", 400, nil},
		{"repeated property", `{"email":"b@example.com","password":"correct-horse","name":5,"name":"B"}`, "", 400, nil},
		{"email with a display name", `{"email":"B <b@example.com>","password":"correct-horse","name":"B"}`, "", 422, []string{"email"}},
		{"two JSON values", `{"email":"b@example.com"} {}`, "", 400, nil},
		{"not an object", `["a"]`, "", 400, nil},
		{"wrong content type", `{"email":"b@example.com","password":"correct-horse","name":"B"}`, "text/plain", 415, nil},
		{"too large", `{"email":"` + strings.Repeat("a", 300<<10) + `@example.com"}`, "", 413, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := e.expect(request{method: "POST", path: "/v1/users", body: tc.body, contentType: tc.ct}, tc.status)
			if tc.props != nil {
				got := strings.Join(errorProperties(t, res), ",")
				if want := strings.Join(tc.props, ","); got != want {
					t.Errorf("properties %q, want %q", got, want)
				}
			}
		})
	}
}

func TestSignInRateLimit(t *testing.T) {
	e := newEnv(t, publicapi.Options{SignInLimit: 2, SignInWindow: time.Minute})
	body := `{"email":"ana@example.com","password":"wrong-password"}`
	for range 2 {
		e.expect(request{method: "POST", path: "/v1/users/tokens", body: body}, http.StatusUnauthorized)
	}
	res := e.expect(request{method: "POST", path: "/v1/users/tokens", body: body}, http.StatusTooManyRequests)
	if got := res.header.Get("Retry-After"); got == "" || got == "0" {
		t.Errorf("Retry-After %q", got)
	}
	// Sign up has its own count.
	e.signUp("ana@example.com", "Ana")
}

func TestRestaurants(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")

	r := e.createRestaurant(ana.Token, `{"name":"Luigi's","description":"Family run","tags":["italian","pasta"]}`)
	id := r["id"].(string)
	if r["creator"].(map[string]any)["name"] != "Ana" {
		t.Errorf("creator %v", r["creator"])
	}
	path := "/v1/restaurants/" + id

	t.Run("create sets Location", func(t *testing.T) {
		res := e.expect(request{method: "POST", path: "/v1/restaurants", body: `{"name":"Other"}`, token: ana.Token}, http.StatusCreated)
		if loc := res.header.Get("Location"); !strings.HasPrefix(loc, "/v1/restaurants/") {
			t.Errorf("Location %q", loc)
		}
		if strings.Contains(string(res.body), `"description"`) || strings.Contains(string(res.body), `"tags"`) {
			t.Errorf("unset properties are in the body: %s", res.body)
		}
	})

	t.Run("create errors", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/restaurants", body: `{"name":"A"}`}, http.StatusUnauthorized)
		e.expect(request{method: "POST", path: "/v1/restaurants", body: `{"name":"A","id":"k3x9m2p7q1w8z4a"}`, token: ana.Token}, http.StatusBadRequest)
		res := e.expect(request{method: "POST", path: "/v1/restaurants",
			body: `{"description":null,"tags":["ok","Bad","ok"]}`, token: ana.Token}, http.StatusUnprocessableEntity)
		if got := strings.Join(errorProperties(t, res), ","); got != "description,name,tags[1],tags[2]" {
			t.Errorf("properties %q", got)
		}
		many := `["` + strings.Repeat(`Bad","`, 999) + `Bad"]`
		res = e.expect(request{method: "POST", path: "/v1/restaurants", body: `{"name":"A","tags":` + many + `}`, token: ana.Token}, http.StatusUnprocessableEntity)
		if got := strings.Join(errorProperties(t, res), ","); got != "tags" {
			t.Errorf("properties %q", got)
		}
	})

	t.Run("get", func(t *testing.T) {
		got := e.expect(request{method: "GET", path: path}, http.StatusOK).object(t)
		if got["name"] != "Luigi's" || got["description"] != "Family run" {
			t.Errorf("get: %v", got)
		}
		e.expect(request{method: "GET", path: "/v1/restaurants/aaaaaaaaaaaaaaa"}, http.StatusNotFound)
		e.expect(request{method: "GET", path: "/v1/restaurants/BAD"}, http.StatusBadRequest)
	})

	t.Run("patch", func(t *testing.T) {
		e.expect(request{method: "PATCH", path: path, body: `{"description":"x"}`, token: joe.Token}, http.StatusForbidden)
		e.expect(request{method: "PATCH", path: path, body: `{"description":"x"}`, token: ana.Token, contentType: "application/json"}, http.StatusUnsupportedMediaType)
		e.expect(request{method: "PATCH", path: path, body: `{"name":null}`, token: ana.Token}, http.StatusUnprocessableEntity)

		got := e.expect(request{method: "PATCH", path: path, body: `{"description":null,"tags":null,"name":"Luigi's Trattoria"}`, token: ana.Token}, http.StatusOK).object(t)
		if _, ok := got["description"]; ok {
			t.Errorf("description is not cleared: %v", got)
		}
		if _, ok := got["tags"]; ok {
			t.Errorf("tags are not cleared: %v", got)
		}
		if got["name"] != "Luigi's Trattoria" {
			t.Errorf("name %v", got["name"])
		}

		time.Sleep(2 * time.Millisecond)
		same := e.expect(request{method: "PATCH", path: path, body: `{"name":"Luigi's Trattoria","tags":null}`, token: ana.Token}, http.StatusOK).object(t)
		if same["updated"] != got["updated"] {
			t.Errorf("a patch without changes changed updated from %v to %v", got["updated"], same["updated"])
		}
	})

	review := e.createReview(joe.Token, id, `{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"Great"}`)

	t.Run("name cannot change with reviews", func(t *testing.T) {
		e.expect(request{method: "PATCH", path: path, body: `{"name":"New name"}`, token: ana.Token}, http.StatusConflict)
		e.expect(request{method: "PATCH", path: path, body: `{"name":"Luigi's Trattoria","tags":["italian"]}`, token: ana.Token}, http.StatusOK)
	})

	t.Run("delete", func(t *testing.T) {
		e.expect(request{method: "DELETE", path: path, token: joe.Token}, http.StatusForbidden)
		e.expect(request{method: "DELETE", path: path, token: ana.Token}, http.StatusConflict)
		e.expect(request{method: "DELETE", path: path + "/reviews/" + review["id"].(string), token: joe.Token}, http.StatusNoContent)
		res := e.expect(request{method: "DELETE", path: path, token: ana.Token}, http.StatusNoContent)
		if len(res.body) != 0 {
			t.Errorf("body %q", res.body)
		}
		e.expect(request{method: "GET", path: path}, http.StatusNotFound)
	})
}

func TestReviews(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	id := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	other := e.createRestaurant(ana.Token, `{"name":"Other"}`)["id"].(string)
	reviews := "/v1/restaurants/" + id + "/reviews"

	good := e.createReview(joe.Token, id, `{"space":4,"service":5,"food":6,"price":7,"cleanliness":3,"title":"Good","feedback":"Nice","tags":["pasta"]}`)
	great := e.createReview(ana.Token, id, `{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"Great food"}`)
	goodPath := reviews + "/" + good["id"].(string)
	otherReviews := "/v1/restaurants/" + other + "/reviews"

	t.Run("responses have the scores", func(t *testing.T) {
		if good["space"] != float64(4) || good["price"] != float64(7) {
			t.Errorf("scores: %v", good)
		}
		if _, ok := good["kind"]; ok {
			t.Errorf("review has kind: %v", good)
		}
		if good["restaurant"] != id || good["creator"].(map[string]any)["name"] != "Joe" {
			t.Errorf("good: %v", good)
		}
	})

	t.Run("create errors", func(t *testing.T) {
		e.expect(request{method: "POST", path: reviews, body: `{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"Again"}`, token: joe.Token}, http.StatusConflict)
		e.expect(request{method: "POST", path: "/v1/restaurants/aaaaaaaaaaaaaaa/reviews", body: `{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"X"}`, token: joe.Token}, http.StatusNotFound)
		e.expect(request{method: "POST", path: otherReviews, body: `{"kind":"joc_de_cartes","space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"X"}`, token: joe.Token}, http.StatusBadRequest)
		e.expect(request{method: "POST", path: otherReviews, body: `{"rating":3,"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"X"}`, token: joe.Token}, http.StatusBadRequest)

		for body, want := range map[string]string{
			`{"space":7,"service":9,"food":8,"price":6,"cleanliness":0,"title":"X"}`:          "cleanliness",
			`{"space":11,"service":9,"food":8,"price":6,"cleanliness":8,"title":"X"}`:         "space",
			`{"space":7,"service":9,"title":"X"}`:                                             "cleanliness,food,price",
			`{"space":9999999999,"service":9,"food":8,"price":6,"cleanliness":8,"title":"X"}`: "space",
			`{"space":7,"service":2.5,"food":8,"price":6,"cleanliness":8,"title":"X"}`:        "service",
		} {
			res := e.expect(request{method: "POST", path: otherReviews, body: body, token: joe.Token}, http.StatusUnprocessableEntity)
			if got := strings.Join(errorProperties(t, res), ","); got != want {
				t.Errorf("%s: properties %q, want %q", body, got, want)
			}
		}
		e.expect(request{method: "POST", path: otherReviews, body: `{"space":"x","space":4,"service":9,"food":8,"price":6,"cleanliness":8,"title":"X"}`, token: joe.Token}, http.StatusBadRequest)
	})

	t.Run("the unique index blocks a second review of a user", func(t *testing.T) {
		collection, err := e.app.FindCollectionByNameOrId("reviews")
		if err != nil {
			t.Fatal(err)
		}
		r := core.NewRecord(collection)
		r.Load(map[string]any{"restaurant": id, "creator": joe.User.ID, "space": 1, "service": 1, "food": 1, "price": 1, "cleanliness": 1, "title": "Copy"})
		err = e.app.Save(r)
		var invalid validation.Errors
		if !errors.As(err, &invalid) || invalid["creator"] == nil {
			t.Errorf("save: %v", err)
		}
	})

	t.Run("get", func(t *testing.T) {
		got := e.expect(request{method: "GET", path: goodPath}, http.StatusOK).object(t)
		if got["restaurant_name"] != "Luigi's" {
			t.Errorf("get: restaurant_name %v", got["restaurant_name"])
		}
		if got["food"] != float64(6) || got["title"] != "Good" {
			t.Errorf("get: %v", got)
		}
		e.expect(request{method: "GET", path: otherReviews + "/" + good["id"].(string)}, http.StatusNotFound)
	})

	t.Run("patch", func(t *testing.T) {
		e.expect(request{method: "PATCH", path: goodPath, body: `{"price":1}`, token: ana.Token}, http.StatusForbidden)
		e.expect(request{method: "PATCH", path: goodPath, body: `{"kind":"joc_de_cartes","price":3}`, token: joe.Token}, http.StatusBadRequest)
		e.expect(request{method: "PATCH", path: goodPath, body: `{"rating":3}`, token: joe.Token}, http.StatusBadRequest)
		e.expect(request{method: "PATCH", path: goodPath, body: `{"title":null}`, token: joe.Token}, http.StatusUnprocessableEntity)
		res := e.expect(request{method: "PATCH", path: goodPath, body: `{"food":0}`, token: joe.Token}, http.StatusUnprocessableEntity)
		if got := strings.Join(errorProperties(t, res), ","); got != "food" {
			t.Errorf("properties %q", got)
		}

		same := e.expect(request{method: "PATCH", path: goodPath, body: `{}`, token: joe.Token}, http.StatusOK).object(t)
		if same["updated"] != good["updated"] || same["price"] != float64(7) {
			t.Errorf("empty patch: %v", same)
		}

		got := e.expect(request{method: "PATCH", path: goodPath, body: `{"price":2,"feedback":null}`, token: joe.Token}, http.StatusOK).object(t)
		if got["restaurant_name"] != "Luigi's" {
			t.Errorf("patch: restaurant_name %v", got["restaurant_name"])
		}
		if got["price"] != float64(2) || got["food"] != float64(6) || got["title"] != "Good" {
			t.Errorf("patch: %v", got)
		}
		if _, ok := got["feedback"]; ok {
			t.Errorf("feedback is not cleared: %v", got)
		}
	})

	t.Run("list filters by score", func(t *testing.T) {
		var page struct {
			Items []map[string]any `json:"items"`
		}
		e.expect(request{method: "GET", path: reviews + "?cleanliness[gte]=4"}, http.StatusOK).decode(t, &page)
		if len(page.Items) != 1 || page.Items[0]["title"] != "Great food" {
			t.Errorf("cleanliness filter: %v", page.Items)
		}
		e.expect(request{method: "GET", path: reviews + "?sort=cleanliness"}, http.StatusOK).decode(t, &page)
		if len(page.Items) != 2 || page.Items[0]["cleanliness"] != float64(3) {
			t.Errorf("cleanliness sort: %v", page.Items)
		}
		e.expect(request{method: "GET", path: reviews + "?food[gte]=7"}, http.StatusOK).decode(t, &page)
		if len(page.Items) != 1 || page.Items[0]["title"] != "Great food" {
			t.Errorf("food filter: %v", page.Items)
		}
		e.expect(request{method: "GET", path: reviews + "?sort=-price"}, http.StatusOK).decode(t, &page)
		if len(page.Items) != 2 || page.Items[0]["price"] != float64(6) {
			t.Errorf("price sort: %v", page.Items)
		}
		e.expect(request{method: "GET", path: reviews + "?sort=price&creator[eq]=" + joe.User.ID}, http.StatusOK).decode(t, &page)
		if len(page.Items) != 1 || page.Items[0]["price"] != float64(2) {
			t.Errorf("price sort with creator filter: %v", page.Items)
		}
		for _, query := range []string{"food[gte]=11", "kind[eq]=basic", "rating[gte]=1", "sort=rating"} {
			e.expect(request{method: "GET", path: reviews + "?" + query}, http.StatusBadRequest)
		}
		e.expect(request{method: "GET", path: "/v1/restaurants/aaaaaaaaaaaaaaa/reviews"}, http.StatusNotFound)
	})

	t.Run("pages of a score sort", func(t *testing.T) {
		sam := e.signUp("sam@example.com", "Sam")
		e.createReview(sam.Token, id, `{"space":1,"service":1,"food":1,"price":9,"cleanliness":8,"title":"Best"}`)
		for sort, want := range map[string]string{
			"-price": "9,6,2",
			"price":  "2,6,9",
		} {
			var got []string
			query := reviews + "?limit=1&sort=" + sort
			for range 4 {
				var page struct {
					Items      []map[string]any `json:"items"`
					NextCursor *string          `json:"next_cursor"`
				}
				e.expect(request{method: "GET", path: query}, http.StatusOK).decode(t, &page)
				for _, it := range page.Items {
					got = append(got, fmt.Sprint(it["price"]))
				}
				if page.NextCursor == nil {
					break
				}
				query = reviews + "?limit=1&sort=" + sort + "&cursor=" + url.QueryEscape(*page.NextCursor)
			}
			if strings.Join(got, ",") != want {
				t.Errorf("sort %s: %v, want %s", sort, got, want)
			}
		}
	})

	t.Run("deleting a user deletes the reviews and keeps the restaurants", func(t *testing.T) {
		user, err := e.app.FindRecordById("users", ana.User.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := e.app.Delete(user); err != nil {
			t.Fatal(err)
		}
		r := e.expect(request{method: "GET", path: "/v1/restaurants/" + id}, http.StatusOK).object(t)
		if c, ok := r["creator"]; !ok || c != nil {
			t.Errorf("creator %v, want null", c)
		}
		e.expect(request{method: "GET", path: reviews + "/" + great["id"].(string)}, http.StatusNotFound)
		e.expect(request{method: "PATCH", path: "/v1/restaurants/" + id, body: `{"description":"x"}`, token: joe.Token}, http.StatusForbidden)
	})
}

func TestListRestaurants(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	for _, r := range []struct{ token, body string }{
		{ana.Token, `{"name":"Zebra","tags":["italian","pasta"]}`},
		{ana.Token, `{"name":"apple","tags":["south-italian"]}`},
		{joe.Token, `{"name":"Mango","tags":["pasta"]}`},
		{joe.Token, `{"name":"banana"}`},
		{joe.Token, `{"name":"Cherry","tags":["italian"]}`},
	} {
		e.createRestaurant(r.token, r.body)
		// The created timestamps have millisecond precision.
		time.Sleep(2 * time.Millisecond)
	}

	type page struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
		NextCursor *string `json:"next_cursor"`
	}
	list := func(query string) page {
		t.Helper()
		var p page
		e.expect(request{method: "GET", path: "/v1/restaurants?" + query}, http.StatusOK).decode(t, &p)
		return p
	}
	names := func(p page) string {
		var out []string
		for _, it := range p.Items {
			out = append(out, it.Name)
		}
		return strings.Join(out, ",")
	}

	t.Run("pages in name order ignore ASCII case", func(t *testing.T) {
		var all []string
		query := "sort=name&limit=2"
		for range 5 {
			p := list(query)
			all = append(all, names(p))
			if p.NextCursor == nil {
				break
			}
			query = "sort=name&limit=2&cursor=" + url.QueryEscape(*p.NextCursor)
		}
		if got := strings.Join(all, "|"); got != "apple,banana|Cherry,Mango|Zebra" {
			t.Errorf("pages %q", got)
		}
	})

	t.Run("descending pages", func(t *testing.T) {
		p := list("sort=-name&limit=3")
		next := list("sort=-name&limit=3&cursor=" + url.QueryEscape(*p.NextCursor))
		if got := names(p) + "|" + names(next); got != "Zebra,Mango,Cherry|banana,apple" {
			t.Errorf("pages %q", got)
		}
		if next.NextCursor != nil {
			t.Errorf("next_cursor %q, want null", *next.NextCursor)
		}
	})

	t.Run("a cursor with values that are not strings or numbers", func(t *testing.T) {
		p := list("sort=name&limit=2")
		raw, err := base64.RawURLEncoding.DecodeString(*p.NextCursor)
		if err != nil {
			t.Fatal(err)
		}
		var c map[string]any
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		c["v"] = []any{map[string]any{}, []any{1}}
		raw, _ = json.Marshal(c)
		e.expect(request{method: "GET", path: "/v1/restaurants?sort=name&limit=2&cursor=" + base64.RawURLEncoding.EncodeToString(raw)}, http.StatusBadRequest)
	})

	t.Run("a cursor needs the same sort and filters", func(t *testing.T) {
		p := list("sort=name&limit=2")
		e.expect(request{method: "GET", path: "/v1/restaurants?sort=-name&cursor=" + url.QueryEscape(*p.NextCursor)}, http.StatusBadRequest)
		e.expect(request{method: "GET", path: "/v1/restaurants?sort=name&tags[has]=pasta&cursor=" + url.QueryEscape(*p.NextCursor)}, http.StatusBadRequest)
		e.expect(request{method: "GET", path: "/v1/restaurants?sort=name&limit=3&cursor=" + url.QueryEscape(*p.NextCursor)}, http.StatusOK)
	})

	t.Run("tag filters compare whole tags", func(t *testing.T) {
		cases := map[string]string{
			"tags[has]=italian": "Cherry,Zebra",
			"tags[has_any]=south-italian&tags[has_any]=pasta": "apple,Mango,Zebra",
			"tags[has_all]=italian&tags[has_all]=pasta":       "Zebra",
			"creator[in]=" + joe.User.ID:                      "banana,Cherry,Mango",
		}
		for query, want := range cases {
			if got := names(list("sort=name&" + query)); got != want {
				t.Errorf("%s: %q, want %q", query, got, want)
			}
		}
	})

	t.Run("default sort is newest first", func(t *testing.T) {
		if got := names(list("")); got != "Cherry,banana,Mango,apple,Zebra" {
			t.Errorf("default sort %q", got)
		}
	})

	t.Run("a binder error names the parameter", func(t *testing.T) {
		res := e.expect(request{method: "GET", path: "/v1/restaurants?limit=x"}, http.StatusBadRequest)
		if got := res.object(t)["detail"]; got != "The parameter limit is not valid." {
			t.Errorf("detail %q", got)
		}
	})

	t.Run("invalid query parameters", func(t *testing.T) {
		for _, query := range []string{
			"limit=0", "limit=101", "limit=x", "sort=name,name", "sort=title",
			"name[eq]=Zebra", "created[foo]=x", "created[gte]=2024-11-04", "tags[has]=Italian",
			"creator[eq]=short", "limit=1&limit=2", "cursor=not-a-cursor",
			strings.Repeat("tags[has_any]=a&", 21) + "sort=name",
		} {
			e.expect(request{method: "GET", path: "/v1/restaurants?" + query}, http.StatusBadRequest)
		}
	})
}

func TestUnknownRoutes(t *testing.T) {
	e := newEnv(t, testOptions)
	e.expect(request{method: "GET", path: "/v1/nothing"}, http.StatusNotFound)
	res := e.expect(request{method: "PUT", path: "/v1/restaurants"}, http.StatusMethodNotAllowed)
	if got := res.header.Get("Allow"); got != "GET, POST" {
		t.Errorf("Allow %q", got)
	}
}

func TestListAllReviews(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	luigi := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	sole := e.createRestaurant(joe.Token, `{"name":"Can Sole"}`)["id"].(string)
	e.createReview(ana.Token, luigi, `{"space":4,"service":4,"food":4,"price":4,"cleanliness":8,"title":"Ana at Luigi"}`)
	time.Sleep(2 * time.Millisecond)
	e.createReview(joe.Token, luigi, `{"space":2,"service":2,"food":2,"price":2,"cleanliness":8,"title":"Joe at Luigi"}`)
	time.Sleep(2 * time.Millisecond)
	created := e.createReview(ana.Token, sole, `{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"Ana at Sole"}`)

	if created["restaurant_name"] != "Can Sole" {
		t.Errorf("create: restaurant_name %v", created["restaurant_name"])
	}

	type page struct {
		Items []struct {
			Title          string `json:"title"`
			RestaurantName string `json:"restaurant_name"`
		} `json:"items"`
		NextCursor *string `json:"next_cursor"`
	}
	list := func(query string) page {
		t.Helper()
		var p page
		e.expect(request{method: "GET", path: "/v1/reviews?" + query}, http.StatusOK).decode(t, &p)
		return p
	}
	titles := func(p page) string {
		var out []string
		for _, it := range p.Items {
			out = append(out, it.Title+"@"+it.RestaurantName)
		}
		return strings.Join(out, ",")
	}

	for query, want := range map[string]string{
		"":                               "Ana at Sole@Can Sole,Joe at Luigi@Luigi's,Ana at Luigi@Luigi's",
		"creator[eq]=" + ana.User.ID:     "Ana at Sole@Can Sole,Ana at Luigi@Luigi's",
		"restaurant[eq]=" + luigi:        "Joe at Luigi@Luigi's,Ana at Luigi@Luigi's",
		"restaurant[in]=" + sole:         "Ana at Sole@Can Sole",
		"sort=space":                     "Joe at Luigi@Luigi's,Ana at Luigi@Luigi's,Ana at Sole@Can Sole",
		"restaurant[eq]=aaaaaaaaaaaaaaa": "",
	} {
		if got := titles(list(query)); got != want {
			t.Errorf("%q: %q, want %q", query, got, want)
		}
	}

	first := list("limit=2")
	second := list("limit=2&cursor=" + url.QueryEscape(*first.NextCursor))
	if got := titles(first) + "|" + titles(second); got != "Ana at Sole@Can Sole,Joe at Luigi@Luigi's|Ana at Luigi@Luigi's" {
		t.Errorf("pages %q", got)
	}

	for _, query := range []string{"restaurant[eq]=BAD", "restaurant[in]=x", "restaurantId=" + luigi} {
		e.expect(request{method: "GET", path: "/v1/reviews?" + query}, http.StatusBadRequest)
	}

	got := e.expect(request{method: "GET", path: "/v1/restaurants/" + luigi + "/reviews"}, http.StatusOK).object(t)
	if items := got["items"].([]any); items[0].(map[string]any)["restaurant_name"] != "Luigi's" {
		t.Errorf("per-restaurant list: %v", items[0])
	}
}
