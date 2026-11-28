package publicapi

import (
	"encoding/json"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pocketbase/pocketbase/core"
)

var pantryItemGuard = fieldGuard{
	collection: "pantry_items",
	noun:       "pantry item",
	schema:     createSchema("PantryItemCreate"),
	properties: pantryItemProperties,
	set:        setText,
}

// pantryItemProperties gets the properties that a response would have. The
// tags are IDs of existing tags. PocketBase checks them.
func pantryItemProperties(r *core.Record, schema *openapi3.Schema) (map[string]json.RawMessage, error) {
	return responseProperties(toPantryItem(r, nil, nil, nil, viewerData{}), schema)
}
