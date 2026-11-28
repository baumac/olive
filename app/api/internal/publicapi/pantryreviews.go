package publicapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	reviewsapi "github.com/baumac/olive/specs/api/sdk/server/go"
	"github.com/pocketbase/pocketbase/core"
)

// ListPantryItemReviews returns 404 when the pantry item does not exist. It
// does not return an empty list.
func (s *Server) ListPantryItemReviews(_ context.Context, req reviewsapi.ListPantryItemReviewsRequestObject) (reviewsapi.ListPantryItemReviewsResponseObject, error) {
	p := req.Params
	all := reviewsapi.ListAllPantryItemReviewsParams{
		Limit: p.Limit, Cursor: p.Cursor, Sort: p.Sort, Search: p.Search, PantryItemEq: &req.PantryItemId,
		Taste: p.Taste, Aroma: p.Aroma, Texture: p.Texture, Value: p.Value,
		Overall: p.Overall, Date: p.Date, Created: p.Created, Updated: p.Updated, CreatorEq: p.CreatorEq, CreatorIn: p.CreatorIn,
	}
	fingerprintOf := func(sort string) []any {
		q := req.Params
		q.Limit, q.Cursor, q.Sort = nil, nil, &sort
		return []any{"pantry-item-reviews", req.PantryItemId, q}
	}
	page, err := s.pantryItemReviewPage(all, fingerprintOf, func() (*core.Record, error) {
		r, err := findParent(s.app, pantryItemSubject, req.PantryItemId)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return r, err
	})
	switch {
	case err != nil:
		return nil, err
	case page.bad != "":
		return reviewsapi.ListPantryItemReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(page.bad)}, nil
	case page.notFound:
		return reviewsapi.ListPantryItemReviews404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(pantryItemNotFound)}, nil
	}
	return reviewsapi.ListPantryItemReviews200JSONResponse(page.list), nil
}

// ListAllPantryItemReviews gives an empty list for an unknown pantry item in
// pantry_item[eq], not 404.
func (s *Server) ListAllPantryItemReviews(_ context.Context, req reviewsapi.ListAllPantryItemReviewsRequestObject) (reviewsapi.ListAllPantryItemReviewsResponseObject, error) {
	page, err := s.pantryItemReviewPage(req.Params, nil, nil)
	if err != nil {
		return nil, err
	}
	if page.bad != "" {
		return reviewsapi.ListAllPantryItemReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(page.bad)}, nil
	}
	return reviewsapi.ListAllPantryItemReviews200JSONResponse(page.list), nil
}

// ListReviewerPantryItemReviews returns 404 when the user does not exist. It
// does not return an empty list.
func (s *Server) ListReviewerPantryItemReviews(ctx context.Context, req reviewsapi.ListReviewerPantryItemReviewsRequestObject) (reviewsapi.ListReviewerPantryItemReviewsResponseObject, error) {
	p := req.Params
	all := reviewsapi.ListAllPantryItemReviewsParams{
		Limit: p.Limit, Cursor: p.Cursor, Sort: p.Sort, Search: p.Search, CreatorEq: &req.ReviewerId,
		PantryItemEq: p.PantryItemEq, PantryItemIn: p.PantryItemIn,
		Taste: p.Taste, Aroma: p.Aroma, Texture: p.Texture, Value: p.Value,
		Overall: p.Overall, Date: p.Date, Created: p.Created, Updated: p.Updated,
	}
	fingerprintOf := func(sort string) []any {
		q := req.Params
		q.Limit, q.Cursor, q.Sort = nil, nil, &sort
		return []any{"reviewer-pantry-item-reviews", req.ReviewerId, q}
	}
	page, err := s.pantryItemReviewPage(all, fingerprintOf, nil)
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
		return reviewsapi.ListReviewerPantryItemReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(page.bad)}, nil
	case page.notFound:
		return reviewsapi.ListReviewerPantryItemReviews404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(reviewerNotFound)}, nil
	}
	return reviewsapi.ListReviewerPantryItemReviews200JSONResponse(page.list), nil
}

