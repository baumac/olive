package publicapi

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

var restaurantSorts = map[string]sortField{
	"name":      textSort("restaurants", "name"),
	"overall":   overallSort,
	"relevance": relevanceSort,
	"created":   columnSort("restaurants", "created"),
	"updated":   columnSort("restaurants", "updated"),
}

// ListRestaurants uses keyset pagination. The cursor has the sort values of
// the last item.
func (s *Server) ListRestaurants(_ context.Context, req reviewsapi.ListRestaurantsRequestObject) (reviewsapi.ListRestaurantsResponseObject, error) {
	p := req.Params
	bad := func(detail string) (reviewsapi.ListRestaurantsResponseObject, error) {
		return reviewsapi.ListRestaurants400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(detail)}, nil
	}

	limit := limitOf(p.Limit)
	search := newSearchPage("restaurants")
	if detail := search.parse(p.Search); detail != "" {
		return bad(detail)
	}
	sortSpec, detail := search.sortWithSearch(p.Sort)
	if detail != "" {
		return bad(detail)
	}
	keys, sort, detail := parseSort(sortSpec, "restaurants", restaurantSorts)
	if detail != "" {
		return bad(detail)
	}
	if detail := overallDetail(p.Overall); detail != "" {
		return bad(detail)
	}
	p.Limit, p.Cursor, p.Sort = nil, nil, &sort
	parts := []any{"restaurants", p}
	// The overall sort keys changed in decision 00012. The version rejects
	// the cursors of the old keys.
	if sortsBy(sort, "overall") {
		parts = append(parts, 2)
	}
	query, err := fingerprint(parts...)
	if err != nil {
		return nil, err
	}
	after, detail := decodeCursor(req.Params.Cursor, query, len(keys))
	if detail != "" {
		return bad(detail)
	}

	w := newWhere()
	search.filter(w)
	if p.Overall != nil || sortsBy(sort, "overall") {
		w.joinOverall()
	}
	w.compare(overallColumn, averageComparisons(p.Overall))
	w.tagFilters(p.TagsHas, p.TagsHasAny, p.TagsHasAll)
	w.idFilters("restaurants", "creator", p.CreatorEq, p.CreatorIn)
	w.compare(column("restaurants", "created"), timestampComparisons(p.Created))
	w.compare(column("restaurants", "updated"), timestampComparisons(p.Updated))
	if after != nil {
		w.keyset(keys, after)
	}

	records, next, err := s.page("restaurants", w, keys, limit, query)
	if err != nil {
		return nil, err
	}
	users, err := s.creators(records...)
	if err != nil {
		return nil, err
	}
	ratings, err := s.ratings(records...)
	if err != nil {
		return nil, err
	}
	items := make([]reviewsapi.Restaurant, len(records))
	for i, r := range records {
		items[i] = toRestaurant(r, users, ratings)
	}
	return reviewsapi.ListRestaurants200JSONResponse{Items: items, NextCursor: next}, nil
}

func sortsBy(sort, field string) bool {
	return slices.ContainsFunc(strings.Split(sort, ","), func(f string) bool { return strings.TrimPrefix(f, "-") == field })
}

// CreateRestaurant sets the user of the bearer token as the creator.
func (s *Server) CreateRestaurant(ctx context.Context, req reviewsapi.CreateRestaurantRequestObject) (reviewsapi.CreateRestaurantResponseObject, error) {
	user := stateFrom(ctx).user
	b := req.Body
	collection, err := s.app.FindCachedCollectionByNameOrId("restaurants")
	if err != nil {
		return nil, err
	}
	r := core.NewRecord(collection)
	r.Set("name", b.Name)
	if b.Description != nil {
		r.Set("description", *b.Description)
	}
	if b.Tags != nil {
		r.Set("tags", *b.Tags)
	}
	r.Set("creator", user.Id)
	if err := s.app.Save(r); err != nil {
		return nil, err
	}

	location := basePath + "/restaurants/" + r.Id
	return reviewsapi.CreateRestaurant201JSONResponse{
		Body:    toRestaurant(r, map[string]*core.Record{user.Id: user}, nil),
		Headers: reviewsapi.CreateRestaurant201ResponseHeaders{Location: &location},
	}, nil
}

