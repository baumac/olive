package publicapi

import (
	"fmt"
	"math"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// SQLite rounds all averages. The overall filter and sort then compare the
// values that the responses show.
const overallAverage = "ROUND(SUM([[space]] + [[service]] + [[food]] + [[price]] + [[cleanliness]]) * 1.0 / (5 * COUNT(*)), 1)"

// overallColumn is NULL for a restaurant without reviews. It needs
// joinOverall.
const overallColumn = "[[overall_of.overall]]"

// joinOverall computes the averages of all restaurants in one grouped scan of
// reviews. A correlated subquery runs again for each use in a row.
func (w *where) joinOverall() {
	w.joins = append(w.joins, join{
		table: "(SELECT [[restaurant]], " + overallAverage + " AS [[overall]] FROM {{reviews}} GROUP BY [[restaurant]]) [[overall_of]]",
		on:    dbx.NewExp("[[overall_of.restaurant]] = [[restaurants.id]]"),
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

// ratings runs one query for all restaurants. A restaurant without reviews
// has no entry.
func (s *Server) ratings(restaurants ...*core.Record) (map[string]restaurantRating, error) {
	if len(restaurants) == 0 {
		return map[string]restaurantRating{}, nil
	}
	ids := make([]any, len(restaurants))
	for i, r := range restaurants {
		ids[i] = r.Id
	}
	columns := []string{"[[restaurant]]", "COUNT(*) AS [[count]]", overallAverage + " AS [[overall]]"}
	for _, name := range []string{"space", "service", "food", "price", "cleanliness"} {
		columns = append(columns, fmt.Sprintf("ROUND(AVG([[%s]]), 1) AS [[%s]]", name, name))
	}
	var rows []restaurantRating
	err := s.app.DB().
		Select(columns...).
		From("reviews").
		Where(dbx.In("restaurant", ids...)).
		GroupBy("restaurant").
		All(&rows)
	if err != nil {
		return nil, err
	}
	out := make(map[string]restaurantRating, len(rows))
	for _, row := range rows {
		out[row.Restaurant] = row
	}
	return out, nil
}

func (r restaurantRating) toAPI() reviewsapi.RestaurantRating {
	return reviewsapi.RestaurantRating{
		Overall: r.Overall, Space: r.Space, Service: r.Service,
		Food: r.Food, Price: r.Price, Cleanliness: r.Cleanliness,
	}
}

// ratedPage has the ratings of one page of restaurants. The overall sort
// reads the cursor values from it, so the page needs one rating query.
type ratedPage struct {
	s       *Server
	ratings map[string]restaurantRating
}

func (p *ratedPage) load(records []*core.Record) error {
	ratings, err := p.s.ratings(records...)
	p.ratings = ratings
	return err
}

// overallSort puts restaurants without reviews last in both directions. Its
// first key is 1 for them and 0 for the others. They have 0 as the second key.
func (p *ratedPage) overallSort(desc bool) []sortKey {
	return []sortKey{
		{
			expr: "(" + overallColumn + " IS NULL)",
			value: func(r *core.Record) any {
				if _, ok := p.ratings[r.Id]; ok {
					return 0
				}
				return 1
			},
		},
		{
			expr:  "COALESCE(" + overallColumn + ", 0)",
			desc:  desc,
			value: func(r *core.Record) any { return p.ratings[r.Id].Overall },
		},
	}
}

// checkAverages also rejects NaN. A NaN value passes the range comparisons.
func checkAverages(name string, cs []comparison) string {
	for _, c := range cs {
		if v := c.value.(float64); math.IsNaN(v) || v < 1 || v > maxScore {
			return fmt.Sprintf("The %s filter values must be between 1 and %d.", name, maxScore)
		}
	}
	return ""
}

func averageComparisons(r *reviewsapi.AverageRange) []comparison {
	if r == nil {
		return nil
	}
	return comparisons(r.Lt, r.Lte, r.Eq, r.Gte, r.Gt)
}
