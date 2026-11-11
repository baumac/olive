package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Reviews have no tags. See decision 00011. FTS5 cannot drop a column. The
// migration makes reviews_fts again. Down cannot restore the deleted tags.
func init() {
	m.Register(func(app core.App) error {
		if err := rebuildReviewIndex(app, false); err != nil {
			return err
		}
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		reviews.Fields.RemoveByName("tags")
		return app.Save(reviews)
	}, func(app core.App) error {
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		at := len(reviews.Fields)
		for i, f := range reviews.Fields {
			if f.GetName() == "feedback" {
				at = i + 1
			}
		}
		reviews.Fields.AddAt(at, &core.JSONField{Name: "tags"})
		if err := app.Save(reviews); err != nil {
			return err
		}
		return rebuildReviewIndex(app, true)
	})
}

// rebuildReviewIndex makes reviews_fts again and fills it from the records.
// The tags column needs the tags field of reviews.
func rebuildReviewIndex(app core.App, withTags bool) error {
	columns, values := "id, title, feedback, restaurant_name", "[[reviews.id]], [[reviews.title]], [[reviews.feedback]], [[restaurants.name]]"
	if withTags {
		columns = "id, title, feedback, tags, restaurant_name"
		values = "[[reviews.id]], [[reviews.title]], [[reviews.feedback]], " + tagText("reviews") + ", [[restaurants.name]]"
	}
	for _, q := range []string{
		"DROP TABLE {{reviews_fts}}",
		"CREATE VIRTUAL TABLE {{reviews_fts}} USING fts5(id UNINDEXED" + strings.TrimPrefix(columns, "id") + ", tokenize = 'unicode61 remove_diacritics 2', prefix = '2 3')",
		"INSERT INTO {{reviews_fts}} (" + columns + ") SELECT " + values +
			" FROM {{reviews}} JOIN {{restaurants}} ON [[restaurants.id]] = [[reviews.restaurant]]",
	} {
		if _, err := app.DB().NewQuery(q).Execute(); err != nil {
			return err
		}
	}
	return nil
}