type pantryItemReviewPageResult struct {
	list     reviewsapi.PantryItemReviewList
	bad      string
	notFound bool
}

// pantryItemReviewPage gets a page of pantry item reviews. A nil
// fingerprintOf uses the parameters of listAllPantryItemReviews.
func (s *Server) pantryItemReviewPage(
	p reviewsapi.ListAllPantryItemReviewsParams,
	fingerprintOf func(sort string) []any,
	item func() (*core.Record, error),
) (pantryItemReviewPageResult, error) {
	if fingerprintOf == nil {
		fingerprintOf = func(sort string) []any {
			q := p
			q.Limit, q.Cursor, q.Sort = nil, nil, &sort
			return []any{"all-pantry-item-reviews", q}
		}
	}
	page, err := s.reviewsOf(pantryItemSubject, reviewQuery{
		limit: p.Limit, cursor: p.Cursor, sort: p.Sort, search: p.Search,
		parentEq: p.PantryItemEq, parentIn: p.PantryItemIn,
		scores: map[string]*reviewsapi.ScoreRange{
			"taste": p.Taste, "aroma": p.Aroma, "texture": p.Texture, "value": p.Value,
		},
		overall: p.Overall, date: p.Date, created: p.Created, updated: p.Updated, creatorEq: p.CreatorEq, creatorIn: p.CreatorIn,
	}, fingerprintOf, item)
	if err != nil || page.bad != "" || page.notFound {
		return pantryItemReviewPageResult{bad: page.bad, notFound: page.notFound}, err
	}
	items := make([]reviewsapi.PantryItemReview, len(page.records))
	for i, r := range page.records {
		if items[i], err = toPantryItemReview(r, page.users[r.GetString("creator")], page.names[r.GetString("pantry_item")], page.photos[r.Id]); err != nil {
			return pantryItemReviewPageResult{}, err
		}
	}
	return pantryItemReviewPageResult{list: reviewsapi.PantryItemReviewList{Items: items, NextCursor: page.next}}, nil
}

// CreatePantryItemReview makes a review of one tasting. A user can write many
// reviews of one pantry item. See decision 00005.
func (s *Server) CreatePantryItemReview(ctx context.Context, req reviewsapi.CreatePantryItemReviewRequestObject) (reviewsapi.CreatePantryItemReviewResponseObject, error) {
	user := stateFrom(ctx).user
	b := req.Body
	in := reviewInput{
		scores: map[string]int{"taste": b.Taste, "aroma": b.Aroma, "texture": b.Texture, "value": b.Value},
		date:   &b.Date, title: &b.Title, feedback: optional(b.Feedback),
	}
	saved, item, p, err := s.createReview(pantryItemSubject, req.PantryItemId, user, in)
	if err != nil {
		return nil, err
	}
	if p != nil {
		if p.status() == http.StatusNotFound {
			return reviewsapi.CreatePantryItemReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	out, err := toPantryItemReview(saved, user, item.GetString("name"), nil)
	if err != nil {
		return nil, err
	}
	location := pantryItemSubject.reviewPath(req.PantryItemId, saved.Id)
	return reviewsapi.CreatePantryItemReview201JSONResponse{
		Body:    out,
		Headers: reviewsapi.CreatePantryItemReview201ResponseHeaders{Location: &location},
	}, nil
}

// GetPantryItemReview returns 404 for a review of a different pantry item.
func (s *Server) GetPantryItemReview(_ context.Context, req reviewsapi.GetPantryItemReviewRequestObject) (reviewsapi.GetPantryItemReviewResponseObject, error) {
	item, r, detail, err := s.findReview(s.app, pantryItemSubject, req.PantryItemId, req.ReviewId)
	if err != nil {
		return nil, err
	}
	if detail != "" {
		return reviewsapi.GetPantryItemReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}, nil
	}
	users, err := s.creators(r)
	if err != nil {
		return nil, err
	}
	photos, err := s.reviewPhotos(pantryItemSubject, r)
	if err != nil {
		return nil, err
	}
	out, err := toPantryItemReview(r, users[r.GetString("creator")], item.GetString("name"), photos[r.Id])
	if err != nil {
		return nil, err
	}
	return reviewsapi.GetPantryItemReview200JSONResponse(out), nil
}

// UpdatePantryItemReview lets only the creator change a review.
func (s *Server) UpdatePantryItemReview(ctx context.Context, req reviewsapi.UpdatePantryItemReviewRequestObject) (reviewsapi.UpdatePantryItemReviewResponseObject, error) {
	user := stateFrom(ctx).user
	b := req.Body
	in := reviewInput{scores: map[string]int{}, date: b.Date, title: b.Title, feedback: b.Feedback}
	setScore(in.scores, "taste", b.Taste)
	setScore(in.scores, "aroma", b.Aroma)
	setScore(in.scores, "texture", b.Texture)
	setScore(in.scores, "value", b.Value)
	saved, item, p, err := s.updateReview(pantryItemSubject, req.PantryItemId, req.ReviewId, user, in)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusNotFound:
			return reviewsapi.UpdatePantryItemReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusForbidden:
			return reviewsapi.UpdatePantryItemReview403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		}
		return nil, p.unexpected()
	}
	photos, err := s.reviewPhotos(pantryItemSubject, saved)
	if err != nil {
		return nil, err
	}
	out, err := toPantryItemReview(saved, user, item.GetString("name"), photos[saved.Id])
	if err != nil {
		return nil, err
	}
	return reviewsapi.UpdatePantryItemReview200JSONResponse(out), nil
}

