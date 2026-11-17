package publicapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	openapi_types "github.com/oapi-codegen/runtime/types"
	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
	"golang.org/x/crypto/bcrypt"
)

// SignUp gives the same response for a new and a used email address. An
// email tells the person the difference. See decision 00037.
func (s *Server) SignUp(_ context.Context, req reviewsapi.SignUpRequestObject) (reviewsapi.SignUpResponseObject, error) {
	b := req.Body
	users, err := s.app.FindCachedCollectionByNameOrId("users")
	if err != nil {
		return nil, err
	}
	existing, err := s.app.FindAuthRecordByEmail(users, string(b.Email))
	if errors.Is(err, sql.ErrNoRows) {
		existing = nil
	} else if err != nil {
		return nil, err
	}
	confirmed := existing != nil && existing.Verified()
	// The user changes only together with a new link. A person who does not
	// own the address then cannot make the last link invalid without end.
	if existing != nil && !confirmed && s.post.atLimit(kindConfirmation, existing.Email()) {
		return reviewsapi.SignUp202Response{}, nil
	}
	// A second sign up replaces a user without confirmation. A person who
	// does not own the address then cannot set its password.
	user := existing
	if existing == nil || confirmed {
		user = core.NewRecord(users)
	}
	user.SetEmail(string(b.Email))
	user.SetPassword(b.Password)
	user.Set("name", b.Name)
	if confirmed {
		// The checks and their time are the same as for a new user.
		invalid := validationErrors(s.app.Validate(user))
		if isNotUnique(invalid, core.FieldNameEmail) {
			delete(invalid, core.FieldNameEmail)
		}
		if len(invalid) > 0 {
			return reviewsapi.SignUp422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: unprocessable(fromPocketBase(invalid))}, nil
		}
		s.post.sendAccountExists(existing)
		return reviewsapi.SignUp202Response{}, nil
	}
	if err := s.app.Save(user); err != nil {
		invalid := validationErrors(err)
		if invalid == nil {
			return nil, err
		}
		// Two sign ups at the same time can both find no user.
		if isNotUnique(invalid, core.FieldNameEmail) {
			return reviewsapi.SignUp202Response{}, nil
		}
		return reviewsapi.SignUp422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: unprocessable(fromPocketBase(invalid))}, nil
	}
	if err := s.post.sendConfirmation(user); err != nil {
		return nil, err
	}
	return reviewsapi.SignUp202Response{}, nil
}

// ConfirmEmail checks the password and signs the user of a confirmation
// token in. A sign up of another person can set that password. See decision
// 00039.
func (s *Server) ConfirmEmail(ctx context.Context, req reviewsapi.ConfirmEmailRequestObject) (reviewsapi.ConfirmEmailResponseObject, error) {
	user := s.userOfEmailToken(req.Body.Token, core.TokenTypeVerification)
	if user == nil {
		return reviewsapi.ConfirmEmail422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: invalidEmailToken()}, nil
	}
	// Only a request with a valid token adds to this key. A person without
	// the link then cannot block the confirmation.
	wait, wrong := s.checkPassword("confirm:"+user.Id, user, req.Body.Password, "password")
	if wait > 0 {
		return reviewsapi.ConfirmEmail429ApplicationProblemPlusJSONResponse{TooManyRequestsApplicationProblemPlusJSONResponse: tooManyPasswords(wait)}, nil
	}
	if wrong != nil {
		return reviewsapi.ConfirmEmail422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: *wrong}, nil
	}
	// A new token key makes the token of the link invalid.
	user.RefreshTokenKey()
	if err := s.confirm(user); err != nil {
		return nil, err
	}
	auth, cookie, err := s.issueToken(stateFrom(ctx), user, req.Params.Cookie)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ConfirmEmail200JSONResponse{Body: auth, Headers: reviewsapi.ConfirmEmail200ResponseHeaders{SetCookie: cookie}}, nil
}

// confirm saves user as confirmed. The superusers get an email about a new
// user that waits for approval.
func (s *Server) confirm(user *core.Record) error {
	first := !user.Verified()
	user.SetVerified(true)
	if err := s.app.Save(user); err != nil {
		return err
	}
	if !first || user.GetBool("approved") {
		return nil
	}
	if err := s.post.sendWaitsForApproval(user); err != nil {
		s.app.Logger().Error("The superusers got no email about a new user", "error", err)
	}
	return nil
}

