package publicapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

const (
	kindBasic       = "basic"
	kindJocDeCartes = "joc_de_cartes"
)

// kindScores has the score properties of each review kind. PocketBase
// stores 0 in the score fields of the other kinds.
var kindScores = map[string][]string{
	kindBasic:       {"rating"},
	kindJocDeCartes: {"space", "service", "food", "price"},
}

var allScores = []string{"rating", "space", "service", "food", "price"}

// reviewInput is a create or patch body of a known kind. The body schema
// checked the values before.
type reviewInput struct {
	kind     string
	scores   map[string]int
	title    *string
	feedback nullable.Nullable[string]
	tags     nullable.Nullable[[]string]
}

var reviewSorts = map[string]sortField{
	"rating":  scoreSort("reviews", kindBasic, "rating"),
	"space":   scoreSort("reviews", kindJocDeCartes, "space"),
	"service": scoreSort("reviews", kindJocDeCartes, "service"),
	"food":    scoreSort("reviews", kindJocDeCartes, "food"),
	"price":   scoreSort("reviews", kindJocDeCartes, "price"),
	"title":   textSort("reviews", "title"),
	"created": timestampSort("reviews", "created"),
	"updated": timestampSort("reviews", "updated"),
}

// ListReviews returns 404 when the restaurant does not exist. It does not
// return an empty list.
func (s *Server) ListReviews(_ context.Context, req reviewsapi.ListReviewsRequestObject) (reviewsapi.ListReviewsResponseObject, error) {
	bad := func(detail string) (reviewsapi.ListReviewsResponseObject, error) {
		return reviewsapi.ListReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(detail)}, nil
	}
	if !recordIDPattern.MatchString(req.RestaurantId) {
		return bad(invalidRestaurantID)
	}

	p := req.Params
	limit, detail := parseLimit(p.Limit)
	if detail != "" {
		return bad(detail)
	}
	keys, sort, detail := parseSort(p.Sort, "reviews", reviewSorts)
	if detail != "" {
		return bad(detail)
	}
	p.Limit, p.Cursor, p.Sort = nil, nil, &sort
	query, err := fingerprint("reviews", req.RestaurantId, p)
	if err != nil {
		return nil, err
	}
	after, detail := decodeCursor(req.Params.Cursor, query, len(keys))
	if detail != "" {
		return bad(detail)
	}

	w := newWhere()
	w.compare(column("reviews", "restaurant"), []comparison{{"=", req.RestaurantId}})
	if p.KindEq != nil {
		w.compare(column("reviews", "kind"), []comparison{{"=", *p.KindEq}})
	}
	if p.KindIn != nil {
		w.in(column("reviews", "kind"), *p.KindIn)
	}
	for _, detail := range []string{
		w.scoreFilter(kindBasic, "rating", 5, ratingComparisons(p.Rating)),
		w.scoreFilter(kindJocDeCartes, "space", 10, scoreComparisons(p.Space)),
		w.scoreFilter(kindJocDeCartes, "service", 10, scoreComparisons(p.Service)),
		w.scoreFilter(kindJocDeCartes, "food", 10, scoreComparisons(p.Food)),
		w.scoreFilter(kindJocDeCartes, "price", 10, scoreComparisons(p.Price)),
		w.tagFilters("reviews", p.TagsHas, p.TagsHasAny, p.TagsHasAll),
		w.creatorFilters("reviews", p.CreatorEq, p.CreatorIn),
		w.timestampFilter("reviews", "created", timestampComparisons(p.Created)),
		w.timestampFilter("reviews", "updated", timestampComparisons(p.Updated)),
	} {
		if detail != "" {
			return bad(detail)
		}
	}
	if after != nil {
		w.keyset(keys, after)
	}

	if _, err := s.findRestaurant(s.app, req.RestaurantId); errors.Is(err, sql.ErrNoRows) {
		return reviewsapi.ListReviews404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(restaurantNotFound)}, nil
	} else if err != nil {
		return nil, err
	}

	records, next, err := s.page("reviews", w, keys, limit, query)
	if err != nil {
		return nil, err
	}
	users, err := s.creators(records...)
	if err != nil {
		return nil, err
	}
	items := make([]reviewsapi.Review, len(records))
	for i, r := range records {
		if items[i], err = toReview(r, users[r.GetString("creator")]); err != nil {
			return nil, err
		}
	}
	return reviewsapi.ListReviews200JSONResponse{Items: items, NextCursor: next}, nil
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
	var saved *core.Record
	err := s.app.RunInTransaction(func(tx core.App) error {
		if _, err := s.findRestaurant(tx, req.RestaurantId); errors.Is(err, sql.ErrNoRows) {
			response = reviewsapi.CreateReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(restaurantNotFound)}
			return nil
		} else if err != nil {
			return err
		}

		in, errs, foreign, err := createInput(req.Body, st.props)
		if err != nil {
			return err
		}
		if foreign != "" {
			response = reviewsapi.CreateReview400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(foreign)}
			return nil
		}
		if len(errs) > 0 {
			response = reviewsapi.CreateReview422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: unprocessable(errs)}
			return nil
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
		r.Set("kind", in.kind)
		in.apply(r)
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

	out, err := toReview(saved, st.user)
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
	r, detail, err := s.findReview(s.app, req.RestaurantId, req.ReviewId)
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
	out, err := toReview(r, users[r.GetString("creator")])
	if err != nil {
		return nil, err
	}
	return reviewsapi.GetReview200JSONResponse(out), nil
}

