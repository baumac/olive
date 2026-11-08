// Package publicapi implements the Public API in specs/api/public-openapi.yaml
// on a PocketBase app.
package publicapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/crypto/bcrypt"
)

const basePath = "/v1"

// Options are the settings of the Public API.
type Options struct {
	// SignInLimit is the number of sign up or sign in requests that one IP
	// address can send in SignInWindow. Each operation has its own count.
	SignInLimit  int
	SignInWindow time.Duration
}

// DefaultOptions are the settings of a production server.
var DefaultOptions = Options{SignInLimit: 10, SignInWindow: time.Minute}

// Server implements reviewsapi.StrictServerInterface.
type Server struct {
	app     core.App
	limiter *rateLimiter
	// dummyHash lets sign in check a password hash for an unknown email.
	// Sign in then takes the same time for all email addresses.
	dummyHash []byte
}

var _ reviewsapi.StrictServerInterface = (*Server)(nil)

// Register mounts the Public API on the router of app under /v1.
func Register(app core.App, opts Options) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		s, err := New(app, opts)
		if err != nil {
			return err
		}
		h := s.Handler()
		se.Router.BindFunc(closeUserAccountRoutes)
		se.Router.Any(basePath+"/{path...}", func(e *core.RequestEvent) error {
			ctx := context.WithValue(e.Request.Context(), clientIPKey{}, e.RealIP())
			// The loadAuthToken middleware of PocketBase already checked the
			// token. A nil record means that the check failed.
			ctx = context.WithValue(ctx, pocketBaseAuthKey{}, e.Auth)
			h.ServeHTTP(e.Response, e.Request.WithContext(ctx))
			return nil
		})
		return se.Next()
	})
}

// New makes a Server for app.
func New(app core.App, opts Options) (*Server, error) {
	cost := bcrypt.DefaultCost
	if users, err := app.FindCollectionByNameOrId("users"); err == nil {
		if f, ok := users.Fields.GetByName(core.FieldNamePassword).(*core.PasswordField); ok && f.Cost > 0 {
			cost = f.Cost
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("not a real password"), cost)
	if err != nil {
		return nil, err
	}
	return &Server{
		app:       app,
		limiter:   newRateLimiter(opts.SignInLimit, opts.SignInWindow),
		dummyHash: hash,
	}, nil
}

// Handler returns the HTTP handler of the Public API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	strict := reviewsapi.NewStrictHandlerWithOptions(s, nil, reviewsapi.StrictHTTPServerOptions{
		// The checks in prepare find all body errors that a client can cause.
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeProblem(w, newProblem(http.StatusBadRequest, "The request body is not valid."))
		},
		ResponseErrorHandlerFunc: s.serverError,
	})
	reviewsapi.HandlerWithOptions(strict, reviewsapi.StdHTTPServerOptions{
		BaseURL:    basePath,
		BaseRouter: mux,
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeProblem(w, newProblem(http.StatusBadRequest, paramErrorDetail(err)))
		},
	})
	mux.HandleFunc(basePath+"/", noRoute(mux))
	return s.prepare(mux)
}

func (s *Server) serverError(w http.ResponseWriter, r *http.Request, err error) {
	s.app.Logger().Error("Public API error", "method", r.Method, "path", r.URL.Path, "error", err)
	writeProblem(w, newProblem(http.StatusInternalServerError, "An unexpected error occurred."))
}

// noRoute returns 405 for a known path with a different method. Otherwise
// it returns 404.
func noRoute(mux *http.ServeMux) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var allowed []string
		for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete} {
			probe := r.Clone(r.Context())
			probe.Method = m
			if _, pattern := mux.Handler(probe); pattern != r.Pattern {
				allowed = append(allowed, m)
			}
		}
		if len(allowed) == 0 {
			writeProblem(w, newProblem(http.StatusNotFound, "The path does not exist."))
			return
		}
		w.Header().Set("Allow", strings.Join(allowed, ", "))
		writeProblem(w, newProblem(http.StatusMethodNotAllowed, "The path does not support the method."))
	}
}

type authMode int

const (
	authOptional authMode = iota
	authRequired
	authIgnored
)

// route has the checks that prepare does for one operation before the
// strict handler decodes the request.
type route struct {
	auth    authMode
	limited bool
	body    *bodySchema
	// query maps each allowed query parameter to its maximum number of
	// values. A nil map does not check the query.
	query map[string]int
}

var routes = map[string]route{
	"POST " + basePath + "/users":                                           {auth: authIgnored, limited: true, body: &userCreateSchema},
	"POST " + basePath + "/users/tokens":                                    {auth: authIgnored, limited: true, body: &credentialsSchema},
	"POST " + basePath + "/users/tokens/refresh":                            {auth: authRequired},
	"GET " + basePath + "/restaurants":                                      {query: restaurantQuery},
	"POST " + basePath + "/restaurants":                                     {auth: authRequired, body: &restaurantCreateSchema},
	"GET " + basePath + "/restaurants/{restaurantId}":                       {},
	"PATCH " + basePath + "/restaurants/{restaurantId}":                     {auth: authRequired, body: &restaurantPatchSchema},
	"DELETE " + basePath + "/restaurants/{restaurantId}":                    {auth: authRequired},
	"GET " + basePath + "/reviews":                                          {query: allReviewQuery},
	"GET " + basePath + "/restaurants/{restaurantId}/reviews":               {query: reviewQuery},
	"POST " + basePath + "/restaurants/{restaurantId}/reviews":              {auth: authRequired, body: &reviewCreateSchema},
	"GET " + basePath + "/restaurants/{restaurantId}/reviews/{reviewId}":    {},
	"PATCH " + basePath + "/restaurants/{restaurantId}/reviews/{reviewId}":  {auth: authRequired, body: &reviewPatchSchema},
	"DELETE " + basePath + "/restaurants/{restaurantId}/reviews/{reviewId}": {auth: authRequired},
}

