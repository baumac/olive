// Command app serves the Public UI, the Public API, the PocketBase API, and
// the PocketBase dashboard.
package main

import (
	"log"
	"os"
	"strconv"

	"github.com/baumac/reviews-app/app/internal/publicapi"
	"github.com/baumac/reviews-app/app/internal/webui"
	_ "github.com/baumac/reviews-app/app/migrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func main() {
	app := pocketbase.New()
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{TemplateLang: migratecmd.TemplateLangGo})
	publicapi.Register(app, options())
	webui.Register(app)
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// options reads REVIEWS_SIGN_IN_LIMIT. The end-to-end tests raise the limit.
// Their parallel sign ups come from one IP address.
func options() publicapi.Options {
	opts := publicapi.DefaultOptions
	if v := os.Getenv("REVIEWS_SIGN_IN_LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			log.Fatalf("REVIEWS_SIGN_IN_LIMIT must be a positive integer, not %q", v)
		}
		opts.SignInLimit = n
	}
	return opts
}
