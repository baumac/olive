package publicapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/baumac/reviews-app/app/internal/publicapi"
	"github.com/pocketbase/pocketbase/core"
)

const (
	anaSignUp     = `{"email":"ana@example.com","password":"correct-horse-battery","name":"Ana"}`
	anaSignIn     = `{"email":"ana@example.com","password":"correct-horse-battery"}`
	tokenProperty = "token"
)

func (e *env) emailCount() int {
	return e.app.TestMailer.TotalSend()
}

func (e *env) signIn(body string, status int) response {
	e.t.Helper()
	return e.expect(request{method: "POST", path: "/v1/users/tokens", body: body}, status)
}

func (e *env) addSuperuser(email string) {
	e.t.Helper()
	superusers, err := e.app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		e.t.Fatal(err)
	}
	su := core.NewRecord(superusers)
	su.SetEmail(email)
	su.SetPassword("correct-horse-battery")
	if err := e.app.Save(su); err != nil {
		e.t.Fatal(err)
	}
}

func TestSignUpConfirmation(t *testing.T) {
	e := newEnv(t, testOptions)
	e.addSuperuser("root@example.com")

	res := e.expect(request{method: "POST", path: "/v1/users", body: anaSignUp}, http.StatusAccepted)
	if len(res.body) != 0 {
		t.Errorf("sign up body %s, want no body", res.body)
	}
	mail := e.lastEmail("ana@example.com")
	if !strings.Contains(mail, "Confirm your email address") || !strings.Contains(mail, "/confirm-email#token=") {
		t.Fatalf("confirmation email: %s", mail)
	}
	// A person who does not own the address can set the name.
	if strings.Contains(mail, "Ana") {
		t.Errorf("the confirmation email has the name of the request: %s", mail)
	}
	first := e.emailToken("ana@example.com")

	t.Run("sign in before the confirmation", func(t *testing.T) {
		e.signIn(`{"email":"ana@example.com","password":"wrong-password"}`, http.StatusUnauthorized)
		before := e.emailCount()
		res := e.signIn(anaSignIn, http.StatusForbidden)
		if code := res.object(t)["code"]; code != "email_not_confirmed" {
			t.Errorf("code %v, want email_not_confirmed", code)
		}
		if e.emailCount() != before+1 {
			t.Errorf("%d emails after the sign in, want %d", e.emailCount(), before+1)
		}
	})

	t.Run("a token that is not valid", func(t *testing.T) {
		for _, token := range []string{"not-a-token", first + "x"} {
			res := e.expect(request{method: "POST", path: "/v1/users/confirmations", body: `{"token":"` + token + `"}`}, http.StatusUnprocessableEntity)
			if got := strings.Join(errorProperties(t, res), ","); got != tokenProperty {
				t.Errorf("properties %q, want token", got)
			}
		}
	})

	t.Run("a second sign up replaces the user without confirmation", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/users",
			body: `{"email":"ANA@example.com","password":"second-horse-battery","name":"Ana B"}`}, http.StatusAccepted)
		e.expect(request{method: "POST", path: "/v1/users/confirmations", body: `{"token":"` + first + `"}`}, http.StatusUnprocessableEntity)
		e.signIn(anaSignIn, http.StatusUnauthorized)
		e.signIn(`{"email":"ana@example.com","password":"second-horse-battery"}`, http.StatusForbidden)
		// The first sign up of this test has the password of anaSignIn again.
		e.expect(request{method: "POST", path: "/v1/users", body: anaSignUp}, http.StatusAccepted)
	})

	token := e.emailToken("ana@example.com")
	res = e.expect(request{method: "POST", path: "/v1/users/confirmations", body: `{"token":"` + token + `"}`}, http.StatusOK)
	var ana authToken
	res.decode(t, &ana)
	if ana.Token == "" || ana.User.Email != "ana@example.com" || ana.User.Name != "Ana" {
		t.Fatalf("confirmation: %+v", ana)
	}

	t.Run("the superusers get an email", func(t *testing.T) {
		mail := e.lastEmail("root@example.com")
		if !strings.Contains(mail, "a new user waits for approval") || !strings.Contains(mail, "Ana (ana@example.com)") {
			t.Errorf("email to the superuser: %s", mail)
		}
	})

	t.Run("a token works one time", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/users/confirmations", body: `{"token":"` + token + `"}`}, http.StatusUnprocessableEntity)
	})

	t.Run("sign in after the confirmation", func(t *testing.T) {
		e.signIn(anaSignIn, http.StatusOK)
	})

	t.Run("sign up with the email of a confirmed user", func(t *testing.T) {
		for _, email := range []string{"ana@example.com", "ANA@Example.com"} {
			before := e.emailCount()
			res := e.expect(request{method: "POST", path: "/v1/users",
				body: `{"email":"` + email + `","password":"another-password","name":"Eve"}`}, http.StatusAccepted)
			if len(res.body) != 0 {
				t.Errorf("sign up body %s, want no body", res.body)
			}
			mail := e.lastEmail("ana@example.com")
			if e.emailCount() != before+1 || !strings.Contains(mail, "You already have an account") || strings.Contains(mail, "token=") {
				t.Errorf("email for a used address: %s", mail)
			}
		}
		e.signIn(anaSignIn, http.StatusOK)
		e.signIn(`{"email":"ana@example.com","password":"another-password"}`, http.StatusUnauthorized)
	})

	t.Run("invalid values get 422 also for a used email", func(t *testing.T) {
		long := strings.Repeat("é", 40)
		for _, email := range []string{"ana@example.com", "new@example.com"} {
			res := e.expect(request{method: "POST", path: "/v1/users",
				body: `{"email":"` + email + `","password":"` + long + `","name":"Ana"}`}, http.StatusUnprocessableEntity)
			if got := strings.Join(errorProperties(t, res), ","); got != "password" {
				t.Errorf("%s: properties %q, want password", email, got)
			}
		}
	})

	t.Run("an approval sends an email to the user", func(t *testing.T) {
		user, err := e.app.FindRecordById("users", ana.User.ID)
		if err != nil {
			t.Fatal(err)
		}
		user.Set("approved", true)
		if err := e.app.Save(user); err != nil {
			t.Fatal(err)
		}
		if mail := e.lastEmail("ana@example.com"); !strings.Contains(mail, "is approved") {
			t.Errorf("email after the approval: %s", mail)
		}
		before := e.emailCount()
		// The dashboard also loads the record for each save.
		if user, err = e.app.FindRecordById("users", ana.User.ID); err != nil {
			t.Fatal(err)
		}
		user.Set("name", "Ana C")
		if err := e.app.Save(user); err != nil {
			t.Fatal(err)
		}
		if e.emailCount() != before {
			t.Error("a later save of an approved user sent an email")
		}
	})
}