// userOfEmailToken returns nil for a token that is not valid, is expired, or
// is not for the current email address of its user.
func (s *Server) userOfEmailToken(token, tokenType string) *core.Record {
	user, err := s.app.FindAuthRecordByToken(token, tokenType)
	if err != nil || user.Collection().Name != "users" {
		return nil
	}
	claims, err := security.ParseUnverifiedJWT(token)
	if err != nil || claims[core.TokenClaimEmail] != user.Email() {
		return nil
	}
	return user
}

func invalidEmailToken() reviewsapi.UnprocessableContentApplicationProblemPlusJSONResponse {
	var errs fieldErrors
	errs.add("token", "is not valid or is expired")
	return unprocessable(errs)
}

// RequestPasswordReset gives the same response for all email addresses.
func (s *Server) RequestPasswordReset(_ context.Context, req reviewsapi.RequestPasswordResetRequestObject) (reviewsapi.RequestPasswordResetResponseObject, error) {
	user, err := s.app.FindAuthRecordByEmail("users", string(req.Body.Email))
	if errors.Is(err, sql.ErrNoRows) {
		return reviewsapi.RequestPasswordReset202Response{}, nil
	}
	if err != nil {
		return nil, err
	}
	if err := s.post.sendPasswordReset(user); err != nil {
		return nil, err
	}
	return reviewsapi.RequestPasswordReset202Response{}, nil
}

// ResetPassword also confirms the email address. The token came from an
// email to this address.
func (s *Server) ResetPassword(_ context.Context, req reviewsapi.ResetPasswordRequestObject) (reviewsapi.ResetPasswordResponseObject, error) {
	user := s.userOfEmailToken(req.Body.Token, core.TokenTypePasswordReset)
	if user == nil {
		return reviewsapi.ResetPassword422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: invalidEmailToken()}, nil
	}
	// SetPassword changes the token key. The reset token and all auth
	// tokens of the user then stop being valid.
	user.SetPassword(req.Body.Password)
	if err := s.confirm(user); err != nil {
		invalid := validationErrors(err)
		if invalid == nil {
			return nil, err
		}
		return reviewsapi.ResetPassword422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: unprocessable(fromPocketBase(invalid))}, nil
	}
	// The wrong passwords from before the reset do not block the new one.
	s.signInFailures.forget(passwordLimitKey(user.Email()))
	return reviewsapi.ResetPassword204Response{}, nil
}

// GetCurrentUser returns the user of the token.
func (s *Server) GetCurrentUser(ctx context.Context, _ reviewsapi.GetCurrentUserRequestObject) (reviewsapi.GetCurrentUserResponseObject, error) {
	return reviewsapi.GetCurrentUser200JSONResponse(apiUser(stateFrom(ctx).user)), nil
}

// UpdateUser changes the name of the user of the token.
func (s *Server) UpdateUser(ctx context.Context, req reviewsapi.UpdateUserRequestObject) (reviewsapi.UpdateUserResponseObject, error) {
	user := stateFrom(ctx).user
	if req.Body.Name != nil {
		user.Set("name", *req.Body.Name)
		if err := s.app.Save(user); err != nil {
			invalid := validationErrors(err)
			if invalid == nil {
				return nil, err
			}
			return reviewsapi.UpdateUser422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: unprocessable(fromPocketBase(invalid))}, nil
		}
	}
	return reviewsapi.UpdateUser200JSONResponse(apiUser(user)), nil
}

// ChangePassword makes all tokens of the user invalid and returns a new one.
func (s *Server) ChangePassword(ctx context.Context, req reviewsapi.ChangePasswordRequestObject) (reviewsapi.ChangePasswordResponseObject, error) {
	st := stateFrom(ctx)
	user := st.user
	wait, wrong := s.checkPassword(passwordLimitKey(user.Email()), user, req.Body.CurrentPassword, "current_password")
	if wait > 0 {
		return reviewsapi.ChangePassword429ApplicationProblemPlusJSONResponse{TooManyRequestsApplicationProblemPlusJSONResponse: tooManyPasswords(wait)}, nil
	}
	if wrong != nil {
		return reviewsapi.ChangePassword422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: *wrong}, nil
	}
	user.SetPassword(req.Body.Password)
	if err := s.app.Save(user); err != nil {
		invalid := validationErrors(err)
		if invalid == nil {
			return nil, err
		}
		return reviewsapi.ChangePassword422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: unprocessable(fromPocketBase(invalid))}, nil
	}
	auth, cookie, err := s.issueToken(st, user, req.Params.Cookie)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ChangePassword200JSONResponse{Body: auth, Headers: reviewsapi.ChangePassword200ResponseHeaders{SetCookie: cookie}}, nil
}

