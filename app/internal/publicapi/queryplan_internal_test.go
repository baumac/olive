package publicapi

import (
	"context"
	"strings"
	"testing"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/pocketbase/dbx"
)

func TestRestaurantListsUseIndexes(t *testing.T) {
	s := newTestServer(t)
	for _, c := range []struct {
		name   string
		sort   string
		filter func(w *where)
		cursor bool
		want   string
	}{
		{name: "sort -overall", sort: "-overall", want: "SCAN restaurant_overall USING INDEX idx_restaurant_overall_desc"},
		{name: "sort overall", sort: "overall", want: "SCAN restaurant_overall USING INDEX idx_restaurant_overall_asc"},
		{name: "sort -overall after cursor", sort: "-overall", cursor: true, want: "SEARCH restaurant_overall USING INDEX idx_restaurant_overall_desc"},
		{name: "sort overall,name after cursor", sort: "overall,name", cursor: true, want: "SEARCH restaurant_overall USING INDEX idx_restaurant_overall_asc"},
		{name: "overall filter", sort: "name", filter: func(w *where) {
			w.joinOverall()
			w.compare(overallColumn, []comparison{{">=", 4.0}})
		}, want: "SEARCH restaurant_overall USING COVERING INDEX idx_restaurant_overall_value"},
		{name: "tags has", sort: "name", filter: func(w *where) {
			w.tagFilters(new("italian"), nil, nil)
		}, want: "SEARCH restaurant_tags USING PRIMARY KEY (tag=?)"},
		{name: "tags has any and all", sort: "-created", filter: func(w *where) {
			w.tagFilters(nil, &[]string{"italian", "pizza"}, &[]string{"cheap", "late"})
		}, want: "SEARCH restaurant_tags USING PRIMARY KEY (tag=?)"},
	} {
		t.Run(c.name, func(t *testing.T) {
			keys, sort, detail := parseSort(&c.sort, "restaurants", restaurantSorts)
			if detail != "" {
				t.Fatal(detail)
			}
			w := newWhere()
			if sortsBy(sort, "overall") {
				w.joinOverall()
			}
			if c.filter != nil {
				c.filter(w)
			}
			if c.cursor {
				after := make([]any, len(keys))
				for i := range after {
					after[i] = 1.0
				}
				w.keyset(keys, after)
			}
			plan := queryPlan(t, s, s.sortQuery("restaurants", w, keys, defaultLimit+1).Build())
			if !strings.Contains(plan, c.want) {
				t.Errorf("plan has no %q:\n%s", c.want, plan)
			}
			for _, bad := range []string{"SCAN restaurants", "reviews", "json_each"} {
				if strings.Contains(plan, bad) {
					t.Errorf("plan has %q:\n%s", bad, plan)
				}
			}
			if strings.TrimPrefix(sort, "-") == "overall" && strings.Contains(plan, "TEMP B-TREE") {
				t.Errorf("plan sorts rows:\n%s", plan)
			}
		})
	}
}

