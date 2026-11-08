// Command app serves the Public UI, the Public API, the PocketBase API, and
// the PocketBase dashboard.
package main

import (
	"log"

	"github.com/baumac/reviews-app/app/internal/publicapi"
	"github.com/baumac/reviews-app/app/internal/webui"
	_ "github.com/baumac/reviews-app/app/migrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func main() {
	app := pocketbase.New()
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{TemplateLang: migratecmd.TemplateLangGo})
	publicapi.Register(app, publicapi.DefaultOptions)
	webui.Register(app)
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
