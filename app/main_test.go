package main

import (
	"testing"

	"github.com/baumac/reviews-app/app/internal/publicapi"
)

func TestOptions(t *testing.T) {
	env := func(vars map[string]string) func(string) string {
		return func(name string) string { return vars[name] }
	}

	opts, err := options(env(nil))
	if err != nil || opts != publicapi.DefaultOptions {
		t.Errorf("no variables: %+v, %v", opts, err)
	}
	if opts.SkipRequestValidation || opts.SkipResponseValidation {
		t.Error("the default turns off the validation")
	}

	opts, err = options(env(map[string]string{
		"REVIEWS_SIGN_IN_LIMIT":      "500",
		"REVIEWS_PHOTO_UPLOAD_LIMIT": "400",
		"REVIEWS_VALIDATE_REQUESTS":  "false",
		"REVIEWS_VALIDATE_RESPONSES": "0",
	}))
	if err != nil || opts.SignInLimit != 500 || opts.PhotoUploadLimit != 400 || !opts.SkipRequestValidation || !opts.SkipResponseValidation {
		t.Errorf("all variables: %+v, %v", opts, err)
	}

	opts, err = options(env(map[string]string{"REVIEWS_VALIDATE_REQUESTS": "true"}))
	if err != nil || opts.SkipRequestValidation {
		t.Errorf("true: %+v, %v", opts, err)
	}

	for _, vars := range []map[string]string{
		{"REVIEWS_SIGN_IN_LIMIT": "0"},
		{"REVIEWS_PHOTO_UPLOAD_LIMIT": "many"},
		{"REVIEWS_VALIDATE_REQUESTS": "off"},
		{"REVIEWS_VALIDATE_RESPONSES": "yes"},
	} {
		if _, err := options(env(vars)); err == nil {
			t.Errorf("%v: no error", vars)
		}
	}
}
