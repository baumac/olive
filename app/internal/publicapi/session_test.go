package publicapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

const anaCredentials = `{"email":"ana@example.com","password":"correct-horse-battery"}`

func sessionCookieOf(t *testing.T, res response) *http.Cookie {
	t.Helper()
	for _, c := range (&http.Response{Header: res.header}).Cookies() {
		if c.Name == "reviews_session" {
			return c
		}
	}
	return nil
}

// cookieToken checks the cookie of a response that sets it. The body then
// has no token.
func cookieToken(t *testing.T, res response) string {
	t.Helper()
	c := sessionCookieOf(t, res)
	if c == nil || c.Value == "" {
		t.Fatalf("no session cookie in %v", res.header.Values("Set-Cookie"))
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/v1" || c.MaxAge <= 0 {
		t.Errorf("cookie %q", c.String())
	}
	if _, ok := res.object(t)["token"]; ok {
		t.Errorf("body has a token: %s", res.body)
	}
	return c.Value
}

func withCookie(token string, header map[string]string) map[string]string {
	h := map[string]string{"Cookie": "reviews_session=" + token, "Sec-Fetch-Site": "same-origin"}
	for k, v := range header {
		h[k] = v
	}
	return h
}

func TestSessionCookie(t *testing.T) {
	e := newEnv(t, testOptions)
	e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")

	res := e.expect(request{method: "POST", path: "/v1/users/tokens?cookie=true", body: anaCredentials}, http.StatusOK)
	ana := cookieToken(t, res)
	if c := sessionCookieOf(t, res); c.Secure {
		t.Errorf("cookie %q has Secure without an https Application URL", c.String())
	}

	t.Run("the cookie authenticates writes", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/restaurants", header: withCookie(ana, nil),
			body: `{"name":"Cookie Bistro","location":` + testLocation + `}`}, http.StatusCreated)
	})

	t.Run("refresh with the cookie sets a new cookie", func(t *testing.T) {
		res := e.expect(request{method: "POST", path: "/v1/users/tokens/refresh", header: withCookie(ana, nil)}, http.StatusOK)
		cookieToken(t, res)
		if res.object(t)["user"].(map[string]any)["name"] != "Ana" {
			t.Errorf("body %s", res.body)
		}
	})

	t.Run("the Authorization header wins over the cookie", func(t *testing.T) {
		res := e.expect(request{method: "POST", path: "/v1/users/tokens/refresh", token: joe.Token, header: withCookie(ana, nil)}, http.StatusOK)
		var tok authToken
		res.decode(t, &tok)
		if tok.Token == "" || tok.User.Name != "Joe" || sessionCookieOf(t, res) != nil {
			t.Errorf("refresh: %s %v", res.body, res.header.Values("Set-Cookie"))
		}
	})

	t.Run("a bearer token moves into the cookie", func(t *testing.T) {
		res := e.expect(request{method: "POST", path: "/v1/users/tokens/refresh?cookie=true", token: joe.Token}, http.StatusOK)
		cookieToken(t, res)
	})

	t.Run("the confirmation sets the cookie", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/users",
			body: `{"email":"eve@example.com","password":"correct-horse-battery","name":"Eve"}`}, http.StatusAccepted)
		res := e.expect(request{method: "POST", path: "/v1/users/confirmations?cookie=true",
			body: `{"token":"` + e.emailToken("eve@example.com") + `"}`}, http.StatusOK)
		cookieToken(t, res)
	})

	t.Run("a cross-origin request has no cookie", func(t *testing.T) {
		for name, header := range map[string]map[string]string{
			"Sec-Fetch-Site": {"Sec-Fetch-Site": "same-site"},
			"Origin":         {"Sec-Fetch-Site": "", "Origin": "https://evil.example"},
		} {
			e.expect(request{method: "POST", path: "/v1/restaurants", header: withCookie(ana, header),
				body: `{"name":"Cross Bistro","location":` + testLocation + `}`}, http.StatusUnauthorized)
			res := e.expect(request{method: "POST", path: "/v1/users/tokens?cookie=true", header: header, body: anaCredentials}, http.StatusOK)
			var tok authToken
			res.decode(t, &tok)
			if tok.Token == "" || sessionCookieOf(t, res) != nil {
				t.Errorf("%s: sign in %s %v", name, res.body, res.header.Values("Set-Cookie"))
			}
		}
		// A read does not change data. It always passes the check.
		e.expect(request{method: "GET", path: "/v1/restaurants",
			header: withCookie(ana, map[string]string{"Sec-Fetch-Site": "cross-site"})}, http.StatusOK)
	})

	t.Run("sign out deletes the cookie", func(t *testing.T) {
		res := e.expect(request{method: "POST", path: "/v1/users/tokens?cookie=true", body: anaCredentials}, http.StatusOK)
		token := cookieToken(t, res)
		res = e.expect(request{method: "DELETE", path: "/v1/users/tokens", header: withCookie(token, nil)}, http.StatusNoContent)
		if c := sessionCookieOf(t, res); c == nil || c.MaxAge >= 0 {
			t.Errorf("sign out: %v", res.header.Values("Set-Cookie"))
		}
		// A read with the old cookie gets 401. The response deletes the cookie.
		res = e.expect(request{method: "GET", path: "/v1/restaurants", header: withCookie(token, nil)}, http.StatusUnauthorized)
		if c := sessionCookieOf(t, res); c == nil || c.MaxAge >= 0 {
			t.Errorf("401: %v", res.header.Values("Set-Cookie"))
		}
	})

	t.Run("a cookie that cannot be refreshed gets deleted", func(t *testing.T) {
		user, err := e.app.FindAuthRecordByEmail("users", "joe@example.com")
		if err != nil {
			t.Fatal(err)
		}
		static, err := user.NewStaticAuthToken(time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		res := e.expect(request{method: "POST", path: "/v1/users/tokens/refresh", header: withCookie(static, nil)}, http.StatusUnauthorized)
		if c := sessionCookieOf(t, res); c == nil || c.MaxAge >= 0 {
			t.Errorf("401: %v", res.header.Values("Set-Cookie"))
		}
	})

	t.Run("a bearer token gets no cookie", func(t *testing.T) {
		res := e.expect(request{method: "DELETE", path: "/v1/users/tokens", token: joe.Token}, http.StatusNoContent)
		if sessionCookieOf(t, res) != nil {
			t.Errorf("sign out: %v", res.header.Values("Set-Cookie"))
		}
		res = e.expect(request{method: "GET", path: "/v1/restaurants", token: joe.Token}, http.StatusUnauthorized)
		if sessionCookieOf(t, res) != nil {
			t.Errorf("401: %v", res.header.Values("Set-Cookie"))
		}
	})
}

// TestSessionCookieSecure uses the host example.com of httptest.NewRequest.
func TestSessionCookieSecure(t *testing.T) {
	e := newEnv(t, testOptions)
	e.signUp("ana@example.com", "Ana")
	for _, tc := range []struct {
		appURL, proto string
		secure        bool
	}{
		{"https://example.com", "", true},
		{"https://reviews.example.com", "", false},
		{"http://example.com", "", false},
		{"", "", false},
		{"https://example.com", "http", false},
		{"", "https", true},
	} {
		e.app.Settings().Meta.AppURL = tc.appURL
		res := e.expect(request{method: "POST", path: "/v1/users/tokens?cookie=true", body: anaCredentials,
			header: map[string]string{"X-Forwarded-Proto": tc.proto}}, http.StatusOK)
		if c := sessionCookieOf(t, res); c == nil || c.Secure != tc.secure || strings.HasPrefix(c.Name, "__Host-") {
			t.Errorf("%+v: cookie %v", tc, res.header.Values("Set-Cookie"))
		}
	}
}
