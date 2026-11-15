// Package publicapi implements the Public API in specs/api/public-openapi.yaml
// on a PocketBase app.
package publicapi

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/getkin/kin-openapi/openapi3filter"
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
	// SignInFailureLimit is the number of wrong passwords that one user can
	// get in SignInFailureWindow. Zero uses 10 in 15 minutes.
	SignInFailureLimit  int
	SignInFailureWindow time.Duration
	// PhotoUploadLimit is the number of photos that one user can add in
	// PhotoUploadWindow.
	PhotoUploadLimit  int
	PhotoUploadWindow time.Duration
	// CreateLimit is the number of restaurants, reviews, or lists that one
	// user can create in WriteWindow. Each operation has its own count.
	// Zero uses 30.
	CreateLimit int
	// EditLimit is the number of updates, deletes, or list changes of one
	// operation that one user can send in WriteWindow. Zero uses 300.
	EditLimit int
	// WriteWindow is the window of CreateLimit and EditLimit. Zero uses one
	// hour.
	WriteWindow time.Duration
	// PlacesURL is the base URL of the Photon geocoder of the place search.
	// An empty value turns the place search off.
	PlacesURL string
	// PlaceSearchLimit is the number of place searches that one user can
	// send in PlaceSearchWindow.
	PlaceSearchLimit  int
	PlaceSearchWindow time.Duration
	// PlaceUpstreamLimit is the number of requests that all users together
	// can cause at the geocoder in PlaceSearchWindow.
	PlaceUpstreamLimit int
	// PhotoMemoryLimit is the number of bytes that all photo uploads can
	// hold in memory. Zero uses 64 MB.
	PhotoMemoryLimit int64
	// SkipRequestValidation turns off the spec validation of the parameters
	// and the body values. Invalid values then reach the handlers.
	SkipRequestValidation bool
	// SkipResponseValidation turns off the spec validation of the
	// responses. The responses then go to the client without a buffer.
	SkipResponseValidation bool
}

// DefaultOptions are the settings of a production server.
var DefaultOptions = Options{
	SignInLimit: 10, SignInWindow: time.Minute,
	SignInFailureLimit: 10, SignInFailureWindow: 15 * time.Minute,
	PhotoUploadLimit: 30, PhotoUploadWindow: time.Hour,
	CreateLimit: 30, EditLimit: 300, WriteWindow: time.Hour,
	PlacesURL:        "https://photon.komoot.io",
	PlaceSearchLimit: 30, PlaceSearchWindow: time.Minute, PlaceUpstreamLimit: 60,
}

// rateLimitNames are the values of the name in the x-rate-limit extension.
var rateLimitNames = []string{"sign-in", "photo-upload", "place-search", "create", "edit"}

// failureRefunds are the limits that give back a request with an error
// status. A failed write changes no data.
var failureRefunds = map[string]bool{"create": true, "edit": true}

// Server implements reviewsapi.StrictServerInterface.
type Server struct {
	app      core.App
	spec     *apiSpec
	opts     Options
	limiters map[string]*rateLimiter
	// signInFailures counts the wrong passwords of each user ID. It has no
	// window cap, so other keys cannot remove the window of a user.
	signInFailures *rateLimiter
	// decodes holds one token for each photo upload that decodes its image.
	decodes     chan struct{}
	photoMemory *photoMemory
	// places is nil when the place search is off.
	places *placeSearch
	// dummyHash lets sign in check a password hash for an unknown email.
	// Sign in then takes the same time for all email addresses.
	dummyHash []byte
}

var _ reviewsapi.StrictServerInterface = (*Server)(nil)