// DeletePantryItemReview lets only the creator delete a review.
func (s *Server) DeletePantryItemReview(ctx context.Context, req reviewsapi.DeletePantryItemReviewRequestObject) (reviewsapi.DeletePantryItemReviewResponseObject, error) {
	p, err := s.deleteReview(pantryItemSubject, req.PantryItemId, req.ReviewId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusNotFound:
			return reviewsapi.DeletePantryItemReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusForbidden:
			return reviewsapi.DeletePantryItemReview403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.DeletePantryItemReview204Response{}, nil
}

// toPantryItemReview gives an empty photos array for nil photos. The spec
// requires the array.
func toPantryItemReview(r *core.Record, creator *core.Record, itemName string, photos []reviewsapi.Photo) (reviewsapi.PantryItemReview, error) {
	if creator == nil {
		return reviewsapi.PantryItemReview{}, fmt.Errorf("pantry item review %s has no creator", r.Id)
	}
	if itemName == "" {
		return reviewsapi.PantryItemReview{}, fmt.Errorf("pantry item review %s has no pantry item", r.Id)
	}
	if photos == nil {
		photos = []reviewsapi.Photo{}
	}
	taste, aroma, texture, value := r.GetInt("taste"), r.GetInt("aroma"), r.GetInt("texture"), r.GetInt("value")
	// TestPantryItemOverallColumnEqualsResponse keeps the overall column
	// equal to this.
	return reviewsapi.PantryItemReview{
		Id: r.Id, PantryItem: r.GetString("pantry_item"), PantryItemName: itemName,
		Date: r.GetString("date"), Overall: float64(taste+aroma+texture+value) / 4,
		Taste: taste, Aroma: aroma, Texture: texture, Value: value,
		Title: r.GetString("title"), Feedback: optionalText(r, "feedback"),
		Creator: toCreator(creator), Photos: photos, Created: apiTimestamp(r.GetDateTime("created")), Updated: apiTimestamp(r.GetDateTime("updated")),
	}, nil
}

// AddPantryItemReviewPhoto lets only the creator of the review add a photo. A
// review has at most 5 photos.
func (s *Server) AddPantryItemReviewPhoto(ctx context.Context, req reviewsapi.AddPantryItemReviewPhotoRequestObject) (reviewsapi.AddPantryItemReviewPhotoResponseObject, error) {
	saved, p, err := s.addPhoto(ctx, pantryItemSubject, req.PantryItemId, req.ReviewId, req.ContentType, req.Body)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusBadRequest:
			return reviewsapi.AddPantryItemReviewPhoto400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: p.badRequest()}, nil
		case http.StatusForbidden:
			return reviewsapi.AddPantryItemReviewPhoto403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		case http.StatusNotFound:
			return reviewsapi.AddPantryItemReviewPhoto404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusConflict:
			return reviewsapi.AddPantryItemReviewPhoto409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: p.conflict()}, nil
		case http.StatusRequestEntityTooLarge:
			return reviewsapi.AddPantryItemReviewPhoto413ApplicationProblemPlusJSONResponse{ContentTooLargeApplicationProblemPlusJSONResponse: p.contentTooLarge()}, nil
		case http.StatusUnsupportedMediaType:
			return reviewsapi.AddPantryItemReviewPhoto415ApplicationProblemPlusJSONResponse{UnsupportedMediaTypeApplicationProblemPlusJSONResponse: p.unsupportedMediaType()}, nil
		case http.StatusUnprocessableEntity:
			return reviewsapi.AddPantryItemReviewPhoto422ApplicationProblemPlusJSONResponse{InvalidImageApplicationProblemPlusJSONResponse: p.invalidImage()}, nil
		case http.StatusTooManyRequests:
			return reviewsapi.AddPantryItemReviewPhoto429ApplicationProblemPlusJSONResponse{TooManyRequestsApplicationProblemPlusJSONResponse: p.tooManyRequests()}, nil
		}
		return nil, p.unexpected()
	}
	location := pantryItemSubject.photoPath(req.PantryItemId, req.ReviewId, saved.Id)
	return reviewsapi.AddPantryItemReviewPhoto201JSONResponse{
		Body:    toPhoto(pantryItemSubject, saved, req.PantryItemId),
		Headers: reviewsapi.AddPantryItemReviewPhoto201ResponseHeaders{Location: &location},
	}, nil
}

