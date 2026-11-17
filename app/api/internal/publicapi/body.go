package publicapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime"
	"net/http"
	"regexp"
	"slices"
	"strings"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"golang.org/x/text/unicode/norm"
)

// maxBodyBytes is far above the largest valid body of about 100 KB. A larger
// body gets 413 before the decoders use memory for it.
const maxBodyBytes = 256 << 10

// limitBody has the ID of the PocketBase body limit. The router then uses it
// in place of that limit. A body over maxUploadBytes gets 413 with Problem
// Details.
func limitBody() *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id:       apis.DefaultBodyLimitMiddlewareId,
		Priority: apis.DefaultBodyLimitMiddlewarePriority,
		Func: func(e *core.RequestEvent) error {
			if e.Request.ContentLength > maxUploadBytes {
				writeProblem(e.Response, bodyTooLarge())
				return nil
			}
			e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, maxUploadBytes)
			return e.Next()
		},
	}
}

func bodyTooLarge() reviewsapi.Problem {
	return newProblem(http.StatusRequestEntityTooLarge, "The request body is too large.")
}

// readBody checks the body against the spec. Without validate, it does only
// the checks that the strict handler needs to decode the body. The strict
// handler then reads the same bytes from r.Body.
func readBody(ctx context.Context, in *openapi3filter.RequestValidationInput, validate bool) (map[string]json.RawMessage, *reviewsapi.Problem) {
	r := in.Request
	body := in.Route.Operation.RequestBody.Value
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	content := body.Content[mediaType]
	if err != nil || content == nil {
		p := newProblem(http.StatusUnsupportedMediaType, "The content type must be "+strings.Join(slices.Sorted(maps.Keys(body.Content)), " or ")+".")
		return nil, &p
	}
	schema := content.Schema.Value

	raw, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, maxBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			p := bodyTooLarge()
			return nil, &p
		}
		p := newProblem(http.StatusBadRequest, "The request body cannot be read.")
		return nil, &p
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))

	props, err := decodeObject(raw)
	if err != nil {
		p := newProblem(http.StatusBadRequest, "The request body must be one JSON object without repeated properties.")
		return nil, &p
	}
	var unknown []string
	for name := range props {
		if _, ok := schema.Properties[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		slices.Sort(unknown)
		p := newProblem(http.StatusBadRequest,
			"The request body has unknown or read-only properties: "+strings.Join(unknown, ", ")+".")
		return nil, &p
	}

	if changeProperties(schema, props, "x-nfc", norm.NFC.String) {
		raw, _ = json.Marshal(props)
		r.Body = io.NopCloser(bytes.NewReader(raw))
	}

	errs := integerLiterals(schema, props)
	if validate {
		err = openapi3filter.ValidateRequestBody(ctx, in, in.Route.Operation.RequestBody.Value)
		r.Body = io.NopCloser(bytes.NewReader(raw))
		errs = append(errs, bodyErrors(err)...)
	}
	if len(errs) > 0 {
		p := reviewsapi.Problem(unprocessable(sortedErrors(errs)))
		return nil, &p
	}
	if changeProperties(schema, props, "x-trim", trimSpaces) {
		trimmed, _ := json.Marshal(props)
		r.Body = io.NopCloser(bytes.NewReader(trimmed))
	}
	return props, nil
}

// trimSpaces removes the ASCII spaces around s. The pattern of a schema with
// x-trim needs one other character. A valid value then does not become empty.
func trimSpaces(s string) string {
	return strings.Trim(s, " ")
}

// changeProperties uses change on each string of a schema with the extension
// ext. It returns true when a value changes.
func changeProperties(schema *openapi3.Schema, props map[string]json.RawMessage, ext string, change func(string) string) bool {
	changed := false
	for name, value := range props {
		property := schema.Properties[name]
		if property == nil {
			continue
		}
		if next := changeValue(property.Value, value, ext, change); !bytes.Equal(next, value) {
			props[name] = next
			changed = true
		}
	}
	return changed
}

func changeValue(schema *openapi3.Schema, raw json.RawMessage, ext string, change func(string) string) json.RawMessage {
	for _, option := range schema.OneOf {
		raw = changeValue(option.Value, raw, ext, change)
	}
	if schema.Extensions[ext] == true {
		var text string
		if json.Unmarshal(raw, &text) != nil || change(text) == text {
			return raw
		}
		next, _ := json.Marshal(change(text))
		return next
	}
	if schema.Items != nil {
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return raw
		}
		changed := false
		for i, item := range items {
			if next := changeValue(schema.Items.Value, item, ext, change); !bytes.Equal(next, item) {
				items[i] = next
				changed = true
			}
		}
		if !changed {
			return raw
		}
		next, _ := json.Marshal(items)
		return next
	}
	if len(schema.Properties) == 0 {
		return raw
	}
	props, err := decodeObject(raw)
	if err != nil || !changeProperties(schema, props, ext, change) {
		return raw
	}
	next, _ := json.Marshal(props)
	return next
}

