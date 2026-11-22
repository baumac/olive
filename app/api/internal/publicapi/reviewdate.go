package publicapi

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pocketbase/pocketbase/core"
)

const reviewDateLayout = "2006-01-02"

const firstReviewDate = "1900-01-01"

// latestZone is UTC+14. A user in any time zone can then give the local date
// of today. See decision 00005.
var latestZone = time.FixedZone("UTC+14", 14*60*60)

// checkReviewDate checks the review-date format of the spec. The pattern of
// the spec checks the form.
func checkReviewDate(s string) error {
	if _, err := time.Parse(reviewDateLayout, s); err != nil {
		return errors.New("not a date")
	}
	if s < firstReviewDate || s > time.Now().In(latestZone).Format(reviewDateLayout) {
		return errors.New("out of range")
	}
	return nil
}

// reviewDateGuard gives the date of a save in the dashboard the checks of the
// Public API.
func reviewDateGuard(sub subject, schema string) fieldGuard {
	return fieldGuard{
		collection: sub.Reviews,
		noun:       sub.noun + " review",
		schema:     createSchema(schema),
		properties: func(r *core.Record, _ *openapi3.Schema) (map[string]json.RawMessage, error) {
			date := r.GetString("date")
			if date == "" {
				return map[string]json.RawMessage{}, nil
			}
			raw, err := json.Marshal(date)
			return map[string]json.RawMessage{"date": raw}, err
		},
		set: setText,
	}
}