// DeleteUser deletes the user of the token. The relations of the
// collections delete the reviews, the photos, and the lists of the user.
func (s *Server) DeleteUser(ctx context.Context, req reviewsapi.DeleteUserRequestObject) (reviewsapi.DeleteUserResponseObject, error) {
	st := stateFrom(ctx)
	wait, wrong := s.checkPassword(passwordLimitKey(st.user.Email()), st.user, req.Body.Password, "password")
	if wait > 0 {
		return reviewsapi.DeleteUser429ApplicationProblemPlusJSONResponse{TooManyRequestsApplicationProblemPlusJSONResponse: tooManyPasswords(wait)}, nil
	}
	if wrong != nil {
		return reviewsapi.DeleteUser422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: *wrong}, nil
	}
	if err := s.app.Delete(st.user); err != nil {
		return nil, err
	}
	var res reviewsapi.DeleteUser204Response
	if st.fromCookie {
		deleted := s.deleteSessionCookie(st.request)
		res.Headers.SetCookie = &deleted
	}
	return res, nil
}

// passwordLimitKey is the key of the wrong passwords of an email address.
// The lookup of a user by email ignores the case of ASCII letters.
func passwordLimitKey(email string) string {
	return strings.ToLower(email)
}

// checkPassword counts a wrong password for key. A stolen token then cannot
// try many passwords. It returns the wait time of a key at the limit, or the
// 422 of a wrong password for property.
func (s *Server) checkPassword(key string, user *core.Record, password, property string) (time.Duration, *reviewsapi.UnprocessableContentApplicationProblemPlusJSONResponse) {
	wait, window := s.signInFailures.allow(key)
	if wait > 0 {
		return wait, nil
	}
	if !user.ValidatePassword(password) {
		var errs fieldErrors
		errs.add(property, "is not correct")
		res := unprocessable(errs)
		return 0, &res
	}
	s.signInFailures.giveBack(key, window)
	return 0, nil
}

func tooManyPasswords(wait time.Duration) reviewsapi.TooManyRequestsApplicationProblemPlusJSONResponse {
	res := tooManyRequests(wait, "Too many wrong passwords. Try again later.")
	code := reviewsapi.ProblemCodeWrongPasswordLimit
	res.Body.Code = &code
	return res
}

// SignIn gives the same responses for an unknown email and for a wrong
// password. It checks a password hash and counts the wrong password for both.
// An email at the wrong password limit gets 429 before the user lookup.
func (s *Server) SignIn(ctx context.Context, req reviewsapi.SignInRequestObject) (reviewsapi.SignInResponseObject, error) {
	key := passwordLimitKey(string(req.Body.Email))
	wait, window := s.signInFailures.allow(key)
	if wait > 0 {
		return reviewsapi.SignIn429ApplicationProblemPlusJSONResponse{TooManyRequestsApplicationProblemPlusJSONResponse: tooManyPasswords(wait)}, nil
	}
	user, err := s.app.FindAuthRecordByEmail("users", string(req.Body.Email))
	if errors.Is(err, sql.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(req.Body.Password))
		return invalidCredentials(), nil
	}
	if err != nil {
		return nil, err
	}
	if !user.ValidatePassword(req.Body.Password) {
		return invalidCredentials(), nil
	}
	s.signInFailures.giveBack(key, window)
	if !user.Verified() {
		if err := s.post.sendConfirmation(user); err != nil {
			return nil, err
		}
		p := newProblem(http.StatusForbidden, "The email address is not confirmed. Use the link in the confirmation email.")
		code := reviewsapi.ProblemCodeEmailNotConfirmed
		p.Code = &code
		return reviewsapi.SignIn403ApplicationProblemPlusJSONResponse{EmailNotConfirmedApplicationProblemPlusJSONResponse: reviewsapi.EmailNotConfirmedApplicationProblemPlusJSONResponse(p)}, nil
	}

	auth, cookie, err := s.issueToken(stateFrom(ctx), user, req.Params.Cookie)
	if err != nil {
		return nil, err
	}
	return reviewsapi.SignIn200JSONResponse{Body: auth, Headers: reviewsapi.SignIn200ResponseHeaders{SetCookie: cookie}}, nil
}