// decodeObject rejects repeated properties. The later decoders read the
// first copy and this map keeps the last copy.
func decodeObject(raw []byte) (map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil, errors.New("not an object")
	}
	props := map[string]json.RawMessage{}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		name, _ := t.(string)
		if _, repeated := props[name]; repeated {
			return nil, fmt.Errorf("repeated property %q", name)
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, err
		}
		props[name] = value
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("more than one JSON value")
	}
	return props, nil
}

// integerLiteral has no fraction and no exponent. The schema allows 5.0 for
// an integer. The strict handler cannot decode it into a Go int.
var integerLiteral = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

func integerLiterals(schema *openapi3.Schema, props map[string]json.RawMessage) fieldErrors {
	var errs fieldErrors
	for name, value := range props {
		s := schema.Properties[name].Value
		isNumber := len(value) > 0 && (value[0] == '-' || (value[0] >= '0' && value[0] <= '9'))
		if s.Type.Includes(openapi3.TypeInteger) && isNumber && !integerLiteral.Match(value) {
			errs.add(name, "must be an integer")
		}
	}
	return errs
}

// bodyErrors changes the schema errors of the body to Public API errors. An
// array with too many items gets only that error. The response size then
// does not grow with the request.
func bodyErrors(err error) fieldErrors {
	var schemaErrs []*openapi3.SchemaError
	collectSchemaErrors(err, &schemaErrs)
	tooMany := map[string]bool{}
	for _, e := range schemaErrs {
		if e.SchemaField == "maxItems" {
			tooMany[propertyPath(e.JSONPointer())] = true
		}
	}
	var errs fieldErrors
	for _, e := range schemaErrs {
		ptr := e.JSONPointer()
		if len(ptr) > 0 && tooMany[propertyPath(ptr[:1])] && e.SchemaField != "maxItems" {
			continue
		}
		errs.add(propertyPath(ptr), schemaMessage(e))
	}
	return errs
}

func collectSchemaErrors(err error, out *[]*openapi3.SchemaError) {
	var multi openapi3.MultiError
	if errors.As(err, &multi) {
		for _, e := range multi {
			collectSchemaErrors(e, out)
		}
		return
	}
	var reqErr *openapi3filter.RequestError
	if errors.As(err, &reqErr) && reqErr.Err != nil {
		collectSchemaErrors(reqErr.Err, out)
		return
	}
	var resErr *openapi3filter.ResponseError
	if errors.As(err, &resErr) && resErr.Err != nil {
		collectSchemaErrors(resErr.Err, out)
		return
	}
	var schemaErr *openapi3.SchemaError
	if errors.As(err, &schemaErr) {
		*out = append(*out, schemaErr)
	}
}

// propertyPath changes a JSON pointer such as tags/1 to tags[1].
func propertyPath(ptr []string) string {
	var b strings.Builder
	for i, part := range ptr {
		if i > 0 && isIndex(part) {
			b.WriteString("[" + part + "]")
			continue
		}
		if i > 0 {
			b.WriteString(".")
		}
		b.WriteString(part)
	}
	return b.String()
}

func isIndex(s string) bool {
	return s != "" && strings.Trim(s, "0123456789") == ""
}

// schemaMessage gives the Public API message of a schema error. A pattern
// error uses the x-pattern-message of the schema.
func schemaMessage(e *openapi3.SchemaError) string {
	s := e.Schema
	switch e.SchemaField {
	case "required":
		return "is required"
	case "nullable":
		return "must not be null"
	case "type":
		return "must be " + typeName(s)
	case "minLength", "maxLength":
		return lengthMessage(s)
	case "pattern":
		if m, ok := s.Extensions["x-pattern-message"].(string); ok {
			return m
		}
	case "format":
		if s.Format == "email" {
			return "must be an email address"
		}
		if s.Format == "tag" {
			return "must not have invisible characters"
		}
	case "minimum", "maximum":
		if s.Min != nil && s.Max != nil {
			return fmt.Sprintf("must be between %v and %v", *s.Min, *s.Max)
		}
	case "maxItems":
		if s.MaxItems != nil {
			return fmt.Sprintf("must have at most %d items", *s.MaxItems)
		}
	case "uniqueItems":
		return "must have unique items"
	}
	return e.Reason
}

func typeName(s *openapi3.Schema) string {
	for _, t := range s.Type.Slice() {
		switch t {
		case openapi3.TypeInteger, openapi3.TypeObject, openapi3.TypeArray:
			return "an " + t
		case openapi3.TypeNull:
		default:
			return "a " + t
		}
	}
	return "a valid value"
}

func lengthMessage(s *openapi3.Schema) string {
	switch {
	case s.MaxLength != nil && s.MinLength > 0:
		return fmt.Sprintf("must have %d to %d characters", s.MinLength, *s.MaxLength)
	case s.MaxLength != nil:
		return fmt.Sprintf("must have at most %d characters", *s.MaxLength)
	default:
		return fmt.Sprintf("must have at least %d characters", s.MinLength)
	}
}

// sortedErrors keeps the first error of each property.
func sortedErrors(errs fieldErrors) fieldErrors {
	slices.SortStableFunc(errs, func(a, b reviewsapi.ValidationError) int { return strings.Compare(a.Property, b.Property) })
	return slices.CompactFunc(errs, func(a, b reviewsapi.ValidationError) bool { return a.Property == b.Property })
}
