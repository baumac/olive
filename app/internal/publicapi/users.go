package publicapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	openapi_types "github.com/oapi-codegen/runtime/types"
	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/pocketbase/pocketbase/tools/types"
	"golang.org/x/crypto/bcrypt"
)

// SignUp creates a user. PocketBase checks that the email is unique.
func (s *Server) SignUp(_ context.Context, req reviewsapi.SignUpRequestObject) (reviewsapi.SignUpResponseObject, error) {
	b := req.Body
	users, err := s.app.FindCachedCollectionByNameOrId("users")
	if err != nil {
		return nil, err
	}
	user := core.NewRecord(users)
	user.SetEmail(string(b.Email))
	user.SetPassword(b.Password)
	user.Set("name", b.Name)
	if err := s.app.Save(user); err != nil {
		var invalid validation.Errors
		if !errors.As(err, &invalid) {
			return nil, err
		}
		if isNotUnique(invalid, core.FieldNameEmail) {
			return reviewsapi.SignUp409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict("A user with this email address exists.")}, nil
		}
		return reviewsapi.SignUp422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: unprocessable(fromPocketBase(invalid))}, nil
	}

	token, err := newAuthToken(user)
	if err != nil {
		return nil, err
	}
	return reviewsapi.SignUp201JSONResponse(token), nil
}

// SignIn checks a password hash also for an unknown email. A user at the
// wrong password limit gets 429 before the password check.
func (s *Server) SignIn(_ context.Context, req reviewsapi.SignInRequestObject) (reviewsapi.SignInResponseObject, error) {
	user, err := s.app.FindAuthRecordByEmail("users", string(req.Body.Email))
	if errors.Is(err, sql.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(req.Body.Password))
		return invalidCredentials(), nil
	}
	if err != nil {
		return nil, err
	}
	wait, window := s.signInFailures.allow(user.Id)
	if wait > 0 {
		return reviewsapi.SignIn429ApplicationProblemPlusJSONResponse{
			TooManyRequestsApplicationProblemPlusJSONResponse: tooManyRequests(wait, "Too many wrong passwords for this user. Try again later."),
		}, nil
	}
	if !user.ValidatePassword(req.Body.Password) {
		return invalidCredentials(), nil
	}
	s.signInFailures.giveBack(user.Id, window)

	token, err := newAuthToken(user)
	if err != nil {
		return nil, err
	}
	return reviewsapi.SignIn200JSONResponse(token), nil
}

// SignOut changes the token key of the user. PocketBase signs each auth
// token with this key. All tokens of the user then stop being valid.
func (s *Server) SignOut(ctx context.Context, _ reviewsapi.SignOutRequestObject) (reviewsapi.SignOutResponseObject, error) {
	user := stateFrom(ctx).user
	user.RefreshTokenKey()
	// A record that breaks a newer field rule must also be able to sign out.
	if err := s.app.SaveNoValidate(user); err != nil {
		return nil, err
	}
	return reviewsapi.SignOut204Response{}, nil
}

// RefreshToken returns a new token for the user of the bearer token.
func (s *Server) RefreshToken(ctx context.Context, _ reviewsapi.RefreshTokenRequestObject) (reviewsapi.RefreshTokenResponseObject, error) {
	st := stateFrom(ctx)
	// A superuser can make a token that cannot be refreshed, for example to
	// act as a user.
	claims, err := security.ParseUnverifiedJWT(st.token)
	if err != nil || claims[core.TokenClaimRefreshable] != true {
		return reviewsapi.RefreshToken401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: reviewsapi.UnauthorizedApplicationProblemPlusJSONResponse{
				Body:    newProblem(http.StatusUnauthorized, "The bearer token cannot be refreshed."),
				Headers: reviewsapi.UnauthorizedResponseHeaders{WWWAuthenticate: `Bearer error="invalid_token"`},
			},
		}, nil
	}

	token, err := newAuthToken(st.user)
	if err != nil {
		return nil, err
	}
	return reviewsapi.RefreshToken200JSONResponse(token), nil
}

func invalidCredentials() reviewsapi.SignIn401ApplicationProblemPlusJSONResponse {
	return reviewsapi.SignIn401ApplicationProblemPlusJSONResponse{
		InvalidCredentialsApplicationProblemPlusJSONResponse: reviewsapi.InvalidCredentialsApplicationProblemPlusJSONResponse{
			Body:    newProblem(http.StatusUnauthorized, "The email or password is not correct."),
			Headers: reviewsapi.InvalidCredentialsResponseHeaders{WWWAuthenticate: "Bearer"},
		},
	}
}

func newAuthToken(user *core.Record) (reviewsapi.AuthToken, error) {
	token, err := user.NewAuthToken()
	if err != nil {
		return reviewsapi.AuthToken{}, err
	}
	claims, err := security.ParseUnverifiedJWT(token)
	if err != nil {
		return reviewsapi.AuthToken{}, err
	}
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		return reviewsapi.AuthToken{}, errors.New("the auth token has no exp claim")
	}
	return reviewsapi.AuthToken{
		Token:   token,
		Expires: exp.UTC().Format(types.DefaultDateLayout),
		User: reviewsapi.User{
			Id:      user.Id,
			Email:   openapi_types.Email(user.Email()),
			Name:    user.GetString("name"),
			Created: user.GetDateTime("created").String(),
			Updated: user.GetDateTime("updated").String(),
		},
	}, nil
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
