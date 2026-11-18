// Package usercleanup deletes old users without approval that have no
// data. Decision 00034 has the details.
package usercleanup

import (
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// JobID is the name of the job in the PocketBase dashboard.
	JobID    = "user-cleanup"
	schedule = "47 3 * * *"
	// MaxAge is the time that a user can wait for approval.
	MaxAge    = 30 * 24 * time.Hour
	batchSize = 500
)

// Register adds the daily job to the cron scheduler of app.
func Register(app core.App) {
	app.Cron().MustAdd(JobID, schedule, func() {
		n, err := Run(app, time.Now())
		if err != nil {
			app.Logger().Error("User cleanup failed", "error", err, "users", n)
			return
		}
		if n > 0 {
			app.Logger().Info("User cleanup deleted old users without approval", "users", n)
		}
	})
}

// Run deletes each user without approval that is at least MaxAge old at
// now. A user with a review, a restaurant, a recipe, or an item in a list
// stays.
// A superuser took the approval of such a user back. It returns the number
// of deleted users.
func Run(app core.App, now time.Time) (int, error) {
	deleted := 0
	for {
		var ids []string
		err := app.DB().NewQuery(`
			SELECT [[id]] FROM {{users}}
			WHERE [[approved]] = FALSE AND [[created]] < {:before}
				AND NOT EXISTS (SELECT 1 FROM {{reviews}} WHERE [[reviews.creator]] = [[users.id]])
				AND NOT EXISTS (SELECT 1 FROM {{restaurants}} WHERE [[restaurants.creator]] = [[users.id]])
				AND NOT EXISTS (SELECT 1 FROM {{recipe_reviews}} WHERE [[recipe_reviews.creator]] = [[users.id]])
				AND NOT EXISTS (SELECT 1 FROM {{recipes}} WHERE [[recipes.creator]] = [[users.id]])
				AND NOT EXISTS (SELECT 1 FROM {{list_items}} INNER JOIN {{lists}} ON [[lists.id]] = [[list_items.list]]
					WHERE [[lists.creator]] = [[users.id]])
				AND NOT EXISTS (SELECT 1 FROM {{list_recipes}} INNER JOIN {{lists}} ON [[lists.id]] = [[list_recipes.list]]
					WHERE [[lists.creator]] = [[users.id]])
			LIMIT {:limit}`).
			Bind(dbx.Params{"before": now.Add(-MaxAge).UTC().Format("2006-01-02 15:04:05.000Z"), "limit": batchSize}).
			Column(&ids)
		if err != nil || len(ids) == 0 {
			return deleted, err
		}
		for _, id := range ids {
			user, err := app.FindRecordById("users", id)
			if err != nil {
				return deleted, err
			}
			if err := app.Delete(user); err != nil {
				return deleted, err
			}
			deleted++
		}
	}
}