// UpdatePantryItemReviewPhoto changes only the alt text. The image never
// changes.
func (s *Server) UpdatePantryItemReviewPhoto(ctx context.Context, req reviewsapi.UpdatePantryItemReviewPhotoRequestObject) (reviewsapi.UpdatePantryItemReviewPhotoResponseObject, error) {
	saved, p, err := s.updatePhoto(pantryItemSubject, req.PantryItemId, req.ReviewId, req.PhotoId, stateFrom(ctx).user, req.Body)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusNotFound:
			return reviewsapi.UpdatePantryItemReviewPhoto404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusForbidden:
			return reviewsapi.UpdatePantryItemReviewPhoto403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.UpdatePantryItemReviewPhoto200JSONResponse(toPhoto(pantryItemSubject, saved, req.PantryItemId)), nil
}

// DeletePantryItemReviewPhoto lets only the creator of the review delete a
// photo.
func (s *Server) DeletePantryItemReviewPhoto(ctx context.Context, req reviewsapi.DeletePantryItemReviewPhotoRequestObject) (reviewsapi.DeletePantryItemReviewPhotoResponseObject, error) {
	p, err := s.deletePhoto(pantryItemSubject, req.PantryItemId, req.ReviewId, req.PhotoId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusNotFound:
			return reviewsapi.DeletePantryItemReviewPhoto404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusForbidden:
			return reviewsapi.DeletePantryItemReviewPhoto403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.DeletePantryItemReviewPhoto204Response{}, nil
}

// GetPantryItemReviewPhotoFile sends the stored JPEG of the variant.
func (s *Server) GetPantryItemReviewPhotoFile(ctx context.Context, req reviewsapi.GetPantryItemReviewPhotoFileRequestObject) (reviewsapi.GetPantryItemReviewPhotoFileResponseObject, error) {
	file, detail, err := s.photoFile(ctx, pantryItemSubject, req.PantryItemId, req.ReviewId, req.PhotoId, req.Variant)
	if err != nil {
		return nil, err
	}
	if detail != "" {
		return reviewsapi.GetPantryItemReviewPhotoFile404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}, nil
	}
	return file, nil
}

