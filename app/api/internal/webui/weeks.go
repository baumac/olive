package webui

import (
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// DefaultTimeZone is the zone of the weeks in the link previews. See
// decision 00013.
const DefaultTimeZone = "CET"

const dayLayout = "2006-01-02"

// firstWeekDay is the first date of a review. See decision 00005.
var firstWeekDay = time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)

// startOfDay returns the first instant on or after 00:00 of the date in loc.
// In a few zones the clocks jump at midnight. The day then starts at the
// first instant after the jump. The UI uses the same rule.
func startOfDay(year int, month time.Month, day int, loc *time.Location) time.Time {
	t := time.Date(year, month, day, 0, 0, 0, 0, loc)
	y, m, d := t.Date()
	zoneStart, zoneEnd := t.ZoneBounds()
	switch {
	case y != year || m != month || d != day:
		// Go gave a time before the jump. Its zone ends at the jump.
		return zoneEnd
	case t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0:
		// Go gave a time after the jump. Its zone starts at the jump.
		return zoneStart
	}
	return firstOccurrence(t, zoneStart)
}

// firstOccurrence returns the earlier instant with the clock time of t. Go
// gives the later one when the clocks go back over t.
func firstOccurrence(t, zoneStart time.Time) time.Time {
	if zoneStart.IsZero() {
		return t
	}
	_, before := zoneStart.Add(-time.Nanosecond).Zone()
	_, after := t.Zone()
	if before <= after {
		return t
	}
	earlier := t.Add(-time.Duration(before-after) * time.Second)
	if earlier.Before(zoneStart) {
		return earlier
	}
	return t
}

// week is Monday 00:00 to the next Monday 00:00 in a zone. monday is a
// date in UTC without a time.
type week struct {
	monday     time.Time
	start, end time.Time
}

// weekOf returns the week of a date in UTC without a time.
func weekOf(day time.Time, loc *time.Location) week {
	monday := day.AddDate(0, 0, -(int(day.Weekday())+6)%7)
	next := monday.AddDate(0, 0, 7)
	return week{
		monday: monday,
		start:  startOfDay(monday.Year(), monday.Month(), monday.Day(), loc),
		end:    startOfDay(next.Year(), next.Month(), next.Day(), loc),
	}
}

func currentWeek(now time.Time, loc *time.Location) week {
	y, m, d := now.In(loc).Date()
	return weekOf(time.Date(y, m, d, 0, 0, 0, 0, time.UTC), loc)
}

// weekCounts has the reviews of each kind in the order of reviewedKinds.
// people counts each person with a review one time.
type weekCounts struct {
	reviews itemCounts
	people  int
}

// countsOfWeek counts with one query. A person with reviews of two kinds
// counts one time.
func countsOfWeek(app core.App, w week) (weekCounts, error) {
	inWeek := " WHERE [[created]] >= {:start} AND [[created]] < {:end}"
	parts := make([]string, len(reviewedKinds))
	creators := make([]string, len(reviewedKinds))
	for i, k := range reviewedKinds {
		parts[i] = "(SELECT COUNT(*) FROM {{" + k.Reviews + "}}" + inWeek + ") AS [[n" + strconv.Itoa(i) + "]]"
		creators[i] = "SELECT [[creator]] FROM {{" + k.Reviews + "}}" + inWeek
	}
	people := "(SELECT COUNT(*) FROM (" + strings.Join(creators, " UNION ") + ")) AS [[people]]"
	row := dbx.NullStringMap{}
	err := app.DB().NewQuery("SELECT " + strings.Join(parts, ", ") + ", " + people).Bind(dbx.Params{
		"start": w.start.UTC().Format(types.DefaultDateLayout),
		"end":   w.end.UTC().Format(types.DefaultDateLayout),
	}).One(row)
	if err != nil {
		return weekCounts{}, err
	}
	out := weekCounts{reviews: make(itemCounts, len(reviewedKinds))}
	for i := range out.reviews {
		out.reviews[i], _ = strconv.Atoi(row["n"+strconv.Itoa(i)].String)
	}
	out.people, _ = strconv.Atoi(row["people"].String)
	return out, nil
}

// weekPage has no record for an invalid date, a date before 1900, or a
// future week. Another day of a week gets the meta of its week.
func (p indexPage) weekPage(app core.App, appURL, date string) (pageMeta, bool, error) {
	day, err := time.Parse(dayLayout, date)
	if err != nil || day.Before(firstWeekDay) {
		return pageMeta{}, true, errNoRecord
	}
	w := weekOf(day, p.zone)
	current := currentWeek(p.now(), p.zone)
	if w.monday.After(current.monday) {
		return pageMeta{}, true, errNoRecord
	}
	counts, err := countsOfWeek(app, w)
	if err != nil {
		return pageMeta{}, true, err
	}
	return p.weekMeta(w, counts, w.monday.Equal(current.monday), publicBaseURL(appURL)), true, nil
}

func (p indexPage) weekMeta(w week, counts weekCounts, inProgress bool, baseURL string) pageMeta {
	title := "Week of " + w.monday.Format("2 January 2006")
	description := "A quiet week on " + p.siteName + "."
	if total := sum(counts.reviews); total > 0 {
		var kinds []string
		for i, n := range counts.reviews {
			if n > 0 {
				kinds = append(kinds, countOf(n, reviewedKinds[i].entry))
			}
		}
		description = plural(total, "entry", "entries") + " from " + plural(counts.people, "person", "people") + ": " + joinAnd(kinds) + "."
	}
	if inProgress {
		description = "So far, " + strings.ToLower(description[:1]) + description[1:]
	}
	meta := p.siteImage
	meta.Title = p.pageTitle(title)
	meta.OGTitle = title
	meta.Description = description
	meta.Type = "article"
	meta.URL = pageURL(baseURL, "/weeks/"+w.monday.Format(dayLayout))
	return meta
}

func sum(counts itemCounts) int {
	total := 0
	for _, n := range counts {
		total += n
	}
	return total
}

func plural(n int, one, other string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + other
}
