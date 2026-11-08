package publicapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

const maxScore = 10

// reviewInput is a create or patch body. The body schema checked the values
// before.
type reviewInput struct {
	scores   map[string]int
	title    *string
	feedback nullable.Nullable[string]
	tags     nullable.Nullable[[]string]
}

var reviewSorts = map[string]sortField{
	"space":       scoreSort("reviews", "space"),
	"service":     scoreSort("reviews", "service"),
	"food":        scoreSort("reviews", "food"),
	"price":       scoreSort("reviews", "price"),
	"cleanliness": scoreSort("reviews", "cleanliness"),
	"title":       textSort("reviews", "title"),
	"created":     timestampSort("reviews", "created"),
	"updated":     timestampSort("reviews", "updated"),
}

// ListReviews returns 404 when the restaurant does not exist. It does not
// return an empty list.
func (s *Server) ListReviews(_ context.Context, req reviewsapi.ListReviewsRequestObject) (reviewsapi.ListReviewsResponseObject, error) {
	if !recordIDPattern.MatchString(req.RestaurantId) {
		return reviewsapi.ListReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(invalidRestaurantID)}, nil
	}
	p := req.Params
	all := reviewsapi.ListAllReviewsParams{
		Limit: p.Limit, Cursor: p.Cursor, Sort: p.Sort, RestaurantEq: &req.RestaurantId,
		Space: p.Space, Service: p.Service, Food: p.Food, Price: p.Price, Cleanliness: p.Cleanliness,
		TagsHas: p.TagsHas, TagsHasAny: p.TagsHasAny, TagsHasAll: p.TagsHasAll,
		Created: p.Created, Updated: p.Updated, CreatorEq: p.CreatorEq, CreatorIn: p.CreatorIn,
	}
	// The cursor keeps the fingerprint of the first version of this
	// operation. Open pages can then still load more after a deploy.
	fingerprintOf := func(sort string) []any {
		q := req.Params
		q.Limit, q.Cursor, q.Sort = nil, nil, &sort
		return []any{"reviews", req.RestaurantId, q}
	}
	page, err := s.reviewPage(all, fingerprintOf, func() (*core.Record, error) {
		r, err := s.findRestaurant(s.app, req.RestaurantId)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return r, err
	})
	switch {
	case err != nil:
		return nil, err
	case page.bad != "":
		return reviewsapi.ListReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(page.bad)}, nil
	case page.notFound:
		return reviewsapi.ListReviews404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(restaurantNotFound)}, nil
	}
	return reviewsapi.ListReviews200JSONResponse(page.list), nil
}

// ListAllReviews lists the reviews of all restaurants. An unknown
// restaurant in restaurant[eq] gives an empty list, not 404.
func (s *Server) ListAllReviews(_ context.Context, req reviewsapi.ListAllReviewsRequestObject) (reviewsapi.ListAllReviewsResponseObject, error) {
	page, err := s.reviewPage(req.Params, nil, nil)
	if err != nil {
		return nil, err
	}
	if page.bad != "" {
		return reviewsapi.ListAllReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(page.bad)}, nil
	}
	return reviewsapi.ListAllReviews200JSONResponse(page.list), nil
}

type reviewPageResult struct {
	list     reviewsapi.ReviewList
	bad      string
	notFound bool
}