func TestOldOverallCursorIsRejected(t *testing.T) {
	s := newTestServer(t)
	sort := "-overall"
	for name, version := range map[string][]any{"no version": nil, "version 2": {2}} {
		t.Run(name, func(t *testing.T) {
			parts := append([]any{"restaurants", reviewsapi.ListRestaurantsParams{Sort: &sort}}, version...)
			old, err := fingerprint(parts...)
			if err != nil {
				t.Fatal(err)
			}
			cursor, err := encodeCursor(old, []any{0.0, 7.4, "abc"})
			if err != nil {
				t.Fatal(err)
			}
			res, err := s.ListRestaurants(context.Background(), reviewsapi.ListRestaurantsRequestObject{
				Params: reviewsapi.ListRestaurantsParams{Sort: &sort, Cursor: &cursor},
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := res.(reviewsapi.ListRestaurants400ApplicationProblemPlusJSONResponse); !ok {
				t.Errorf("response %T, want 400", res)
			}
		})
	}
}

// queryPlan gets the lines of EXPLAIN QUERY PLAN for q.
func queryPlan(t *testing.T, s *Server, q *dbx.Query) string {
	t.Helper()
	var rows []struct {
		Detail string `db:"detail"`
	}
	if err := s.app.DB().NewQuery("EXPLAIN QUERY PLAN " + q.SQL()).Bind(q.Params()).All(&rows); err != nil {
		t.Fatal(err)
	}
	lines := make([]string, len(rows))
	for i, r := range rows {
		lines[i] = r.Detail
	}
	return strings.Join(lines, "\n")
}

func TestGalleryUsesIndexes(t *testing.T) {
	s := newTestServer(t)
	w := galleryWhere(false, "k3x9m2p7q1w8z4a")
	w.keyset(galleryKeys, []any{7.6, "2024-11-04 01:26:55.002Z", "r1b8c5e2g9i6k3m", "2024-11-04 01:26:55.002Z", "p4h7j1l5n9q2s6v"})
	for name, p := range map[string]string{
		"page":  queryPlan(t, s, s.sortQuery("review_photos", w, galleryKeys, defaultLimit+1).Build()),
		"cover": queryPlan(t, s, s.coverQuery([]string{"k3x9m2p7q1w8z4a", "u5n1c8r3t6y0b2e"})),
	} {
		for _, want := range []string{"SEARCH reviews USING INDEX idx_reviews_restaurant", "idx_review_photos_review_created (review=?)"} {
			if !strings.Contains(p, want) {
				t.Errorf("%s plan has no %q:\n%s", name, want, p)
			}
		}
		if strings.Contains(p, "SCAN") && !strings.Contains(p, "SCAN (subquery") {
			t.Errorf("%s plan scans a table:\n%s", name, p)
		}
	}
}

func TestReviewOverallUsesIndexes(t *testing.T) {
	s := newTestServer(t)
	restaurant := func(w *where) { w.idFilters("reviews", "restaurant", new("k3x9m2p7q1w8z4a"), nil) }
	creator := func(w *where) { w.idFilters("reviews", "creator", new("u5n1c8r3t6y0b2e"), nil) }
	const cursorRange, filterRange = "(overall,id)", "overall>"
	for _, c := range []struct {
		name    string
		sort    string
		filter  func(w *where)
		overall bool
		cursor  bool
		want    []string
	}{
		{name: "restaurant", sort: "-overall", filter: restaurant, want: []string{"idx_reviews_restaurant_overall"}},
		{name: "restaurant after cursor", sort: "-overall", filter: restaurant, cursor: true, want: []string{"idx_reviews_restaurant_overall", cursorRange}},
		{name: "restaurant ascending after cursor", sort: "overall", filter: restaurant, cursor: true, want: []string{"idx_reviews_restaurant_overall", cursorRange}},
		{name: "restaurant with overall filter", sort: "-overall", filter: restaurant, overall: true, want: []string{"idx_reviews_restaurant_overall", filterRange}},
		{name: "reviewer", sort: "-overall", filter: creator, want: []string{"idx_reviews_creator_overall"}},
		{name: "reviewer after cursor", sort: "overall", filter: creator, cursor: true, want: []string{"idx_reviews_creator_overall", cursorRange}},
		{name: "all", sort: "-overall", want: []string{"idx_reviews_overall"}},
		{name: "all after cursor", sort: "-overall", cursor: true, want: []string{"idx_reviews_overall", cursorRange}},
		{name: "all with overall filter", sort: "-overall", overall: true, want: []string{"idx_reviews_overall", filterRange}},
		{name: "restaurant by created with overall filter", sort: "-created", filter: restaurant, overall: true, want: []string{"idx_reviews_restaurant_created"}},
		{name: "all by created with overall filter", sort: "-created", overall: true, want: []string{"idx_reviews_created"}},
		{name: "all by title with overall filter", sort: "title", overall: true, want: []string{"idx_reviews_title"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			keys, sort, detail := parseSort(&c.sort, "reviews", reviewSorts)
			if detail != "" {
				t.Fatal(detail)
			}
			w := newWhere()
			if c.filter != nil {
				c.filter(w)
			}
			if c.overall {
				w.compare(overallFilter(sort), []comparison{{">=", 8.0}})
			}
			if c.cursor {
				w.keyset(keys, []any{7.6, "r1b8c5e2g9i6k3m"})
			}
			plan := queryPlan(t, s, s.sortQuery("reviews", w, keys, defaultLimit+1).Build())
			for _, want := range c.want {
				if !strings.Contains(plan, want) {
					t.Errorf("plan has no %q:\n%s", want, plan)
				}
			}
			if strings.Contains(plan, "TEMP B-TREE") {
				t.Errorf("plan sorts rows:\n%s", plan)
			}
		})
	}
}

func TestReviewerCountUsesIndex(t *testing.T) {
	s := newTestServer(t)
	q := s.app.DB().Select("COUNT(*)").From("reviews").Where(dbx.HashExp{"creator": "u5n1c8r3t6y0b2e"}).Build()
	if plan := queryPlan(t, s, q); !strings.Contains(plan, "idx_reviews_creator_overall") {
		t.Errorf("plan has no creator index:\n%s", plan)
	}
}
