package publicapi

import (
	"strings"
	"testing"

	reviewsapi "github.com/baumac/olive/specs/api/sdk/server/go"
)

func TestPantryItemQueriesUseIndexes(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	item := func(w *where) { w.idFilters("pantry_item_reviews", "pantry_item", new("p2k7m4q9s1v6x3z"), nil) }
	creator := func(w *where) { w.idFilters("pantry_item_reviews", "creator", new("u5n1c8r3t6y0b2e"), nil) }
	for _, c := range []struct {
		name, table, sort string
		sorts             map[string]sortField
		filter            func(w *where)
		want              string
	}{
		{"pantry items -overall", "pantry_items", "-overall", pantryItemSorts, func(w *where) { w.joinOverall(pantryItemSubject) }, "idx_pantry_item_overall_desc"},
		{"pantry items by name", "pantry_items", "name", pantryItemSorts, nil, "idx_pantry_items_name"},
		{"pantry items of a category", "pantry_items", "-created", pantryItemSorts,
			func(w *where) { w.categoryFilters(new(reviewsapi.Wine), nil) }, "idx_pantry_items_category_created (category=?)"},
		{"pantry items of a creator", "pantry_items", "-created", pantryItemSorts,
			func(w *where) { w.idFilters("pantry_items", "creator", new("u5n1c8r3t6y0b2e"), nil) }, "idx_pantry_items_creator_created"},
		{"pantry items with a tag", "pantry_items", "-created", pantryItemSorts,
			func(w *where) { w.tagFilters(pantryItemSubject, new("aged"), nil, nil) }, "SEARCH pantry_item_tags USING PRIMARY KEY (tag=?)"},
		{"pantry items of a list", "pantry_items", "-created", pantryItemSorts,
			func(w *where) { w.inList(pantryItemSubject, new("l7d2f9h4j1k8m3p")) }, "idx_list_pantry_items_list_pantry_item (list=?)"},
		{"reviews of a pantry item", "pantry_item_reviews", "-overall", pantryItemSubject.reviewSorts, item, "idx_pantry_item_reviews_pantry_item_overall"},
		{"reviews of a reviewer by date", "pantry_item_reviews", "-date", pantryItemSubject.reviewSorts, creator, "idx_pantry_item_reviews_creator_date"},
		{"all reviews by date", "pantry_item_reviews", "-date", pantryItemSubject.reviewSorts, nil, "idx_pantry_item_reviews_date"},
	} {
		t.Run(c.name, func(t *testing.T) {
			keys, _, detail := parseSort(&c.sort, c.table, c.sorts)
			if detail != "" {
				t.Fatal(detail)
			}
			w := newWhere()
			if c.filter != nil {
				c.filter(w)
			}
			plan := queryPlan(t, s, s.sortQuery(c.table, w, keys, defaultLimit+1).Build())
			if !strings.Contains(plan, c.want) {
				t.Errorf("plan has no %q:\n%s", c.want, plan)
			}
		})
	}

	t.Run("reviewer stats", func(t *testing.T) {
		plan := queryPlan(t, s, s.reviewerStatsQuery("u5n1c8r3t6y0b2e"))
		for _, want := range []string{"idx_pantry_item_reviews_creator_overall", "COVERING INDEX idx_pantry_items_creator_created"} {
			if !strings.Contains(plan, want) {
				t.Errorf("plan has no %q:\n%s", want, plan)
			}
		}
	})
}