// reviewPage checks the parameters, then calls restaurant, then gets the
// page. A nil restaurant skips the check. This order gives 400 before 404.
// A nil fingerprintOf uses the parameters of listAllReviews.
func (s *Server) reviewPage(
	p reviewsapi.ListAllReviewsParams,
	fingerprintOf func(sort string) []any,
	restaurant func() (*core.Record, error),
) (reviewPageResult, error) {
	limit, detail := parseLimit(p.Limit)
	if detail != "" {
		return reviewPageResult{bad: detail}, nil
	}
	keys, sort, detail := parseSort(p.Sort, "reviews", reviewSorts)
	if detail != "" {
		return reviewPageResult{bad: detail}, nil
	}
	cursor := p.Cursor
	p.Limit, p.Cursor, p.Sort = nil, nil, &sort
	parts := []any{"reviews", p}
	if fingerprintOf != nil {
		parts = fingerprintOf(sort)
	}
	query, err := fingerprint(parts...)
	if err != nil {
		return reviewPageResult{}, err
	}
	after, detail := decodeCursor(cursor, query, len(keys))
	if detail != "" {
		return reviewPageResult{bad: detail}, nil
	}

	w := newWhere()
	for _, detail := range []string{
		w.restaurantFilters(p.RestaurantEq, p.RestaurantIn),
		w.scoreFilter("space", scoreComparisons(p.Space)),
		w.scoreFilter("service", scoreComparisons(p.Service)),
		w.scoreFilter("food", scoreComparisons(p.Food)),
		w.scoreFilter("price", scoreComparisons(p.Price)),
		w.scoreFilter("cleanliness", scoreComparisons(p.Cleanliness)),
		w.tagFilters("reviews", p.TagsHas, p.TagsHasAny, p.TagsHasAll),
		w.creatorFilters("reviews", p.CreatorEq, p.CreatorIn),
		w.timestampFilter("reviews", "created", timestampComparisons(p.Created)),
		w.timestampFilter("reviews", "updated", timestampComparisons(p.Updated)),
	} {
		if detail != "" {
			return reviewPageResult{bad: detail}, nil
		}
	}
	if after != nil {
		w.keyset(keys, after)
	}

	known := map[string]string{}
	if restaurant != nil {
		r, err := restaurant()
		if err != nil {
			return reviewPageResult{}, err
		}
		if r == nil {
			return reviewPageResult{notFound: true}, nil
		}
		known[r.Id] = r.GetString("name")
	}

	records, next, err := s.page("reviews", w, keys, limit, query, nil)
	if err != nil {
		return reviewPageResult{}, err
	}
	users, err := s.creators(records...)
	if err != nil {
		return reviewPageResult{}, err
	}
	names, err := s.restaurantNames(known, records...)
	if err != nil {
		return reviewPageResult{}, err
	}
	items := make([]reviewsapi.Review, len(records))
	for i, r := range records {
		if items[i], err = toReview(r, users[r.GetString("creator")], names[r.GetString("restaurant")]); err != nil {
			return reviewPageResult{}, err
		}
	}
	return reviewPageResult{list: reviewsapi.ReviewList{Items: items, NextCursor: next}}, nil
}

func (w *where) restaurantFilters(eq *string, in *[]string) string {
	ids := []string{}
	if eq != nil {
		ids = append(ids, *eq)
	}
	if in != nil {
		ids = append(ids, *in...)
	}
	for _, id := range ids {
		if !recordIDPattern.MatchString(id) {
			return fmt.Sprintf("The restaurant ID %q is not valid.", id)
		}
	}
	if eq != nil {
		w.compare(column("reviews", "restaurant"), []comparison{{"=", *eq}})
	}
	if in != nil {
		w.in(column("reviews", "restaurant"), *in)
	}
	return ""
}

// restaurantNames reads only the restaurants that are not in known. The
// per-restaurant list then needs no extra query.
func (s *Server) restaurantNames(known map[string]string, reviews ...*core.Record) (map[string]string, error) {
	out := maps.Clone(known)
	ids := map[string]bool{}
	for _, r := range reviews {
		if _, ok := out[r.GetString("restaurant")]; !ok {
			ids[r.GetString("restaurant")] = true
		}
	}
	if len(ids) == 0 {
		return out, nil
	}
	restaurants, err := s.app.FindRecordsByIds("restaurants", slices.Collect(maps.Keys(ids)))
	if err != nil {
		return nil, err
	}
	for _, r := range restaurants {
		out[r.Id] = r.GetString("name")
	}
	return out, nil
}

