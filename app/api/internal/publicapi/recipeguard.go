package publicapi

import (
	"encoding/json"
	"slices"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// recipeLists are the JSON fields of a recipe. The guard checks their stored
// JSON.
var recipeLists = []string{"tags", "ingredients", "steps"}

var recipeGuard = fieldGuard{
	collection: "recipes",
	noun:       "recipe",
	schema:     createSchema("RecipeCreate"),
	properties: recipeProperties,
	set: func(r *core.Record, name string, value json.RawMessage) error {
		if slices.Contains(recipeLists, name) {
			r.Set(name, types.JSONRaw(value))
			return nil
		}
		return setText(r, name, value)
	},
}

// recipeProperties gets the properties that a response would have. The JSON
// fields are the stored JSON. A stored string or object then gets an error.
func recipeProperties(r *core.Record, schema *openapi3.Schema) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(toRecipe(r, nil, nil, nil))
	if err != nil {
		return nil, err
	}
	all, err := decodeObject(raw)
	if err != nil {
		return nil, err
	}
	for _, name := range recipeLists {
		all[name] = nil
		if v, ok := r.Get(name).(types.JSONRaw); ok {
			all[name] = json.RawMessage(v)
		}
	}
	props := map[string]json.RawMessage{}
	for name, value := range all {
		if _, ok := schema.Properties[name]; ok && len(value) > 0 && string(value) != "null" {
			props[name] = value
		}
	}
	return props, nil
}
