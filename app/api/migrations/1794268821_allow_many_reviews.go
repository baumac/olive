package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// reviewKind has the names of one review table of this migration.
type reviewKind struct {
	reviews, parent, fts string
}

var reviewKinds = []reviewKind{
	{reviews: "reviews", parent: "restaurant", fts: "reviews_fts"},
	{reviews: "recipe_reviews", parent: "recipe", fts: "recipe_reviews_fts"},
}

// A review is one visit or one cook. A user can have many reviews of one
// subject. See decision 00050.
func init() {
	m.Register(func(app core.App) error {
		for _, k := range reviewKinds {
			if err := allowManyReviews(app, k); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for _, k := range reviewKinds {
			if err := allowOneReview(app, k); err != nil {
				return err
			}
		}
		return nil
	})
}

func allowManyReviews(app core.App, k reviewKind) error {
	c, err := app.FindCollectionByNameOrId(k.reviews)
	if err != nil {
		return err
	}
	c.Fields.Add(&core.TextField{Name: "date", Required: true, Min: 10, Max: 10, Pattern: `^\d{4}-\d{2}-\d{2}$`})
	c.AddIndex(creatorIndexName(k), false, k.parent+", creator", "")
	c.AddIndex("idx_"+k.reviews+"_"+k.parent+"_date", false, k.parent+", date, created, id", "")
	c.AddIndex("idx_"+k.reviews+"_creator_date", false, "creator, date, created, id", "")
	c.AddIndex("idx_"+k.reviews+"_date", false, "date, created, id", "")
	if err := app.Save(c); err != nil {
		return err
	}
	// The created column has the UTC timestamp of PocketBase, such as
	// 2024-11-06 19:02:11.120Z.
	return execAll(app, "UPDATE {{"+k.reviews+"}} SET [[date]] = substr([[created]], 1, 10)")
}

// allowOneReview keeps the review with the latest date of each user and
// subject. It deletes the other reviews and their photos.
func allowOneReview(app core.App, k reviewKind) error {
	var ids []string
	err := app.DB().NewQuery("SELECT [[r.id]] FROM {{" + k.reviews + "}} r WHERE EXISTS (" +
		"SELECT 1 FROM {{" + k.reviews + "}} o WHERE [[o." + k.parent + "]] = [[r." + k.parent + "]] AND [[o.creator]] = [[r.creator]]" +
		" AND ([[o.date]], [[o.created]], [[o.id]]) > ([[r.date]], [[r.created]], [[r.id]]))").Column(&ids)
	if err != nil {
		return err
	}
	for _, id := range ids {
		r, err := app.FindRecordById(k.reviews, id)
		if err != nil {
			return err
		}
		if err := app.Delete(r); err != nil {
			return err
		}
		// The hooks of searchindex can be off in a migration.
		if _, err := app.DB().Delete(k.fts, dbx.HashExp{"id": id}).Execute(); err != nil {
			return err
		}
	}

	c, err := app.FindCollectionByNameOrId(k.reviews)
	if err != nil {
		return err
	}
	c.RemoveIndex("idx_" + k.reviews + "_" + k.parent + "_date")
	c.RemoveIndex("idx_" + k.reviews + "_creator_date")
	c.RemoveIndex("idx_" + k.reviews + "_date")
	c.Fields.RemoveByName("date")
	c.AddIndex(creatorIndexName(k), true, k.parent+", creator", "")
	return app.Save(c)
}

func creatorIndexName(k reviewKind) string {
	return "idx_" + k.reviews + "_" + k.parent + "_creator"
}