// Register mounts the Public API on the router of app under /v1.
func Register(app core.App, opts Options) {
	guardReviewOverall(app)
	registerFavorites(app)
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
		}).Bind(limitBody())
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
	spec, err := loadSpec()
	if err != nil {
		return nil, err
	}
	switch {
	case opts.SkipRequestValidation && !opts.SkipResponseValidation:
		// A saved invalid value makes each response with it fail.
		app.Logger().Warn("The spec validation of Public API requests is off. Responses with invalid saved values get 500.")
	case opts.SkipRequestValidation:
		app.Logger().Warn("The spec validation of Public API requests and responses is off")
	case opts.SkipResponseValidation:
		app.Logger().Warn("The spec validation of Public API responses is off")
	}
	signInFailures := newRateLimiter(
		cmp.Or(opts.SignInFailureLimit, DefaultOptions.SignInFailureLimit),
		cmp.Or(opts.SignInFailureWindow, DefaultOptions.SignInFailureWindow),
	)
	signInFailures.maxWindows = 0
	writeWindow := cmp.Or(opts.WriteWindow, DefaultOptions.WriteWindow)
	var places *placeSearch
	if opts.PlacesURL != "" {
		// The geocoder can use the URL to contact the operator of the app.
		userAgent := func() string { return "reviews-app (" + app.Settings().Meta.AppURL + ")" }
		places = newPlaceSearch(opts.PlacesURL, userAgent,
			cmp.Or(opts.PlaceUpstreamLimit, DefaultOptions.PlaceUpstreamLimit),
			cmp.Or(opts.PlaceSearchWindow, DefaultOptions.PlaceSearchWindow))
	}
	return &Server{
		app:  app,
		spec: spec,
		opts: opts,
		limiters: map[string]*rateLimiter{
			"sign-in":      newRateLimiter(opts.SignInLimit, opts.SignInWindow),
			"photo-upload": newRateLimiter(opts.PhotoUploadLimit, opts.PhotoUploadWindow),
			"place-search": newRateLimiter(
				cmp.Or(opts.PlaceSearchLimit, DefaultOptions.PlaceSearchLimit),
				cmp.Or(opts.PlaceSearchWindow, DefaultOptions.PlaceSearchWindow)),
			"create": newRateLimiter(cmp.Or(opts.CreateLimit, DefaultOptions.CreateLimit), writeWindow),
			"edit":   newRateLimiter(cmp.Or(opts.EditLimit, DefaultOptions.EditLimit), writeWindow),
		},
		signInFailures: signInFailures,
		decodes:        make(chan struct{}, maxParallelDecodes),
		photoMemory:    &photoMemory{limit: cmp.Or(opts.PhotoMemoryLimit, defaultPhotoMemoryLimit)},
		places:         places,
		dummyHash:      hash,
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
		for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
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

// notApproved is the detail of the 403 for a user without approval.
const notApproved = "The account is not approved. A superuser must approve it."

type clientIPKey struct{}

type pocketBaseAuthKey struct{}

type stateKey struct{}

type state struct {
	user  *core.Record
	token string
	// fromCookie is true when the token came from the session cookie.
	fromCookie bool
	// sameOrigin is false for an unsafe cross-origin request of a browser.
	sameOrigin bool
	props      map[string]json.RawMessage
	// request lets a file response read the Range and If-None-Match headers.
	request *http.Request
	// limited is the rate limit count of the request. It is nil without a
	// limit.
	limited *limitedRequest
}

type limitedRequest struct {
	limiter *rateLimiter
	key     string
	start   time.Time
}

// giveBackLimit removes the request from its rate limit count. A request
// that the server cannot do then does not use the limit of the client.
func (st *state) giveBackLimit() {
	if st.limited != nil {
		st.limited.limiter.giveBack(st.limited.key, st.limited.start)
	}
}

func stateFrom(ctx context.Context) *state {
	if st, ok := ctx.Value(stateKey{}).(*state); ok {
		return st
	}
	return &state{}
}

// prepare does the checks that come before the operation handler. Decision
// 00004 lists them in order. It also checks each response against the spec.
func (s *Server) prepare(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A panic gets Problem Details. The PocketBase recovery would answer
		// with its own JSON format.
		tw := &trackingWriter{ResponseWriter: w}
		defer func() {
			if v := recover(); v != nil {
				if tw.wrote {
					// A second response would mix with the first one. The
					// connection closes instead.
					s.app.Logger().Error("Public API panic after the response started", "error", v)
					panic(http.ErrAbortHandler)
				}
				s.serverError(tw, r, fmt.Errorf("panic: %v", v))
			}
		}()
		_, pattern := mux.Handler(r)
		op, ok := s.spec.operations[pattern]
		if !ok {
			if pattern != basePath+"/" {
				// An operation without a spec operation would skip all checks.
				s.serverError(tw, r, errors.New("no spec operation for "+pattern))
				return
			}
			mux.ServeHTTP(tw, r)
			return
		}
		in, err := op.input(r)
		if err != nil {
			writeProblem(tw, newProblem(http.StatusNotFound, "The path does not exist."))
			return
		}
		if s.opts.SkipResponseValidation {
			s.serveOperation(tw, mux, op, in)
			return
		}
		if op.fileResponse {
			hw := &headerCheckWriter{ResponseWriter: tw, check: func(status int, header http.Header) error {
				return checkResponseHeaders(r.Context(), in, status, header)
			}}
			s.serveOperation(hw, mux, op, in)
			hw.WriteHeader(http.StatusOK)
			if hw.err != nil {
				s.serverError(tw, r, errors.New("the response does not match the spec: "+responseProblem(hw.err)))
			}
			return
		}

		bw := newBufferedWriter(tw)
		s.serveOperation(bw, mux, op, in)
		// The handler can read the body. The response check does not need it.
		r.Body = http.NoBody
		if err := checkResponse(r.Context(), in, bw.Status(), bw.Header(), bw.body.Bytes()); err != nil {
			s.serverError(tw, r, errors.New("the response does not match the spec: "+responseProblem(err)))
			return
		}
		if err := bw.Flush(); err != nil {
			s.app.Logger().Debug("Public API response write failed", "error", err)
		}
	})
}

