package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// The patterns are the patterns of the Website, Phone, and OpeningHours
// schemas. The dashboard then cannot save a value that the API refuses.
const (
	hiddenCharacters    = `\x{00AD}\x{061C}\x{115F}\x{1160}\x{180E}\x{200B}\x{200E}\x{200F}\x{202A}-\x{202E}\x{2060}-\x{206F}\x{3164}\x{FEFF}\x{FFA0}\x{FFF9}-\x{FFFB}`
	websitePattern      = `^https?://[^\x00-\x20\x7F-\x9F` + hiddenCharacters + `/?#][^\x00-\x20\x7F-\x9F` + hiddenCharacters + `]*$`
	phonePattern        = `^\+?\(?[0-9](?:[ ()./-]*[0-9]){2,19}$`
	openingHoursPattern = `^[^\x00-\x08\x0B\x0C\x0E-\x1F\x7F-\x9F` + hiddenCharacters + `]*$`
)

// maxPhotoFileBytes is far above the largest JPEG that the Public API makes.
const maxPhotoFileBytes = 5 << 20

// newRestaurants requires the address of each location. A required number
// field refuses the valid coordinate 0. A price_level of 0 is "not set".
func newRestaurants(users *core.Collection) *core.Collection {
	c := core.NewBaseCollection("restaurants")
	c.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 255},
		&core.TextField{Name: "description", Max: 10000},
		&core.JSONField{Name: "tags"},
		&core.NumberField{Name: "latitude"},
		&core.NumberField{Name: "longitude"},
		&core.TextField{Name: "address", Required: true, Max: 255},
		&core.TextField{Name: "place_id", Max: 80},
		&core.TextField{Name: "website", Max: 2048, Pattern: websitePattern},
		&core.TextField{Name: "phone", Max: 40, Pattern: phonePattern},
		&core.TextField{Name: "opening_hours", Max: 500, Pattern: openingHoursPattern},
		&core.NumberField{Name: "price_level", OnlyInt: true, Min: types.Pointer(1.0), Max: types.Pointer(4.0)},
		&core.RelationField{Name: "creator", CollectionId: users.Id, MaxSelect: 1},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_restaurants_name", false, "name COLLATE NOCASE, id", "")
	c.AddIndex("idx_restaurants_created", false, "created, id", "")
	c.AddIndex("idx_restaurants_place_id", true, "place_id", "place_id != ''")
	c.AddIndex("idx_restaurants_latitude", false, "latitude, longitude", "")
	c.AddIndex("idx_restaurants_creator_created", false, "creator, created, id", "")
	return c
}

// newRecipes has the Website pattern of the spec for source. A record hook
// of the Public API sets source_key from source on each save.
func newRecipes(users *core.Collection) *core.Collection {
	c := core.NewBaseCollection("recipes")
	c.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 255},
		&core.TextField{Name: "description", Max: 10000},
		&core.JSONField{Name: "tags"},
		&core.JSONField{Name: "ingredients", Required: true},
		&core.JSONField{Name: "steps", Required: true},
		&core.TextField{Name: "source", Max: 2048, Pattern: websitePattern},
		&core.TextField{Name: "source_key", Max: 2048},
		&core.NumberField{Name: "total_time", OnlyInt: true, Min: types.Pointer(1.0), Max: types.Pointer(10080.0)},
		&core.NumberField{Name: "servings", OnlyInt: true, Min: types.Pointer(1.0), Max: types.Pointer(100.0)},
		&core.RelationField{Name: "creator", CollectionId: users.Id, MaxSelect: 1},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_recipes_name", false, "name COLLATE NOCASE, id", "")
	c.AddIndex("idx_recipes_created", false, "created, id", "")
	c.AddIndex("idx_recipes_creator_created", false, "creator, created, id", "")
	c.AddIndex("idx_recipes_source_key", true, "source_key", "source_key != ''")
	return c
}