func (f photoFileResponse) VisitGetPantryItemReviewPhotoFileResponse(w http.ResponseWriter) error {
	return f.VisitGetRestaurantReviewPhotoFileResponse(w)
}

func (s *Server) ListPantryItemPhotos(_ context.Context, req reviewsapi.ListPantryItemPhotosRequestObject) (reviewsapi.ListPantryItemPhotosResponseObject, error) {
	rows, next, p, err := s.galleryPage(pantryItemSubject, req.PantryItemId, req.Params.Cursor, req.Params.Limit)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusBadRequest:
			return reviewsapi.ListPantryItemPhotos400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: p.badRequest()}, nil
		case http.StatusNotFound:
			return reviewsapi.ListPantryItemPhotos404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	items := make([]reviewsapi.PantryItemPhoto, len(rows))
	for i, r := range rows {
		items[i] = r.pantryItemPhoto()
	}
	return reviewsapi.ListPantryItemPhotos200JSONResponse{Items: items, NextCursor: next}, nil
}

func (r galleryRow) pantryItemPhoto() reviewsapi.PantryItemPhoto {
	p := photoOf(pantryItemSubject, r.photoRow, r.Parent)
	return reviewsapi.PantryItemPhoto{
		Id: p.Id, Alt: p.Alt, Width: p.Width, Height: p.Height, Urls: p.Urls, Created: p.Created,
		Review:  r.Review,
		Creator: reviewsapi.Creator{Id: r.Creator, Name: r.CreatorName},
	}
}

// PantryItemPhotoFilePath returns the path of the GetPantryItemReviewPhotoFile
// operation.
func PantryItemPhotoFilePath(pantryItemID, reviewID, photoID string, variant reviewsapi.PhotoVariant) string {
	return pantryItemSubject.photoFilePath(pantryItemID, reviewID, photoID, variant)
}

func (s *Server) ListListPantryItems(ctx context.Context, req reviewsapi.ListListPantryItemsRequestObject) (reviewsapi.ListListPantryItemsResponseObject, error) {
	records, next, p, err := s.listItemPage(ctx, pantryItemSubject, req.ListId, req.Params.Cursor, req.Params.Limit)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusBadRequest:
			return reviewsapi.ListListPantryItems400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: p.badRequest()}, nil
		case http.StatusNotFound:
			return reviewsapi.ListListPantryItems404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	items, err := s.pantryItems(stateFrom(ctx).user, records...)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ListListPantryItems200JSONResponse{Items: items, NextCursor: next}, nil
}

// AddListPantryItem gives 204 also for a pantry item that the list has.
func (s *Server) AddListPantryItem(ctx context.Context, req reviewsapi.AddListPantryItemRequestObject) (reviewsapi.AddListPantryItemResponseObject, error) {
	p, err := s.addListItem(pantryItemSubject, req.ListRef, req.PantryItemId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusForbidden:
			return reviewsapi.AddListPantryItem403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		case http.StatusNotFound:
			return reviewsapi.AddListPantryItem404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusConflict:
			return reviewsapi.AddListPantryItem409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: p.conflict()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.AddListPantryItem204Response{}, nil
}

func (s *Server) RemoveListPantryItem(ctx context.Context, req reviewsapi.RemoveListPantryItemRequestObject) (reviewsapi.RemoveListPantryItemResponseObject, error) {
	p, err := s.removeListItem(pantryItemSubject, req.ListRef, req.PantryItemId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusForbidden:
			return reviewsapi.RemoveListPantryItem403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		case http.StatusNotFound:
			return reviewsapi.RemoveListPantryItem404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.RemoveListPantryItem204Response{}, nil
}
