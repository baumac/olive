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
func (s *Server) GetReviewer(ctx context.Context, req reviewsapi.GetReviewerRequestObject) (reviewsapi.GetReviewerResponseObject, error) {
	user, err := s.findReviewer(stateFrom(ctx).user, req.ReviewerId)
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
		ReviewCount: stats.Reviews, AverageOverall: stats.average(), RestaurantCount: stats.Restaurants, ListCount: stats.Lists,
	}, nil
}

type reviewerStats struct {
	Reviews     int             `db:"reviews"`
	Overall     sql.NullFloat64 `db:"overall"`
	Restaurants int             `db:"restaurants"`
	Lists       int             `db:"lists"`
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
			(SELECT COUNT(*) FROM {{restaurants}} WHERE [[creator]] = {:user}) AS [[restaurants]],
			(SELECT COUNT(*) FROM {{lists}} WHERE [[creator]] = {:user}) AS [[lists]]
		FROM {{reviews}} WHERE [[creator]] = {:user}`).Bind(dbx.Params{"user": userID})
}

// ListReviewerReviews returns 404 when the user does not exist. It does not
// return an empty list.
func (s *Server) ListReviewerReviews(ctx context.Context, req reviewsapi.ListReviewerReviewsRequestObject) (reviewsapi.ListReviewerReviewsResponseObject, error) {
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
	if err == nil && page.bad == "" {
		// The lookup runs after the parameter checks. 400 then comes before
		// 404.
		var user *core.Record
		user, err = s.findReviewer(stateFrom(ctx).user, req.ReviewerId)
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

// findReviewer returns a nil record and no error when the user does not
// exist. A user without approval has a public profile only for the own
// token. See decision 00034.
func (s *Server) findReviewer(viewer *core.Record, id string) (*core.Record, error) {
	user, err := s.app.FindRecordById("users", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil || visibleTo(viewer, user) {
		return user, err
	}
	return nil, nil
}

// visibleTo is false for a user without approval that viewer is not.
func visibleTo(viewer, user *core.Record) bool {
	return user.GetBool("approved") || (viewer != nil && viewer.Id == user.Id)
}
