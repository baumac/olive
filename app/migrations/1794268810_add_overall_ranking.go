package migrations

import (
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The overall sort uses the ranking column. The overall column stays the
// value of the responses. See decision 00029.
func init() {
	m.Register(func(app core.App) error {
		return execAll(app, slices.Concat(
			dropOverallSort(),
			[]string{
				"ALTER TABLE {{restaurant_overall}} DROP COLUMN [[overall_or_0]]",
				"ALTER TABLE {{restaurant_overall}} ADD COLUMN [[ranking]] REAL",
				"ALTER TABLE {{restaurant_overall}} ADD COLUMN [[ranking_or_0]] REAL AS (COALESCE([[ranking]], 0))",
				"UPDATE {{restaurant_overall}} SET [[ranking]] = (" + selectOfReviews(rankingAverage, "[[restaurant_overall.restaurant]]") + ")",
			},
			makeOverallSort("ranking_or_0", "[[overall]], [[ranking]]", overallAverage+", "+rankingAverage),
		)...)
	}, func(app core.App) error {
		return execAll(app, slices.Concat(
			dropOverallSort(),
			[]string{
				"ALTER TABLE {{restaurant_overall}} DROP COLUMN [[ranking_or_0]]",
				"ALTER TABLE {{restaurant_overall}} DROP COLUMN [[ranking]]",
				"ALTER TABLE {{restaurant_overall}} ADD COLUMN [[overall_or_0]] REAL AS (COALESCE([[overall]], 0))",
			},
			makeOverallSort("overall_or_0", "[[overall]]", overallAverage),
		)...)
	})
}

// A restaurant starts with rankingPriorReviews reviews of rankingPriorScore.
// A change of these values needs a new migration that computes all rows again.
const (
	rankingPriorReviews = "5"
	rankingPriorScore   = "5.5"
)

// rankingAverage is NULL without reviews.
const rankingAverage = "(" + rankingPriorReviews + " * " + rankingPriorScore +
	" + " + scoreSum + " / 5.0) / (" + rankingPriorReviews + " + COUNT(*))"

func selectOfReviews(values, restaurant string) string {
	return "SELECT " + values + " FROM {{reviews}} WHERE [[reviews.restaurant]] = " + restaurant
}

// SQLite refuses to drop a column that an index or a trigger uses.
func dropOverallSort() []string {
	return []string{
		"DROP TRIGGER {{restaurant_index_insert}}",
		"DROP TRIGGER {{review_overall_insert}}",
		"DROP TRIGGER {{review_overall_update}}",
		"DROP TRIGGER {{review_overall_delete}}",
		"DROP INDEX {{idx_restaurant_overall_asc}}",
		"DROP INDEX {{idx_restaurant_overall_desc}}",
	}
}

// makeOverallSort needs one aggregate of the reviews in values for each
// column of columns.
func makeOverallSort(sortColumn, columns, values string) []string {
	update := func(row string) string {
		restaurant := row + ".[[restaurant]]"
		return "UPDATE {{restaurant_overall}} SET (" + columns + ") = (" + selectOfReviews(values, restaurant) + ") WHERE [[restaurant]] = " + restaurant + ";"
	}
	return []string{
		"CREATE INDEX {{idx_restaurant_overall_asc}} ON {{restaurant_overall}} ([[unrated]], [[" + sortColumn + "]], [[restaurant]])",
		"CREATE INDEX {{idx_restaurant_overall_desc}} ON {{restaurant_overall}} ([[rated]], [[" + sortColumn + "]], [[restaurant]])",
		"CREATE TRIGGER {{restaurant_index_insert}} AFTER INSERT ON {{restaurants}} BEGIN " +
			"INSERT INTO {{restaurant_overall}} ([[restaurant]], " + columns + ") " +
			selectOfReviews("NEW.[[id]], "+values, "NEW.[[id]]") + "; " +
			insertTags("NEW") + " END",
		"CREATE TRIGGER {{review_overall_insert}} AFTER INSERT ON {{reviews}} BEGIN " + update("NEW") + " END",
		"CREATE TRIGGER {{review_overall_update}} AFTER UPDATE OF " + strings.Join(overallColumns, ", ") + " ON {{reviews}} WHEN " + anyChanged(overallColumns) + " BEGIN " +
			update("NEW") + " " + update("OLD") + " END",
		"CREATE TRIGGER {{review_overall_delete}} AFTER DELETE ON {{reviews}} BEGIN " + update("OLD") + " END",
	}
}
