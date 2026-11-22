package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Triggers keep these tables equal to the records. Each restaurant has one
// restaurant_overall row. See decision 00007.
func init() {
	m.Register(func(app core.App) error {
		return execAll(app,
			// SQLite uses an index on columns for a row value range. It does
			// not use an index on expressions for it.
			"CREATE TABLE {{restaurant_overall}} ([[restaurant]] TEXT PRIMARY KEY NOT NULL, [[overall]] REAL, "+
				"[[unrated]] INTEGER AS ([[overall]] IS NULL), [[rated]] INTEGER AS ([[overall]] IS NOT NULL), "+
				"[[overall_or_0]] REAL AS (COALESCE([[overall]], 0))) WITHOUT ROWID",
			"CREATE INDEX {{idx_restaurant_overall_asc}} ON {{restaurant_overall}} ([[unrated]], [[overall_or_0]], [[restaurant]])",
			"CREATE INDEX {{idx_restaurant_overall_desc}} ON {{restaurant_overall}} ([[rated]], [[overall_or_0]], [[restaurant]])",
			"CREATE INDEX {{idx_restaurant_overall_value}} ON {{restaurant_overall}} ([[overall]])",
			"CREATE TABLE {{restaurant_tags}} ([[tag]] TEXT NOT NULL, [[restaurant]] TEXT NOT NULL, PRIMARY KEY ([[tag]], [[restaurant]])) WITHOUT ROWID",
			"CREATE INDEX {{idx_restaurant_tags_restaurant}} ON {{restaurant_tags}} ([[restaurant]])",

			"INSERT INTO {{restaurant_overall}} ([[restaurant]], [[overall]]) SELECT [[id]], "+overallOf("[[restaurants.id]]")+" FROM {{restaurants}}",
			"INSERT OR IGNORE INTO {{restaurant_tags}} ([[tag]], [[restaurant]]) SELECT [[value]], [[restaurants.id]] FROM {{restaurants}}, "+
				stringTags("[[restaurants.tags]]"),

			"CREATE TRIGGER {{restaurant_index_insert}} AFTER INSERT ON {{restaurants}} BEGIN "+
				"INSERT INTO {{restaurant_overall}} ([[restaurant]], [[overall]]) VALUES (NEW.[[id]], "+overallOf("NEW.[[id]]")+"); "+
				insertTags("NEW")+" END",
			"CREATE TRIGGER {{restaurant_index_update}} AFTER UPDATE OF [[tags]] ON {{restaurants}} WHEN OLD.[[tags]] IS NOT NEW.[[tags]] BEGIN "+
				"DELETE FROM {{restaurant_tags}} WHERE [[restaurant]] = OLD.[[id]]; "+
				insertTags("NEW")+" END",
			"CREATE TRIGGER {{restaurant_index_delete}} AFTER DELETE ON {{restaurants}} BEGIN "+
				"DELETE FROM {{restaurant_overall}} WHERE [[restaurant]] = OLD.[[id]]; "+
				"DELETE FROM {{restaurant_tags}} WHERE [[restaurant]] = OLD.[[id]]; END",
			"CREATE TRIGGER {{review_overall_insert}} AFTER INSERT ON {{reviews}} BEGIN "+updateOverall("NEW")+" END",
			"CREATE TRIGGER {{review_overall_update}} AFTER UPDATE OF "+strings.Join(fiveRestaurantScores.triggerColumns, ", ")+" ON {{reviews}} WHEN "+anyChanged(fiveRestaurantScores.triggerColumns)+" BEGIN "+
				updateOverall("NEW")+" "+updateOverall("OLD")+" END",
			"CREATE TRIGGER {{review_overall_delete}} AFTER DELETE ON {{reviews}} BEGIN "+updateOverall("OLD")+" END",
		)
	}, func(app core.App) error {
		return execAll(app,
			"DROP TRIGGER {{restaurant_index_insert}}",
			"DROP TRIGGER {{restaurant_index_update}}",
			"DROP TRIGGER {{restaurant_index_delete}}",
			"DROP TRIGGER {{review_overall_insert}}",
			"DROP TRIGGER {{review_overall_update}}",
			"DROP TRIGGER {{review_overall_delete}}",
			"DROP TABLE {{restaurant_overall}}",
			"DROP TABLE {{restaurant_tags}}",
		)
	})
}

// overallOf is the rounded overall average of the reviews of a restaurant.
// It is NULL without reviews.
func overallOf(restaurant string) string {
	return "(SELECT " + fiveRestaurantScores.overall() + " " +
		"FROM {{reviews}} WHERE [[reviews.restaurant]] = " + restaurant + ")"
}

// updateOverall changes no row for a deleted restaurant.
func updateOverall(row string) string {
	restaurant := row + ".[[restaurant]]"
	return "UPDATE {{restaurant_overall}} SET [[overall]] = " + overallOf(restaurant) + " WHERE [[restaurant]] = " + restaurant + ";"
}

// stringTags keeps only the strings of a JSON array. The API accepts only
// arrays of strings. The dashboard accepts all JSON.
func stringTags(tags string) string {
	return "json_each(CASE WHEN json_valid(" + tags + ") AND json_type(" + tags + ") = 'array' THEN " + tags + " ELSE '[]' END) WHERE [[type]] = 'text'"
}

func insertTags(row string) string {
	return "INSERT OR IGNORE INTO {{restaurant_tags}} ([[tag]], [[restaurant]]) SELECT [[value]], " + row + ".[[id]] FROM " +
		stringTags(row+".[[tags]]") + ";"
}

func anyChanged(columns []string) string {
	parts := make([]string, len(columns))
	for i, c := range columns {
		parts[i] = "OLD." + c + " IS NOT NEW." + c
	}
	return strings.Join(parts, " OR ")
}

func execAll(app core.App, queries ...string) error {
	for _, q := range queries {
		if _, err := app.DB().NewQuery(q).Execute(); err != nil {
			return err
		}
	}
	return nil
}
