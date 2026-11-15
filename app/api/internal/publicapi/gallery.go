package publicapi

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// galleryKeys put the photos of the best review first. The first photo of a
// restaurant is its cover. See decision 00030.
var galleryKeys = []sortKey{
	{expr: reviewOverall, desc: true},
	{expr: column("reviews", "created"), desc: true},
	{expr: column("reviews", "id"), desc: true},
	{expr: column("review_photos", "created")},
	{expr: column("review_photos", "id"), unique: true},
}

// galleryColumns are the columns of galleryRow. They need joinReviews with
// the creators.
var galleryColumns = []string{
	column("review_photos", "id") + " AS [[id]]",
	column("review_photos", "review") + " AS [[review]]",
	column("review_photos", "alt") + " AS [[alt]]",
	column("review_photos", "width") + " AS [[width]]",
	column("review_photos", "height") + " AS [[height]]",
	column("review_photos", "created") + " AS [[created]]",
	column("reviews", "restaurant") + " AS [[restaurant]]",
	column("users", "id") + " AS [[creator]]",
	column("users", "name") + " AS [[creator_name]]",
}

type galleryRow struct {
	photoRow
	Restaurant  string `db:"restaurant"`
	Creator     string `db:"creator"`
	CreatorName string `db:"creator_name"`
}

func (r galleryRow) toAPI() reviewsapi.RestaurantPhoto {
	p := photoOf(r.photoRow, r.Restaurant)
	return reviewsapi.RestaurantPhoto{
		Id: p.Id, Alt: p.Alt, Width: p.Width, Height: p.Height, Urls: p.Urls, Created: p.Created,
		Review:  r.Review,
		Creator: reviewsapi.Creator{Id: r.Creator, Name: r.CreatorName},
	}
}

// joinReviews joins the review of each photo. With creators, it also joins
// the user of the review.
func (w *where) joinReviews(creators bool) {
	w.joins = append(w.joins, join{inner: true, table: "{{reviews}}",
		on: dbx.NewExp(column("reviews", "id") + " = " + column("review_photos", "review"))})
	if creators {
		w.joins = append(w.joins, join{inner: true, table: "{{users}}",
			on: dbx.NewExp(column("users", "id") + " = " + column("reviews", "creator"))})
	}
}

func (s *Server) ListRestaurantPhotos(_ context.Context, req reviewsapi.ListRestaurantPhotosRequestObject) (reviewsapi.ListRestaurantPhotosResponseObject, error) {
	query, err := fingerprint("restaurant_photos", req.RestaurantId)
	if err != nil {
		return nil, err
	}
	after, detail := decodeCursor(req.Params.Cursor, query, len(galleryKeys))
	if detail != "" {
		return reviewsapi.ListRestaurantPhotos400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(detail)}, nil
	}
	_, err = s.findRestaurant(s.app, req.RestaurantId)
	if errors.Is(err, sql.ErrNoRows) {
		return reviewsapi.ListRestaurantPhotos404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(restaurantNotFound)}, nil
	}
	if err != nil {
		return nil, err
	}

	w := galleryWhere(false, req.RestaurantId)
	if after != nil {
		w.keyset(galleryKeys, after)
	}
	ids, next, err := s.pageIDs("review_photos", w, galleryKeys, limitOf(req.Params.Limit), query)
	if err != nil {
		return nil, err
	}
	items, err := s.restaurantPhotos(ids)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ListRestaurantPhotos200JSONResponse{Items: items, NextCursor: next}, nil
}

// galleryWhere keeps the photos of the reviews of the restaurants.
func galleryWhere(creators bool, restaurantIDs ...string) *where {
	w := newWhere()
	w.joinReviews(creators)
	w.in(column("reviews", "restaurant"), restaurantIDs)
	return w
}

// restaurantPhotos gets the photos in the order of ids. A photo that is
// deleted after the page query is not in the result.
func (s *Server) restaurantPhotos(ids []string) ([]reviewsapi.RestaurantPhoto, error) {
	out := make([]reviewsapi.RestaurantPhoto, 0, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	w := newWhere()
	w.joinReviews(true)
	w.in(column("review_photos", "id"), ids)
	var rows []galleryRow
	if err := s.selectWhere("review_photos", w, galleryColumns...).All(&rows); err != nil {
		return nil, err
	}
	byID := make(map[string]galleryRow, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}
	for _, id := range ids {
		if r, ok := byID[id]; ok {
			out = append(out, r.toAPI())
		}
	}
	return out, nil
}

// covers gets the first gallery photo of each restaurant in one query. A
// restaurant without photos has no entry.
func (s *Server) covers(restaurants ...*core.Record) (map[string]reviewsapi.RestaurantPhoto, error) {
	out := map[string]reviewsapi.RestaurantPhoto{}
	if len(restaurants) == 0 {
		return out, nil
	}
	ids := make([]string, len(restaurants))
	for i, r := range restaurants {
		ids[i] = r.Id
	}
	var rows []galleryRow
	if err := s.coverQuery(ids).All(&rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.Restaurant] = r.toAPI()
	}
	return out, nil
}

// coverQuery numbers the photos of each restaurant in the gallery order. It
// keeps the first photo of each.
func (s *Server) coverQuery(restaurantIDs []string) *dbx.Query {
	rank := "ROW_NUMBER() OVER (PARTITION BY " + column("reviews", "restaurant") +
		" ORDER BY " + strings.Join(orderTerms(galleryKeys), ", ") + ") AS [[n]]"
	ranked := s.selectWhere("review_photos", galleryWhere(true, restaurantIDs...), slices.Concat(galleryColumns, []string{rank})...).Build()
	return s.app.DB().NewQuery("SELECT * FROM (" + ranked.SQL() + ") WHERE [[n]] = 1").Bind(ranked.Params())
}
