// Package photocleanup deletes the photo folders of the review photo
// collections that have no record. Decision 00009 has the details.
package photocleanup

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/baumac/reviews-app/app/api/internal/subjects"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/list"
)

const (
	// JobID is the name of the job in the PocketBase dashboard.
	JobID    = "photo-cleanup"
	schedule = "17 3 * * *"
	// minAge protects an upload in progress. PocketBase writes the files of
	// a new record before the record is in the database.
	minAge    = time.Hour
	batchSize = 500
)

// Register adds the daily job to the cron scheduler of app.
func Register(app core.App) {
	app.Cron().MustAdd(JobID, schedule, func() {
		n, err := Run(app, time.Now())
		if err != nil {
			app.Logger().Error("Photo cleanup failed", "error", err, "folders", n)
			return
		}
		if n > 0 {
			app.Logger().Info("Photo cleanup deleted photo folders without a record", "folders", n)
		}
	})
}

// Run deletes each photo folder without a record whose newest file is at
// least minAge old at now. It returns the number of deleted folders. A
// failed folder stays. Run then continues with the others.
func Run(app core.App, now time.Time) (int, error) {
	deleted := 0
	var errs []error
	for _, k := range subjects.All {
		n, err := runCollection(app, k.Photos, now)
		deleted += n
		errs = append(errs, err)
	}
	return deleted, errors.Join(errs...)
}

func runCollection(app core.App, collection string, now time.Time) (int, error) {
	photos, err := app.FindCachedCollectionByNameOrId(collection)
	if err != nil {
		return 0, err
	}
	fsys, err := app.NewFilesystem()
	if err != nil {
		return 0, err
	}
	defer fsys.Close()

	// The filesystem API of PocketBase cannot list only the folders. The
	// list then has each file of the collection.
	files, err := fsys.List(photos.Id + "/")
	if err != nil {
		return 0, err
	}
	newest := map[string]time.Time{}
	for _, f := range files {
		parts := strings.Split(f.Key, "/")
		if len(parts) < 3 {
			continue
		}
		if t, ok := newest[parts[1]]; !ok || f.ModTime.After(t) {
			newest[parts[1]] = f.ModTime
		}
	}
	var old []string
	for id, t := range newest {
		if now.Sub(t) >= minAge {
			old = append(old, id)
		}
	}

	existing, err := existingIDs(app, photos.Name, old)
	if err != nil {
		return 0, err
	}
	deleted := 0
	var errs []error
	for _, id := range old {
		if existing[id] {
			continue
		}
		if failed := fsys.DeletePrefix(photos.Id + "/" + id + "/"); len(failed) > 0 {
			errs = append(errs, fmt.Errorf("folder %s: %w", id, errors.Join(failed...)))
			continue
		}
		deleted++
	}
	return deleted, errors.Join(errs...)
}

func existingIDs(app core.App, collection string, ids []string) (map[string]bool, error) {
	out := map[string]bool{}
	for start := 0; start < len(ids); start += batchSize {
		batch := list.ToInterfaceSlice(ids[start:min(start+batchSize, len(ids))])
		var found []string
		if err := app.DB().Select("id").From(collection).Where(dbx.In("id", batch...)).Column(&found); err != nil {
			return nil, err
		}
		for _, id := range found {
			out[id] = true
		}
	}
	return out, nil
}
