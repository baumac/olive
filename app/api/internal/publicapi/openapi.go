package publicapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"slices"
	"strings"
	"sync"
	"unicode"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"golang.org/x/text/unicode/norm"
)

// operation has the spec of one operation and the checks that prepare does
// before the spec validation.
type operation struct {
	pattern string
	route   *routers.Route
	query   map[string]bool
	// nfcQuery has the query parameters with the x-nfc extension.
	nfcQuery map[string]bool
	auth     authMode
	// approval is true when the operation needs an approved user.
	approval bool
	limit    *rateLimit
	// fileBody and fileResponse mark binary bodies. The JSON checks of the
	// body and the response body check skip them.
	fileBody     bool
	fileResponse bool
}

// rateLimit is the x-rate-limit extension of an operation. The key "ip"
// counts each client network. The key "user" counts each user.
type rateLimit struct {
	key  string
	name string
}

type authMode int

const (
	authOptional authMode = iota
	authRequired
	authIgnored
)

// apiSpec has the operations of the embedded spec by ServeMux pattern.
type apiSpec struct {
	operations map[string]*operation
}

// embeddedSpec decodes the spec that the Go SDK embeds one time. The callers
// only read it.
var embeddedSpec = sync.OnceValues(reviewsapi.GetSwagger)

// loadSpec reads the spec that the Go SDK embeds.
func loadSpec() (*apiSpec, error) {
	doc, err := embeddedSpec()
	if err != nil {
		return nil, err
	}
	s := &apiSpec{operations: map[string]*operation{}}
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			pattern := method + " " + basePath + path
			limit, err := rateLimitOf(op)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", pattern, err)
			}
			if limit != nil && limit.key == "user" && authOf(doc, op) != authRequired {
				return nil, fmt.Errorf("%s: x-rate-limit with the key user needs a bearer token", pattern)
			}
			if limit != nil && op.Responses.Value("429") == nil {
				return nil, fmt.Errorf("%s: an operation with x-rate-limit needs a 429 response", pattern)
			}
			approval := needsApproval(doc, op)
			if approval && op.Responses.Value("403") == nil {
				return nil, fmt.Errorf("%s: an operation that needs an approved user needs a 403 response", pattern)
			}
			s.operations[pattern] = &operation{
				pattern:      pattern,
				approval:     approval,
				route:        &routers.Route{Spec: doc, Path: path, PathItem: item, Method: method, Operation: op},
				query:        queryNames(item, op),
				nfcQuery:     nfcQueryNames(item, op),
				auth:         authOf(doc, op),
				limit:        limit,
				fileBody:     op.RequestBody != nil && !hasJSON(op.RequestBody.Value.Content),
				fileResponse: hasFileResponse(op),
			}
		}
	}
	return s, nil
}

// needsApproval is true for the operations that need a token. The
// x-allow-unapproved extension is the exception. A new operation then needs
// approval without a change here. See decision 00034.
func needsApproval(doc *openapi3.T, op *openapi3.Operation) bool {
	return authOf(doc, op) == authRequired && op.Extensions["x-allow-unapproved"] != true
}

func rateLimitOf(op *openapi3.Operation) (*rateLimit, error) {
	raw, ok := op.Extensions["x-rate-limit"]
	if !ok {
		return nil, nil
	}
	m, _ := raw.(map[string]any)
	key, _ := m["key"].(string)
	name, _ := m["name"].(string)
	if (key != "ip" && key != "user") || !slices.Contains(rateLimitNames, name) {
		return nil, fmt.Errorf("x-rate-limit %v is not valid", raw)
	}
	return &rateLimit{key: key, name: name}, nil
}

func hasJSON(content openapi3.Content) bool {
	for mediaType := range content {
		if strings.HasPrefix(mediaType, "application/") && strings.HasSuffix(mediaType, "json") {
			return true
		}
	}
	return false
}

// hasFileResponse is true when a success response has a body that is not
// JSON.
func hasFileResponse(op *openapi3.Operation) bool {
	for status, ref := range op.Responses.Map() {
		if strings.HasPrefix(status, "2") && len(ref.Value.Content) > 0 && !hasJSON(ref.Value.Content) {
			return true
		}
	}
	return false
}

