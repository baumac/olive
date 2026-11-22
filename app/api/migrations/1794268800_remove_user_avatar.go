package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The Public API has no profile pictures. PocketBase serves uploads from the
// origin of the UI. The CSP of decision 00002 trusts this origin.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.Fields.RemoveByName("avatar")
		users.OAuth2.MappedFields.AvatarURL = ""
		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.Fields.Add(&core.FileField{
			Name:      "avatar",
			MaxSelect: 1,
			MimeTypes: []string{"image/jpeg", "image/png", "image/svg+xml", "image/gif", "image/webp"},
		})
		users.OAuth2.MappedFields.AvatarURL = "avatar"
		return app.Save(users)
	})
}
