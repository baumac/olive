// Package searchindex keeps the FTS5 tables of the reviewed things and their
// reviews equal to the records.
package searchindex

import (
	"errors"
	"slices"
	"strings"

	"github.com/baumac/reviews-app/app/api/internal/subjects"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The FTS5 tables of restaurants and their reviews. Their id column has the
// record ID.
var (
	Restaurants = subjects.Restaurants.FTS
	Reviews     = subjects.Restaurants.ReviewsFTS
)

// Register adds record hooks to app. Each hook runs the save or delete and
// the index change in one transaction. Saves in the PocketBase dashboard and
// cascade deletes also run them.
func Register(app core.App) {
	var collections []string
	for _, k := range subjects.All {
		collections = append(collections, k.Table, k.Reviews)
	}
	app.OnRecordCreateExecute(collections...).BindFunc(func(e *core.RecordEvent) error {
		return inTransaction(e, nil, func(tx core.App) error {
			return Put(tx, e.Record)
		})
	})
	app.OnRecordUpdateExecute(collections...).BindFunc(func(e *core.RecordEvent) error {
		// Original keeps the loaded values after a save of the same record.
		// The stored record has the values of the index row.
		var stored *core.Record
		readStored := func(tx core.App) (err error) {
			stored, err = tx.FindRecordById(e.Record.Collection(), e.Record.Id)
			return err
		}
		return inTransaction(e, readStored, func(tx core.App) error {
			k, isParent, err := kindOf(e.Record)
			if err != nil {
				return err
			}
			if !changed(stored, e.Record, indexedFields(k, isParent)...) {
				return nil
			}
			if err := Put(tx, e.Record); err != nil {
				return err
			}
			// The API and the dashboard can change the name of a parent
			// that has reviews.
			if isParent && changed(stored, e.Record, "name") {
				return renameInReviews(tx, k, e.Record)
			}
			return nil
		})
	})
	app.OnRecordDeleteExecute(collections...).BindFunc(func(e *core.RecordEvent) error {
		return inTransaction(e, nil, func(tx core.App) error {
			k, isParent, err := kindOf(e.Record)
			if err != nil {
				return err
			}
			return remove(tx, tableOf(k, isParent), e.Record.Id)
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
func indexedFields(k subjects.Tables, isParent bool) []string {
	if isParent {
		return append(slices.Clone(k.IndexedFields), "tags")
	}
	return []string{"title", "feedback", k.Parent}
}

func changed(original, r *core.Record, fields ...string) bool {
	for _, f := range fields {
		if original.GetString(f) != r.GetString(f) {
			return true
		}
	}
	return false
}

// kindOf returns the kind of a parent or a review record. The bool is false
// for a review.
func kindOf(r *core.Record) (subjects.Tables, bool, error) {
	name := r.Collection().Name
	for _, k := range subjects.All {
		if name == k.Table {
			return k, true, nil
		}
		if name == k.Reviews {
			return k, false, nil
		}
	}
	return subjects.Tables{}, false, errors.New("searchindex: the " + name + " collection has no index")
}

func tableOf(k subjects.Tables, isParent bool) string {
	if isParent {
		return k.FTS
	}
	return k.ReviewsFTS
}

// Put replaces the index row of a parent or a review. A failed parent lookup
// keeps the old row.
func Put(app core.App, r *core.Record) error {
	k, isParent, err := kindOf(r)
	if err != nil {
		return err
	}
	row := dbx.Params{"id": r.Id}
	if isParent {
		row["tags"] = tagText(r)
		for _, f := range k.IndexedFields {
			row[f] = r.GetString(f)
		}
	} else {
		parent, err := app.FindRecordById(k.Table, r.GetString(k.Parent))
		if err != nil {
			return err
		}
		row["title"] = r.GetString("title")
		row["feedback"] = r.GetString("feedback")
		row[k.Parent+"_name"] = parent.GetString("name")
	}
	table := tableOf(k, isParent)
	if err := remove(app, table, r.Id); err != nil {
		return err
	}
	_, err = app.DB().Insert(table, row).Execute()
	return err
}

func renameInReviews(app core.App, k subjects.Tables, parent *core.Record) error {
	_, err := app.DB().NewQuery(
		"UPDATE {{" + k.ReviewsFTS + "}} SET [[" + k.Parent + "_name]] = {:name} WHERE [[id]] IN (SELECT [[id]] FROM {{" + k.Reviews + "}} WHERE [[" + k.Parent + "]] = {:id})",
	).Bind(dbx.Params{"name": parent.GetString("name"), "id": parent.Id}).Execute()
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
