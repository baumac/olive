package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// stepSpace has the characters that the API removes around a step.
const stepSpace = "' ' || char(9) || char(10) || char(13)"

// storedSteps is an empty array for steps that are not a JSON array. A save
// without validation can store them. json_each then gets valid JSON.
const storedSteps = "CASE WHEN json_valid([[steps]]) AND json_type([[steps]]) = 'array' THEN [[steps]] ELSE '[]' END"

// The API removes the ASCII spaces, tabs, and line breaks around each step of
// a request. The stored steps get the same rule. See decision 00001.
func init() {
	m.Register(func(app core.App) error {
		return execAll(app,
			"UPDATE {{recipes}} SET [[steps]] = ("+
				"SELECT json_group_array(TRIM([[value]], "+stepSpace+") ORDER BY [[key]]) FROM json_each("+storedSteps+")"+
				") WHERE EXISTS ("+
				"SELECT 1 FROM json_each("+storedSteps+") WHERE [[value]] != TRIM([[value]], "+stepSpace+"))"+
				// A step that is not text keeps the array as it is. TRIM would
				// change it to text.
				" AND NOT EXISTS (SELECT 1 FROM json_each("+storedSteps+") WHERE [[type]] != 'text')",
		)
	}, func(core.App) error {
		return nil
	})
}
