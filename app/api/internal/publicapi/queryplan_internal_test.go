package publicapi

import (
	"context"
	"strings"
	"testing"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestRestaurantListsUseIndexes(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	for _, c := range []struct {
		name   string
		sort   string
		filter func(w *where)
		cursor bool
		// sorts allows a temporary B-tree for the overall sort. A list has
		// at most 500 restaurants.
		sorts bool
		want  string
	}{
		{name: "sort -overall", sort: "-overall", want: "SCAN restaurant_overall USING INDEX idx_restaurant_overall_desc"},
		{name: "sort overall", sort: "overall", want: "SCAN restaurant_overall USING INDEX idx_restaurant_overall_asc"},
		{name: "sort -overall after cursor", sort: "-overall", cursor: true, want: "SEARCH restaurant_overall USING INDEX idx_restaurant_overall_desc"},
		{name: "sort overall,name after cursor", sort: "overall,name", cursor: true, want: "SEARCH restaurant_overall USING INDEX idx_restaurant_overall_asc"},
		{name: "overall filter", sort: "name", filter: func(w *where) {
			w.joinOverall(restaurantSubject)
			w.compare(restaurantSubject.overallColumn(), []comparison{{">=", 4.0}})
		}, want: "SEARCH restaurant_overall USING COVERING INDEX idx_restaurant_overall_value"},
		{name: "tags has", sort: "name", filter: func(w *where) {
			w.tagFilters(restaurantSubject, new("italian"), nil, nil)
		}, want: "SEARCH restaurant_tags USING PRIMARY KEY (tag=?)"},
		{name: "tags has any and all", sort: "-created", filter: func(w *where) {
			w.tagFilters(restaurantSubject, nil, &[]string{"italian", "pizza"}, &[]string{"cheap", "late"})
		}, want: "SEARCH restaurant_tags USING PRIMARY KEY (tag=?)"},
		{name: "list -created", sort: "-created", filter: func(w *where) { w.inList(restaurantSubject, new("k3x9m2p7q1w8z4a")) }, want: "SEARCH list_items USING COVERING INDEX idx_list_items_list_restaurant (list=?)"},
		{name: "list -overall", sort: "-overall", filter: func(w *where) { w.inList(restaurantSubject, new("k3x9m2p7q1w8z4a")) }, sorts: true, want: "SEARCH list_items USING COVERING INDEX idx_list_items_list_restaurant (list=?)"},
		{name: "list name", sort: "name", filter: func(w *where) { w.inList(restaurantSubject, new("k3x9m2p7q1w8z4a")) }, want: "SEARCH list_items USING COVERING INDEX idx_list_items_list_restaurant (list=?)"},
	} {
		t.Run(c.name, func(t *testing.T) {
			keys, sort, detail := parseSort(&c.sort, "restaurants", restaurantSorts)
			if detail != "" {
				t.Fatal(detail)
			}
			w := newWhere()
			if sortsBy(sort, "overall") {
				w.joinOverall(restaurantSubject)
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
			if !c.sorts && strings.TrimPrefix(sort, "-") == "overall" && strings.Contains(plan, "TEMP B-TREE") {
				t.Errorf("plan sorts rows:\n%s", plan)
			}
		})
	}
}

func TestOldOverallCursorIsRejected(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	s := newTestServer(t)
	w := galleryWhere(restaurantSubject, false, "k3x9m2p7q1w8z4a")
	w.keyset(restaurantSubject.galleryKeys, []any{7.6, "2024-11-04 01:26:55.002Z", "r1b8c5e2g9i6k3m", "2024-11-04 01:26:55.002Z", "p4h7j1l5n9q2s6v"})
	for name, p := range map[string]string{
		"page":  queryPlan(t, s, s.sortQuery("review_photos", w, restaurantSubject.galleryKeys, defaultLimit+1).Build()),
		"cover": queryPlan(t, s, s.coverQuery(restaurantSubject, []string{"k3x9m2p7q1w8z4a", "u5n1c8r3t6y0b2e"})),
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
	t.Parallel()
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
			keys, sort, detail := parseSort(&c.sort, "reviews", restaurantSubject.reviewSorts)
			if detail != "" {
				t.Fatal(detail)
			}
			w := newWhere()
			if c.filter != nil {
				c.filter(w)
			}
			if c.overall {
				w.compare(restaurantSubject.overallFilter(sort), []comparison{{">=", 8.0}})
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

func TestReviewerStatsUseIndexes(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	plan := queryPlan(t, s, s.reviewerStatsQuery("u5n1c8r3t6y0b2e"))
	for _, want := range []string{"idx_reviews_creator_overall", "COVERING INDEX idx_restaurants_creator_created"} {
		if !strings.Contains(plan, want) {
			t.Errorf("plan has no %q:\n%s", want, plan)
		}
	}
}

func TestReviewerRestaurantsUseIndex(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	sort := "-created"
	keys, _, detail := parseSort(&sort, "restaurants", restaurantSorts)
	if detail != "" {
		t.Fatal(detail)
	}
	for name, cursor := range map[string]bool{"first page": false, "after cursor": true} {
		t.Run(name, func(t *testing.T) {
			w := newWhere()
			w.idFilters("restaurants", "creator", new("u5n1c8r3t6y0b2e"), nil)
			if cursor {
				w.keyset(keys, []any{"2024-11-04 01:26:55.002Z", "k3x9m2p7q1w8z4a"})
			}
			plan := queryPlan(t, s, s.sortQuery("restaurants", w, keys, 7).Build())
			if !strings.Contains(plan, "idx_restaurants_creator_created") || strings.Contains(plan, "TEMP B-TREE") {
				t.Errorf("plan does not read the creator index in order:\n%s", plan)
			}
		})
	}
}

func TestListsUseIndexes(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	restaurants, err := s.app.FindCollectionByNameOrId("restaurants")
	if err != nil {
		t.Fatal(err)
	}
	page := make([]*core.Record, 20)
	for i := range page {
		page[i] = core.NewRecord(restaurants)
	}
	w := listItemsWhere(restaurantSubject, "l7d2f9h4j1k8m3p")
	w.keyset(restaurantSubject.listItemKeys, []any{"2024-11-04 01:26:55.002Z", "k3x9m2p7q1w8z4a"})
	for name, c := range map[string]struct {
		plan string
		want []string
	}{
		"favorites of a page":   {queryPlan(t, s, s.favoritesQuery(restaurantSubject, "u5n1c8r3t6y0b2e", page...).Build()), []string{"idx_lists_favorites", "idx_list_items_list_restaurant"}},
		"restaurants of a list": {queryPlan(t, s, s.sortQuery("list_items", w, restaurantSubject.listItemKeys, defaultLimit+1).Build()), []string{"idx_list_items_list_created"}},
	} {
		for _, want := range c.want {
			if !strings.Contains(c.plan, want) {
				t.Errorf("%s: plan has no %q:\n%s", name, want, c.plan)
			}
		}
		if strings.Contains(c.plan, "TEMP B-TREE") || strings.Contains(c.plan, "SCAN") {
			t.Errorf("%s: plan sorts or scans:\n%s", name, c.plan)
		}
	}
}