func (s *Server) GetRestaurant(_ context.Context, req reviewsapi.GetRestaurantRequestObject) (reviewsapi.GetRestaurantResponseObject, error) {
	r, err := s.findRestaurant(s.app, req.RestaurantId)
	if errors.Is(err, sql.ErrNoRows) {
		return reviewsapi.GetRestaurant404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(restaurantNotFound)}, nil
	}
	if err != nil {
		return nil, err
	}
	out, err := s.restaurantResponse(r)
	if err != nil {
		return nil, err
	}
	return reviewsapi.GetRestaurant200JSONResponse(out), nil
}

// UpdateRestaurant runs the review check and the save in one transaction. A
// review that another request adds cannot make the name check wrong.
func (s *Server) UpdateRestaurant(ctx context.Context, req reviewsapi.UpdateRestaurantRequestObject) (reviewsapi.UpdateRestaurantResponseObject, error) {
	user := stateFrom(ctx).user
	b := req.Body

	var response reviewsapi.UpdateRestaurantResponseObject
	var saved *core.Record
	err := s.app.RunInTransaction(func(tx core.App) error {
		r, err := s.findRestaurant(tx, req.RestaurantId)
		if errors.Is(err, sql.ErrNoRows) {
			response = reviewsapi.UpdateRestaurant404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(restaurantNotFound)}
			return nil
		}
		if err != nil {
			return err
		}
		if detail, err := restaurantChangeForbidden(tx, r, user.Id); err != nil {
			return err
		} else if detail != "" {
			response = reviewsapi.UpdateRestaurant403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(detail)}
			return nil
		}

		if b.Name != nil && *b.Name != r.GetString("name") {
			n, err := countReviewsOfOtherUsers(tx, r.Id, user.Id)
			if err != nil {
				return err
			}
			if n > 0 {
				response = reviewsapi.UpdateRestaurant409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict("The name cannot change because the restaurant has reviews of other users.")}
				return nil
			}
		}
		ch := &changes{r: r}
		if b.Name != nil {
			ch.set("name", *b.Name)
		}
		ch.patchText("description", b.Description)
		ch.patchTags(b.Tags)
		saved = r
		if !ch.changed {
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
	out, err := s.restaurantResponse(saved)
	if err != nil {
		return nil, err
	}
	return reviewsapi.UpdateRestaurant200JSONResponse(out), nil
}

// DeleteRestaurant runs the review check and the delete in one transaction.
// It also deletes the review of the user.
func (s *Server) DeleteRestaurant(ctx context.Context, req reviewsapi.DeleteRestaurantRequestObject) (reviewsapi.DeleteRestaurantResponseObject, error) {
	user := stateFrom(ctx).user

	var response reviewsapi.DeleteRestaurantResponseObject = reviewsapi.DeleteRestaurant204Response{}
	err := s.app.RunInTransaction(func(tx core.App) error {
		r, err := s.findRestaurant(tx, req.RestaurantId)
		if errors.Is(err, sql.ErrNoRows) {
			response = reviewsapi.DeleteRestaurant404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(restaurantNotFound)}
			return nil
		}
		if err != nil {
			return err
		}
		if detail, err := restaurantChangeForbidden(tx, r, user.Id); err != nil {
			return err
		} else if detail != "" {
			response = reviewsapi.DeleteRestaurant403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(detail)}
			return nil
		}
		n, err := countReviewsOfOtherUsers(tx, r.Id, user.Id)
		if err != nil {
			return err
		}
		if n > 0 {
			response = reviewsapi.DeleteRestaurant409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict("The restaurant has reviews of other users. It cannot be deleted.")}
			return nil
		}
		own, err := tx.FindAllRecords("reviews", dbx.HashExp{"restaurant": r.Id})
		if err != nil {
			return err
		}
		for _, review := range own {
			if err := tx.Delete(review); err != nil {
				return err
			}
		}
		return tx.Delete(r)
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

const restaurantNotFound = "The restaurant does not exist."

