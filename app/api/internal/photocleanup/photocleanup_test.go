package photocleanup_test

import (
	"bytes"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/baumac/reviews-app/app/api/internal/photocleanup"
	_ "github.com/baumac/reviews-app/app/api/migrations"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/cron"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func testJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4)), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func save(t *testing.T, app core.App, collection string, data map[string]any) *core.Record {
	t.Helper()
	c, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(data)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

// age sets the time of each file in the folder to the past.
func age(t *testing.T, dir string, by time.Duration) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-by)
	for _, e := range entries {
		if err := os.Chtimes(filepath.Join(dir, e.Name()), old, old); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunDeletesOnlyOldFoldersWithoutARecord(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	user := save(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	restaurant := save(t, app, "restaurants", map[string]any{"name": "Luigi's"})
	review := save(t, app, "reviews", map[string]any{
		"restaurant": restaurant.Id, "creator": user.Id, "space": 7, "service": 7, "food": 7, "price": 7, "cleanliness": 7, "title": "Good",
	})
	full, err := filesystem.NewFileFromBytes(testJPEG(t), "full.jpg")
	if err != nil {
		t.Fatal(err)
	}
	thumb, err := filesystem.NewFileFromBytes(testJPEG(t), "thumb.jpg")
	if err != nil {
		t.Fatal(err)
	}
	kept := save(t, app, "review_photos", map[string]any{"review": review.Id, "width": 4, "height": 4, "full": full, "thumb": thumb})

	photos, err := app.FindCollectionByNameOrId("review_photos")
	if err != nil {
		t.Fatal(err)
	}
	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fsys.Close()
	for _, id := range []string{"orphanold000001", "orphannew000001"} {
		for _, name := range []string{"full_a.jpg", "thumb_a.jpg"} {
			if err := fsys.Upload(testJPEG(t), photos.Id+"/"+id+"/"+name); err != nil {
				t.Fatal(err)
			}
		}
	}
	storage := filepath.Join(app.DataDir(), "storage", photos.Id)
	age(t, filepath.Join(storage, "orphanold000001"), 2*time.Hour)
	age(t, filepath.Join(storage, kept.Id), 2*time.Hour)

	n, err := photocleanup.Run(app, time.Now())
	if err != nil || n != 1 {
		t.Fatalf("Run: %d folders, %v", n, err)
	}
	for id, want := range map[string]bool{"orphanold000001": false, "orphannew000001": true, kept.Id: true} {
		_, err := os.Stat(filepath.Join(storage, id))
		if exists := err == nil; exists != want {
			t.Errorf("folder %s exists %v, want %v", id, exists, want)
		}
	}

	// The new orphan goes once it is old enough.
	if n, err := photocleanup.Run(app, time.Now().Add(2*time.Hour)); err != nil || n != 1 {
		t.Errorf("second Run: %d folders, %v", n, err)
	}
}

func TestRegisterAddsTheJob(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	photocleanup.Register(app)
	if !slices.ContainsFunc(app.Cron().Jobs(), func(j *cron.Job) bool { return j.Id() == photocleanup.JobID }) {
		t.Errorf("no job %s", photocleanup.JobID)
	}
}

// orphan makes an old photo folder without a record. It returns the folder.
func orphan(t *testing.T, app core.App, id string) string {
	t.Helper()
	photos, err := app.FindCollectionByNameOrId("review_photos")
	if err != nil {
		t.Fatal(err)
	}
	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fsys.Close()
	if err := fsys.Upload(testJPEG(t), photos.Id+"/"+id+"/full_a.jpg"); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(app.DataDir(), "storage", photos.Id, id)
	age(t, dir, 2*time.Hour)
	return dir
}

func TestRunKeepsAFolderThatItCannotDelete(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can delete files in a read-only folder")
	}
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	locked := orphan(t, app, "orphanlocked001")
	free := orphan(t, app, "orphanfree00001")
	if err := os.Chmod(locked, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

	n, err := photocleanup.Run(app, time.Now())
	if err == nil || n != 1 {
		t.Errorf("Run: %d folders, %v. Want 1 folder and an error.", n, err)
	}
	if _, err := os.Stat(filepath.Join(locked, "full_a.jpg")); err != nil {
		t.Errorf("the locked folder lost its file: %v", err)
	}
	if _, err := os.Stat(free); !os.IsNotExist(err) {
		t.Errorf("the free folder stays: %v", err)
	}
}

// TestSuperusersCanStartTheJob uses the cron API of the dashboard page
// Settings, Crons.
func TestSuperusersCanStartTheJob(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	photocleanup.Register(app)
	dir := orphan(t, app, "orphanapi000001")

	su := save(t, app, core.CollectionNameSuperusers, map[string]any{"email": "admin@example.com", "password": "correct-horse-battery"})
	token, err := su.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	h, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}

	list := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/crons", nil)
	r.Header.Set("Authorization", token)
	h.ServeHTTP(list, r)
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(photocleanup.JobID)) {
		t.Fatalf("list: %d %s", list.Code, list.Body)
	}

	run := httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/api/crons/"+photocleanup.JobID, nil)
	r.Header.Set("Authorization", token)
	h.ServeHTTP(run, r)
	if run.Code != http.StatusNoContent {
		t.Fatalf("run: %d %s", run.Code, run.Body)
	}
	// The API starts the job in the background.
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return
		}
	}
	t.Error("the job did not delete the orphan folder")
}

// The job logs the error of Run. It does not stop the app.
func TestRegisteredJobWithoutPhotoCollection(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	photocleanup.Register(app)
	photos, err := app.FindCollectionByNameOrId("review_photos")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Delete(photos); err != nil {
		t.Fatal(err)
	}
	if _, err := photocleanup.Run(app, time.Now()); err == nil {
		t.Fatal("Run without the review_photos collection gives no error")
	}
	jobs := app.Cron().Jobs()
	i := slices.IndexFunc(jobs, func(j *cron.Job) bool { return j.Id() == photocleanup.JobID })
	if i < 0 {
		t.Fatalf("no job %s", photocleanup.JobID)
	}
	jobs[i].Run()
	if !slices.ContainsFunc(app.Cron().Jobs(), func(j *cron.Job) bool { return j.Id() == photocleanup.JobID }) {
		t.Error("the failed run removed the job")
	}
}
