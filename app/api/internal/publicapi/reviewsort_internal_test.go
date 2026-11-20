package publicapi

import (
	"slices"
	"testing"
)

func TestDateSortHasNoRepeatedKey(t *testing.T) {
	t.Parallel()
	for _, spec := range []string{"-date,created", "-date,-created", "date,title"} {
		keys, _, detail := parseSort(&spec, "reviews", restaurantSubject.reviewSorts)
		if detail != "" {
			t.Fatal(detail)
		}
		var exprs []string
		for _, k := range keys {
			exprs = append(exprs, k.expr)
		}
		want := []string{"[[reviews.date]]", "[[reviews.created]]"}
		if spec == "date,title" {
			want = append(want, keys[2].expr)
		}
		want = append(want, "[[reviews.id]]")
		if !slices.Equal(exprs, want) {
			t.Errorf("%s: keys %v, want %v", spec, exprs, want)
		}
		if spec != "date,title" && !keys[1].desc {
			t.Errorf("%s: created is ascending, want the direction of date", spec)
		}
	}
}

func TestDefaultReviewSortKeepsALegacyCursor(t *testing.T) {
	t.Parallel()
	fingerprintOf := func(sort string) []any { return []any{"reviews", "k3x9m2p7q1w8z4a", sort} }
	cursorOf := func(sort string, values ...any) *string {
		t.Helper()
		query, err := fingerprint(fingerprintOf(sort)...)
		if err != nil {
			t.Fatal(err)
		}
		c, err := encodeCursor(query, values)
		if err != nil {
			t.Fatal(err)
		}
		return &c
	}
	garbage := "not a cursor"
	for name, c := range map[string]struct {
		cursor *string
		want   string
	}{
		"no cursor":        {nil, reviewDefaultSort},
		"a legacy cursor":  {cursorOf(legacyReviewSort, "2024-11-06 19:02:11.120Z", "r1b8c5e2g9i6k3m"), legacyReviewSort},
		"a date cursor":    {cursorOf(reviewDefaultSort, "2024-11-06", "2024-11-06 19:02:11.120Z", "r1b8c5e2g9i6k3m"), reviewDefaultSort},
		"an invalid value": {&garbage, reviewDefaultSort},
	} {
		if got := defaultReviewSort(c.cursor, fingerprintOf); got != c.want {
			t.Errorf("%s: sort %s, want %s", name, got, c.want)
		}
	}
}
