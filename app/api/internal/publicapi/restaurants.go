package publicapi

import (
	"context"
	"database/sql"
	"errors"
	"maps"
	"math"
	"slices"
	"strings"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
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
func (s *Server) ListRestaurants(ctx context.Context, req reviewsapi.ListRestaurantsRequestObject) (reviewsapi.ListRestaurantsResponseObject, error) {
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
	sorts := restaurantSorts
	if p.Near != nil {
		// The spec validation can be off. The sort key has the numbers in
		// its SQL text.
		if math.IsNaN(p.Near.Latitude+p.Near.Longitude) || math.IsInf(p.Near.Latitude+p.Near.Longitude, 0) {
			return bad("The near values must be numbers.")
		}
		sorts = maps.Clone(restaurantSorts)
		sorts["distance"] = distanceSort(p.Near)
	} else if sortSpec != nil && sortsBy(*sortSpec, "distance") {
		return bad("The distance sort needs the near parameter.")
	}
	keys, sort, detail := parseSort(sortSpec, "restaurants", sorts)
	if detail != "" {
		return bad(detail)
	}
	if detail := overallDetail(p.Overall); detail != "" {
		return bad(detail)
	}
	if p.Within != nil && p.Within.South > p.Within.North {
		return bad("The within[south] parameter must not be more than within[north].")
	}
	p.Limit, p.Cursor, p.Sort = nil, nil, &sort
	parts := []any{"restaurants", p}
	// The overall sort keys changed in decisions 00012 and 00029. The version
	// rejects the cursors of the old keys.
	if sortsBy(sort, "overall") {
		parts = append(parts, 3)
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
	w.near(p.Near)
	w.within(p.Within)
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
	items, err := s.restaurantsOf(stateFrom(ctx).user, records...)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ListRestaurants200JSONResponse{Items: items, NextCursor: next}, nil
}

func sortsBy(sort, field string) bool {
	return slices.ContainsFunc(strings.Split(sort, ","), func(f string) bool { return strings.TrimPrefix(f, "-") == field })
}

// CreateRestaurant sets the user of the bearer token as the creator. The
// place check and the save run in one transaction.
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
	setLocation(r, b.Location)
	setOptionalText(r, "website", b.Website)
	setOptionalText(r, "phone", b.Phone)
	setOptionalText(r, "opening_hours", b.OpeningHours)
	if b.PriceLevel != nil {
		r.Set("price_level", float64(*b.PriceLevel))
	}
	r.Set("creator", user.Id)
	var duplicate string
	err = s.app.RunInTransaction(func(tx core.App) error {
		if duplicate, err = restaurantWithPlace(tx, r); err != nil || duplicate != "" {
			return err
		}
		return tx.Save(r)
	})
	if err != nil {
		return nil, err
	}
	if duplicate != "" {
		return reviewsapi.CreateRestaurant409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: placeConflict(duplicate)}, nil
	}

	location := basePath + "/restaurants/" + r.Id
	return reviewsapi.CreateRestaurant201JSONResponse{
		Body:    toRestaurant(r, map[string]*core.Record{user.Id: user}, nil, nil, nil),
		Headers: reviewsapi.CreateRestaurant201ResponseHeaders{Location: &location},
	}, nil
}

