package migrations

import (
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The rating of a restaurant or a recipe is the average of its reviewers.
// The overall sort uses this value and then the number of reviewers. See
// decision 00005.
func init() {
	kinds := []overallKind{restaurantOverall, recipeOverall}
	m.Register(func(app core.App) error {
		var queries []string
		for _, k := range kinds {
			queries = append(queries, k.dropSort()...)
			queries = append(queries,
				k.alter("DROP COLUMN [[ranking_or_0]]"),
				k.alter("DROP COLUMN [[ranking]]"),
				k.alter("ADD COLUMN [[reviewers]] INTEGER NOT NULL DEFAULT 0"),
				k.alter("ADD COLUMN [[review_count]] INTEGER NOT NULL DEFAULT 0"),
			)
			for _, name := range k.scores.names {
				queries = append(queries, k.alter("ADD COLUMN [["+name+"]] REAL"))
			}
			queries = append(queries,
				k.alter("ADD COLUMN [[overall_or_0]] REAL AS (COALESCE([[overall]], 0))"),
				"UPDATE {{"+k.table+"}} SET ("+k.columns()+") = ("+k.byReviewer("", k.row())+")",
			)
			queries = append(queries, k.makeSort()...)
		}
		return execAll(app, queries...)
	}, func(app core.App) error {
		var queries []string
		for _, k := range kinds {
			queries = append(queries, k.dropSort()...)
			queries = append(queries,
				k.alter("DROP COLUMN [[overall_or_0]]"),
				k.alter("DROP COLUMN [[review_count]]"),
				k.alter("DROP COLUMN [[reviewers]]"),
			)
			for _, name := range k.scores.names {
				queries = append(queries, k.alter("DROP COLUMN [["+name+"]]"))
			}
			queries = append(queries,
				k.alter("ADD COLUMN [[ranking]] REAL"),
				k.alter("ADD COLUMN [[ranking_or_0]] REAL AS (COALESCE([[ranking]], 0))"),
				"UPDATE {{"+k.table+"}} SET ([[overall]], [[ranking]]) = ("+k.ranking(k.row())+")",
			)
			queries = append(queries, k.makeRankingSort()...)
		}
		return execAll(app, queries...)
	})
}

// overallKind has the names of the overall table of one kind of parent.
type overallKind struct {
	table          string
	parent         string
	parents        string
	reviews        string
	reviewTriggers string
	scores         reviewScores
	insertTags     func(row string) string
	// ranking and makeRankingSort make the table of decision 00005 again.
	ranking         func(parent string) string
	makeRankingSort func() []string
}

var (
	restaurantOverall = overallKind{
		table: "restaurant_overall", parent: "restaurant", parents: "restaurants", reviews: "reviews",
		reviewTriggers: "review_overall", scores: fourRestaurantScores, insertTags: insertTags,
		ranking: func(parent string) string {
			return selectOfReviews(fourRestaurantScores.overall()+", "+fourRestaurantScores.ranking(), parent)
		},
		makeRankingSort: func() []string {
			return makeOverallSort(fourRestaurantScores, "ranking_or_0", "[[overall]], [[ranking]]",
				fourRestaurantScores.overall()+", "+fourRestaurantScores.ranking())
		},
	}
	recipeOverall = overallKind{
		table: "recipe_overall", parent: "recipe", parents: "recipes", reviews: "recipe_reviews",
		reviewTriggers: "recipe_review_overall", scores: recipeScores, insertTags: insertRecipeTags,
		ranking:         func(parent string) string { return selectOfRecipeReviews(recipeRankingValues, parent) },
		makeRankingSort: makeRecipeRankingSort,
	}
)

func (k overallKind) alter(change string) string {
	return "ALTER TABLE {{" + k.table + "}} " + change
}

func (k overallKind) row() string {
	return "[[" + k.table + "." + k.parent + "]]"
}

// columns are the stored values of byReviewer in the same order.
func (k overallKind) columns() string {
	columns := []string{"[[overall]]", "[[reviewers]]", "[[review_count]]"}
	for _, name := range k.scores.names {
		columns = append(columns, "[["+name+"]]")
	}
	return strings.Join(columns, ", ")
}

// byReviewer selects the values of columns for one parent. Each creator
// counts once with the average of the own reviews.
func (k overallKind) byReviewer(first, parent string) string {
	outer := []string{"ROUND(AVG([[reviewer_overall]]), 1)", "COUNT(*)", "COALESCE(SUM([[reviewer_reviews]]), 0)"}
	inner := []string{
		"SUM(" + k.scores.sum + ") * 1.0 / (" + k.scores.count + " * COUNT(*)) AS [[reviewer_overall]]",
		"COUNT(*) AS [[reviewer_reviews]]",
	}
	for _, name := range k.scores.names {
		outer = append(outer, "ROUND(AVG([["+name+"]]), 1)")
		inner = append(inner, "AVG([["+name+"]]) AS [["+name+"]]")
	}
	return "SELECT " + first + strings.Join(outer, ", ") + " FROM (SELECT " + strings.Join(inner, ", ") +
		" FROM {{" + k.reviews + "}} WHERE [[" + k.reviews + "." + k.parent + "]] = " + parent + " GROUP BY [[creator]])"
}

func (k overallKind) dropSort() []string {
	return []string{
		"DROP TRIGGER {{" + k.parent + "_index_insert}}",
		"DROP TRIGGER {{" + k.reviewTriggers + "_insert}}",
		"DROP TRIGGER {{" + k.reviewTriggers + "_update}}",
		"DROP TRIGGER {{" + k.reviewTriggers + "_delete}}",
		"DROP INDEX {{idx_" + k.table + "_asc}}",
		"DROP INDEX {{idx_" + k.table + "_desc}}",
	}
}

// makeSort watches the creator too. A change of the creator moves a review
// to another reviewer.
func (k overallKind) makeSort() []string {
	watched := append(slices.Clone(k.scores.triggerColumns), "[[creator]]")
	update := func(row string) string {
		parent := row + ".[[" + k.parent + "]]"
		return "UPDATE {{" + k.table + "}} SET (" + k.columns() + ") = (" + k.byReviewer("", parent) + ") WHERE [[" + k.parent + "]] = " + parent + ";"
	}
	sortColumns := "[[overall_or_0]], [[reviewers]], [[" + k.parent + "]]"
	return []string{
		"CREATE INDEX {{idx_" + k.table + "_asc}} ON {{" + k.table + "}} ([[unrated]], " + sortColumns + ")",
		"CREATE INDEX {{idx_" + k.table + "_desc}} ON {{" + k.table + "}} ([[rated]], " + sortColumns + ")",
		"CREATE TRIGGER {{" + k.parent + "_index_insert}} AFTER INSERT ON {{" + k.parents + "}} BEGIN " +
			"INSERT INTO {{" + k.table + "}} ([[" + k.parent + "]], " + k.columns() + ") " + k.byReviewer("NEW.[[id]], ", "NEW.[[id]]") + "; " +
			k.insertTags("NEW") + " END",
		"CREATE TRIGGER {{" + k.reviewTriggers + "_insert}} AFTER INSERT ON {{" + k.reviews + "}} BEGIN " + update("NEW") + " END",
		"CREATE TRIGGER {{" + k.reviewTriggers + "_update}} AFTER UPDATE OF " + strings.Join(watched, ", ") +
			" ON {{" + k.reviews + "}} WHEN " + anyChanged(watched) + " BEGIN " + update("NEW") + " " + update("OLD") + " END",
		"CREATE TRIGGER {{" + k.reviewTriggers + "_delete}} AFTER DELETE ON {{" + k.reviews + "}} BEGIN " + update("OLD") + " END",
	}
}
