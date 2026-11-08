package publicapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime"
	"net/http"
	"net/mail"
	"regexp"
	"slices"
	"strings"

	reviewsapi "github.com/baumac/reviews-app/sdk/go"
)

type propType int

const (
	typeString propType = iota
	typeEmail
	typeInteger
	typeTags
)

type prop struct {
	typ      propType
	required bool
	nullable bool
	// check checks the value rules after the type check passes.
	check func(errs *fieldErrors, name string, value json.RawMessage)
}

func (p prop) isRequired() prop {
	p.required = true
	return p
}

func (p prop) isNullable() prop {
	p.nullable = true
	return p
}

// bodySchema has the media type, the JSON types, and the value rules of a
// request body.
type bodySchema struct {
	mediaType string
	props     map[string]prop
}

const (
	mediaJSON       = "application/json"
	mediaMergePatch = "application/merge-patch+json"
)

func textRule(rule func(errs *fieldErrors, name, v string)) func(*fieldErrors, string, json.RawMessage) {
	return func(errs *fieldErrors, name string, value json.RawMessage) {
		var s string
		_ = json.Unmarshal(value, &s)
		rule(errs, name, s)
	}
}

func scoreRule(maxValue int) func(*fieldErrors, string, json.RawMessage) {
	return func(errs *fieldErrors, name string, value json.RawMessage) {
		var v int
		_ = json.Unmarshal(value, &v)
		checkScore(errs, name, v, maxValue)
	}
}

func tagsRule(errs *fieldErrors, name string, value json.RawMessage) {
	var tags []string
	_ = json.Unmarshal(value, &tags)
	checkTags(errs, name, tags)
}

var (
	nameProp     = prop{typ: typeString, check: textRule(checkName)}
	longTextProp = prop{typ: typeString, check: textRule(checkLongText)}
	tagsProp     = prop{typ: typeTags, check: tagsRule}
	kindProp     = prop{typ: typeString, required: true}
	ratingProp   = prop{typ: typeInteger, check: scoreRule(5)}
	jocScoreProp = prop{typ: typeInteger, check: scoreRule(10)}
)

var (
	userCreateSchema = bodySchema{mediaJSON, map[string]prop{
		"email":    {typ: typeEmail, required: true, check: textRule(checkEmail)},
		"password": {typ: typeString, required: true, check: textRule(checkPassword)},
		"name":     nameProp.isRequired(),
	}}
	credentialsSchema = bodySchema{mediaJSON, map[string]prop{
		"email":    {typ: typeEmail, required: true},
		"password": {typ: typeString, required: true},
	}}
	restaurantCreateSchema = bodySchema{mediaJSON, map[string]prop{
		"name":        nameProp.isRequired(),
		"description": longTextProp,
		"tags":        tagsProp,
	}}
	restaurantPatchSchema = bodySchema{mediaMergePatch, map[string]prop{
		"name":        nameProp,
		"description": longTextProp.isNullable(),
		"tags":        tagsProp.isNullable(),
	}}
	// The review schemas have the properties of all kinds. The handlers
	// reject the properties of other kinds.
	reviewCreateSchema = bodySchema{mediaJSON, map[string]prop{
		"kind":     kindProp,
		"rating":   ratingProp,
		"space":    jocScoreProp,
		"service":  jocScoreProp,
		"food":     jocScoreProp,
		"price":    jocScoreProp,
		"title":    nameProp.isRequired(),
		"feedback": longTextProp,
		"tags":     tagsProp,
	}}
	reviewPatchSchema = bodySchema{mediaMergePatch, map[string]prop{
		"kind":     kindProp,
		"rating":   ratingProp,
		"space":    jocScoreProp,
		"service":  jocScoreProp,
		"food":     jocScoreProp,
		"price":    jocScoreProp,
		"title":    nameProp,
		"feedback": longTextProp.isNullable(),
		"tags":     tagsProp.isNullable(),
	}}
)

// readBody checks the body against schema. The strict handler then reads
// the same bytes from r.Body.
func readBody(r *http.Request, schema *bodySchema) (map[string]json.RawMessage, *reviewsapi.Problem) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != schema.mediaType {
		p := newProblem(http.StatusUnsupportedMediaType, "The content type must be "+schema.mediaType+".")
		return nil, &p
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			p := newProblem(http.StatusRequestEntityTooLarge, "The request body is too large.")
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
		if _, ok := schema.props[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		slices.Sort(unknown)
		p := newProblem(http.StatusBadRequest,
			"The request body has unknown or read-only properties: "+strings.Join(unknown, ", ")+".")
		return nil, &p
	}

	var errs fieldErrors
	for _, name := range slices.Sorted(maps.Keys(schema.props)) {
		pr := schema.props[name]
		value, ok := props[name]
		if !ok {
			if pr.required {
				errs.add(name, "is required")
			}
			continue
		}
		if checkType(&errs, name, pr, value) && pr.check != nil && string(value) != "null" {
			pr.check(&errs, name, value)
		}
	}
	if len(errs) > 0 {
		p := reviewsapi.Problem(unprocessable(errs))
		return nil, &p
	}
	return props, nil
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

var integerLiteral = regexp.MustCompile(`^-?(0|[1-9][0-9]{0,17})$`)

func checkType(errs *fieldErrors, name string, pr prop, value json.RawMessage) bool {
	n := len(*errs)
	if string(value) == "null" {
		if !pr.nullable {
			errs.add(name, "must not be null")
		}
		return len(*errs) == n
	}
	switch pr.typ {
	case typeString:
		var s string
		if json.Unmarshal(value, &s) != nil {
			errs.add(name, "must be a string")
		}
	case typeEmail:
		var s string
		if json.Unmarshal(value, &s) != nil {
			errs.add(name, "must be a string")
		} else if !isEmailAddress(s) {
			errs.add(name, "must be an email address")
		}
	case typeInteger:
		if !integerLiteral.Match(value) {
			errs.add(name, "must be an integer")
		}
	case typeTags:
		var items []json.RawMessage
		if json.Unmarshal(value, &items) != nil {
			errs.add(name, "must be an array")
			break
		}
		for i, item := range items {
			var s string
			if json.Unmarshal(item, &s) != nil {
				errs.add(fmt.Sprintf("%s[%d]", name, i), "must be a string")
			}
		}
	}
	return len(*errs) == n
}

// isEmailAddress accepts only a bare address. The strict handler uses
// mail.ParseAddress, which also accepts a display name.
func isEmailAddress(s string) bool {
	a, err := mail.ParseAddress(s)
	return err == nil && a.Name == "" && a.Address == s
}