func (s *Server) GetRestaurant(ctx context.Context, req reviewsapi.GetRestaurantRequestObject) (reviewsapi.GetRestaurantResponseObject, error) {
	r, err := s.findRestaurant(s.app, req.RestaurantId)
	if errors.Is(err, sql.ErrNoRows) {
		return reviewsapi.GetRestaurant404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(restaurantNotFound)}, nil
	}
	if err != nil {
		return nil, err
	}
	out, err := s.restaurantResponse(stateFrom(ctx).user, r)
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
			response = reviewsapi.UpdateRestaurant403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: notCreator(detail)}
			return nil
		}

		hadLocation := r.GetString("address") != ""
		renames := b.Name != nil && *b.Name != r.GetString("name")
		ch := &changes{r: r}
		if b.Name != nil {
			ch.set("name", *b.Name)
		}
		ch.patchText("description", b.Description)
		ch.patchTags(b.Tags)
		ch.patchText("website", b.Website)
		ch.patchText("phone", b.Phone)
		ch.patchText("opening_hours", b.OpeningHours)
		ch.patchPriceLevel(b.PriceLevel)
		moves := b.Location != nil && !sameLocation(r, *b.Location)
		if moves {
			setLocation(r, *b.Location)
			ch.changed = true
		}
		if renames || (moves && hadLocation) {
			n, err := countReviewsOfOtherUsers(tx, r.Id, user.Id)
			if err != nil {
				return err
			}
			if n > 0 {
				property := "name"
				if !renames {
					property = "location"
				}
				response = reviewsapi.UpdateRestaurant409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: propertyConflict(
					reviewsapi.ProblemCodeOtherReviews, property, "The "+property+" cannot change because the restaurant has reviews of other users.")}
				return nil
			}
		}
		if moves {
			duplicate, err := restaurantWithPlace(tx, r)
			if err != nil {
				return err
			}
			if duplicate != "" {
				response = reviewsapi.UpdateRestaurant409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: placeConflict(duplicate)}
				return nil
			}
		}
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
	out, err := s.restaurantResponse(user, saved)
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
			response = reviewsapi.DeleteRestaurant403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: notCreator(detail)}
			return nil
		}
		n, err := countReviewsOfOtherUsers(tx, r.Id, user.Id)
		if err != nil {
			return err
		}
		if n > 0 {
			response = reviewsapi.DeleteRestaurant409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict(reviewsapi.ProblemCodeOtherReviews, "The restaurant has reviews of other users. It cannot be deleted.")}
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

// setLocation replaces the complete location. A location without a place
// ID clears the stored place ID.
func setLocation(r *core.Record, l reviewsapi.Location) {
	r.Set("latitude", l.Latitude)
	r.Set("longitude", l.Longitude)
	r.Set("address", l.Address)
	r.Set("place_id", placeIDOf(l))
}

func sameLocation(r *core.Record, l reviewsapi.Location) bool {
	return r.GetFloat("latitude") == l.Latitude && r.GetFloat("longitude") == l.Longitude &&
		r.GetString("address") == l.Address && r.GetString("place_id") == placeIDOf(l)
}

func placeIDOf(l reviewsapi.Location) string {
	if l.PlaceId == nil {
		return ""
	}
	return *l.PlaceId
}

