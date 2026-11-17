package photocleanup

import (
	"fmt"
	"testing"

	"github.com/baumac/reviews-app/app/api/internal/testapp"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// TestExistingIDsReadsEachBatch puts real IDs at the start, in the middle,
// and at the end of more than two batches.
func TestExistingIDsReadsEachBatch(t *testing.T) {
	app, err := tests.NewTestApp(testapp.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 2*batchSize+1)
	for i := range ids {
		ids[i] = fmt.Sprintf("missing%08d", i)
	}
	var real []string
	for i, at := range []int{0, batchSize, len(ids) - 1} {
		u := core.NewRecord(users)
		u.Load(map[string]any{"email": fmt.Sprintf("u%d@example.com", i), "password": "correct-horse", "name": "U"})
		if err := app.Save(u); err != nil {
			t.Fatal(err)
		}
		ids[at] = u.Id
		real = append(real, u.Id)
	}

	got, err := existingIDs(app, "users", ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(real) {
		t.Errorf("found %d IDs, want %d", len(got), len(real))
	}
	for _, id := range real {
		if !got[id] {
			t.Errorf("did not find %s", id)
		}
	}
}
