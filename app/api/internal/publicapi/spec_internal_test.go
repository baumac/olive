package publicapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

func TestSpecRules(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		change func(doc *openapi3.T)
		want   string
	}{
		{"an unknown rate limit", func(doc *openapi3.T) {
			doc.Paths.Find("/restaurants").Post.Extensions["x-rate-limit"] = map[string]any{"key": "user", "name": "other"}
		}, "is not valid"},
		{"a user rate limit without a token", func(doc *openapi3.T) {
			doc.Paths.Find("/users").Post.Extensions["x-rate-limit"] = map[string]any{"key": "user", "name": "create"}
		}, "needs a bearer token"},
		{"a rate limit without 429", func(doc *openapi3.T) {
			doc.Paths.Find("/restaurants").Post.Responses.Delete("429")
		}, "needs a 429 response"},
		{"an approval without 403", func(doc *openapi3.T) {
			doc.Paths.Find("/restaurants").Post.Responses.Delete("403")
		}, "needs a 403 response"},
	} {
		doc, err := reviewsapi.GetSwagger()
		if err != nil {
			t.Fatal(err)
		}
		tc.change(doc)
		if _, err := specOf(doc); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %v, want %q", tc.name, err, tc.want)
		}
	}
}

func TestPathParams(t *testing.T) {
	t.Parallel()
	pattern := "GET /v1/restaurants/{restaurantId}"
	if got, err := pathParams(pattern, "/v1/restaurants/a%20b"); err != nil || got["restaurantId"] != "a b" {
		t.Errorf("params %v, error %v", got, err)
	}
	for _, path := range []string{"/v1/restaurants/a/b", "/v1/restaurants/%zz"} {
		if _, err := pathParams(pattern, path); err == nil {
			t.Errorf("%s: no error", path)
		}
	}
}

func TestDecodeObjectRefusesBrokenJSON(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{`{"a":1 "b":2}`, `{"a":}`, `{"a":1`, `{"a":1]`, `{"a":1}{}`, `[]`, `{"a":1,"a":2}`} {
		if _, err := decodeObject([]byte(raw)); err == nil {
			t.Errorf("%s: no error", raw)
		}
	}
}

func TestCollectSchemaErrorsUnwrapsValidationErrors(t *testing.T) {
	t.Parallel()
	schemaErr := &openapi3.SchemaError{SchemaField: "maxLength"}
	var got []*openapi3.SchemaError
	collectSchemaErrors(openapi3.MultiError{
		&openapi3filter.RequestError{Err: schemaErr},
		&openapi3filter.ResponseError{Err: schemaErr},
		errors.New("not a schema error"),
	}, &got)
	if len(got) != 2 || got[0] != schemaErr || got[1] != schemaErr {
		t.Errorf("schema errors %v", got)
	}
}

func TestLengthMessage(t *testing.T) {
	t.Parallel()
	if got := lengthMessage(&openapi3.Schema{MinLength: 3}); got != "must have at least 3 characters" {
		t.Errorf("message %q", got)
	}
}

func TestResponseProblemOfOtherError(t *testing.T) {
	t.Parallel()
	if got := responseProblem(errors.New("x")); got != "*errors.errorString" {
		t.Errorf("problem %q", got)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func TestReadBodyOfBrokenConnection(t *testing.T) {
	t.Parallel()
	spec, err := loadSpec()
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, basePath+"/users", failingReader{})
	r.Header.Set("Content-Type", "application/json")
	in, err := spec.operations["POST "+basePath+"/users"].input(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, p := readBody(context.Background(), in, true); p == nil || p.Status != http.StatusBadRequest {
		t.Errorf("problem %+v, want 400", p)
	}
}
