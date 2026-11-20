package publicapi

import (
	"encoding/json"
	"strings"
	"testing"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/pocketbase/pocketbase/core"
)

func TestWithDetailsKeepsTheSummary(t *testing.T) {
	t.Parallel()
	var sum reviewsapi.RecipeSummary
	if err := json.Unmarshal([]byte(`{"id":"c4r8e2p6i0s3t7a","name":"Dal","description":"Red","tags":["lentils"],
		"source":"https://dal.example","total_time":30,"servings":2,"creator":{"id":"u5n1c8r3t6y0b2e","name":"Ana"},
		"review_count":1,"rating":{"overall":8,"taste":8,"ease":8,"instructions":8,"value":8},"cover":null,"favorite":true,
		"created":"2024-11-04T01:26:55.002Z","updated":"2024-11-04T01:26:55.002Z"}`), &sum); err != nil {
		t.Fatal(err)
	}
	c := core.NewBaseCollection("recipes")
	c.Fields.Add(&core.JSONField{Name: "ingredients"}, &core.JSONField{Name: "steps"})
	r := core.NewRecord(c)
	r.Set("ingredients", []string{"1 cup of lentils"})
	r.Set("steps", []string{"Boil."})
	want, _ := json.Marshal(sum)
	got, _ := json.Marshal(withDetails(sum, r))
	var all map[string]json.RawMessage
	if err := json.Unmarshal(got, &all); err != nil {
		t.Fatal(err)
	}
	if string(all["ingredients"]) != `["1 cup of lentils"]` || string(all["steps"]) != `["Boil."]` {
		t.Errorf("recipe %s, want the ingredients and the steps", got)
	}
	delete(all, "ingredients")
	delete(all, "steps")
	if rest, _ := json.Marshal(all); string(rest) != string(want) {
		t.Errorf("recipe %s, want the summary %s", rest, want)
	}
}

func TestSourceKey(t *testing.T) {
	t.Parallel()
	for source, want := range map[string]string{
		"":                                  "",
		"https://Recipes.Example/Dal/":      "https://recipes.example/Dal",
		"HTTP://recipes.example?id=Dal#top": "http://recipes.example?id=Dal",
		"https://recipes.example/dal#":      "https://recipes.example/dal",
		"https://recipes.example":           "https://recipes.example",
		"https://\u0130.example/P":          "https://\u0130.example/P",
	} {
		if got := sourceKey(source); got != want {
			t.Errorf("sourceKey(%q) is %q, want %q", source, got, want)
		}
	}
}

func TestRecipeQueriesUseIndexes(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	recipe := func(w *where) { w.idFilters("recipe_reviews", "recipe", new("c4r8e2p6i0s3t7a"), nil) }
	creator := func(w *where) { w.idFilters("recipe_reviews", "creator", new("u5n1c8r3t6y0b2e"), nil) }
	for _, c := range []struct {
		name, table, sort string
		sorts             map[string]sortField
		filter            func(w *where)
		want              string
	}{
		{"recipes -overall", "recipes", "-overall", recipeSorts, func(w *where) { w.joinOverall(recipeSubject) }, "idx_recipe_overall_desc"},
		{"recipes overall", "recipes", "overall", recipeSorts, func(w *where) { w.joinOverall(recipeSubject) }, "idx_recipe_overall_asc"},
		{"recipes by name", "recipes", "name", recipeSorts, nil, "idx_recipes_name"},
		{"recipes of a creator", "recipes", "-created", recipeSorts, func(w *where) { w.idFilters("recipes", "creator", new("u5n1c8r3t6y0b2e"), nil) }, "idx_recipes_creator_created"},
		{"recipes with a tag", "recipes", "-created", recipeSorts, func(w *where) { w.tagFilters(recipeSubject, new("dal"), nil, nil) }, "SEARCH recipe_tags USING PRIMARY KEY (tag=?)"},
		{"recipes of a list", "recipes", "-created", recipeSorts, func(w *where) { w.inList(recipeSubject, new("l7d2f9h4j1k8m3p")) }, "idx_list_recipes_list_recipe (list=?)"},
		{"reviews of a recipe", "recipe_reviews", "-overall", recipeSubject.reviewSorts, recipe, "idx_recipe_reviews_recipe_overall"},
		{"reviews of a recipe by created", "recipe_reviews", "-created", recipeSubject.reviewSorts, recipe, "idx_recipe_reviews_recipe_created"},
		{"reviews of a reviewer", "recipe_reviews", "-overall", recipeSubject.reviewSorts, creator, "idx_recipe_reviews_creator_overall"},
		{"all reviews", "recipe_reviews", "-overall", recipeSubject.reviewSorts, nil, "idx_recipe_reviews_overall"},
		{"all reviews by title", "recipe_reviews", "title", recipeSubject.reviewSorts, nil, "idx_recipe_reviews_title"},
		{"reviews of a recipe by date", "recipe_reviews", "-date", recipeSubject.reviewSorts, recipe, "idx_recipe_reviews_recipe_date"},
		{"reviews of a reviewer by date", "recipe_reviews", "-date", recipeSubject.reviewSorts, creator, "idx_recipe_reviews_creator_date"},
		{"all reviews by date", "recipe_reviews", "-date", recipeSubject.reviewSorts, nil, "idx_recipe_reviews_date"},
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

	t.Run("gallery", func(t *testing.T) {
		w := galleryWhere(recipeSubject, false, "c4r8e2p6i0s3t7a")
		for name, p := range map[string]string{
			"page":  queryPlan(t, s, s.sortQuery("recipe_review_photos", w, recipeSubject.galleryKeys, defaultLimit+1).Build()),
			"cover": queryPlan(t, s, s.coverQuery(recipeSubject, []string{"c4r8e2p6i0s3t7a"})),
		} {
			for _, want := range []string{"SEARCH recipe_reviews USING INDEX idx_recipe_reviews_recipe", "idx_recipe_review_photos_review_created (review=?)"} {
				if !strings.Contains(p, want) {
					t.Errorf("%s plan has no %q:\n%s", name, want, p)
				}
			}
		}
	})

	t.Run("favorites and list pages", func(t *testing.T) {
		w := listItemsWhere(recipeSubject, "l7d2f9h4j1k8m3p")
		for name, c := range map[string]struct {
			plan, want string
		}{
			"favorites": {queryPlan(t, s, s.favoritesQuery(recipeSubject, "u5n1c8r3t6y0b2e").Build()), "idx_lists_favorites"},
			"page":      {queryPlan(t, s, s.sortQuery("list_recipes", w, recipeSubject.listItemKeys, defaultLimit+1).Build()), "idx_list_recipes_list_created"},
		} {
			if !strings.Contains(c.plan, c.want) {
				t.Errorf("%s plan has no %q:\n%s", name, c.want, c.plan)
			}
		}
	})

	t.Run("reviewer stats", func(t *testing.T) {
		plan := queryPlan(t, s, s.reviewerStatsQuery("u5n1c8r3t6y0b2e"))
		for _, want := range []string{"idx_recipe_reviews_creator_overall", "COVERING INDEX idx_recipes_creator_created"} {
			if !strings.Contains(plan, want) {
				t.Errorf("plan has no %q:\n%s", want, plan)
			}
		}
	})
}