// authOf reads the security requirements of op. An empty requirement allows
// a request without a token. An operation without bearerAuth ignores the
// Authorization header.
func authOf(doc *openapi3.T, op *openapi3.Operation) authMode {
	reqs := doc.Security
	if op.Security != nil {
		reqs = *op.Security
	}
	anonymous, bearer := false, false
	for _, req := range reqs {
		if len(req) == 0 {
			anonymous = true
		}
		if _, ok := req["bearerAuth"]; ok {
			bearer = true
		}
	}
	switch {
	case !bearer:
		return authIgnored
	case anonymous:
		return authOptional
	default:
		return authRequired
	}
}

// queryNames gives the names that a client can send. A deepObject
// parameter has one name for each property, such as created[lt].
func queryNames(item *openapi3.PathItem, op *openapi3.Operation) map[string]bool {
	names := map[string]bool{}
	for _, ref := range slices.Concat(item.Parameters, op.Parameters) {
		p := ref.Value
		if p.In != openapi3.ParameterInQuery {
			continue
		}
		if p.Style != openapi3.SerializationDeepObject {
			names[p.Name] = true
			continue
		}
		for prop := range p.Schema.Value.Properties {
			names[p.Name+"["+prop+"]"] = true
		}
	}
	return names
}

var validationOptions = &openapi3filter.Options{
	MultiError: true,
	// The defaults would change the request. The cursor fingerprint
	// then would not match the request of the client.
	SkipSettingDefaults: true,
	// prepare checks the bearer token before the validation.
	AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
	SchemaValidationOptions: []openapi3.SchemaValidationOption{
		openapi3.WithStringFormatValidator("email", openapi3.NewCallbackValidator(checkEmailFormat)),
		openapi3.WithStringFormatValidator("tag", openapi3.NewCallbackValidator(checkTagFormat)),
	},
}

var paramOptions = func() *openapi3filter.Options {
	opts := *validationOptions
	opts.ExcludeRequestBody = true
	return &opts
}()

// responseOptions accept each stored email. PocketBase checked it, also for
// a save in the dashboard.
var responseOptions = &openapi3filter.Options{
	MultiError:            true,
	IncludeResponseStatus: true,
	SchemaValidationOptions: []openapi3.SchemaValidationOption{
		openapi3.WithStringFormatValidator("email", openapi3.NewCallbackValidator(func(string) error { return nil })),
	},
}

// fileResponseOptions skip the body of a file response. The image bytes
// then do not need a buffer.
var fileResponseOptions = func() *openapi3filter.Options {
	opts := *responseOptions
	opts.ExcludeResponseBody = true
	return &opts
}()

// responseProblem names the place and the rule of each response error. It
// leaves out the values. The log then does not get tokens or emails.
func responseProblem(err error) string {
	var schemaErrs []*openapi3.SchemaError
	collectSchemaErrors(err, &schemaErrs)
	var parts []string
	for _, e := range schemaErrs {
		parts = append(parts, "/"+strings.Join(e.JSONPointer(), "/")+" "+e.SchemaField)
	}
	var resErr *openapi3filter.ResponseError
	if errors.As(err, &resErr) {
		parts = append([]string{resErr.Reason}, parts...)
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%T", err)
	}
	return strings.Join(parts, "; ")
}

// checkEmailFormat accepts only a bare address. mail.ParseAddress also
// accepts a display name, such as Ana <ana@example.com>.
func checkEmailFormat(s string) error {
	if a, err := mail.ParseAddress(s); err == nil && a.Name == "" && a.Address == s {
		return nil
	}
	return errors.New("not a bare email address")
}

// checkTagFormat refuses the default ignorable code points. A tag with one
// looks like the tag without it. The tag pattern refuses the ones in Cf.
func checkTagFormat(s string) error {
	if strings.ContainsFunc(s, func(r rune) bool {
		return unicode.In(r, unicode.Other_Default_Ignorable_Code_Point, unicode.Variation_Selector)
	}) {
		return errors.New("has a default ignorable code point")
	}
	return nil
}

// input makes the validation input of r. The ServeMux pattern gives the
// path parameters.
func (op *operation) input(r *http.Request) (*openapi3filter.RequestValidationInput, error) {
	params, err := pathParams(op.pattern, r.URL.EscapedPath())
	if err != nil {
		return nil, err
	}
	return &openapi3filter.RequestValidationInput{
		Request:    r,
		PathParams: params,
		Route:      op.route,
		Options:    validationOptions,
	}, nil
}

