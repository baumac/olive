// Command app serves the Public UI, the Public API, the PocketBase API, and
// the PocketBase dashboard.
package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/baumac/reviews-app/app/internal/photocleanup"
	"github.com/baumac/reviews-app/app/internal/publicapi"
	"github.com/baumac/reviews-app/app/internal/searchindex"
	"github.com/baumac/reviews-app/app/internal/webui"
	_ "github.com/baumac/reviews-app/app/migrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func main() {
	opts, err := options(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	app := pocketbase.New()
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{TemplateLang: migratecmd.TemplateLangGo})
	searchindex.Register(app)
	publicapi.Register(app, opts)
	photocleanup.Register(app)
	webui.Register(app)
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// options reads the settings of the Public API from the environment. The
// Production section of AGENTS.md lists the variables.
func options(getenv func(string) string) (publicapi.Options, error) {
	opts := publicapi.DefaultOptions
	for name, limit := range map[string]*int{
		"REVIEWS_SIGN_IN_LIMIT":         &opts.SignInLimit,
		"REVIEWS_SIGN_IN_FAILURE_LIMIT": &opts.SignInFailureLimit,
		"REVIEWS_PHOTO_UPLOAD_LIMIT":    &opts.PhotoUploadLimit,
		"REVIEWS_PLACE_SEARCH_LIMIT":    &opts.PlaceSearchLimit,
	} {
		v := getenv(name)
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return opts, fmt.Errorf("%s must be a positive integer, not %q", name, v)
		}
		*limit = n
	}
	for name, skip := range map[string]*bool{
		"REVIEWS_VALIDATE_REQUESTS":  &opts.SkipRequestValidation,
		"REVIEWS_VALIDATE_RESPONSES": &opts.SkipResponseValidation,
	} {
		v := getenv(name)
		if v == "" {
			continue
		}
		validate, err := strconv.ParseBool(v)
		if err != nil {
			return opts, fmt.Errorf("%s must be true or false, not %q", name, v)
		}
		*skip = !validate
	}
	switch v := getenv("REVIEWS_PLACES_URL"); {
	case v == "off":
		opts.PlacesURL = ""
	case v != "":
		u, err := url.Parse(v)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return opts, fmt.Errorf("REVIEWS_PLACES_URL must be an http or https URL or off, not %q", v)
		}
		opts.PlacesURL = v
	}
	return opts, nil
}
