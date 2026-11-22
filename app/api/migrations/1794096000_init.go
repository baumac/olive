// Package migrations has the PocketBase migrations of the collections that
// the Public API uses.
package migrations

import (
	"errors"
	"slices"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/dbutils"
	"github.com/pocketbase/pocketbase/tools/types"
)

// cspReportRule limits the CSP reports of decision 00002 per IP address.
// Anybody can send them. The limit keeps the log small.
var cspReportRule = core.RateLimitRule{Label: "POST /csp-report", MaxRequests: 20, Duration: 60}

func init() {
	m.Register(up, down)
}

func up(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("restaurants"); err == nil {
		return errors.New("the data folder has the collections of the migrations before 1794096000_init.go: delete it and start the app again")
	}
	if err := setRateLimits(app, true); err != nil {
		return err
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	if err := setupUsers(app, users); err != nil {
		return err
	}
	lists := newLists(users)
	if err := app.Save(lists); err != nil {
		return err
	}
	for _, k := range subjectKinds {
		if err := k.create(app, users, lists); err != nil {
			return err
		}
	}
	return nil
}

func down(app core.App) error {
	for _, k := range slices.Backward(subjectKinds) {
		if err := k.drop(app); err != nil {
			return err
		}
	}
	lists, err := app.FindCollectionByNameOrId("lists")
	if err != nil {
		return err
	}
	if err := app.Delete(lists); err != nil {
		return err
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	if err := restoreUsers(app, users); err != nil {
		return err
	}
	return setRateLimits(app, false)
}

// setRateLimits changes the rate limits of PocketBase. Their default rules
// limit the auth routes of /api and the superuser sign in. The Public API has
// its own limit.
func setRateLimits(app core.App, enabled bool) error {
	settings := app.Settings()
	settings.RateLimits.Enabled = enabled
	settings.RateLimits.Rules = slices.DeleteFunc(settings.RateLimits.Rules, func(r core.RateLimitRule) bool {
		return r.Label == cspReportRule.Label
	})
	if enabled {
		settings.RateLimits.Rules = append(settings.RateLimits.Rules, cspReportRule)
	}
	return app.Save(settings)
}

// setupUsers removes all API rules of users. A nil AuthRule also stops the
// PocketBase auth endpoints. All sign ins then go through the Public API.
// A superuser approves each user in the dashboard. See decision 00003.
func setupUsers(app core.App, users *core.Collection) error {
	users.ListRule = nil
	users.ViewRule = nil
	users.CreateRule = nil
	users.UpdateRule = nil
	users.DeleteRule = nil
	users.AuthRule = nil
	users.PasswordAuth.Enabled = true
	users.PasswordAuth.IdentityFields = []string{core.FieldNameEmail}

	name, ok := users.Fields.GetByName("name").(*core.TextField)
	if !ok {
		name = &core.TextField{Name: "name"}
		users.Fields.Add(name)
	}
	name.Required = true
	name.Max = 255

	// The Public API has no profile pictures. PocketBase serves uploads from
	// the origin of the UI. The CSP of decision 00002 trusts this origin.
	users.Fields.RemoveByName("avatar")
	users.OAuth2.MappedFields.AvatarURL = ""

	users.Fields.Add(&core.BoolField{Name: "approved"})
	users.AddIndex("idx_users_approved", false, "approved, created", "")
	setEmailIndex(users, "email COLLATE NOCASE")
	return app.Save(users)
}

// restoreUsers sets the users collection back to the PocketBase defaults.
func restoreUsers(app core.App, users *core.Collection) error {
	owner := types.Pointer("id = @request.auth.id")
	users.ListRule, users.ViewRule, users.UpdateRule, users.DeleteRule = owner, owner, owner, owner
	users.CreateRule = types.Pointer("")
	users.AuthRule = types.Pointer("")
	if name, ok := users.Fields.GetByName("name").(*core.TextField); ok {
		name.Required = false
	}
	users.Fields.Add(&core.FileField{
		Name:      "avatar",
		MaxSelect: 1,
		MimeTypes: []string{"image/jpeg", "image/png", "image/svg+xml", "image/gif", "image/webp"},
	})
	users.OAuth2.MappedFields.AvatarURL = "avatar"
	users.RemoveIndex("idx_users_approved")
	users.Fields.RemoveByName("approved")
	setEmailIndex(users, "email")
	return app.Save(users)
}

// setEmailIndex replaces the unique email index. With COLLATE NOCASE, email
// uniqueness and the sign in lookup ignore ASCII case.
func setEmailIndex(users *core.Collection, columns string) {
	name := "idx_email_" + users.Id
	if idx, ok := dbutils.FindSingleColumnUniqueIndex(users.Indexes, core.FieldNameEmail); ok {
		name = idx.IndexName
		users.RemoveIndex(name)
	}
	users.AddIndex(name, true, columns, "email != ''")
}