func TestPasswordReset(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	request202 := func(email string) {
		t.Helper()
		res := e.expect(request{method: "POST", path: "/v1/users/password-resets", body: `{"email":"` + email + `"}`}, http.StatusAccepted)
		if len(res.body) != 0 {
			t.Errorf("body %s, want no body", res.body)
		}
	}

	before := e.emailCount()
	request202("nobody@example.com")
	if e.emailCount() != before {
		t.Error("an unknown email address got an email")
	}
	request202("Ana@example.com")
	mail := e.lastEmail("ana@example.com")
	if !strings.Contains(mail, "Set a new password") || !strings.Contains(mail, "/reset-password#token=") {
		t.Fatalf("reset email: %s", mail)
	}
	token := e.emailToken("ana@example.com")

	reset := func(token, password string, status int) response {
		t.Helper()
		return e.expect(request{method: "PUT", path: "/v1/users/password",
			body: `{"token":"` + token + `","password":"` + password + `"}`}, status)
	}
	if got := strings.Join(errorProperties(t, reset("not-a-token", "new-horse-battery", 422)), ","); got != tokenProperty {
		t.Errorf("properties %q, want token", got)
	}
	if got := strings.Join(errorProperties(t, reset(token, "short", 422)), ","); got != "password" {
		t.Errorf("properties %q, want password", got)
	}
	reset(token, "new-horse-battery", http.StatusNoContent)

	e.signIn(anaSignIn, http.StatusUnauthorized)
	e.signIn(`{"email":"ana@example.com","password":"new-horse-battery"}`, http.StatusOK)
	e.expect(request{method: "POST", path: "/v1/users/tokens/refresh", token: ana.Token}, http.StatusUnauthorized)
	reset(token, "third-horse-battery", http.StatusUnprocessableEntity)

	t.Run("a confirmation token is not a reset token", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/users", body: `{"email":"joe@example.com","password":"correct-horse-battery","name":"Joe"}`}, http.StatusAccepted)
		reset(e.emailToken("joe@example.com"), "new-horse-battery", http.StatusUnprocessableEntity)
	})

	t.Run("a reset confirms the email address", func(t *testing.T) {
		request202("joe@example.com")
		reset(e.emailToken("joe@example.com"), "new-horse-battery", http.StatusNoContent)
		e.signIn(`{"email":"joe@example.com","password":"new-horse-battery"}`, http.StatusOK)
	})
}

