package publicapi

import (
	"fmt"
	"math"
	"slices"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// overallColumn is NULL for a parent without reviews. It needs joinOverall.
// SQLite triggers keep the overall table equal to the records. See decision
// 00012.
func (sub subject) overallColumn() string {
	return column(sub.Overall, "overall")
}

// joinOverall reads the stored averages. Each parent has a row. The inner
// join lets SQLite read the parents in the order of an overall index.
func (w *where) joinOverall(sub subject) {
	w.joins = append(w.joins, join{
		inner: true,
		table: "{{" + sub.Overall + "}}",
		on:    dbx.NewExp(column(sub.Overall, sub.Parent) + " = " + column(sub.Table, "id")),
	})
}

type restaurantRating struct {
	Restaurant  string  `db:"restaurant"`
	Count       int     `db:"count"`
	Overall     float64 `db:"overall"`
	Space       float64 `db:"space"`
	Service     float64 `db:"service"`
	Food        float64 `db:"food"`
	Price       float64 `db:"price"`
	Cleanliness float64 `db:"cleanliness"`
}

func (s *Server) ratings(restaurants ...*core.Record) (map[string]restaurantRating, error) {
	return ratingsOf(s, restaurantSubject, restaurants, func(r restaurantRating) string { return r.Restaurant })
}

// ratingsOf runs one query for all parents. A parent without reviews has no
// entry. The overall is the stored value. The overall filter then uses the
// value that the response shows. R has a db field for each score and for
// the parent, count, and overall columns.
func ratingsOf[R any](s *Server, sub subject, parents []*core.Record, parentOf func(R) string) (map[string]R, error) {
	if len(parents) == 0 {
		return map[string]R{}, nil
	}
	ids := make([]any, len(parents))
	for i, r := range parents {
		ids[i] = r.Id
	}
	parent := column(sub.Reviews, sub.Parent)
	columns := []string{parent + " AS [[" + sub.Parent + "]]", "COUNT(*) AS [[count]]", "MAX(" + sub.overallColumn() + ") AS [[overall]]"}
	for _, name := range sub.scores {
		columns = append(columns, fmt.Sprintf("ROUND(AVG([[%s]]), 1) AS [[%s]]", name, name))
	}
	var rows []R
	err := s.app.DB().
		Select(columns...).
		From(sub.Reviews).
		LeftJoin(sub.Overall, dbx.NewExp(column(sub.Overall, sub.Parent)+" = "+parent)).
		Where(dbx.In(sub.Reviews+"."+sub.Parent, ids...)).
		GroupBy(sub.Reviews + "." + sub.Parent).
		All(&rows)
	if err != nil {
		return nil, err
	}
	out := make(map[string]R, len(rows))
	for _, row := range rows {
		out[parentOf(row)] = row
	}
	return out, nil
}

func (r restaurantRating) toAPI() reviewsapi.RestaurantRating {
	return reviewsapi.RestaurantRating{
		Overall: r.Overall, Space: r.Space, Service: r.Service,
		Food: r.Food, Price: r.Price, Cleanliness: r.Cleanliness,
	}
}

// overallSort puts parents without reviews last in both directions. Its
// first key puts them last. All keys have one direction. They are the
// columns of one index of the migrations. The ranking is the Bayesian average
// of decision 00029.
func (sub subject) overallSort(desc bool) []sortKey {
	missing := column(sub.Overall, "unrated")
	if desc {
		missing = column(sub.Overall, "rated")
	}
	return []sortKey{
		{expr: missing, desc: desc},
		{expr: column(sub.Overall, "ranking_or_0"), desc: desc},
		{expr: column(sub.Overall, sub.Parent), desc: desc, unique: true},
	}
}

// overallDetail returns the detail of a 400 for a NaN value. The spec
// validation can be off. The fingerprint cannot marshal NaN.
func overallDetail(r *reviewsapi.AverageRange) string {
	if slices.ContainsFunc(averageComparisons(r), func(c comparison) bool { return math.IsNaN(c.value.(float64)) }) {
		return "The overall filter values must be numbers."
	}
	return ""
}

func averageComparisons(r *reviewsapi.AverageRange) []comparison {
	if r == nil {
		return nil
	}
	return comparisons(r.Lt, r.Lte, r.Eq, r.Gte, r.Gt)
}