type clientIPKey struct{}

type pocketBaseAuthKey struct{}

type stateKey struct{}

type state struct {
	user  *core.Record
	token string
	props map[string]json.RawMessage
}

func stateFrom(ctx context.Context) *state {
	if st, ok := ctx.Value(stateKey{}).(*state); ok {
		return st
	}
	return &state{}
}

// prepare does the checks that come before the operation handler. Decision
// 00004 lists them in order.
func (s *Server) prepare(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A panic gets Problem Details. The PocketBase recovery would answer
		// with its own JSON format.
		tw := &trackingWriter{ResponseWriter: w}
		w = tw
		defer func() {
			if v := recover(); v != nil {
				if tw.wrote {
					// A second response would mix with the first one. The
					// connection closes instead.
					s.app.Logger().Error("Public API panic after the response started", "error", v)
					panic(http.ErrAbortHandler)
				}
				s.serverError(w, r, fmt.Errorf("panic: %v", v))
			}
		}()
		_, pattern := mux.Handler(r)
		rt, ok := routes[pattern]
		if !ok {
			if pattern != basePath+"/" {
				// An operation without a routes entry would skip all checks.
				s.serverError(w, r, errors.New("no routes entry for "+pattern))
				return
			}
			mux.ServeHTTP(w, r)
			return
		}
		st := &state{}

		if rt.limited {
			if wait := s.limiter.allow(pattern + " " + clientIP(r)); wait > 0 {
				w.Header().Set("Retry-After", retryAfter(wait))
				writeProblem(w, newProblem(http.StatusTooManyRequests, "Too many requests. Try again later."))
				return
			}
		}

		if rt.auth != authIgnored {
			user, token, ok := s.authenticate(r)
			if !ok {
				w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
				writeProblem(w, newProblem(http.StatusUnauthorized, "The bearer token is not valid or is expired."))
				return
			}
			if user == nil && rt.auth == authRequired {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeProblem(w, newProblem(http.StatusUnauthorized, "The operation requires a bearer token."))
				return
			}
			st.user, st.token = user, token
		}

		if rt.query != nil {
			if detail := checkQueryNames(r.URL.Query(), rt.query); detail != "" {
				writeProblem(w, newProblem(http.StatusBadRequest, detail))
				return
			}
		}

		if rt.body != nil {
			props, p := readBody(r, rt.body)
			if p != nil {
				writeProblem(w, *p)
				return
			}
			st.props = props
		}

		next := r.WithContext(context.WithValue(r.Context(), stateKey{}, st))
		mux.ServeHTTP(w, next)
	})
}

// authenticate returns the user of the bearer token. It returns a nil user
// and true when the request has no Authorization header.
func (s *Server) authenticate(r *http.Request) (*core.Record, string, bool) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return nil, "", true
	}
	scheme, token, found := strings.Cut(header, " ")
	token = strings.TrimSpace(token)
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return nil, "", false
	}
	user, checked := r.Context().Value(pocketBaseAuthKey{}).(*core.Record)
	if !checked {
		var err error
		if user, err = s.app.FindAuthRecordByToken(token, core.TokenTypeAuth); err != nil {
			return nil, "", false
		}
	}
	if user == nil || user.Collection().Name != "users" {
		return nil, "", false
	}
	return user, token, true
}

func clientIP(r *http.Request) string {
	if ip, ok := r.Context().Value(clientIPKey{}).(string); ok && ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func checkQueryNames(query map[string][]string, allowed map[string]int) string {
	for name, values := range query {
		maxValues, ok := allowed[name]
		if !ok {
			return "The query parameter " + name + " is not supported."
		}
		if len(values) > maxValues {
			return fmt.Sprintf("The query parameter %s can occur at most %d times.", name, maxValues)
		}
	}
	return ""
}

// paramErrorDetail names the parameter of a binder error. The binder text
// names Go parser details. These are not for people.
func paramErrorDetail(err error) string {
	var format *reviewsapi.InvalidParamFormatError
	if errors.As(err, &format) {
		return "The parameter " + format.ParamName + " is not valid."
	}
	var many *reviewsapi.TooManyValuesForParamError
	if errors.As(err, &many) {
		return "The query parameter " + many.ParamName + " can occur only one time."
	}
	return "The request has a parameter that is not valid."
}

// trackingWriter records if the response has started.
type trackingWriter struct {
	http.ResponseWriter
	wrote bool
}

func (t *trackingWriter) WriteHeader(status int) {
	t.wrote = true
	t.ResponseWriter.WriteHeader(status)
}

func (t *trackingWriter) Write(b []byte) (int, error) {
	t.wrote = true
	return t.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the PocketBase writer.
func (t *trackingWriter) Unwrap() http.ResponseWriter {
	return t.ResponseWriter
}
