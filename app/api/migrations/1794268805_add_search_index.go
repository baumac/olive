package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The searchindex package keeps these tables equal to the records. The
// tokenizer ignores case and accents. The prefix indexes serve prefix terms.
func init() {
	m.Register(func(app core.App) error {
		for _, q := range []string{
			"CREATE VIRTUAL TABLE {{restaurants_fts}} USING fts5(id UNINDEXED, name, description, tags, tokenize = 'unicode61 remove_diacritics 2', prefix = '2 3')",
			"CREATE VIRTUAL TABLE {{reviews_fts}} USING fts5(id UNINDEXED, title, feedback, tags, restaurant_name, tokenize = 'unicode61 remove_diacritics 2', prefix = '2 3')",
			"INSERT INTO {{restaurants_fts}} (id, name, description, tags) SELECT [[id]], [[name]], [[description]], " + tagText("restaurants") + " FROM {{restaurants}}",
			"INSERT INTO {{reviews_fts}} (id, title, feedback, tags, restaurant_name) SELECT [[reviews.id]], [[reviews.title]], [[reviews.feedback]], " + tagText("reviews") +
				", [[restaurants.name]] FROM {{reviews}} JOIN {{restaurants}} ON [[restaurants.id]] = [[reviews.restaurant]]",
		} {
			if _, err := app.DB().NewQuery(q).Execute(); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for _, q := range []string{"DROP TABLE {{restaurants_fts}}", "DROP TABLE {{reviews_fts}}"} {
			if _, err := app.DB().NewQuery(q).Execute(); err != nil {
				return err
			}
		}
		return nil
	})
}

// tagText joins the tags of a row with spaces, like the searchindex package.
func tagText(table string) string {
	return "COALESCE((SELECT group_concat(value, ' ') FROM json_each(CASE WHEN json_valid([[" + table + ".tags]]) THEN [[" + table + ".tags]] ELSE '[]' END)), '')"
}
