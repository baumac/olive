// Package history keeps a version of each create and each change of a
// restaurant or a recipe. See decision 00055.
package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/baumac/reviews-app/app/api/internal/recordvalues"
	"github.com/baumac/reviews-app/app/api/internal/subjects"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The values of the action field of a version.
const (
	ActionBaseline = "baseline"
	ActionCreate   = "create"
	ActionUpdate   = "update"
)

// The values of the source field of a version.
const (
	SourceUser  = "user"
	SourceAdmin = "admin"
)

type editorKey struct{}

// WithEditor sets the user of the saves that use ctx. A save without it
// gets the source admin.
func WithEditor(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, editorKey{}, userID)
}

func editorOf(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(editorKey{}).(string)
	return id
}

// Property is one property of the snapshot of a version. Its value has the
// JSON shape of the create body.
type Property struct {
	Name  string
	value func(r *core.Record) any
}

// Kind has the versions table and the snapshot properties of one parent
// collection.
type Kind struct {
	subjects.Tables
	Properties []Property
}

var (
	// Restaurants has the properties of RestaurantCreate.
	Restaurants = Kind{subjects.Restaurants, []Property{
		text("name"), optionalText("description"), optionalList("tags"), {"location", location},
		optionalText("website"), optionalText("phone"), optionalText("opening_hours"), optionalNumber("price_level"),
	}}
	// Recipes has the properties of RecipeCreate.
	Recipes = Kind{subjects.Recipes, []Property{
		text("name"), optionalText("description"), optionalList("tags"), list("ingredients"), list("steps"),
		optionalText("source"), optionalNumber("total_time"), optionalNumber("servings"),
	}}
	kinds = []Kind{Restaurants, Recipes}
)

func text(name string) Property {
	return Property{name, func(r *core.Record) any { return r.GetString(name) }}
}

func optionalText(name string) Property {
	return Property{name, func(r *core.Record) any { return recordvalues.OptionalText(r, name) }}
}

func optionalNumber(name string) Property {
	return Property{name, func(r *core.Record) any { return recordvalues.OptionalNumber(r, name) }}
}

func optionalList(name string) Property {
	return Property{name, func(r *core.Record) any { return recordvalues.OptionalList(r, name) }}
}

func list(name string) Property {
	return Property{name, func(r *core.Record) any { return recordvalues.List(r, name) }}
}

func location(r *core.Record) any {
	return recordvalues.LocationOf(r)
}

// Snapshot gets the values of the properties of k from r. The values have
// the types of a JSON decode. They then compare with a stored snapshot.
func (k Kind) Snapshot(r *core.Record) (map[string]any, error) {
	values := make(map[string]any, len(k.Properties))
	for _, p := range k.Properties {
		values[p.Name] = p.value(r)
	}
	b, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	return out, json.Unmarshal(b, &out)
}

// Changed gets the names of the properties that differ between two
// snapshots in the order of k.Properties.
func (k Kind) Changed(before, after map[string]any) []string {
	changed := []string{}
	for _, p := range k.Properties {
		if !reflect.DeepEqual(before[p.Name], after[p.Name]) {
			changed = append(changed, p.Name)
		}
	}
	return changed
}

// Register adds the record hooks. Each hook runs the save and the version
// write in one transaction. Saves in the dashboard and in migrations also
// run them.
func Register(app core.App) {
	names := make([]string, len(kinds))
	for i, k := range kinds {
		names[i] = k.Table
	}
	app.OnRecordCreateExecute(names...).BindFunc(func(e *core.RecordEvent) error {
		return inTransaction(e, ActionCreate)
	})
	app.OnRecordUpdateExecute(names...).BindFunc(func(e *core.RecordEvent) error {
		return inTransaction(e, ActionUpdate)
	})
}

// inTransaction gets the newest snapshot before the save. PocketBase starts
// no transaction for a save without one.
func inTransaction(e *core.RecordEvent, action string) error {
	k, err := kindOf(e.Record)
	if err != nil {
		return err
	}
	app := e.App
	defer func() { e.App = app }()
	return app.RunInTransaction(func(tx core.App) error {
		e.App = tx
		var last *version
		if action == ActionUpdate {
			if last, err = newest(tx, k, e.Record.Id); err != nil {
				return err
			}
			if last == nil {
				if last, err = writeBaseline(tx, k, e.Record.Id); err != nil {
					return err
				}
			}
		}
		if err := e.Next(); err != nil {
			return err
		}
		return write(tx, k, e.Context, e.Record, action, last)
	})
}

func kindOf(r *core.Record) (Kind, error) {
	for _, k := range kinds {
		if r.Collection().Name == k.Table {
			return k, nil
		}
	}
	return Kind{}, errors.New("history: the " + r.Collection().Name + " collection has no history")
}

type version struct {
	number   int
	snapshot map[string]any
}

func newest(app core.App, k Kind, parentID string) (*version, error) {
	var row struct {
		Number   int    `db:"number"`
		Snapshot string `db:"snapshot"`
	}
	err := app.DB().Select("number", "snapshot").From(k.Versions).
		Where(dbx.HashExp{k.Parent: parentID}).OrderBy("number DESC").Limit(1).One(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := &version{number: row.Number}
	return v, json.Unmarshal([]byte(row.Snapshot), &v.snapshot)
}

// writeBaseline saves the stored values of a parent that has no version.
// A migration that inserts a parent with SQL makes such a parent.
func writeBaseline(app core.App, k Kind, parentID string) (*version, error) {
	stored, err := app.FindRecordById(k.Table, parentID)
	if err != nil {
		return nil, err
	}
	snapshot, err := k.Snapshot(stored)
	if err != nil {
		return nil, err
	}
	v := &version{number: 1, snapshot: snapshot}
	return v, save(app, k, parentID, v, ActionBaseline, "", []string{})
}

func write(app core.App, k Kind, ctx context.Context, r *core.Record, action string, last *version) error {
	snapshot, err := k.Snapshot(r)
	if err != nil {
		return err
	}
	next := &version{number: 1, snapshot: snapshot}
	var changed []string
	if last == nil {
		changed = k.Changed(map[string]any{}, snapshot)
	} else {
		next.number = last.number + 1
		changed = k.Changed(last.snapshot, snapshot)
		if len(changed) == 0 {
			return nil
		}
	}
	return save(app, k, r.Id, next, action, editorOf(ctx), changed)
}

func save(app core.App, k Kind, parentID string, v *version, action, editor string, changed []string) error {
	c, err := app.FindCachedCollectionByNameOrId(k.Versions)
	if err != nil {
		return err
	}
	source := SourceAdmin
	if editor != "" {
		source = SourceUser
	}
	r := core.NewRecord(c)
	r.Set(k.Parent, parentID)
	r.Set("number", v.number)
	r.Set("action", action)
	r.Set("source", source)
	r.Set("editor", editor)
	r.Set("changed", changed)
	r.Set("snapshot", v.snapshot)
	return app.Save(r)
}
