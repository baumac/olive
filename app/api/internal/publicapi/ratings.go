package publicapi

import (
	"math"
	"slices"

	reviewsapi "github.com/baumac/olive/specs/api/sdk/server/go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// overallColumn is NULL for a parent without reviews. It needs joinOverall.
// SQLite triggers keep the overall table equal to the records. See decision
// 00005.
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
	Restaurant string  `db:"restaurant"`
	Count      int     `db:"count"`
	Overall    float64 `db:"overall"`
	Space      float64 `db:"space"`
	Service    float64 `db:"service"`
	Food       float64 `db:"food"`
	Price      float64 `db:"price"`
}

func (s *Server) ratings(restaurants ...*core.Record) (map[string]restaurantRating, error) {
	return ratingsOf(s, restaurantSubject, restaurants, func(r restaurantRating) string { return r.Restaurant })
}

// ratingsOf reads the stored rating of each parent. A parent without
// reviews has no entry. R has a db field for each score and for the parent,
// count, and overall columns.
func ratingsOf[R any](s *Server, sub subject, parents []*core.Record, parentOf func(R) string) (map[string]R, error) {
	if len(parents) == 0 {
		return map[string]R{}, nil
	}
	ids := make([]any, len(parents))
	for i, r := range parents {
		ids[i] = r.Id
	}
	var rows []R
	if err := s.ratingsQuery(sub, ids).All(&rows); err != nil {
		return nil, err
	}
	out := make(map[string]R, len(rows))
	for _, row := range rows {
		out[parentOf(row)] = row
	}
	return out, nil
}

// ratingsQuery reads the values that the triggers of decision 00005 store.
func (s *Server) ratingsQuery(sub subject, ids []any) *dbx.Query {
	columns := []string{column(sub.Overall, sub.Parent), column(sub.Overall, "review_count") + " AS [[count]]", sub.overallColumn()}
	for _, name := range sub.Scores {
		columns = append(columns, column(sub.Overall, name))
	}
	return s.app.DB().
		Select(columns...).
		From(sub.Overall).
		Where(dbx.In(column(sub.Overall, sub.Parent), ids...)).
		AndWhere(dbx.NewExp(column(sub.Overall, "rated"))).
		Build()
}

func (r restaurantRating) toAPI() reviewsapi.RestaurantRating {
	return reviewsapi.RestaurantRating{
		Overall: r.Overall, Space: r.Space, Service: r.Service,
		Food: r.Food, Price: r.Price,
	}
}

// overallSort puts parents without reviews last in both directions. Its
// keys are the columns of one index with one direction. See decision 00005.
func (sub subject) overallSort(desc bool) []sortKey {
	missing := column(sub.Overall, "unrated")
	if desc {
		missing = column(sub.Overall, "rated")
	}
	return []sortKey{
		{expr: missing, desc: desc},
		{expr: column(sub.Overall, "overall_or_0"), desc: desc},
		{expr: column(sub.Overall, "reviewers"), desc: desc},
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