// CreateReview runs the duplicate check and the save in one transaction.
// The unique index on restaurant and creator also blocks a second review.
func (s *Server) CreateReview(ctx context.Context, req reviewsapi.CreateReviewRequestObject) (reviewsapi.CreateReviewResponseObject, error) {
	if !recordIDPattern.MatchString(req.RestaurantId) {
		return reviewsapi.CreateReview400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(invalidRestaurantID)}, nil
	}
	st := stateFrom(ctx)
	duplicate := reviewsapi.CreateReview409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict("The user already has a review for this restaurant.")}

	var response reviewsapi.CreateReviewResponseObject
	var saved, restaurant *core.Record
	err := s.app.RunInTransaction(func(tx core.App) error {
		var err error
		if restaurant, err = s.findRestaurant(tx, req.RestaurantId); errors.Is(err, sql.ErrNoRows) {
			response = reviewsapi.CreateReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(restaurantNotFound)}
			return nil
		} else if err != nil {
			return err
		}

		n, err := tx.CountRecords("reviews", dbx.HashExp{"restaurant": req.RestaurantId, "creator": st.user.Id})
		if err != nil {
			return err
		}
		if n > 0 {
			response = duplicate
			return nil
		}

		collection, err := tx.FindCachedCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		r := core.NewRecord(collection)
		r.Set("restaurant", req.RestaurantId)
		r.Set("creator", st.user.Id)
		createInput(req.Body).apply(r)
		saved = r
		return tx.Save(r)
	})
	var invalid validation.Errors
	if errors.As(err, &invalid) && isNotUnique(invalid, "creator") {
		return duplicate, nil
	}
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response, nil
	}

	out, err := toReview(saved, st.user, restaurant.GetString("name"))
	if err != nil {
		return nil, err
	}
	location := basePath + "/restaurants/" + req.RestaurantId + "/reviews/" + saved.Id
	return reviewsapi.CreateReview201JSONResponse{
		Body:    out,
		Headers: reviewsapi.CreateReview201ResponseHeaders{Location: &location},
	}, nil
}

// GetReview returns 404 for a review of a different restaurant.
func (s *Server) GetReview(_ context.Context, req reviewsapi.GetReviewRequestObject) (reviewsapi.GetReviewResponseObject, error) {
	if detail := checkReviewIDs(req.RestaurantId, req.ReviewId); detail != "" {
		return reviewsapi.GetReview400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(detail)}, nil
	}
	restaurant, r, detail, err := s.findReview(s.app, req.RestaurantId, req.ReviewId)
	if err != nil {
		return nil, err
	}
	if detail != "" {
		return reviewsapi.GetReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}, nil
	}
	users, err := s.creators(r)
	if err != nil {
		return nil, err
	}
	out, err := toReview(r, users[r.GetString("creator")], restaurant.GetString("name"))
	if err != nil {
		return nil, err
	}
	return reviewsapi.GetReview200JSONResponse(out), nil
}

// UpdateReview lets only the creator change a review.
func (s *Server) UpdateReview(ctx context.Context, req reviewsapi.UpdateReviewRequestObject) (reviewsapi.UpdateReviewResponseObject, error) {
	if detail := checkReviewIDs(req.RestaurantId, req.ReviewId); detail != "" {
		return reviewsapi.UpdateReview400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(detail)}, nil
	}
	st := stateFrom(ctx)

	var response reviewsapi.UpdateReviewResponseObject
	var saved, restaurant *core.Record
	err := s.app.RunInTransaction(func(tx core.App) error {
		var r *core.Record
		var detail string
		var err error
		restaurant, r, detail, err = s.findReview(tx, req.RestaurantId, req.ReviewId)
		if err != nil {
			return err
		}
		if detail != "" {
			response = reviewsapi.UpdateReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}
			return nil
		}
		if r.GetString("creator") != st.user.Id {
			response = reviewsapi.UpdateReview403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden("Only the creator of the review can change it.")}
			return nil
		}

		saved = r
		if !patchInput(req.Body).apply(r) {
			return nil
		}
		return tx.Save(r)
	})
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response, nil
	}

	out, err := toReview(saved, st.user, restaurant.GetString("name"))
	if err != nil {
		return nil, err
	}
	return reviewsapi.UpdateReview200JSONResponse(out), nil
}

// DeleteReview lets only the creator delete a review.
func (s *Server) DeleteReview(ctx context.Context, req reviewsapi.DeleteReviewRequestObject) (reviewsapi.DeleteReviewResponseObject, error) {
	if detail := checkReviewIDs(req.RestaurantId, req.ReviewId); detail != "" {
		return reviewsapi.DeleteReview400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(detail)}, nil
	}
	user := stateFrom(ctx).user

	var response reviewsapi.DeleteReviewResponseObject = reviewsapi.DeleteReview204Response{}
	err := s.app.RunInTransaction(func(tx core.App) error {
		_, r, detail, err := s.findReview(tx, req.RestaurantId, req.ReviewId)
		if err != nil {
			return err
		}
		if detail != "" {
			response = reviewsapi.DeleteReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}
			return nil
		}
		if r.GetString("creator") != user.Id {
			response = reviewsapi.DeleteReview403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden("Only the creator of the review can delete it.")}
			return nil
		}
		return tx.Delete(r)
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func checkReviewIDs(restaurantID, reviewID string) string {
	if !recordIDPattern.MatchString(restaurantID) {
		return invalidRestaurantID
	}
	if !recordIDPattern.MatchString(reviewID) {
		return "The review ID is not valid."
	}
	return ""
}