// serveOperation does the request checks of op and then runs the handler.
func (s *Server) serveOperation(w http.ResponseWriter, mux *http.ServeMux, op *operation, in *openapi3filter.RequestValidationInput) {
	r := in.Request
	st := &state{request: r, sameOrigin: sameOrigin(r)}
	if op.limit != nil && op.limit.key == "ip" && !s.allow(w, op, st, clientKey(clientIP(r))) {
		return
	}

	if op.auth != authIgnored {
		cookie := cookieToken(st)
		user, token, ok := s.authenticate(r, cookie)
		if !ok {
			if cookie != "" {
				w.Header().Set("Set-Cookie", s.deleteSessionCookie(r))
			}
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			writeProblem(w, newProblem(http.StatusUnauthorized, "The bearer token is not valid or is expired."))
			return
		}
		if user == nil && op.auth == authRequired {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeProblem(w, newProblem(http.StatusUnauthorized, "The operation requires a bearer token."))
			return
		}
		st.user, st.token, st.fromCookie = user, token, cookie != ""
	}
	if op.approval && !st.user.GetBool("approved") {
		p := newProblem(http.StatusForbidden, notApproved)
		code := reviewsapi.NotApproved
		p.Code = &code
		writeProblem(w, p)
		return
	}
	// The key is the user. The token check therefore runs first.
	if op.limit != nil && op.limit.key == "user" && !s.allow(w, op, st, st.user.Id) {
		return
	}
	if st.limited != nil && failureRefunds[op.limit.name] {
		sw := &statusWriter{ResponseWriter: w}
		w = sw
		defer func() {
			if sw.status >= http.StatusBadRequest {
				st.giveBackLimit()
			}
		}()
	}

	if detail := op.checkQuery(r); detail != "" {
		writeProblem(w, newProblem(http.StatusBadRequest, detail))
		return
	}
	validate := !s.opts.SkipRequestValidation
	if validate {
		if detail := op.checkParams(r.Context(), in); detail != "" {
			writeProblem(w, newProblem(http.StatusBadRequest, detail))
			return
		}
	}

	if op.route.Operation.RequestBody != nil && !op.fileBody {
		props, p := readBody(r.Context(), in, validate)
		if p != nil {
			writeProblem(w, *p)
			return
		}
		st.props = props
	}

	mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), stateKey{}, st)))
}

// allow counts the request of key in st. It writes 429 when key reaches the
// limit.
func (s *Server) allow(w http.ResponseWriter, op *operation, st *state, key string) bool {
	limiter := s.limiters[op.limit.name]
	key = op.pattern + " " + key
	wait, start := limiter.allow(key)
	if wait == 0 {
		st.limited = &limitedRequest{limiter: limiter, key: key, start: start}
		return true
	}
	res := tooManyRequests(wait, "Too many requests. Try again later.")
	w.Header().Set("Retry-After", strconv.Itoa(res.Headers.RetryAfter))
	writeProblem(w, res.Body)
	return false
}

// authenticate returns the user of the bearer token or of the cookie token.
// It returns a nil user and true when the request has neither.
func (s *Server) authenticate(r *http.Request, cookieToken string) (*core.Record, string, bool) {
	header := r.Header.Get("Authorization")
	if header == "" {
		if cookieToken == "" {
			return nil, "", true
		}
		// PocketBase reads only the Authorization header.
		user, err := s.app.FindAuthRecordByToken(cookieToken, core.TokenTypeAuth)
		if err != nil || user.Collection().Name != "users" {
			return nil, "", false
		}
		return user, cookieToken, true
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

// statusWriter keeps the first status of the response.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(status int) {
	if s.status == 0 {
		s.status = status
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

func (s *statusWriter) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}
