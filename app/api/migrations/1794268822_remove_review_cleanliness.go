package migrations

import (
	"errors"
	"slices"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

var fourRestaurantScores = newReviewScores("restaurant", "space", "service", "food", "price")

// Restaurant reviews lose the cleanliness score. The overall is the average
// of the other four scores. See decision 00052.
func init() {
	m.Register(func(app core.App) error {
		if err := execAll(app, dropReviewOverall()...); err != nil {
			return err
		}
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		reviews.Fields.RemoveByName("cleanliness")
		if err := app.Save(reviews); err != nil {
			return err
		}
		return execAll(app, makeReviewOverall(fourRestaurantScores)...)
	}, func(app core.App) error {
		if err := execAll(app, dropReviewOverall()...); err != nil {
			return err
		}
		reviews, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		price := slices.IndexFunc(reviews.Fields, func(f core.Field) bool { return f.GetName() == "price" })
		if price < 0 {
			return errors.New("the reviews collection has no price field")
		}
		cleanliness := score("cleanliness", 10)
		reviews.Fields.AddAt(price+1, cleanliness)
		if err := app.Save(reviews); err != nil {
			return err
		}
		// Migration 1794268804 gave the old reviews the same value.
		if err := execAll(app, "UPDATE {{reviews}} SET [[cleanliness]] = [[space]]"); err != nil {
			return err
		}
		cleanliness.Required = true
		if err := app.Save(reviews); err != nil {
			return err
		}
		return execAll(app, makeReviewOverall(fiveRestaurantScores)...)
	})
}

// dropReviewOverall drops the generated overall column of decision 00031.
// It also drops the triggers of decision 00012 that use the scores.
func dropReviewOverall() []string {
	return append(dropOverallSort(),
		"DROP INDEX {{idx_reviews_restaurant_overall}}",
		"DROP INDEX {{idx_reviews_creator_overall}}",
		"DROP INDEX {{idx_reviews_overall}}",
		"ALTER TABLE {{reviews}} DROP COLUMN [[overall]]",
	)
}

// makeReviewOverall also computes each restaurant_overall row again.
func makeReviewOverall(scores reviewScores) []string {
	values := scores.overall() + ", " + scores.ranking()
	return append([]string{
		"ALTER TABLE {{reviews}} ADD COLUMN [[overall]] REAL AS " + scores.column(),
		"CREATE INDEX {{idx_reviews_restaurant_overall}} ON {{reviews}} ([[restaurant]], [[overall]], [[id]])",
		"CREATE INDEX {{idx_reviews_creator_overall}} ON {{reviews}} ([[creator]], [[overall]], [[id]])",
		"CREATE INDEX {{idx_reviews_overall}} ON {{reviews}} ([[overall]], [[id]])",
		"UPDATE {{restaurant_overall}} SET ([[overall]], [[ranking]]) = (" +
			selectOfReviews(values, "[[restaurant_overall.restaurant]]") + ")",
	}, makeOverallSort(scores, "ranking_or_0", "[[overall]], [[ranking]]", values)...)
}
