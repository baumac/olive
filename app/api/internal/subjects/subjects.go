// Package subjects has the tables of each kind of thing that users review.
// The Public API and the search index use them. See decision 00004.
package subjects

// TagCollection has the tags of all kinds. The tags field of each parent is
// a relation to it. See decision 00011.
const TagCollection = "tags"

// Tables has the table names of one kind.
type Tables struct {
	// Table is the collection.
	Table string
	// Path is the first segment of the API and UI paths of a parent.
	Path string
	// Parent is the relation field of a review and of a list item.
	Parent    string
	Reviews   string
	Photos    string
	ListItems string
	// Versions has the history of each record of Table. See decision 00004.
	Versions string
	Overall  string
	// Tags has one row for each tag ID of each parent.
	Tags       string
	FTS        string
	ReviewsFTS string
	// IndexedFields are the text fields of Table in its FTS5 row. The row
	// also has the tags.
	IndexedFields []string
	// ListFields are the JSON arrays of strings of Table in its FTS5 row.
	ListFields []string
	// Scores are the score fields of a review. The overall is their
	// average.
	Scores []string
}

var Restaurants = Tables{
	Table:         "restaurants",
	Path:          "restaurants",
	Parent:        "restaurant",
	Reviews:       "reviews",
	Photos:        "review_photos",
	ListItems:     "list_items",
	Versions:      "restaurant_versions",
	Overall:       "restaurant_overall",
	Tags:          "restaurant_tags",
	FTS:           "restaurants_fts",
	ReviewsFTS:    "reviews_fts",
	IndexedFields: []string{"name", "description", "address"},
	Scores:        []string{"space", "service", "food", "price"},
}

var Recipes = Tables{
	Table:         "recipes",
	Path:          "recipes",
	Parent:        "recipe",
	Reviews:       "recipe_reviews",
	Photos:        "recipe_review_photos",
	ListItems:     "list_recipes",
	Versions:      "recipe_versions",
	Overall:       "recipe_overall",
	Tags:          "recipe_tags",
	FTS:           "recipes_fts",
	ReviewsFTS:    "recipe_reviews_fts",
	IndexedFields: []string{"name", "description"},
	ListFields:    []string{"ingredients"},
	Scores:        []string{"taste", "ease", "instructions", "value"},
}

// PantryItems are the wines, beers, cheeses, and chocolates of decision
// 00012.
var PantryItems = Tables{
	Table:         "pantry_items",
	Path:          "pantry",
	Parent:        "pantry_item",
	Reviews:       "pantry_item_reviews",
	Photos:        "pantry_item_review_photos",
	ListItems:     "list_pantry_items",
	Versions:      "pantry_item_versions",
	Overall:       "pantry_item_overall",
	Tags:          "pantry_item_tags",
	FTS:           "pantry_items_fts",
	ReviewsFTS:    "pantry_item_reviews_fts",
	IndexedFields: []string{"name", "description", "producer", "style", "origin"},
	Scores:        []string{"taste", "aroma", "texture", "value"},
}

// All has each kind. The search index adds its hooks for each one.
var All = []Tables{Restaurants, Recipes, PantryItems}
