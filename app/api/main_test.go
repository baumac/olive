package main

import (
	"slices"
	"testing"

	"github.com/baumac/reviews-app/app/api/internal/photocleanup"
	"github.com/baumac/reviews-app/app/api/internal/publicapi"
	"github.com/baumac/reviews-app/app/api/internal/usercleanup"
	"github.com/baumac/reviews-app/app/api/internal/webui"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/cron"
)

func TestOptions(t *testing.T) {
	t.Parallel()
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
		"REVIEWS_SIGN_IN_LIMIT":         "500",
		"REVIEWS_SIGN_IN_FAILURE_LIMIT": "7",
		"REVIEWS_PHOTO_UPLOAD_LIMIT":    "400",
		"REVIEWS_CREATE_LIMIT":          "8",
		"REVIEWS_EDIT_LIMIT":            "9",
		"REVIEWS_VALIDATE_REQUESTS":     "false",
		"REVIEWS_VALIDATE_RESPONSES":    "0",
	}))
	if err != nil || opts.SignInLimit != 500 || opts.SignInFailureLimit != 7 || opts.PhotoUploadLimit != 400 ||
		opts.CreateLimit != 8 || opts.EditLimit != 9 || !opts.SkipRequestValidation || !opts.SkipResponseValidation {
		t.Errorf("all variables: %+v, %v", opts, err)
	}

	opts, err = options(env(map[string]string{"REVIEWS_VALIDATE_REQUESTS": "true"}))
	if err != nil || opts.SkipRequestValidation {
		t.Errorf("true: %+v, %v", opts, err)
	}

	opts, err = options(env(map[string]string{"REVIEWS_PLACES_URL": "off", "REVIEWS_PLACE_SEARCH_LIMIT": "5"}))
	if err != nil || opts.PlacesURL != "" || opts.PlaceSearchLimit != 5 {
		t.Errorf("place search off: %+v, %v", opts, err)
	}
	opts, err = options(env(map[string]string{"REVIEWS_PLACES_URL": "http://127.0.0.1:2322"}))
	if err != nil || opts.PlacesURL != "http://127.0.0.1:2322" {
		t.Errorf("place search URL: %+v, %v", opts, err)
	}

	ui, err := uiOptions(env(nil))
	if err != nil || ui != (webui.Options{TileURL: webui.DefaultTileURL}) {
		t.Errorf("no map variables: %+v, %v", ui, err)
	}
	ui, err = uiOptions(env(map[string]string{"REVIEWS_MAP_TILE_URL": "off"}))
	if err != nil || ui.TileURL != "" {
		t.Errorf("map off: %+v, %v", ui, err)
	}
	ui, err = uiOptions(env(map[string]string{"REVIEWS_MAP_TILE_URL": "https://t.example.com/{z}/{x}/{y}.png", "REVIEWS_MAP_ATTRIBUTION": "© Example"}))
	if err != nil || ui.TileURL != "https://t.example.com/{z}/{x}/{y}.png" || ui.TileAttribution != "© Example" {
		t.Errorf("map variables: %+v, %v", ui, err)
	}
	if _, err := uiOptions(env(map[string]string{"REVIEWS_MAP_TILE_URL": "https://t.example.com/"})); err == nil {
		t.Error("a tile URL without the parts: no error")
	}

	for _, vars := range []map[string]string{
		{"REVIEWS_PLACES_URL": "photon.komoot.io"},
		{"REVIEWS_PLACES_URL": "ftp://photon.komoot.io"},
		{"REVIEWS_PLACE_SEARCH_LIMIT": "0"},
		{"REVIEWS_SIGN_IN_LIMIT": "0"},
		{"REVIEWS_SIGN_IN_FAILURE_LIMIT": "-1"},
		{"REVIEWS_PHOTO_UPLOAD_LIMIT": "many"},
		{"REVIEWS_CREATE_LIMIT": "0"},
		{"REVIEWS_EDIT_LIMIT": "-3"},
		{"REVIEWS_VALIDATE_REQUESTS": "off"},
		{"REVIEWS_VALIDATE_RESPONSES": "yes"},
	} {
		if _, err := options(env(vars)); err == nil {
			t.Errorf("%v: no error", vars)
		}
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()
	env := func(vars map[string]string) func(string) string {
		return func(name string) string { return vars[name] }
	}
	newApp := func() *pocketbase.PocketBase {
		return pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	}

	app := newApp()
	if err := setup(app, env(nil)); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{photocleanup.JobID, usercleanup.JobID} {
		if !slices.ContainsFunc(app.Cron().Jobs(), func(j *cron.Job) bool { return j.Id() == id }) {
			t.Errorf("no job %s", id)
		}
	}
	if cmd, _, err := app.RootCmd.Find([]string{"migrate"}); err != nil || cmd.Name() != "migrate" {
		t.Errorf("no migrate command: %v", err)
	}

	for _, vars := range []map[string]string{
		{"REVIEWS_SIGN_IN_LIMIT": "many"},
		{"REVIEWS_MAP_TILE_URL": "https://t.example.com/"},
	} {
		if err := setup(newApp(), env(vars)); err == nil {
			t.Errorf("%v: no error", vars)
		}
	}
}
