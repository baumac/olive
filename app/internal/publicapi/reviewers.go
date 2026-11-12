package publicapi

import (
	"context"
	"database/sql"
	"errors"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/pocketbase/dbx"
)

const reviewerNotFound = "The user does not exist."

// GetReviewer returns only the public fields of a user.
func (s *Server) GetReviewer(_ context.Context, req reviewsapi.GetReviewerRequestObject) (reviewsapi.GetReviewerResponseObject, error) {
	user, err := s.app.FindRecordById("users", req.ReviewerId)
	if errors.Is(err, sql.ErrNoRows) {
		return reviewsapi.GetReviewer404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(reviewerNotFound)}, nil
	}
	if err != nil {
		return nil, err
	}
	n, err := s.app.CountRecords("reviews", dbx.HashExp{"creator": user.Id})
	if err != nil {
		return nil, err
	}
	creator := toCreator(user)
	return reviewsapi.GetReviewer200JSONResponse{Id: creator.Id, Name: creator.Name, ReviewCount: int(n)}, nil
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
		page.notFound, err = s.userMissing(req.ReviewerId)
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

func (s *Server) userMissing(id string) (bool, error) {
	_, err := s.app.FindRecordById("users", id)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	return false, err
}
