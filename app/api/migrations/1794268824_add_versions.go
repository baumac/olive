package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// versionKind has the SQL of the baseline snapshot of one parent
// collection. The history package makes the same snapshot. See decision
// 00006.
type versionKind struct {
	parent   string
	table    string
	versions string
	values   []string
}

func optionalText(name string) string {
	return "'" + name + "', NULLIF([[" + name + "]], '')"
}

func optionalNumber(name string) string {
	return "'" + name + "', NULLIF([[" + name + "]], 0)"
}

// optionalList is null for a field that is not a JSON array.
func optionalList(name string) string {
	return "'" + name + "', CASE WHEN json_valid([[" + name + "]]) AND json_type([[" + name + "]]) = 'array' THEN json([[" + name + "]]) END"
}

func list(name string) string {
	return "'" + name + "', CASE WHEN json_valid([[" + name + "]]) AND json_type([[" + name + "]]) = 'array' THEN json([[" + name + "]]) ELSE json('[]') END"
}

// location is null for a restaurant without an address. Such a restaurant is
// from before the location field.
const location = "'location', CASE WHEN [[address]] != '' THEN " +
	"CASE WHEN [[place_id]] != '' " +
	"THEN json_object('latitude', [[latitude]], 'longitude', [[longitude]], 'address', [[address]], 'place_id', [[place_id]]) " +
	"ELSE json_object('latitude', [[latitude]], 'longitude', [[longitude]], 'address', [[address]]) END END"

var versionKinds = []versionKind{
	{
		parent: "restaurant", table: "restaurants", versions: "restaurant_versions",
		values: []string{
			"'name', [[name]]", optionalText("description"), optionalList("tags"), location,
			optionalText("website"), optionalText("phone"), optionalText("opening_hours"), optionalNumber("price_level"),
		},
	},
	{
		parent: "recipe", table: "recipes", versions: "recipe_versions",
		values: []string{
			"'name', [[name]]", optionalText("description"), optionalList("tags"), list("ingredients"), list("steps"),
			optionalText("source"), optionalNumber("total_time"), optionalNumber("servings"),
		},
	},
}

// Each restaurant and recipe gets a baseline version with its snapshot. The
// history of the next change starts from it. See decision 00006.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		for _, k := range versionKinds {
			parent, err := app.FindCollectionByNameOrId(k.table)
			if err != nil {
				return err
			}
			c := core.NewBaseCollection(k.versions)
			c.Fields.Add(
				&core.RelationField{Name: k.parent, CollectionId: parent.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
				&core.NumberField{Name: "number", Required: true, OnlyInt: true, Min: types.Pointer(1.0)},
				&core.SelectField{Name: "action", Values: []string{"baseline", "create", "update"}, MaxSelect: 1, Required: true},
				&core.SelectField{Name: "source", Values: []string{"user", "admin"}, MaxSelect: 1, Required: true},
				&core.RelationField{Name: "editor", CollectionId: users.Id, MaxSelect: 1},
				&core.JSONField{Name: "changed"},
				&core.JSONField{Name: "snapshot", Required: true},
				&core.AutodateField{Name: "created", OnCreate: true},
			)
			c.AddIndex("idx_"+k.versions+"_number", true, k.parent+", number", "")
			c.AddIndex("idx_"+k.versions+"_editor", false, "editor", "")
			if err := app.Save(c); err != nil {
				return err
			}
			// The id column has a default value.
			if err := execAll(app, "INSERT INTO {{"+k.versions+"}} ([["+k.parent+"]], [[number]], [[action]], [[source]], [[editor]], [[changed]], [[snapshot]], [[created]]) "+
				"SELECT [[id]], 1, 'baseline', 'admin', '', '[]', json_object("+strings.Join(k.values, ", ")+"), [[updated]] FROM {{"+k.table+"}}"); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for _, k := range versionKinds {
			c, err := app.FindCollectionByNameOrId(k.versions)
			if err != nil {
				return err
			}
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		return nil
	})
}
