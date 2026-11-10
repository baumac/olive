package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// maxPhotoFileBytes is far above the largest JPEG that the Public API makes.
const maxPhotoFileBytes = 5 << 20

// The review_photos collection has no API rules. The protected file fields
// then need a superuser for each download under /api/files.
func init() {
	m.Register(func(app core.App) error {
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection("review_photos")
		c.Fields.Add(
			&core.RelationField{Name: "review", CollectionId: reviews.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.TextField{Name: "alt", Max: 255},
			&core.NumberField{Name: "width", OnlyInt: true, Required: true},
			&core.NumberField{Name: "height", OnlyInt: true, Required: true},
			photoFile("full"),
			photoFile("thumb"),
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		c.AddIndex("idx_review_photos_review_created", false, "review, created, id", "")
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("review_photos")
		if err != nil {
			return err
		}
		return app.Delete(c)
	})
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
