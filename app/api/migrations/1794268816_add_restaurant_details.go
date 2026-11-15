package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
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

// A price_level of 0 is "not set". See decision 00038.
func init() {
	m.Register(func(app core.App) error {
		restaurants, err := app.FindCollectionByNameOrId("restaurants")
		if err != nil {
			return err
		}
		at := len(restaurants.Fields)
		for i, f := range restaurants.Fields {
			if f.GetName() == "place_id" {
				at = i + 1
			}
		}
		restaurants.Fields.AddAt(at,
			&core.TextField{Name: "website", Max: 2048, Pattern: websitePattern},
			&core.TextField{Name: "phone", Max: 40, Pattern: phonePattern},
			&core.TextField{Name: "opening_hours", Max: 500, Pattern: openingHoursPattern},
			&core.NumberField{Name: "price_level", OnlyInt: true, Min: types.Pointer(1.0), Max: types.Pointer(4.0)},
		)
		return app.Save(restaurants)
	}, func(app core.App) error {
		restaurants, err := app.FindCollectionByNameOrId("restaurants")
		if err != nil {
			return err
		}
		for _, name := range []string{"website", "phone", "opening_hours", "price_level"} {
			restaurants.Fields.RemoveByName(name)
		}
		return app.Save(restaurants)
	})
}
