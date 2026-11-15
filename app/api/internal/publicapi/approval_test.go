package publicapi_test

import (
	"net/http"
	"testing"
)

func TestUserApproval(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	review := e.createReview(ana.Token, restaurant, `{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"Good"}`)["id"].(string)
	list := e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Date night"}`, token: ana.Token}, http.StatusCreated).object(t)["id"].(string)

	e.expect(request{method: "POST", path: "/v1/users",
		body: `{"email":"joe@example.com","password":"correct-horse-battery","name":"Joe"}`}, http.StatusAccepted)
	res := e.expect(request{method: "POST", path: "/v1/users/confirmations",
		body: `{"token":"` + e.emailToken("joe@example.com") + `"}`}, http.StatusOK)
	var joe authToken
	res.decode(t, &joe)
	if res.object(t)["user"].(map[string]any)["approved"] != false {
		t.Errorf("confirmation: user %v, want approved false", res.object(t)["user"])
	}

	reviewPath := "/v1/restaurants/" + restaurant + "/reviews/" + review
	writes := []request{
		{method: "POST", path: "/v1/restaurants", body: `{"name":"X","location":` + testLocation + `}`},
		{method: "PATCH", path: "/v1/restaurants/" + restaurant, body: `{"description":"X"}`},
		{method: "DELETE", path: "/v1/restaurants/" + restaurant},
		{method: "POST", path: "/v1/restaurants/" + restaurant + "/reviews", body: `{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"X"}`},
		{method: "PATCH", path: reviewPath, body: `{"title":"X"}`},
		{method: "DELETE", path: reviewPath},
		{method: "DELETE", path: reviewPath + "/photos/aaaaaaaaaaaaaaa"},
		{method: "GET", path: "/v1/places?q=luigi"},
		{method: "POST", path: "/v1/lists", body: `{"name":"X"}`},
		{method: "PATCH", path: "/v1/lists/" + list, body: `{"name":"X"}`},
		{method: "DELETE", path: "/v1/lists/" + list},
		{method: "PUT", path: "/v1/lists/favorites/restaurants/" + restaurant},
		{method: "DELETE", path: "/v1/lists/favorites/restaurants/" + restaurant},
	}
	t.Run("a user without approval cannot write", func(t *testing.T) {
		for _, req := range writes {
			req.token = joe.Token
			got := e.expect(req, http.StatusForbidden).object(t)
			if got["detail"] != "The account is not approved. A superuser must approve it." || got["code"] != "not_approved" {
				t.Errorf("%s %s: problem %v", req.method, req.path, got)
			}
		}
	})

	t.Run("a user without approval can read, refresh, and sign in", func(t *testing.T) {
		for _, path := range []string{"/v1/restaurants", "/v1/restaurants/" + restaurant, reviewPath, "/v1/reviewers/" + joe.User.ID, "/v1/reviewers/" + joe.User.ID + "/lists"} {
			e.expect(request{method: "GET", path: path, token: joe.Token}, http.StatusOK)
		}
		refreshed := e.expect(request{method: "POST", path: "/v1/users/tokens/refresh", token: joe.Token}, http.StatusOK).object(t)
		if refreshed["user"].(map[string]any)["approved"] != false {
			t.Errorf("refresh: %v", refreshed["user"])
		}
		e.expect(request{method: "POST", path: "/v1/users/tokens", body: `{"email":"joe@example.com","password":"correct-horse-battery"}`}, http.StatusOK)
	})

	setApproved := func(approved bool) {
		t.Helper()
		user, err := e.app.FindRecordById("users", joe.User.ID)
		if err != nil {
			t.Fatal(err)
		}
		user.Set("approved", approved)
		if err := e.app.Save(user); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("an approved user can write with the same token", func(t *testing.T) {
		setApproved(true)
		refreshed := e.expect(request{method: "POST", path: "/v1/users/tokens/refresh", token: joe.Token}, http.StatusOK).object(t)
		if refreshed["user"].(map[string]any)["approved"] != true {
			t.Errorf("refresh: %v", refreshed["user"])
		}
		e.expect(request{method: "PUT", path: "/v1/lists/favorites/restaurants/" + restaurant, token: joe.Token}, http.StatusNoContent)
		e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Lunch"}`, token: joe.Token}, http.StatusCreated)
	})

	t.Run("a superuser can take the approval back", func(t *testing.T) {
		setApproved(false)
		e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Dinner"}`, token: joe.Token}, http.StatusForbidden)
	})

	t.Run("other people do not see a user without approval", func(t *testing.T) {
		profile := "/v1/reviewers/" + joe.User.ID
		var favorites string
		for _, l := range e.listsAs(joe.User.ID, joe.Token).Items {
			favorites = "/v1/lists/" + l.ID
		}
		paths := []string{profile, profile + "/reviews", profile + "/lists", favorites, favorites + "/restaurants"}
		for _, path := range paths {
			e.expect(request{method: "GET", path: path}, http.StatusNotFound)
			e.expect(request{method: "GET", path: path, token: ana.Token}, http.StatusNotFound)
			e.expect(request{method: "GET", path: path, token: joe.Token}, http.StatusOK)
		}
		e.expect(request{method: "GET", path: profile + "/reviews?limit=0", token: ana.Token}, http.StatusBadRequest)
		setApproved(true)
		for _, path := range paths {
			e.expect(request{method: "GET", path: path}, http.StatusOK)
		}
		setApproved(false)
	})

	t.Run("a user without approval can sign out", func(t *testing.T) {
		e.expect(request{method: "DELETE", path: "/v1/users/tokens", token: joe.Token}, http.StatusNoContent)
	})

	t.Run("a request without a token gets 401 before 403", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"X"}`}, http.StatusUnauthorized)
	})
}