// newLists has a unique index that keeps one favorites list for each user.
// See decision 00006.
func newLists(users *core.Collection) *core.Collection {
	c := core.NewBaseCollection("lists")
	c.Fields.Add(
		&core.RelationField{Name: "creator", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
		&core.TextField{Name: "name", Required: true, Max: 255},
		&core.SelectField{Name: "kind", Values: []string{"favorites", "custom"}, MaxSelect: 1, Required: true},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	c.AddIndex("idx_lists_creator_created", false, "creator, created, id", "")
	c.AddIndex("idx_lists_creator_name", true, "creator, name COLLATE NOCASE", "")
	c.AddIndex("idx_lists_favorites", true, "creator", "kind = 'favorites'")
	return c
}

// newReviews has no cascade delete of the subject. The Public API deletes
// the reviews of the creator with the subject. See decision 00004.
func (k subjectKind) newReviews(users, parent *core.Collection) *core.Collection {
	c := core.NewBaseCollection(k.reviews)
	c.Fields.Add(
		&core.RelationField{Name: k.parent, CollectionId: parent.Id, MaxSelect: 1, Required: true},
		&core.RelationField{Name: "creator", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
	)
	for _, name := range k.scores {
		c.Fields.Add(&core.NumberField{Name: name, OnlyInt: true, Required: true, Min: types.Pointer(1.0), Max: types.Pointer(10.0)})
	}
	c.Fields.Add(
		&core.TextField{Name: "title", Required: true, Max: 255},
		&core.TextField{Name: "feedback", Max: 10000},
		&core.TextField{Name: "date", Required: true, Min: 10, Max: 10, Pattern: `^\d{4}-\d{2}-\d{2}$`},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	prefix := "idx_" + k.reviews + "_"
	c.AddIndex(prefix+k.parent+"_created", false, k.parent+", created, id", "")
	c.AddIndex(prefix+"title", false, "title COLLATE NOCASE, id", "")
	c.AddIndex(prefix+"created", false, "created, id", "")
	c.AddIndex(prefix+k.parent+"_creator", false, k.parent+", creator", "")
	c.AddIndex(prefix+k.parent+"_date", false, k.parent+", date, created, id", "")
	c.AddIndex(prefix+"creator_date", false, "creator, date, created, id", "")
	c.AddIndex(prefix+"date", false, "date, created, id", "")
	return c
}

// newPhotos has no API rules. The protected file fields then need a
// superuser for each download under /api/files. See decision 00007.
func (k subjectKind) newPhotos(reviews *core.Collection) *core.Collection {
	c := core.NewBaseCollection(k.photos)
	c.Fields.Add(
		&core.RelationField{Name: "review", CollectionId: reviews.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
		&core.TextField{Name: "alt", Max: 255},
		&core.NumberField{Name: "width", OnlyInt: true, Required: true},
		&core.NumberField{Name: "height", OnlyInt: true, Required: true},
		photoFile("full"),
		photoFile("thumb"),
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	c.AddIndex("idx_"+k.photos+"_review_created", false, "review, created, id", "")
	return c
}

func photoFile(name string) *core.FileField {
	return &core.FileField{
		Name:      name,
		MaxSelect: 1,
		MaxSize:   maxPhotoFileBytes,
		MimeTypes: []string{"image/jpeg"},
		Protected: true,
		Required:  true,
	}
}

func (k subjectKind) newListItems(lists, parent *core.Collection) *core.Collection {
	c := core.NewBaseCollection(k.listItems)
	c.Fields.Add(
		&core.RelationField{Name: "list", CollectionId: lists.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
		&core.RelationField{Name: k.parent, CollectionId: parent.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	prefix := "idx_" + k.listItems + "_"
	c.AddIndex(prefix+"list_"+k.parent, true, "list, "+k.parent, "")
	c.AddIndex(prefix+"list_created", false, "list, created, "+k.parent, "")
	c.AddIndex(prefix+k.parent, false, k.parent+", list", "")
	return c
}

// newVersions has the history of decision 00004.
func (k subjectKind) newVersions(users, parent *core.Collection) *core.Collection {
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
	return c
}
