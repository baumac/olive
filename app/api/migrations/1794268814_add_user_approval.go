package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The users that exist before this migration are approved. A superuser
// approves each later user in the dashboard. See decision 00003.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.Fields.Add(&core.BoolField{Name: "approved"})
		users.AddIndex("idx_users_approved", false, "approved, created", "")
		if err := app.Save(users); err != nil {
			return err
		}
		return execAll(app, "UPDATE {{users}} SET [[approved]] = TRUE")
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.RemoveIndex("idx_users_approved")
		users.Fields.RemoveByName("approved")
		return app.Save(users)
	})
}
