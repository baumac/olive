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

// Register adds record hooks to app. Each hook changes the index in the
// transaction of the save or delete. Saves in the PocketBase dashboard and
// cascade deletes also run them.
func Register(app core.App) {
	app.OnRecordCreateExecute("restaurants", "reviews").BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		return Put(e.App, e.Record)
	})
	app.OnRecordUpdateExecute("restaurants", "reviews").BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		if !changed(e.Record, indexedFields(e.Record)...) {
			return nil
		}
		if err := Put(e.App, e.Record); err != nil {
			return err
		}
		// The API keeps the name of a restaurant with reviews. A superuser
		// can still change it in the dashboard.
		if tableOf(e.Record) == Restaurants && changed(e.Record, "name") {
			return renameInReviews(e.App, e.Record)
		}
		return nil
	})
	app.OnRecordDeleteExecute("restaurants", "reviews").BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		return remove(e.App, tableOf(e.Record), e.Record.Id)
	})
}

// indexedFields are the record fields that the index row uses.
func indexedFields(r *core.Record) []string {
	if tableOf(r) == Restaurants {
		return []string{"name", "description", "tags"}
	}
	return []string{"title", "feedback", "tags", "restaurant"}
}

func changed(r *core.Record, fields ...string) bool {
	for _, f := range fields {
		if r.Original().GetString(f) != r.GetString(f) {
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

// Put replaces the index row of a restaurant or a review.
func Put(app core.App, r *core.Record) error {
	table := tableOf(r)
	if err := remove(app, table, r.Id); err != nil {
		return err
	}
	row := dbx.Params{
		"id":   r.Id,
		"tags": tagText(r),
	}
	if table == Restaurants {
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