func TestEmailLimit(t *testing.T) {
	e := newEnv(t, publicapi.Options{SignInLimit: 1000, SignInWindow: time.Minute, EmailLimit: 2})
	e.signUp("ana@example.com", "Ana")
	// The confirmation email has its own count.
	before := e.emailCount()
	for range 3 {
		e.expect(request{method: "POST", path: "/v1/users/password-resets", body: `{"email":"ana@example.com"}`}, http.StatusAccepted)
	}
	if got := e.emailCount() - before; got != 2 {
		t.Errorf("%d reset emails, want 2", got)
	}

	t.Run("a sign up at the limit does not change a user without confirmation", func(t *testing.T) {
		signUp := func(password string) {
			t.Helper()
			e.expect(request{method: "POST", path: "/v1/users",
				body: `{"email":"joe@example.com","password":"` + password + `","name":"Joe"}`}, http.StatusAccepted)
		}
		signUp("first-horse-battery")
		signUp("second-horse-battery")
		token := e.emailToken("joe@example.com")
		before := e.emailCount()
		signUp("third-horse-battery")
		if e.emailCount() != before {
			t.Error("a sign up at the limit sent an email")
		}
		e.expect(request{method: "POST", path: "/v1/users/confirmations", body: `{"token":"` + token + `"}`}, http.StatusOK)
		e.signIn(`{"email":"joe@example.com","password":"second-horse-battery"}`, http.StatusOK)
	})
}

func TestEmailWithoutSMTP(t *testing.T) {
	e := newEnv(t, testOptions)
	e.app.Settings().SMTP.Enabled = false
	e.expect(request{method: "POST", path: "/v1/users", body: anaSignUp}, http.StatusAccepted)
	if got := e.emailCount(); got != 0 {
		t.Errorf("%d emails without SMTP, want 0", got)
	}
}

func TestUpdateUser(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUpPending("ana@example.com", "Ana")
	patch := func(body string, status int) response {
		t.Helper()
		return e.expect(request{method: "PATCH", path: "/v1/users/me", body: body, token: ana.Token}, status)
	}

	user := patch(`{"name":"Ana Maria"}`, http.StatusOK).object(t)
	if user["name"] != "Ana Maria" || user["email"] != "ana@example.com" || user["id"] != ana.User.ID {
		t.Errorf("user %v", user)
	}
	if user := patch(`{}`, http.StatusOK).object(t); user["name"] != "Ana Maria" {
		t.Errorf("user after an empty patch: %v", user)
	}
	if got := strings.Join(errorProperties(t, patch(`{"name":"  "}`, 422)), ","); got != "name" {
		t.Errorf("properties %q, want name", got)
	}
	patch(`{"email":"eve@example.com"}`, http.StatusBadRequest)
	e.expect(request{method: "PATCH", path: "/v1/users/me", body: `{"name":"Eve"}`}, http.StatusUnauthorized)
}