// UpdateReview needs the stored kind in the body. A different kind causes
// 422.
func (s *Server) UpdateReview(ctx context.Context, req reviewsapi.UpdateReviewRequestObject) (reviewsapi.UpdateReviewResponseObject, error) {
	if detail := checkReviewIDs(req.RestaurantId, req.ReviewId); detail != "" {
		return reviewsapi.UpdateReview400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(detail)}, nil
	}
	st := stateFrom(ctx)

	var response reviewsapi.UpdateReviewResponseObject
	var saved *core.Record
	err := s.app.RunInTransaction(func(tx core.App) error {
		r, detail, err := s.findReview(tx, req.RestaurantId, req.ReviewId)
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

		in, errs, foreign, err := patchInput(req.Body, st.props)
		if err != nil {
			return err
		}
		if foreign != "" {
			response = reviewsapi.UpdateReview400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(foreign)}
			return nil
		}
		if stored := r.GetString("kind"); in.kind != "" && in.kind != stored {
			errs = append(fieldErrors{{Property: "kind", Message: "must be equal to the stored kind " + stored}}, errs...)
		}
		if len(errs) > 0 {
			response = reviewsapi.UpdateReview422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: unprocessable(errs)}
			return nil
		}

		saved = r
		if !in.apply(r) {
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

	out, err := toReview(saved, st.user)
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
		r, detail, err := s.findReview(tx, req.RestaurantId, req.ReviewId)
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
func (s *Server) findReview(app core.App, restaurantID, reviewID string) (*core.Record, string, error) {
	if _, err := s.findRestaurant(app, restaurantID); errors.Is(err, sql.ErrNoRows) {
		return nil, restaurantNotFound, nil
	} else if err != nil {
		return nil, "", err
	}
	r, err := app.FindRecordById("reviews", reviewID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && r.GetString("restaurant") != restaurantID) {
		return nil, "The review does not exist.", nil
	}
	if err != nil {
		return nil, "", err
	}
	return r, "", nil
}

// checkKind returns the field errors for an unknown kind. It also returns a
// detail for the 400 response if the body has scores of other kinds.
func checkKind(kind string, props map[string]json.RawMessage) (fieldErrors, string) {
	scores, ok := kindScores[kind]
	if !ok {
		var errs fieldErrors
		errs.add("kind", "must be one of "+kindBasic+", "+kindJocDeCartes)
		return errs, ""
	}
	var foreign []string
	for _, name := range allScores {
		if _, inBody := props[name]; inBody && !slices.Contains(scores, name) {
			foreign = append(foreign, name)
		}
	}
	if len(foreign) > 0 {
		return nil, fmt.Sprintf("A %s review does not have the properties: %s.", kind, strings.Join(foreign, ", "))
	}
	return nil, ""
}

func createInput(body *reviewsapi.ReviewCreate, props map[string]json.RawMessage) (reviewInput, fieldErrors, string, error) {
	kind, err := body.Discriminator()
	if err != nil {
		return reviewInput{}, nil, "", err
	}
	if errs, foreign := checkKind(kind, props); errs != nil || foreign != "" {
		return reviewInput{}, errs, foreign, nil
	}

	in := reviewInput{kind: kind}
	switch kind {
	case kindBasic:
		v, err := body.AsBasicReviewCreate()
		if err != nil {
			return in, nil, "", err
		}
		in.scores = map[string]int{"rating": v.Rating}
		in.title, in.feedback, in.tags = &v.Title, optional(v.Feedback), optional(v.Tags)
	case kindJocDeCartes:
		v, err := body.AsJocDeCartesReviewCreate()
		if err != nil {
			return in, nil, "", err
		}
		in.scores = map[string]int{"space": v.Space, "service": v.Service, "food": v.Food, "price": v.Price}
		in.title, in.feedback, in.tags = &v.Title, optional(v.Feedback), optional(v.Tags)
	}

	var errs fieldErrors
	for _, name := range kindScores[kind] {
		if _, ok := props[name]; !ok {
			errs.add(name, "is required")
		}
	}
	return in, errs, "", nil
}

func patchInput(body *reviewsapi.ReviewPatch, props map[string]json.RawMessage) (reviewInput, fieldErrors, string, error) {
	kind, err := body.Discriminator()
	if err != nil {
		return reviewInput{}, nil, "", err
	}
	if errs, foreign := checkKind(kind, props); errs != nil || foreign != "" {
		return reviewInput{}, errs, foreign, nil
	}

	in := reviewInput{kind: kind, scores: map[string]int{}}
	switch kind {
	case kindBasic:
		v, err := body.AsBasicReviewPatch()
		if err != nil {
			return in, nil, "", err
		}
		setScore(in.scores, "rating", v.Rating)
		in.title, in.feedback, in.tags = v.Title, v.Feedback, v.Tags
	case kindJocDeCartes:
		v, err := body.AsJocDeCartesReviewPatch()
		if err != nil {
			return in, nil, "", err
		}
		setScore(in.scores, "space", v.Space)
		setScore(in.scores, "service", v.Service)
		setScore(in.scores, "food", v.Food)
		setScore(in.scores, "price", v.Price)
		in.title, in.feedback, in.tags = v.Title, v.Feedback, v.Tags
	}
	return in, nil, "", nil
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

func toReview(r *core.Record, creator *core.Record) (reviewsapi.Review, error) {
	if creator == nil {
		return reviewsapi.Review{}, fmt.Errorf("review %s has no creator", r.Id)
	}
	var (
		out      reviewsapi.Review
		err      error
		c        = toCreator(creator)
		feedback = optionalText(r, "feedback")
		tags     = tagsOf(r)
		created  = r.GetDateTime("created").String()
		updated  = r.GetDateTime("updated").String()
		kind     = r.GetString("kind")
	)
	switch kind {
	case kindBasic:
		err = out.FromBasicReview(reviewsapi.BasicReview{
			Id: r.Id, Restaurant: r.GetString("restaurant"), Kind: kindBasic,
			Rating: r.GetInt("rating"),
			Title:  r.GetString("title"), Feedback: feedback, Tags: tags,
			Creator: c, Created: created, Updated: updated,
		})
	case kindJocDeCartes:
		err = out.FromJocDeCartesReview(reviewsapi.JocDeCartesReview{
			Id: r.Id, Restaurant: r.GetString("restaurant"), Kind: kindJocDeCartes,
			Space: r.GetInt("space"), Service: r.GetInt("service"), Food: r.GetInt("food"), Price: r.GetInt("price"),
			Title: r.GetString("title"), Feedback: feedback, Tags: tags,
			Creator: c, Created: created, Updated: updated,
		})
	default:
		err = out.FromOtherReview(reviewsapi.OtherReview{
			Id: r.Id, Restaurant: r.GetString("restaurant"), Kind: kind,
			Title: r.GetString("title"), Feedback: feedback, Tags: tags,
			Creator: c, Created: created, Updated: updated,
		})
	}
	return out, err
}

// scoreFilter also compares kind. The stored 0 of other kinds then does not
// match.
func (w *where) scoreFilter(kind, name string, maxValue int, cs []comparison) string {
	if len(cs) == 0 {
		return ""
	}
	for _, c := range cs {
		if v := c.value.(int); v < 1 || v > maxValue {
			return fmt.Sprintf("The %s filter values must be between 1 and %d.", name, maxValue)
		}
	}
	w.compare(column("reviews", "kind"), []comparison{{"=", kind}})
	w.compare(column("reviews", name), cs)
	return ""
}

func ratingComparisons(r *reviewsapi.RatingRange) []comparison {
	if r == nil {
		return nil
	}
	return comparisons(r.Lt, r.Lte, r.Eq, r.Gte, r.Gt)
}

func scoreComparisons(r *reviewsapi.ScoreRange) []comparison {
	if r == nil {
		return nil
	}
	return comparisons(r.Lt, r.Lte, r.Eq, r.Gte, r.Gt)
}