// restaurantWithPlace returns the ID of a different restaurant that has the
// place ID of r. It returns an empty string if no restaurant has it.
func restaurantWithPlace(app core.App, r *core.Record) (string, error) {
	placeID := r.GetString("place_id")
	if placeID == "" {
		return "", nil
	}
	var id string
	err := app.DB().Select("id").From("restaurants").
		Where(dbx.HashExp{"place_id": placeID}).AndWhere(dbx.Not(dbx.HashExp{"id": r.Id})).
		Limit(1).Row(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// locationOf returns null for a restaurant without an address. Such a
// restaurant is from before decision 00017.
func locationOf(r *core.Record) nullable.Nullable[reviewsapi.Location] {
	address := r.GetString("address")
	if address == "" {
		return nullable.NewNullNullable[reviewsapi.Location]()
	}
	l := reviewsapi.Location{Latitude: r.GetFloat("latitude"), Longitude: r.GetFloat("longitude"), Address: address}
	if id := r.GetString("place_id"); id != "" {
		l.PlaceId = &id
	}
	return nullable.NewNullableWithValue(l)
}

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

func (s *Server) restaurantResponse(user, r *core.Record) (reviewsapi.Restaurant, error) {
	out, err := s.restaurantsOf(user, r)
	if err != nil {
		return reviewsapi.Restaurant{}, err
	}
	return out[0], nil
}

// restaurantsOf gets the creators, ratings, and covers of records. A nil
// user has no favorite.
func (s *Server) restaurantsOf(user *core.Record, records ...*core.Record) ([]reviewsapi.Restaurant, error) {
	users, err := s.creators(records...)
	if err != nil {
		return nil, err
	}
	ratings, err := s.ratings(records...)
	if err != nil {
		return nil, err
	}
	covers, err := s.covers(records...)
	if err != nil {
		return nil, err
	}
	favorites, err := s.favoriteIDs(user, records...)
	if err != nil {
		return nil, err
	}
	items := make([]reviewsapi.Restaurant, len(records))
	for i, r := range records {
		items[i] = toRestaurant(r, users, ratings, covers, favorites)
	}
	return items, nil
}

// toRestaurant gives a restaurant that ratings does not list no reviews.
func toRestaurant(r *core.Record, users map[string]*core.Record, ratings map[string]restaurantRating, covers map[string]reviewsapi.RestaurantPhoto, favorites map[string]bool) reviewsapi.Restaurant {
	out := reviewsapi.Restaurant{
		Id:           r.Id,
		Name:         r.GetString("name"),
		Description:  optionalText(r, "description"),
		Tags:         tagsOf(r),
		Location:     locationOf(r),
		Website:      optionalText(r, "website"),
		Phone:        optionalText(r, "phone"),
		OpeningHours: optionalText(r, "opening_hours"),
		PriceLevel:   priceLevelOf(r),
		Creator:      nullable.NewNullNullable[reviewsapi.Creator](),
		Rating:       nullable.NewNullNullable[reviewsapi.RestaurantRating](),
		Cover:        nullable.NewNullNullable[reviewsapi.RestaurantPhoto](),
		Favorite:     favorites[r.Id],
		Created:      apiTimestamp(r.GetDateTime("created")),
		Updated:      apiTimestamp(r.GetDateTime("updated")),
	}
	if u, ok := users[r.GetString("creator")]; ok {
		out.Creator = nullable.NewNullableWithValue(toCreator(u))
	}
	if rating, ok := ratings[r.Id]; ok {
		out.ReviewCount = rating.Count
		out.Rating = nullable.NewNullableWithValue(rating.toAPI())
	}
	if cover, ok := covers[r.Id]; ok {
		out.Cover = nullable.NewNullableWithValue(cover)
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

func setOptionalText(r *core.Record, field string, v *string) {
	if v != nil {
		r.Set(field, *v)
	}
}

// priceLevelOf returns nil for the stored 0 of a restaurant without a price
// level.
func priceLevelOf(r *core.Record) *reviewsapi.PriceLevel {
	if v := r.GetInt("price_level"); v != 0 {
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

// patchPriceLevel stores 0 for null.
func (c *changes) patchPriceLevel(v nullable.Nullable[reviewsapi.PriceLevel]) {
	if v.IsSpecified() {
		level, _ := v.Get()
		c.set("price_level", float64(level))
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
	cs := comparisons(r.Lt, r.Lte, r.Eq, r.Gte, r.Gt)
	for i := range cs {
		cs[i].value = storedTimestamp(cs[i].value.(string))
	}
	return cs
}

// timestampLayout is the RFC 3339 format of the API. PocketBase stores a
// space in the place of the T.
const timestampLayout = "2006-01-02T15:04:05.000Z"

// apiTimestamp returns an empty string for a record without the value. The
// response validation then refuses the response.
func apiTimestamp(t types.DateTime) string {
	if t.IsZero() {
		return ""
	}
	return t.Time().UTC().Format(timestampLayout)
}

// storedTimestamp changes a timestamp of the API to the format of the
// database. The spec pattern makes sure of the format.
func storedTimestamp(value string) string {
	return strings.Replace(value, "T", " ", 1)
}
