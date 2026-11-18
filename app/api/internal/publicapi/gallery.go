package publicapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"strings"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// makeGalleryKeys puts the photos of the best review first. The first photo
// of a parent is its cover. See decision 00030.
func makeGalleryKeys(sub subject) []sortKey {
	return []sortKey{
		{expr: sub.reviewOverall(), desc: true},
		{expr: column(sub.Reviews, "created"), desc: true},
		{expr: column(sub.Reviews, "id"), desc: true},
		{expr: column(sub.Photos, "created")},
		{expr: column(sub.Photos, "id"), unique: true},
	}
}

// makeGalleryColumns gives the columns of galleryRow. They need joinReviews
// with the creators.
func makeGalleryColumns(sub subject) []string {
	return []string{
		column(sub.Photos, "id") + " AS [[id]]",
		column(sub.Photos, "review") + " AS [[review]]",
		column(sub.Photos, "alt") + " AS [[alt]]",
		column(sub.Photos, "width") + " AS [[width]]",
		column(sub.Photos, "height") + " AS [[height]]",
		column(sub.Photos, "created") + " AS [[created]]",
		column(sub.Reviews, sub.Parent) + " AS [[parent]]",
		column("users", "id") + " AS [[creator]]",
		column("users", "name") + " AS [[creator_name]]",
	}
}

type galleryRow struct {
	photoRow
	Parent      string `db:"parent"`
	Creator     string `db:"creator"`
	CreatorName string `db:"creator_name"`
}

func (r galleryRow) restaurantPhoto() reviewsapi.RestaurantPhoto {
	p := photoOf(restaurantSubject, r.photoRow, r.Parent)
	return reviewsapi.RestaurantPhoto{
		Id: p.Id, Alt: p.Alt, Width: p.Width, Height: p.Height, Urls: p.Urls, Created: p.Created,
		Review:  r.Review,
		Creator: reviewsapi.Creator{Id: r.Creator, Name: r.CreatorName},
	}
}

// joinReviews joins the review of each photo. With creators, it also joins
// the user of the review.
func (w *where) joinReviews(sub subject, creators bool) {
	w.joins = append(w.joins, join{inner: true, table: "{{" + sub.Reviews + "}}",
		on: dbx.NewExp(column(sub.Reviews, "id") + " = " + column(sub.Photos, "review"))})
	if creators {
		w.joins = append(w.joins, join{inner: true, table: "{{users}}",
			on: dbx.NewExp(column("users", "id") + " = " + column(sub.Reviews, "creator"))})
	}
}

func (s *Server) ListRestaurantPhotos(_ context.Context, req reviewsapi.ListRestaurantPhotosRequestObject) (reviewsapi.ListRestaurantPhotosResponseObject, error) {
	rows, next, p, err := s.galleryPage(restaurantSubject, req.RestaurantId, req.Params.Cursor, req.Params.Limit)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusBadRequest:
			return reviewsapi.ListRestaurantPhotos400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: p.badRequest()}, nil
		case http.StatusNotFound:
			return reviewsapi.ListRestaurantPhotos404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	items := make([]reviewsapi.RestaurantPhoto, len(rows))
	for i, r := range rows {
		items[i] = r.restaurantPhoto()
	}
	return reviewsapi.ListRestaurantPhotos200JSONResponse{Items: items, NextCursor: next}, nil
}

// galleryPage gets one page of the photos of all reviews of a parent. The
// problem is a 400 or a 404.
func (s *Server) galleryPage(sub subject, parentID string, cursor *string, limit *int) ([]galleryRow, nullable.Nullable[string], *problem, error) {
	keys := sub.galleryKeys
	query, err := fingerprint(sub.Parent+"_photos", parentID)
	if err != nil {
		return nil, nil, nil, err
	}
	after, detail := decodeCursor(cursor, query, len(keys))
	if detail != "" {
		return nil, nil, badRequestProblem(detail), nil
	}
	_, err = findParent(s.app, sub, parentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, notFoundProblem(sub.notFound), nil
	}
	if err != nil {
		return nil, nil, nil, err
	}

	w := galleryWhere(sub, false, parentID)
	if after != nil {
		w.keyset(keys, after)
	}
	ids, next, err := s.pageIDs(sub.Photos, w, keys, limitOf(limit), query)
	if err != nil {
		return nil, nil, nil, err
	}
	rows, err := s.galleryRows(sub, ids)
	return rows, next, nil, err
}

// galleryWhere keeps the photos of the reviews of the parents.
func galleryWhere(sub subject, creators bool, parentIDs ...string) *where {
	w := newWhere()
	w.joinReviews(sub, creators)
	w.in(column(sub.Reviews, sub.Parent), parentIDs)
	return w
}

// galleryRows gets the photos in the order of ids. A photo that is deleted
// after the page query is not in the result.
func (s *Server) galleryRows(sub subject, ids []string) ([]galleryRow, error) {
	out := make([]galleryRow, 0, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	w := newWhere()
	w.joinReviews(sub, true)
	w.in(column(sub.Photos, "id"), ids)
	var rows []galleryRow
	if err := s.selectWhere(sub.Photos, w, sub.galleryColumns...).All(&rows); err != nil {
		return nil, err
	}
	byID := make(map[string]galleryRow, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}
	for _, id := range ids {
		if r, ok := byID[id]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *Server) covers(restaurants ...*core.Record) (map[string]reviewsapi.RestaurantPhoto, error) {
	rows, err := s.coverRows(restaurantSubject, restaurants...)
	out := make(map[string]reviewsapi.RestaurantPhoto, len(rows))
	for id, r := range rows {
		out[id] = r.restaurantPhoto()
	}
	return out, err
}

// coverRows gets the first gallery photo of each parent in one query. A
// parent without photos has no entry.
func (s *Server) coverRows(sub subject, parents ...*core.Record) (map[string]galleryRow, error) {
	out := map[string]galleryRow{}
	if len(parents) == 0 {
		return out, nil
	}
	ids := make([]string, len(parents))
	for i, r := range parents {
		ids[i] = r.Id
	}
	var rows []galleryRow
	if err := s.coverQuery(sub, ids).All(&rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.Parent] = r
	}
	return out, nil
}

// coverQuery numbers the photos of each parent in the gallery order. It
// keeps the first photo of each.
func (s *Server) coverQuery(sub subject, parentIDs []string) *dbx.Query {
	rank := "ROW_NUMBER() OVER (PARTITION BY " + column(sub.Reviews, sub.Parent) +
		" ORDER BY " + strings.Join(orderTerms(sub.galleryKeys), ", ") + ") AS [[n]]"
	ranked := s.selectWhere(sub.Photos, galleryWhere(sub, true, parentIDs...), slices.Concat(sub.galleryColumns, []string{rank})...).Build()
	return s.app.DB().NewQuery("SELECT * FROM (" + ranked.SQL() + ") WHERE [[n]] = 1").Bind(ranked.Params())
}