// restaurantChangeForbidden returns a detail for the 403 response if the user
// cannot change or delete the restaurant. A restaurant without a creator
// belongs to each user who has a review of it.
func restaurantChangeForbidden(app core.App, r *core.Record, userID string) (string, error) {
	if creator := r.GetString("creator"); creator != "" {
		if creator != userID {
			return "Only the creator of the restaurant can change or delete it.", nil
		}
		return "", nil
	}
	n, err := app.CountRecords("reviews", dbx.HashExp{"restaurant": r.Id, "creator": userID})
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "The restaurant has no creator. Only a user with a review of it can change or delete it.", nil
	}
	return "", nil
}

func countReviewsOfOtherUsers(app core.App, restaurantID, userID string) (int64, error) {
	return app.CountRecords("reviews", dbx.HashExp{"restaurant": restaurantID}, dbx.Not(dbx.HashExp{"creator": userID}))
}

func (s *Server) findRestaurant(app core.App, id string) (*core.Record, error) {
	return app.FindRecordById("restaurants", id)
}

func (s *Server) restaurantResponse(r *core.Record) (reviewsapi.Restaurant, error) {
	users, err := s.creators(r)
	if err != nil {
		return reviewsapi.Restaurant{}, err
	}
	ratings, err := s.ratings(r)
	if err != nil {
		return reviewsapi.Restaurant{}, err
	}
	return toRestaurant(r, users, ratings), nil
}

// toRestaurant gives a restaurant that ratings does not list no reviews.
func toRestaurant(r *core.Record, users map[string]*core.Record, ratings map[string]restaurantRating) reviewsapi.Restaurant {
	out := reviewsapi.Restaurant{
		Id:          r.Id,
		Name:        r.GetString("name"),
		Description: optionalText(r, "description"),
		Tags:        tagsOf(r),
		Creator:     nullable.NewNullNullable[reviewsapi.Creator](),
		Rating:      nullable.NewNullNullable[reviewsapi.RestaurantRating](),
		Created:     r.GetDateTime("created").String(),
		Updated:     r.GetDateTime("updated").String(),
	}
	if u, ok := users[r.GetString("creator")]; ok {
		out.Creator = nullable.NewNullableWithValue(toCreator(u))
	}
	if rating, ok := ratings[r.Id]; ok {
		out.ReviewCount = rating.Count
		out.Rating = nullable.NewNullableWithValue(rating.toAPI())
	}
	return out
}

func toCreator(user *core.Record) reviewsapi.Creator {
	return reviewsapi.Creator{Id: user.Id, Name: user.GetString("name")}
}

// optionalText returns nil for an empty text. PocketBase stores an empty
// string for a text that is not set.
func optionalText(r *core.Record, field string) *string {
	if v := r.GetString(field); v != "" {
		return &v
	}
	return nil
}

func tagsOf(r *core.Record) *reviewsapi.Tags {
	var tags []string
	if err := r.UnmarshalJSONField("tags", &tags); err != nil || tags == nil {
		return nil
	}
	return &tags
}

// changes sets only the fields that get a different value. A merge patch
// without changes then does not change the updated timestamp.
type changes struct {
	r       *core.Record
	changed bool
}

func (c *changes) set(field string, v any) {
	if c.r.Get(field) != v {
		c.r.Set(field, v)
		c.changed = true
	}
}

func (c *changes) patchText(field string, v nullable.Nullable[string]) {
	if v.IsSpecified() {
		text, _ := v.Get()
		c.set(field, text)
	}
}

func (c *changes) patchTags(v nullable.Nullable[reviewsapi.Tags]) {
	if !v.IsSpecified() {
		return
	}
	next, _ := v.Get()
	old := tagsOf(c.r)
	if v.IsNull() == (old == nil) && (old == nil || slices.Equal(*old, next)) {
		return
	}
	if v.IsNull() {
		c.r.Set("tags", nil)
	} else {
		c.r.Set("tags", next)
	}
	c.changed = true
}

func timestampComparisons(r *reviewsapi.TimestampRange) []comparison {
	if r == nil {
		return nil
	}
	return comparisons(r.Lt, r.Lte, r.Eq, r.Gte, r.Gt)
}
