package webui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestWeekStarts(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("../../../../specs/api/week-starts.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases struct {
		Weeks []struct {
			Zone, Date, Monday, Start, End string
			Hours                          int
		}
		Days []struct{ Zone, Date, Start string }
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases.Weeks {
		loc, err := time.LoadLocation(c.Zone)
		if err != nil {
			t.Fatal(err)
		}
		day, _ := time.Parse(dayLayout, c.Date)
		w := weekOf(day, loc)
		got := fmt.Sprintf("%s %s %s %.0f", w.monday.Format(dayLayout), w.start.UTC().Format(time.RFC3339), w.end.UTC().Format(time.RFC3339), w.end.Sub(w.start).Hours())
		if want := fmt.Sprintf("%s %s %s %d", c.Monday, c.Start, c.End, c.Hours); got != want {
			t.Errorf("week of %s in %s: %s, want %s", c.Date, c.Zone, got, want)
		}
	}
	for _, c := range cases.Days {
		loc, err := time.LoadLocation(c.Zone)
		if err != nil {
			t.Fatal(err)
		}
		day, _ := time.Parse(dayLayout, c.Date)
		if got := startOfDay(day.Year(), day.Month(), day.Day(), loc).UTC().Format(time.RFC3339); got != c.Start {
			t.Errorf("start of %s in %s: %s, want %s", c.Date, c.Zone, got, c.Start)
		}
	}
}

func TestCurrentWeek(t *testing.T) {
	t.Parallel()
	// Monday 23 November 2026 at 00:30 in CET.
	now := time.Date(2026, time.November, 22, 23, 30, 0, 0, time.UTC)
	for zone, want := range map[string]string{"CET": "2026-11-23", "America/New_York": "2026-11-16", "UTC": "2026-11-16"} {
		loc, err := time.LoadLocation(zone)
		if err != nil {
			t.Fatal(err)
		}
		if got := currentWeek(now, loc).monday.Format(dayLayout); got != want {
			t.Errorf("%s: %s, want %s", zone, got, want)
		}
	}
}

// setCreated moves a record to another time. PocketBase sets created on each
// save.
func setCreated(t *testing.T, app core.App, r *core.Record, created time.Time) {
	t.Helper()
	_, err := app.DB().Update(r.Collection().Name, dbx.Params{"created": created.UTC().Format(types.DefaultDateLayout)}, dbx.HashExp{"id": r.Id}).Execute()
	if err != nil {
		t.Fatal(err)
	}
}

func TestWeekPages(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	cet, err := time.LoadLocation(DefaultTimeZone)
	if err != nil {
		t.Fatal(err)
	}
	ana := save(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana", "approved": true})
	joe := save(t, app, "users", map[string]any{"email": "joe@example.com", "password": "correct-horse", "name": "Joe", "approved": true})
	restaurant := save(t, app, "restaurants", map[string]any{"name": "Luigi's", "address": "Carrer 1", "creator": ana.Id})
	recipe := save(t, app, "recipes", map[string]any{"name": "Shakshuka", "ingredients": []string{"eggs"}, "steps": []string{"Cook."}, "creator": ana.Id})
	item := save(t, app, "pantry_items", map[string]any{"name": "Rioja", "category": "wine", "creator": ana.Id})

	visit := func(user *core.Record, created time.Time) {
		r := save(t, app, "reviews", map[string]any{"restaurant": restaurant.Id, "creator": user.Id, "date": "2024-11-06", "title": "Visit",
			"space": 7, "service": 7, "food": 7, "price": 7})
		setCreated(t, app, r, created)
	}
	cook := func(user *core.Record, created time.Time) {
		r := save(t, app, "recipe_reviews", map[string]any{"recipe": recipe.Id, "creator": user.Id, "date": "2024-11-06", "title": "Cook",
			"taste": 7, "ease": 7, "instructions": 7, "value": 7})
		setCreated(t, app, r, created)
	}
	taste := func(user *core.Record, created time.Time) {
		r := save(t, app, "pantry_item_reviews", map[string]any{"pantry_item": item.Id, "creator": user.Id, "date": "2024-11-06", "title": "Taste",
			"taste": 7, "aroma": 7, "texture": 7, "value": 7})
		setCreated(t, app, r, created)
	}

	// The week of 17 November 2025 in CET starts at 16 November 23:00 UTC.
	start := time.Date(2025, time.November, 16, 23, 0, 0, 0, time.UTC)
	visit(ana, start)
	visit(joe, start.Add(time.Hour))
	cook(ana, start.Add(2*time.Hour))
	taste(ana, start.Add(7*24*time.Hour-time.Millisecond))
	// These are in the weeks before and after.
	visit(joe, start.Add(-time.Millisecond))
	cook(joe, start.Add(7*24*time.Hour))

	// Wednesday 25 November 2026 is in the week of 23 November.
	now := time.Date(2026, time.November, 25, 12, 0, 0, 0, time.UTC)
	current := currentWeek(now, cet)
	visit(ana, current.start)

	h := newHandlerWith(t, app, fstest.MapFS{"index.html": {Data: []byte(testIndex)}}, Options{now: func() time.Time { return now }})
	for _, tc := range []struct {
		path, title, description, url string
	}{
		{
			"/weeks/2025-11-17", "Week of 17 November 2025",
			"4 entries from 2 people: 2 visits, 1 cook, and 1 tasting.", "/weeks/2025-11-17",
		},
		{"/WEEKS/2025-11-23/", "Week of 17 November 2025", "4 entries from 2 people: 2 visits, 1 cook, and 1 tasting.", "/weeks/2025-11-17"},
		{"/weeks/2025-11-10", "Week of 10 November 2025", "1 entry from 1 person: 1 visit.", "/weeks/2025-11-10"},
		{"/weeks/2025-08-04", "Week of 4 August 2025", "A quiet week on Olive.", "/weeks/2025-08-04"},
		{"/weeks/1900-01-01", "Week of 1 January 1900", "A quiet week on Olive.", "/weeks/1900-01-01"},
		{"/weeks/2026-11-29", "Week of 23 November 2026", "So far, 1 entry from 1 person: 1 visit.", "/weeks/2026-11-23"},
	} {
		w := get(h, tc.path)
		body := w.Body.String()
		for _, want := range []string{
			"<title>" + tc.title + " · Olive</title>",
			`<meta property="og:title" content="` + tc.title + `" />`,
			`<meta property="og:type" content="article" />`,
			`<meta name="description" content="` + tc.description + `" />`,
			`<meta property="og:url" content="https://reviews.example.com` + tc.url + `" />`,
			`<meta property="og:image" content="https://site.example.com/og-image.png" />`,
		} {
			if w.Code != 200 || !strings.Contains(body, want) {
				t.Errorf("%s: %d, no %s\n%s", tc.path, w.Code, want, body)
			}
		}
	}

	for _, path := range []string{"/weeks/2026-11-30", "/weeks/2026-02-30", "/weeks/1899-12-31", "/weeks/abc", "/weeks/2026-1-5", "/weeks/9999-12-27"} {
		if w := get(h, path); w.Code != 404 || w.Body.String() != testIndex {
			t.Errorf("%s: %d\n%s", path, w.Code, w.Body.String())
		}
	}
	if w := get(h, "/weeks"); w.Code != 200 || w.Body.String() != testIndex {
		t.Errorf("/weeks: %d, want the default page\n%s", w.Code, w.Body.String())
	}
}

func TestWeekPageUsesTheZone(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	ana := save(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana", "approved": true})
	restaurant := save(t, app, "restaurants", map[string]any{"name": "Luigi's", "address": "Carrer 1", "creator": ana.Id})
	r := save(t, app, "reviews", map[string]any{"restaurant": restaurant.Id, "creator": ana.Id, "date": "2025-11-16", "title": "Late",
		"space": 7, "service": 7, "food": 7, "price": 7})
	// Monday 17 November 2025 at 00:30 in CET.
	setCreated(t, app, r, time.Date(2025, time.November, 16, 23, 30, 0, 0, time.UTC))

	index, _ := readIndex(fstest.MapFS{"index.html": {Data: []byte(testIndex)}})
	index.now = time.Now
	for zone, want := range map[string]string{"CET": "weeks/2025-11-17", "America/New_York": "weeks/2025-11-10"} {
		var err error
		if index.zone, err = time.LoadLocation(zone); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"weeks/2025-11-10", "weeks/2025-11-17"} {
			description := "A quiet week on Olive."
			if path == want {
				description = "1 entry from 1 person: 1 visit."
			}
			if meta, _, err := index.metaFor(app, app.Settings().Meta.AppURL, path); err != nil || meta.Description != description {
				t.Errorf("%s in %s: %q, %v", path, zone, meta.Description, err)
			}
		}
	}
}
