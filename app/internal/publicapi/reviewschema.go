package publicapi

import (
	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

var overallScores = []string{"space", "service", "food", "price", "cleanliness"}

// guardReviewOverall rejects a change of the reviews collection that SQLite
// refuses because of the generated overall column. See decision 00031.
func guardReviewOverall(app core.App) {
	app.OnCollectionValidate("reviews").BindFunc(func(e *core.CollectionEvent) error {
		var generated bool
		err := e.App.DB().NewQuery("SELECT EXISTS (SELECT 1 FROM pragma_table_xinfo('reviews') WHERE [[name]] = 'overall' AND [[hidden]] != 0)").Row(&generated)
		if err != nil {
			return err
		}
		if !generated {
			return e.Next()
		}
		if e.Collection.Fields.GetByName("overall") != nil {
			return reviewSchemaError("The reviews table already has a generated overall column. Use a different field name.")
		}
		for _, name := range overallScores {
			if e.Collection.Fields.GetByName(name) == nil {
				return reviewSchemaError("The generated overall column of the reviews table uses the " + name +
					" field. A migration must drop the column first. See decision 00031.")
			}
		}
		return e.Next()
	})
}

func reviewSchemaError(message string) error {
	return validation.Errors{"fields": validation.NewError("validation_review_overall", message)}
}