// findReview returns a detail for the 404 response if the restaurant or the
// review does not exist. A review of a different restaurant does not exist.
func (s *Server) findReview(app core.App, restaurantID, reviewID string) (restaurant, review *core.Record, detail string, err error) {
	restaurant, err = s.findRestaurant(app, restaurantID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, restaurantNotFound, nil
	} else if err != nil {
		return nil, nil, "", err
	}
	review, err = app.FindRecordById("reviews", reviewID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && review.GetString("restaurant") != restaurantID) {
		return nil, nil, "The review does not exist.", nil
	}
	if err != nil {
		return nil, nil, "", err
	}
	return restaurant, review, "", nil
}

func createInput(body *reviewsapi.ReviewCreate) reviewInput {
	return reviewInput{
		scores: map[string]int{"space": body.Space, "service": body.Service, "food": body.Food, "price": body.Price, "cleanliness": body.Cleanliness},
		title:  &body.Title, feedback: optional(body.Feedback), tags: optional(body.Tags),
	}
}

func patchInput(body *reviewsapi.ReviewPatch) reviewInput {
	in := reviewInput{scores: map[string]int{}, title: body.Title, feedback: body.Feedback, tags: body.Tags}
	setScore(in.scores, "space", body.Space)
	setScore(in.scores, "service", body.Service)
	setScore(in.scores, "food", body.Food)
	setScore(in.scores, "price", body.Price)
	setScore(in.scores, "cleanliness", body.Cleanliness)
	return in
}

func setScore(scores map[string]int, name string, v *int) {
	if v != nil {
		scores[name] = *v
	}
}

func optional[T any](v *T) nullable.Nullable[T] {
	if v == nil {
		return nil
	}
	return nullable.NewNullableWithValue(*v)
}

// apply returns true if a field gets a different value.
func (in reviewInput) apply(r *core.Record) bool {
	ch := &changes{r: r}
	for name, v := range in.scores {
		ch.set(name, float64(v))
	}
	if in.title != nil {
		ch.set("title", *in.title)
	}
	ch.patchText("feedback", in.feedback)
	ch.patchTags(in.tags)
	return ch.changed
}

func toReview(r *core.Record, creator *core.Record, restaurantName string) (reviewsapi.Review, error) {
	if creator == nil {
		return reviewsapi.Review{}, fmt.Errorf("review %s has no creator", r.Id)
	}
	if restaurantName == "" {
		return reviewsapi.Review{}, fmt.Errorf("review %s has no restaurant", r.Id)
	}
	return reviewsapi.Review{
		Id: r.Id, Restaurant: r.GetString("restaurant"), RestaurantName: restaurantName,
		Space: r.GetInt("space"), Service: r.GetInt("service"), Food: r.GetInt("food"), Price: r.GetInt("price"),
		Cleanliness: r.GetInt("cleanliness"),
		Title:       r.GetString("title"), Feedback: optionalText(r, "feedback"), Tags: tagsOf(r),
		Creator: toCreator(creator), Created: r.GetDateTime("created").String(), Updated: r.GetDateTime("updated").String(),
	}, nil
}

func (w *where) scoreFilter(name string, cs []comparison) string {
	if len(cs) == 0 {
		return ""
	}
	for _, c := range cs {
		if v := c.value.(int); v < 1 || v > maxScore {
			return fmt.Sprintf("The %s filter values must be between 1 and %d.", name, maxScore)
		}
	}
	w.compare(column("reviews", name), cs)
	return ""
}

func scoreComparisons(r *reviewsapi.ScoreRange) []comparison {
	if r == nil {
		return nil
	}
	return comparisons(r.Lt, r.Lte, r.Eq, r.Gte, r.Gt)
}
