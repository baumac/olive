package publicapi

import (
	"github.com/pocketbase/dbx"
	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

// guardReviewOverall rejects a change of the reviews collection that SQLite
// refuses because of the generated overall column. See decision 00007.
func guardReviewOverall(app core.App, sub subject) {
	app.OnCollectionValidate(sub.Reviews).BindFunc(func(e *core.CollectionEvent) error {
		var generated bool
		err := e.App.DB().NewQuery("SELECT EXISTS (SELECT 1 FROM pragma_table_xinfo({:table}) WHERE [[name]] = 'overall' AND [[hidden]] != 0)").
			Bind(dbx.Params{"table": sub.Reviews}).Row(&generated)
		if err != nil {
			return err
		}
		if !generated {
			return e.Next()
		}
		if e.Collection.Fields.GetByName("overall") != nil {
			return reviewSchemaError("The " + sub.Reviews + " table already has a generated overall column. Use a different field name.")
		}
		for _, name := range sub.Scores {
			if e.Collection.Fields.GetByName(name) == nil {
				return reviewSchemaError("The generated overall column of the " + sub.Reviews + " table uses the " + name +
					" field. A migration must drop the column first. See decision 00007.")
			}
		}
		return e.Next()
	})
}

func reviewSchemaError(message string) error {
	return validation.Errors{"fields": validation.NewError("validation_review_overall", message)}
}
