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

func restaurantSorts(rated *ratedPage, search *searchPage) map[string]sortField {
	return map[string]sortField{
		"name":      textSort("restaurants", "name"),
		"overall":   rated.overallSort,
		"relevance": search.relevanceSort,
		"created":   timestampSort("restaurants", "created"),
		"updated":   timestampSort("restaurants", "updated"),
	}
}

// ListRestaurants uses keyset pagination. The cursor has the sort values of
// the last item.
func (s *Server) ListRestaurants(_ context.Context, req reviewsapi.ListRestaurantsRequestObject) (reviewsapi.ListRestaurantsResponseObject, error) {
	p := req.Params
	bad := func(detail string) (reviewsapi.ListRestaurantsResponseObject, error) {
		return reviewsapi.ListRestaurants400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(detail)}, nil
	}

	limit, detail := parseLimit(p.Limit)
	if detail != "" {
		return bad(detail)
	}
	rated := &ratedPage{s: s}
	search := newSearchPage(s, "restaurants")
	if detail := search.parse(p.Search); detail != "" {
		return bad(detail)
	}
	sortSpec, detail := search.sortWithSearch(p.Sort)
	if detail != "" {
		return bad(detail)
	}
	keys, sort, detail := parseSort(sortSpec, "restaurants", restaurantSorts(rated, search))
	if detail != "" {
		return bad(detail)
	}
	// fingerprint cannot marshal NaN. The check runs first.
	if detail := checkAverages("overall", averageComparisons(p.Overall)); detail != "" {
		return bad(detail)
	}
	p.Limit, p.Cursor, p.Sort = nil, nil, &sort
	query, err := fingerprint("restaurants", p)
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
	for _, detail := range []string{
		w.tagFilters("restaurants", p.TagsHas, p.TagsHasAny, p.TagsHasAll),
		w.creatorFilters("restaurants", p.CreatorEq, p.CreatorIn),
		w.timestampFilter("restaurants", "created", timestampComparisons(p.Created)),
		w.timestampFilter("restaurants", "updated", timestampComparisons(p.Updated)),
	} {
		if detail != "" {
			return bad(detail)
		}
	}
	if after != nil {
		w.keyset(keys, after)
	}

	records, next, err := s.page("restaurants", w, keys, limit, query, loadAll(rated.load, search.load))
	if err != nil {
		return nil, err
	}
	users, err := s.creators(records...)
	if err != nil {
		return nil, err
	}
	items := make([]reviewsapi.Restaurant, len(records))
	for i, r := range records {
		items[i] = toRestaurant(r, users, rated.ratings)
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

// GetRestaurant returns 400 for an ID that does not match the RecordId
// pattern. The other operations with IDs do the same.
func (s *Server) GetRestaurant(_ context.Context, req reviewsapi.GetRestaurantRequestObject) (reviewsapi.GetRestaurantResponseObject, error) {
	if !recordIDPattern.MatchString(req.RestaurantId) {
		return reviewsapi.GetRestaurant400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(invalidRestaurantID)}, nil
	}
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
	if !recordIDPattern.MatchString(req.RestaurantId) {
		return reviewsapi.UpdateRestaurant400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(invalidRestaurantID)}, nil
	}
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
		if r.GetString("creator") != user.Id {
			response = reviewsapi.UpdateRestaurant403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden("Only the creator of the restaurant can change it.")}
			return nil
		}

		if b.Name != nil && *b.Name != r.GetString("name") {
			n, err := tx.CountRecords("reviews", dbx.HashExp{"restaurant": r.Id})
			if err != nil {
				return err
			}
			if n > 0 {
				response = reviewsapi.UpdateRestaurant409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict("The name cannot change because the restaurant has reviews.")}
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
func (s *Server) DeleteRestaurant(ctx context.Context, req reviewsapi.DeleteRestaurantRequestObject) (reviewsapi.DeleteRestaurantResponseObject, error) {
	if !recordIDPattern.MatchString(req.RestaurantId) {
		return reviewsapi.DeleteRestaurant400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(invalidRestaurantID)}, nil
	}
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
		if r.GetString("creator") != user.Id {
			response = reviewsapi.DeleteRestaurant403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden("Only the creator of the restaurant can delete it.")}
			return nil
		}
		n, err := tx.CountRecords("reviews", dbx.HashExp{"restaurant": r.Id})
		if err != nil {
			return err
		}
		if n > 0 {
			response = reviewsapi.DeleteRestaurant409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict("The restaurant has reviews. Only a restaurant without reviews can be deleted.")}
			return nil
		}
		return tx.Delete(r)
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

const (
	invalidRestaurantID = "The restaurant ID is not valid."
	restaurantNotFound  = "The restaurant does not exist."
)

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
