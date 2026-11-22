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
