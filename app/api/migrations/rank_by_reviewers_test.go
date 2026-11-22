package migrations

import (
	"math"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
)

const rankByReviewersFile = "1794268823_rank_by_the_average_of_the_reviewers.go"

func TestRankByReviewersComputesExistingRows(t *testing.T) {
	t.Parallel()
	app := newIndexApp(t)
	redo := downTo(t, app, rankByReviewersFile)
	ana := newRecord(t, app, "users", map[string]any{"email": "ana@example.com", "password": "correct-horse", "name": "Ana"})
	joe := newRecord(t, app, "users", map[string]any{"email": "joe@example.com", "password": "correct-horse", "name": "Joe"})
	rated := newRecord(t, app, "restaurants", map[string]any{"name": "Rated"})
	unrated := newRecord(t, app, "restaurants", map[string]any{"name": "Unrated"})
	dal := newRecord(t, app, "recipes", map[string]any{"name": "Dal", "ingredients": []string{"lentils"}, "steps": []string{"Boil."}})
	for range 3 {
		newRecord(t, app, "reviews", review(ana, rated, scores(10)))
	}
	newRecord(t, app, "reviews", review(joe, rated, scores(6)))
	newRecord(t, app, "recipe_reviews", recipeReview(ana, dal, 10))
	newRecord(t, app, "recipe_reviews", recipeReview(ana, dal, 10))
	newRecord(t, app, "recipe_reviews", recipeReview(joe, dal, 2))
	wantOverall(t, app, rated.Id, 9)
	wantRanking(t, app, rated.Id, (5*5.5+36)/9)

	redo()
	wantOverall(t, app, rated.Id, 8)
	wantReviewers(t, app, rated.Id, 2)
	wantNoOverall(t, app, unrated.Id, "up")
	wantReviewers(t, app, unrated.Id, 0)
	// Ana has 8.5 twice. Joe has 6.5.
	if overall, reviewers, _ := storedRecipeOverall(t, app, dal.Id); overall.Float64 != 7.5 || reviewers != 2 {
		t.Errorf("recipe up: overall %v, reviewers %d", overall, reviewers)
	}
	added := newRecord(t, app, "restaurants", map[string]any{"name": "Added"})
	wantNoOverall(t, app, added.Id, "new restaurant")
	wantReviewers(t, app, added.Id, 0)

	downTo(t, app, rankByReviewersFile)
	wantOverall(t, app, rated.Id, 9)
	wantRanking(t, app, rated.Id, (5*5.5+36)/9)
	wantNoRanking(t, app, unrated.Id, "down")
	var ranking float64
	err := app.DB().Select("ranking").From("recipe_overall").Where(dbx.HashExp{"recipe": dal.Id}).Row(&ranking)
	if err != nil || math.Abs(ranking-(5*5.5+8.5+8.5+6.5)/8) > 1e-9 {
		t.Errorf("recipe down: ranking %v, error %v", ranking, err)
	}
	newRecord(t, app, "reviews", review(joe, unrated, scores(4)))
	wantRanking(t, app, unrated.Id, (5*5.5+4)/6)
}

func TestRatingOfReviewersUsesTheCreatorIndex(t *testing.T) {
	t.Parallel()
	app := newIndexApp(t)
	for _, c := range []struct {
		kind overallKind
		want string
	}{
		{restaurantOverall, "SEARCH reviews USING INDEX idx_reviews_restaurant_creator (restaurant=?)"},
		{recipeOverall, "SEARCH recipe_reviews USING INDEX idx_recipe_reviews_recipe_creator (recipe=?)"},
	} {
		var rows []struct {
			Detail string `db:"detail"`
		}
		q := app.DB().NewQuery("EXPLAIN QUERY PLAN " + c.kind.byReviewer("", "{:id}")).Bind(dbx.Params{"id": "k3x9m2p7q1w8z4a"})
		if err := q.All(&rows); err != nil {
			t.Fatal(err)
		}
		var plan []string
		for _, r := range rows {
			plan = append(plan, r.Detail)
		}
		if got := strings.Join(plan, "\n"); !strings.Contains(got, c.want) {
			t.Errorf("%s plan has no %q:\n%s", c.kind.table, c.want, got)
		}
	}
}
