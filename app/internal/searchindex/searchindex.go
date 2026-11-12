// Package searchindex keeps the FTS5 tables of restaurants and reviews equal
// to the records.
package searchindex

import (
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The FTS5 tables. Their id column has the record ID.
const (
	Restaurants = "restaurants_fts"
	Reviews     = "reviews_fts"
)

// Register adds record hooks to app. Each hook runs the save or delete and
// the index change in one transaction. Saves in the PocketBase dashboard and
// cascade deletes also run them.
func Register(app core.App) {
	app.OnRecordCreateExecute("restaurants", "reviews").BindFunc(func(e *core.RecordEvent) error {
		return inTransaction(e, nil, func(tx core.App) error {
			return Put(tx, e.Record)
		})
	})
	app.OnRecordUpdateExecute("restaurants", "reviews").BindFunc(func(e *core.RecordEvent) error {
		// Original keeps the loaded values after a save of the same record.
		// The stored record has the values of the index row.
		var stored *core.Record
		readStored := func(tx core.App) (err error) {
			stored, err = tx.FindRecordById(e.Record.Collection(), e.Record.Id)
			return err
		}
		return inTransaction(e, readStored, func(tx core.App) error {
			if !changed(stored, e.Record, indexedFields(e.Record)...) {
				return nil
			}
			if err := Put(tx, e.Record); err != nil {
				return err
			}
			// The API and the dashboard can change the name of a restaurant
			// that has reviews.
			if tableOf(e.Record) == Restaurants && changed(stored, e.Record, "name") {
				return renameInReviews(tx, e.Record)
			}
			return nil
		})
	})
	app.OnRecordDeleteExecute("restaurants", "reviews").BindFunc(func(e *core.RecordEvent) error {
		return inTransaction(e, nil, func(tx core.App) error {
			return remove(tx, tableOf(e.Record), e.Record.Id)
		})
	})
}

// inTransaction runs before, the next hooks, and index in one transaction.
// PocketBase starts no transaction for a save without one. A failed index
// change then also removes the record change.
func inTransaction(e *core.RecordEvent, before, index func(tx core.App) error) error {
	app := e.App
	defer func() { e.App = app }()
	return app.RunInTransaction(func(tx core.App) error {
		e.App = tx
		if before != nil {
			if err := before(tx); err != nil {
				return err
			}
		}
		if err := e.Next(); err != nil {
			return err
		}
		return index(tx)
	})
}

// indexedFields are the record fields that the index row uses.
func indexedFields(r *core.Record) []string {
	if tableOf(r) == Restaurants {
		return []string{"name", "description", "tags"}
	}
	return []string{"title", "feedback", "restaurant"}
}

func changed(original, r *core.Record, fields ...string) bool {
	for _, f := range fields {
		if original.GetString(f) != r.GetString(f) {
			return true
		}
	}
	return false
}

func tableOf(r *core.Record) string {
	if r.Collection().Name == "restaurants" {
		return Restaurants
	}
	return Reviews
}

// Put replaces the index row of a restaurant or a review. A failed restaurant
// lookup keeps the old row.
func Put(app core.App, r *core.Record) error {
	table := tableOf(r)
	row := dbx.Params{"id": r.Id}
	if table == Restaurants {
		row["tags"] = tagText(r)
		row["name"] = r.GetString("name")
		row["description"] = r.GetString("description")
	} else {
		restaurant, err := app.FindRecordById("restaurants", r.GetString("restaurant"))
		if err != nil {
			return err
		}
		row["title"] = r.GetString("title")
		row["feedback"] = r.GetString("feedback")
		row["restaurant_name"] = restaurant.GetString("name")
	}
	if err := remove(app, table, r.Id); err != nil {
		return err
	}
	_, err := app.DB().Insert(table, row).Execute()
	return err
}

func renameInReviews(app core.App, restaurant *core.Record) error {
	_, err := app.DB().NewQuery(
		"UPDATE {{" + Reviews + "}} SET [[restaurant_name]] = {:name} WHERE [[id]] IN (SELECT [[id]] FROM {{reviews}} WHERE [[restaurant]] = {:id})",
	).Bind(dbx.Params{"name": restaurant.GetString("name"), "id": restaurant.Id}).Execute()
	return err
}

func remove(app core.App, table, id string) error {
	_, err := app.DB().Delete(table, dbx.HashExp{"id": id}).Execute()
	return err
}

// tagText joins the tags with spaces. The tokenizer splits a tag such as
// date-night into two words.
func tagText(r *core.Record) string {
	var tags []string
	if err := r.UnmarshalJSONField("tags", &tags); err != nil {
		return ""
	}
	return strings.Join(tags, " ")
}
