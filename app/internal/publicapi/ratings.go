package publicapi

import (
	"fmt"
	"math"
	"slices"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The migrations make these tables. SQLite triggers keep them equal to the
// records. See decision 00012.
const (
	overallTable = "restaurant_overall"
	tagsTable    = "restaurant_tags"
)

// overallColumn is NULL for a restaurant without reviews. It needs
// joinOverall.
const overallColumn = "[[" + overallTable + ".overall]]"

// joinOverall reads the stored averages. Each restaurant has a row. The inner
// join lets SQLite read restaurants in the order of an overall index.
func (w *where) joinOverall() {
	w.joins = append(w.joins, join{
		inner: true,
		table: "{{" + overallTable + "}}",
		on:    dbx.NewExp(column(overallTable, "restaurant") + " = [[restaurants.id]]"),
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
// has no entry. The overall is the stored value. The overall sort and filter
// then use the value that the response shows.
func (s *Server) ratings(restaurants ...*core.Record) (map[string]restaurantRating, error) {
	if len(restaurants) == 0 {
		return map[string]restaurantRating{}, nil
	}
	ids := make([]any, len(restaurants))
	for i, r := range restaurants {
		ids[i] = r.Id
	}
	columns := []string{"[[reviews.restaurant]] AS [[restaurant]]", "COUNT(*) AS [[count]]", "MAX(" + overallColumn + ") AS [[overall]]"}
	for _, name := range []string{"space", "service", "food", "price", "cleanliness"} {
		columns = append(columns, fmt.Sprintf("ROUND(AVG([[%s]]), 1) AS [[%s]]", name, name))
	}
	var rows []restaurantRating
	err := s.app.DB().
		Select(columns...).
		From("reviews").
		LeftJoin(overallTable, dbx.NewExp(column(overallTable, "restaurant")+" = [[reviews.restaurant]]")).
		Where(dbx.In("reviews.restaurant", ids...)).
		GroupBy("reviews.restaurant").
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

// overallSort puts restaurants without reviews last in both directions. Its
// first key puts them last. All keys have one direction. They are the
// columns of one index of the migrations.
func overallSort(desc bool) []sortKey {
	missing := column(overallTable, "unrated")
	if desc {
		missing = column(overallTable, "rated")
	}
	return []sortKey{
		{expr: missing, desc: desc},
		{expr: column(overallTable, "overall_or_0"), desc: desc},
		{expr: column(overallTable, "restaurant"), desc: desc, unique: true},
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
