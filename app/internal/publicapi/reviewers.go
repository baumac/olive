package publicapi

import (
	"context"
	"database/sql"
	"errors"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const reviewerNotFound = "The user does not exist."

// GetReviewer returns only the public fields of a user.
func (s *Server) GetReviewer(_ context.Context, req reviewsapi.GetReviewerRequestObject) (reviewsapi.GetReviewerResponseObject, error) {
	user, err := s.findUser(req.ReviewerId)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return reviewsapi.GetReviewer404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(reviewerNotFound)}, nil
	}
	stats, err := s.findReviewerStats(user.Id)
	if err != nil {
		return nil, err
	}
	creator := toCreator(user)
	return reviewsapi.GetReviewer200JSONResponse{
		Id: creator.Id, Name: creator.Name,
		ReviewCount: stats.Reviews, AverageOverall: stats.average(), RestaurantCount: stats.Restaurants,
	}, nil
}

type reviewerStats struct {
	Reviews     int             `db:"reviews"`
	Overall     sql.NullFloat64 `db:"overall"`
	Restaurants int             `db:"restaurants"`
}

func (r reviewerStats) average() nullable.Nullable[reviewsapi.AverageScore] {
	if !r.Overall.Valid {
		return nullable.NewNullNullable[reviewsapi.AverageScore]()
	}
	return nullable.NewNullableWithValue(r.Overall.Float64)
}

func (s *Server) findReviewerStats(userID string) (reviewerStats, error) {
	var stats reviewerStats
	err := s.reviewerStatsQuery(userID).One(&stats)
	return stats, err
}

// reviewerStatsQuery is one statement. The count and the average come from
// the same reviews.
func (s *Server) reviewerStatsQuery(userID string) *dbx.Query {
	return s.app.DB().NewQuery(`
		SELECT COUNT(*) AS [[reviews]],
			ROUND(AVG([[overall]]), 1) AS [[overall]],
			(SELECT COUNT(*) FROM {{restaurants}} WHERE [[creator]] = {:user}) AS [[restaurants]]
		FROM {{reviews}} WHERE [[creator]] = {:user}`).Bind(dbx.Params{"user": userID})
}

// ListReviewerReviews returns 404 when the user does not exist. It does not
// return an empty list.
func (s *Server) ListReviewerReviews(_ context.Context, req reviewsapi.ListReviewerReviewsRequestObject) (reviewsapi.ListReviewerReviewsResponseObject, error) {
	p := req.Params
	all := reviewsapi.ListAllReviewsParams{
		Limit: p.Limit, Cursor: p.Cursor, Sort: p.Sort, Search: p.Search, CreatorEq: &req.ReviewerId,
		RestaurantEq: p.RestaurantEq, RestaurantIn: p.RestaurantIn,
		Space: p.Space, Service: p.Service, Food: p.Food, Price: p.Price, Cleanliness: p.Cleanliness,
		Overall: p.Overall, Created: p.Created, Updated: p.Updated,
	}
	fingerprintOf := func(sort string) []any {
		q := req.Params
		q.Limit, q.Cursor, q.Sort = nil, nil, &sort
		return []any{"reviewer-reviews", req.ReviewerId, q}
	}
	page, err := s.reviewPage(all, fingerprintOf, nil)
	if err == nil && page.bad == "" && len(page.list.Items) == 0 {
		// A page with a review proves that the user exists.
		var user *core.Record
		user, err = s.findUser(req.ReviewerId)
		page.notFound = user == nil
	}
	switch {
	case err != nil:
		return nil, err
	case page.bad != "":
		return reviewsapi.ListReviewerReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(page.bad)}, nil
	case page.notFound:
		return reviewsapi.ListReviewerReviews404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(reviewerNotFound)}, nil
	}
	return reviewsapi.ListReviewerReviews200JSONResponse(page.list), nil
}

// findUser returns a nil record and no error when the user does not exist.
func (s *Server) findUser(id string) (*core.Record, error) {
	user, err := s.app.FindRecordById("users", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return user, err
}