func TestChangePassword(t *testing.T) {
	e := newEnv(t, publicapi.Options{SignInLimit: 1000, SignInWindow: time.Minute, SignInFailureLimit: 3})
	ana := e.signUpPending("ana@example.com", "Ana")
	change := func(token, current, password string, status int) response {
		t.Helper()
		return e.expect(request{method: "PUT", path: "/v1/users/me/password", token: token,
			body: `{"current_password":"` + current + `","password":"` + password + `"}`}, status)
	}

	if got := strings.Join(errorProperties(t, change(ana.Token, "wrong-password", "new-horse-battery", 422)), ","); got != "current_password" {
		t.Errorf("properties %q, want current_password", got)
	}
	if got := strings.Join(errorProperties(t, change(ana.Token, "correct-horse-battery", "short", 422)), ","); got != "password" {
		t.Errorf("properties %q, want password", got)
	}
	var next authToken
	change(ana.Token, "correct-horse-battery", "new-horse-battery", http.StatusOK).decode(t, &next)
	if next.Token == "" || next.Token == ana.Token || next.User.ID != ana.User.ID {
		t.Fatalf("new token: %+v", next)
	}
	e.expect(request{method: "POST", path: "/v1/users/tokens/refresh", token: ana.Token}, http.StatusUnauthorized)
	e.expect(request{method: "POST", path: "/v1/users/tokens/refresh", token: next.Token}, http.StatusOK)
	e.signIn(anaSignIn, http.StatusUnauthorized)
	e.signIn(`{"email":"ana@example.com","password":"new-horse-battery"}`, http.StatusOK)

	t.Run("wrong passwords count for the limit of sign in", func(t *testing.T) {
		// The limit is three. Two wrong passwords are above.
		change(next.Token, "wrong-password", "third-horse-battery", http.StatusUnprocessableEntity)
		res := change(next.Token, "new-horse-battery", "third-horse-battery", http.StatusTooManyRequests)
		if got := res.header.Get("Retry-After"); got == "" || got == "0" {
			t.Errorf("Retry-After %q", got)
		}
		if code := res.object(t)["code"]; code != "wrong_password_limit" {
			t.Errorf("code %v, want wrong_password_limit", code)
		}
		e.expect(request{method: "POST", path: "/v1/users/me/deletion", token: next.Token, body: `{"password":"new-horse-battery"}`}, http.StatusTooManyRequests)
		e.signIn(`{"email":"ana@example.com","password":"new-horse-battery"}`, http.StatusTooManyRequests)
	})

	t.Run("a password reset removes the wrong password count", func(t *testing.T) {
		e.expect(request{method: "POST", path: "/v1/users/password-resets", body: `{"email":"ana@example.com"}`}, http.StatusAccepted)
		e.expect(request{method: "PUT", path: "/v1/users/password",
			body: `{"token":"` + e.emailToken("ana@example.com") + `","password":"reset-horse-battery"}`}, http.StatusNoContent)
		e.signIn(`{"email":"ana@example.com","password":"reset-horse-battery"}`, http.StatusOK)
	})
}

func TestDeleteUser(t *testing.T) {
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	review := `{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"Good"}`
	e.createReview(ana.Token, restaurant, review)
	e.createReview(joe.Token, restaurant, review)
	list := e.expect(request{method: "POST", path: "/v1/lists", body: `{"name":"Date night"}`, token: ana.Token}, http.StatusCreated).object(t)["id"].(string)

	remove := func(password string, status int) response {
		t.Helper()
		return e.expect(request{method: "POST", path: "/v1/users/me/deletion", token: ana.Token, body: `{"password":"` + password + `"}`}, status)
	}
	if got := strings.Join(errorProperties(t, remove("wrong-password", 422)), ","); got != "password" {
		t.Errorf("properties %q, want password", got)
	}
	remove("correct-horse-battery", http.StatusNoContent)

	e.signIn(anaSignIn, http.StatusUnauthorized)
	e.expect(request{method: "GET", path: "/v1/restaurants", token: ana.Token}, http.StatusUnauthorized)
	e.expect(request{method: "GET", path: "/v1/lists/" + list}, http.StatusNotFound)
	e.expect(request{method: "GET", path: "/v1/reviewers/" + ana.User.ID}, http.StatusNotFound)
	got := e.expect(request{method: "GET", path: "/v1/restaurants/" + restaurant}, http.StatusOK).object(t)
	if got["creator"] != nil {
		t.Errorf("creator %v after the delete, want none", got["creator"])
	}
	reviews := e.expect(request{method: "GET", path: "/v1/restaurants/" + restaurant + "/reviews"}, http.StatusOK).object(t)["items"].([]any)
	if len(reviews) != 1 {
		t.Errorf("%d reviews after the delete, want 1", len(reviews))
	}

	t.Run("a user without approval can delete the account", func(t *testing.T) {
		eve := e.signUpPending("eve@example.com", "Eve")
		e.expect(request{method: "POST", path: "/v1/users/me/deletion", token: eve.Token, body: `{"password":"correct-horse-battery"}`}, http.StatusNoContent)
	})

	t.Run("the cookie is deleted with the user", func(t *testing.T) {
		e.signUp("kim@example.com", "Kim")
		res := e.expect(request{method: "POST", path: "/v1/users/tokens?cookie=true", body: `{"email":"kim@example.com","password":"correct-horse-battery"}`}, http.StatusOK)
		res = e.expect(request{method: "POST", path: "/v1/users/me/deletion", header: withCookie(cookieToken(t, res), nil),
			body: `{"password":"correct-horse-battery"}`}, http.StatusNoContent)
		if got := res.header.Get("Set-Cookie"); !strings.Contains(got, "Max-Age=0") {
			t.Errorf("Set-Cookie %q, want a deleted cookie", got)
		}
	})
}
