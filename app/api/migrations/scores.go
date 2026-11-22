package migrations

import "strconv"

// reviewScores makes the SQL of the overall of one set of review scores.
type reviewScores struct {
	// triggerColumns are the parent column and the score columns. SQLite
	// refuses to drop them while the triggers of decision 00005 exist.
	triggerColumns []string
	names          []string
	sum            string
	count          string
}

func newReviewScores(parent string, scores ...string) reviewScores {
	s := reviewScores{triggerColumns: []string{"[[" + parent + "]]"}, names: scores, count: strconv.Itoa(len(scores))}
	for i, name := range scores {
		if i > 0 {
			s.sum += " + "
		}
		s.sum += "[[" + name + "]]"
		s.triggerColumns = append(s.triggerColumns, "[["+name+"]]")
	}
	return s
}

// fiveRestaurantScores are the scores of restaurant reviews before migration
// 1794268822. Only the migrations before it and its down use them.
var fiveRestaurantScores = newReviewScores("restaurant", "space", "service", "food", "price", "cleanliness")

// column is the generated overall column of decision 00005. The division in
// a double gives the value of the responses.
func (s reviewScores) column() string {
	return "((" + s.sum + ") / " + s.count + ".0)"
}

// overall is the rounded average of the reviews of a parent. It is NULL
// without reviews.
func (s reviewScores) overall() string {
	return "ROUND(SUM(" + s.sum + ") * 1.0 / (" + s.count + " * COUNT(*)), 1)"
}

// ranking is the Bayesian average of decision 00005. It is NULL without
// reviews.
func (s reviewScores) ranking() string {
	return "(" + rankingPriorReviews + " * " + rankingPriorScore + " + SUM(" + s.sum + ") / " + s.count + ".0) / (" +
		rankingPriorReviews + " + COUNT(*))"
}
