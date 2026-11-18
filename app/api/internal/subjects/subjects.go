// Package subjects has the tables of each kind of thing that users review.
// The Public API and the search index use them. See decision 00047.
package subjects

// Tables has the table names of one kind.
type Tables struct {
	// Table is the collection. Its name is also the plural noun of the
	// problem details.
	Table string
	// Parent is the relation field of a review and of a list item.
	Parent     string
	Reviews    string
	Photos     string
	ListItems  string
	Overall    string
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
	Parent:        "restaurant",
	Reviews:       "reviews",
	Photos:        "review_photos",
	ListItems:     "list_items",
	Overall:       "restaurant_overall",
	Tags:          "restaurant_tags",
	FTS:           "restaurants_fts",
	ReviewsFTS:    "reviews_fts",
	IndexedFields: []string{"name", "description", "address"},
	Scores:        []string{"space", "service", "food", "price", "cleanliness"},
}

var Recipes = Tables{
	Table:         "recipes",
	Parent:        "recipe",
	Reviews:       "recipe_reviews",
	Photos:        "recipe_review_photos",
	ListItems:     "list_recipes",
	Overall:       "recipe_overall",
	Tags:          "recipe_tags",
	FTS:           "recipes_fts",
	ReviewsFTS:    "recipe_reviews_fts",
	IndexedFields: []string{"name", "description"},
	ListFields:    []string{"ingredients"},
	Scores:        []string{"taste", "ease", "instructions", "value"},
}

// All has each kind. The search index adds its hooks for each one.
var All = []Tables{Restaurants, Recipes}