// SignOut changes the token key of the user. PocketBase signs each auth
// token with this key. All tokens of the user then stop being valid.
func (s *Server) SignOut(ctx context.Context, _ reviewsapi.SignOutRequestObject) (reviewsapi.SignOutResponseObject, error) {
	st := stateFrom(ctx)
	user := st.user
	user.RefreshTokenKey()
	// A record that breaks a newer field rule must also be able to sign out.
	if err := s.app.SaveNoValidate(user); err != nil {
		return nil, err
	}
	var res reviewsapi.SignOut204Response
	if st.fromCookie {
		deleted := s.deleteSessionCookie(st.request)
		res.Headers.SetCookie = &deleted
	}
	return res, nil
}

// RefreshToken returns a new token for the user of the bearer token.
func (s *Server) RefreshToken(ctx context.Context, req reviewsapi.RefreshTokenRequestObject) (reviewsapi.RefreshTokenResponseObject, error) {
	st := stateFrom(ctx)
	// A superuser can make a token that cannot be refreshed, for example to
	// act as a user.
	claims, err := security.ParseUnverifiedJWT(st.token)
	if err != nil || claims[core.TokenClaimRefreshable] != true {
		headers := reviewsapi.UnauthorizedResponseHeaders{WWWAuthenticate: `Bearer error="invalid_token"`}
		if st.fromCookie {
			deleted := s.deleteSessionCookie(st.request)
			headers.SetCookie = &deleted
		}
		return reviewsapi.RefreshToken401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: reviewsapi.UnauthorizedApplicationProblemPlusJSONResponse{
				Body:    newProblem(http.StatusUnauthorized, "The bearer token cannot be refreshed."),
				Headers: headers,
			},
		}, nil
	}

	auth, cookie, err := s.issueToken(st, st.user, req.Params.Cookie)
	if err != nil {
		return nil, err
	}
	return reviewsapi.RefreshToken200JSONResponse{Body: auth, Headers: reviewsapi.RefreshToken200ResponseHeaders{SetCookie: cookie}}, nil
}

func invalidCredentials() reviewsapi.SignIn401ApplicationProblemPlusJSONResponse {
	return reviewsapi.SignIn401ApplicationProblemPlusJSONResponse{
		InvalidCredentialsApplicationProblemPlusJSONResponse: reviewsapi.InvalidCredentialsApplicationProblemPlusJSONResponse{
			Body:    newProblem(http.StatusUnauthorized, "The email or password is not correct."),
			Headers: reviewsapi.InvalidCredentialsResponseHeaders{WWWAuthenticate: "Bearer"},
		},
	}
}

// issueToken makes a token for user. With the cookie, the token is in the
// returned Set-Cookie value and not in the body.
func (s *Server) issueToken(st *state, user *core.Record, cookieParam *bool) (reviewsapi.AuthToken, *string, error) {
	auth, expires, err := newAuthToken(user)
	if err != nil || !wantsCookie(st, cookieParam) {
		return auth, nil, err
	}
	cookie := s.sessionCookie(st.request, *auth.Token, expires)
	auth.Token = nil
	return auth, &cookie, nil
}

func newAuthToken(user *core.Record) (reviewsapi.AuthToken, time.Time, error) {
	token, err := user.NewAuthToken()
	if err != nil {
		return reviewsapi.AuthToken{}, time.Time{}, err
	}
	claims, err := security.ParseUnverifiedJWT(token)
	if err != nil {
		return reviewsapi.AuthToken{}, time.Time{}, err
	}
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		return reviewsapi.AuthToken{}, time.Time{}, errors.New("the auth token has no exp claim")
	}
	return reviewsapi.AuthToken{
		Token:   &token,
		Expires: exp.UTC().Format(timestampLayout),
		User:    apiUser(user),
	}, exp.Time, nil
}

func apiUser(user *core.Record) reviewsapi.User {
	return reviewsapi.User{
		Id:       user.Id,
		Email:    openapi_types.Email(user.Email()),
		Name:     user.GetString("name"),
		Approved: user.GetBool("approved"),
		Created:  apiTimestamp(user.GetDateTime("created")),
		Updated:  apiTimestamp(user.GetDateTime("updated")),
	}
}

// validationErrors returns nil for an error that is not a validation error.
func validationErrors(err error) validation.Errors {
	var invalid validation.Errors
	if !errors.As(err, &invalid) {
		return nil
	}
	return invalid
}

func isNotUnique(invalid validation.Errors, field string) bool {
	var e validation.Error
	return errors.As(invalid[field], &e) && e.Code() == "validation_not_unique"
}

// fromPocketBase changes PocketBase validation errors to Public API errors.
// The Public API checks catch most errors before PocketBase does.
func fromPocketBase(invalid validation.Errors) fieldErrors {
	var errs fieldErrors
	for field, err := range invalid {
		errs.add(field, err.Error())
	}
	return errs
}