func pathParams(pattern, escapedPath string) (map[string]string, error) {
	_, path, _ := strings.Cut(pattern, " ")
	names := strings.Split(path, "/")
	values := strings.Split(escapedPath, "/")
	if len(names) != len(values) {
		return nil, fmt.Errorf("the path %s does not match %s", escapedPath, pattern)
	}
	params := map[string]string{}
	for i, name := range names {
		if strings.HasPrefix(name, "{") && strings.HasSuffix(name, "}") {
			v, err := url.PathUnescape(values[i])
			if err != nil {
				return nil, err
			}
			params[strings.Trim(name, "{}")] = v
		}
	}
	return params, nil
}

func nfcQueryNames(item *openapi3.PathItem, op *openapi3.Operation) map[string]bool {
	names := map[string]bool{}
	for _, ref := range slices.Concat(item.Parameters, op.Parameters) {
		p := ref.Value
		if p.In != openapi3.ParameterInQuery || p.Style == openapi3.SerializationDeepObject {
			continue
		}
		s := p.Schema.Value
		if s.Items != nil {
			s = s.Items.Value
		}
		if s.Extensions["x-nfc"] == true {
			names[p.Name] = true
		}
	}
	return names
}

// normalizeQuery changes the values of nfcQuery to Unicode normalization
// form C. The validation and the handler then get the changed values.
func (op *operation) normalizeQuery(r *http.Request) {
	if len(op.nfcQuery) == 0 {
		return
	}
	query := r.URL.Query()
	changed := false
	for name, values := range query {
		if !op.nfcQuery[name] {
			continue
		}
		for i, v := range values {
			if n := norm.NFC.String(v); n != v {
				values[i] = n
				changed = true
			}
		}
	}
	if changed {
		r.URL.RawQuery = query.Encode()
	}
}

// checkQuery returns a detail for the 400 response. The spec does not
// refuse unknown query parameters.
func (op *operation) checkQuery(r *http.Request) string {
	names := make([]string, 0, len(r.URL.Query()))
	for name := range r.URL.Query() {
		if !op.query[name] {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	slices.Sort(names)
	return "The query parameter " + names[0] + " is not supported."
}

// checkParams validates the path and query parameters. It returns a
// detail for the 400 response.
func (op *operation) checkParams(ctx context.Context, in *openapi3filter.RequestValidationInput) string {
	params := *in
	params.Options = paramOptions
	// The security check of kin-openapi reads the full body, also with
	// ExcludeRequestBody. The parameters do not need the body.
	req := *in.Request
	req.Body = http.NoBody
	params.Request = &req
	err := openapi3filter.ValidateRequest(ctx, &params)
	if err == nil {
		return ""
	}
	var reqErr *openapi3filter.RequestError
	if !errors.As(err, &reqErr) || reqErr.Parameter == nil {
		return "The request has a parameter that is not valid."
	}
	p := reqErr.Parameter
	name := p.Name
	if p.Style == openapi3.SerializationDeepObject {
		var schemaErr *openapi3.SchemaError
		if errors.As(reqErr.Err, &schemaErr) && len(schemaErr.JSONPointer()) > 0 {
			name += "[" + schemaErr.JSONPointer()[0] + "]"
		}
		var parseErr *openapi3filter.ParseError
		if errors.As(reqErr.Err, &parseErr) && len(parseErr.Path()) > 0 {
			name += fmt.Sprintf("[%v]", parseErr.Path()[0])
		}
	}
	return fmt.Sprintf("The %s parameter %s is not valid.", p.In, name)
}

// checkResponse validates the status, the headers, and the body of the
// response to the request of in.
func checkResponse(ctx context.Context, in *openapi3filter.RequestValidationInput, status int, header http.Header, body []byte) error {
	res := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: in,
		Status:                 status,
		Header:                 header,
		Options:                responseOptions,
	}
	res.SetBodyBytes(body)
	return openapi3filter.ValidateResponse(ctx, res)
}

// checkResponseHeaders validates the status and the headers of a file
// response.
func checkResponseHeaders(ctx context.Context, in *openapi3filter.RequestValidationInput, status int, header http.Header) error {
	return openapi3filter.ValidateResponse(ctx, &openapi3filter.ResponseValidationInput{
		RequestValidationInput: in,
		Status:                 status,
		Header:                 header,
		Options:                fileResponseOptions,
	})
}
