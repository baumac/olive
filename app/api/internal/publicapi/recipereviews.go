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

// ListRecipeReviews returns 404 when the recipe does not exist. It does not
// return an empty list.
func (s *Server) ListRecipeReviews(_ context.Context, req reviewsapi.ListRecipeReviewsRequestObject) (reviewsapi.ListRecipeReviewsResponseObject, error) {
	p := req.Params
	all := reviewsapi.ListAllRecipeReviewsParams{
		Limit: p.Limit, Cursor: p.Cursor, Sort: p.Sort, Search: p.Search, RecipeEq: &req.RecipeId,
		Taste: p.Taste, Ease: p.Ease, Instructions: p.Instructions, Value: p.Value,
		Overall: p.Overall, Created: p.Created, Updated: p.Updated, CreatorEq: p.CreatorEq, CreatorIn: p.CreatorIn,
	}
	fingerprintOf := func(sort string) []any {
		q := req.Params
		q.Limit, q.Cursor, q.Sort = nil, nil, &sort
		return []any{"recipe-reviews", req.RecipeId, q}
	}
	page, err := s.recipeReviewPage(all, fingerprintOf, func() (*core.Record, error) {
		r, err := findParent(s.app, recipeSubject, req.RecipeId)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return r, err
	})
	switch {
	case err != nil:
		return nil, err
	case page.bad != "":
		return reviewsapi.ListRecipeReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(page.bad)}, nil
	case page.notFound:
		return reviewsapi.ListRecipeReviews404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(recipeNotFound)}, nil
	}
	return reviewsapi.ListRecipeReviews200JSONResponse(page.list), nil
}

// ListAllRecipeReviews gives an empty list for an unknown recipe in
// recipe[eq], not 404.
func (s *Server) ListAllRecipeReviews(_ context.Context, req reviewsapi.ListAllRecipeReviewsRequestObject) (reviewsapi.ListAllRecipeReviewsResponseObject, error) {
	page, err := s.recipeReviewPage(req.Params, nil, nil)
	if err != nil {
		return nil, err
	}
	if page.bad != "" {
		return reviewsapi.ListAllRecipeReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(page.bad)}, nil
	}
	return reviewsapi.ListAllRecipeReviews200JSONResponse(page.list), nil
}

// ListReviewerRecipeReviews returns 404 when the user does not exist. It
// does not return an empty list.
func (s *Server) ListReviewerRecipeReviews(ctx context.Context, req reviewsapi.ListReviewerRecipeReviewsRequestObject) (reviewsapi.ListReviewerRecipeReviewsResponseObject, error) {
	p := req.Params
	all := reviewsapi.ListAllRecipeReviewsParams{
		Limit: p.Limit, Cursor: p.Cursor, Sort: p.Sort, Search: p.Search, CreatorEq: &req.ReviewerId,
		RecipeEq: p.RecipeEq, RecipeIn: p.RecipeIn,
		Taste: p.Taste, Ease: p.Ease, Instructions: p.Instructions, Value: p.Value,
		Overall: p.Overall, Created: p.Created, Updated: p.Updated,
	}
	fingerprintOf := func(sort string) []any {
		q := req.Params
		q.Limit, q.Cursor, q.Sort = nil, nil, &sort
		return []any{"reviewer-recipe-reviews", req.ReviewerId, q}
	}
	page, err := s.recipeReviewPage(all, fingerprintOf, nil)
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
		return reviewsapi.ListReviewerRecipeReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(page.bad)}, nil
	case page.notFound:
		return reviewsapi.ListReviewerRecipeReviews404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(reviewerNotFound)}, nil
	}
	return reviewsapi.ListReviewerRecipeReviews200JSONResponse(page.list), nil
}

type recipeReviewPageResult struct {
	list     reviewsapi.RecipeReviewList
	bad      string
	notFound bool
}

// recipeReviewPage gets a page of recipe reviews. A nil fingerprintOf uses
// the parameters of listAllRecipeReviews.
func (s *Server) recipeReviewPage(
	p reviewsapi.ListAllRecipeReviewsParams,
	fingerprintOf func(sort string) []any,
	recipe func() (*core.Record, error),
) (recipeReviewPageResult, error) {
	if fingerprintOf == nil {
		fingerprintOf = func(sort string) []any {
			q := p
			q.Limit, q.Cursor, q.Sort = nil, nil, &sort
			return []any{"all-recipe-reviews", q}
		}
	}
	page, err := s.reviewsOf(recipeSubject, reviewQuery{
		limit: p.Limit, cursor: p.Cursor, sort: p.Sort, search: p.Search,
		parentEq: p.RecipeEq, parentIn: p.RecipeIn,
		scores: map[string]*reviewsapi.ScoreRange{
			"taste": p.Taste, "ease": p.Ease, "instructions": p.Instructions, "value": p.Value,
		},
		overall: p.Overall, created: p.Created, updated: p.Updated, creatorEq: p.CreatorEq, creatorIn: p.CreatorIn,
	}, fingerprintOf, recipe)
	if err != nil || page.bad != "" || page.notFound {
		return recipeReviewPageResult{bad: page.bad, notFound: page.notFound}, err
	}
	items := make([]reviewsapi.RecipeReview, len(page.records))
	for i, r := range page.records {
		if items[i], err = toRecipeReview(r, page.users[r.GetString("creator")], page.names[r.GetString("recipe")], page.photos[r.Id]); err != nil {
			return recipeReviewPageResult{}, err
		}
	}
	return recipeReviewPageResult{list: reviewsapi.RecipeReviewList{Items: items, NextCursor: page.next}}, nil
}

// CreateRecipeReview makes a review of one cook. A user can write many
// reviews of one recipe. See decision 00005.
func (s *Server) CreateRecipeReview(ctx context.Context, req reviewsapi.CreateRecipeReviewRequestObject) (reviewsapi.CreateRecipeReviewResponseObject, error) {
	user := stateFrom(ctx).user
	b := req.Body
	in := reviewInput{
		scores: map[string]int{"taste": b.Taste, "ease": b.Ease, "instructions": b.Instructions, "value": b.Value},
		date:   &b.Date, title: &b.Title, feedback: optional(b.Feedback),
	}
	saved, recipe, p, err := s.createReview(recipeSubject, req.RecipeId, user, in)
	if err != nil {
		return nil, err
	}
	if p != nil {
		if p.status() == http.StatusNotFound {
			return reviewsapi.CreateRecipeReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	out, err := toRecipeReview(saved, user, recipe.GetString("name"), nil)
	if err != nil {
		return nil, err
	}
	location := recipeSubject.reviewPath(req.RecipeId, saved.Id)
	return reviewsapi.CreateRecipeReview201JSONResponse{
		Body:    out,
		Headers: reviewsapi.CreateRecipeReview201ResponseHeaders{Location: &location},
	}, nil
}

// GetRecipeReview returns 404 for a review of a different recipe.
func (s *Server) GetRecipeReview(_ context.Context, req reviewsapi.GetRecipeReviewRequestObject) (reviewsapi.GetRecipeReviewResponseObject, error) {
	recipe, r, detail, err := s.findReview(s.app, recipeSubject, req.RecipeId, req.ReviewId)
	if err != nil {
		return nil, err
	}
	if detail != "" {
		return reviewsapi.GetRecipeReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}, nil
	}
	users, err := s.creators(r)
	if err != nil {
		return nil, err
	}
	photos, err := s.reviewPhotos(recipeSubject, r)
	if err != nil {
		return nil, err
	}
	out, err := toRecipeReview(r, users[r.GetString("creator")], recipe.GetString("name"), photos[r.Id])
	if err != nil {
		return nil, err
	}
	return reviewsapi.GetRecipeReview200JSONResponse(out), nil
}

// UpdateRecipeReview lets only the creator change a review.
func (s *Server) UpdateRecipeReview(ctx context.Context, req reviewsapi.UpdateRecipeReviewRequestObject) (reviewsapi.UpdateRecipeReviewResponseObject, error) {
	user := stateFrom(ctx).user
	b := req.Body
	in := reviewInput{scores: map[string]int{}, date: b.Date, title: b.Title, feedback: b.Feedback}
	setScore(in.scores, "taste", b.Taste)
	setScore(in.scores, "ease", b.Ease)
	setScore(in.scores, "instructions", b.Instructions)
	setScore(in.scores, "value", b.Value)
	saved, recipe, p, err := s.updateReview(recipeSubject, req.RecipeId, req.ReviewId, user, in)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusNotFound:
			return reviewsapi.UpdateRecipeReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusForbidden:
			return reviewsapi.UpdateRecipeReview403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		}
		return nil, p.unexpected()
	}
	photos, err := s.reviewPhotos(recipeSubject, saved)
	if err != nil {
		return nil, err
	}
	out, err := toRecipeReview(saved, user, recipe.GetString("name"), photos[saved.Id])
	if err != nil {
		return nil, err
	}
	return reviewsapi.UpdateRecipeReview200JSONResponse(out), nil
}

// DeleteRecipeReview lets only the creator delete a review.
func (s *Server) DeleteRecipeReview(ctx context.Context, req reviewsapi.DeleteRecipeReviewRequestObject) (reviewsapi.DeleteRecipeReviewResponseObject, error) {
	p, err := s.deleteReview(recipeSubject, req.RecipeId, req.ReviewId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusNotFound:
			return reviewsapi.DeleteRecipeReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusForbidden:
			return reviewsapi.DeleteRecipeReview403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.DeleteRecipeReview204Response{}, nil
}

// toRecipeReview gives an empty photos array for nil photos. The spec
// requires the array.
func toRecipeReview(r *core.Record, creator *core.Record, recipeName string, photos []reviewsapi.Photo) (reviewsapi.RecipeReview, error) {
	if creator == nil {
		return reviewsapi.RecipeReview{}, fmt.Errorf("recipe review %s has no creator", r.Id)
	}
	if recipeName == "" {
		return reviewsapi.RecipeReview{}, fmt.Errorf("recipe review %s has no recipe", r.Id)
	}
	if photos == nil {
		photos = []reviewsapi.Photo{}
	}
	taste, ease, instructions, value := r.GetInt("taste"), r.GetInt("ease"), r.GetInt("instructions"), r.GetInt("value")
	// TestRecipeOverallColumnEqualsResponse keeps the overall column equal
	// to this.
	return reviewsapi.RecipeReview{
		Id: r.Id, Recipe: r.GetString("recipe"), RecipeName: recipeName,
		Date: r.GetString("date"), Overall: float64(taste+ease+instructions+value) / 4,
		Taste: taste, Ease: ease, Instructions: instructions, Value: value,
		Title: r.GetString("title"), Feedback: optionalText(r, "feedback"),
		Creator: toCreator(creator), Photos: photos, Created: apiTimestamp(r.GetDateTime("created")), Updated: apiTimestamp(r.GetDateTime("updated")),
	}, nil
}

// AddRecipeReviewPhoto lets only the creator of the review add a photo. A
// review has at most 5 photos.
func (s *Server) AddRecipeReviewPhoto(ctx context.Context, req reviewsapi.AddRecipeReviewPhotoRequestObject) (reviewsapi.AddRecipeReviewPhotoResponseObject, error) {
	saved, p, err := s.addPhoto(ctx, recipeSubject, req.RecipeId, req.ReviewId, req.ContentType, req.Body)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusBadRequest:
			return reviewsapi.AddRecipeReviewPhoto400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: p.badRequest()}, nil
		case http.StatusForbidden:
			return reviewsapi.AddRecipeReviewPhoto403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		case http.StatusNotFound:
			return reviewsapi.AddRecipeReviewPhoto404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusConflict:
			return reviewsapi.AddRecipeReviewPhoto409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: p.conflict()}, nil
		case http.StatusRequestEntityTooLarge:
			return reviewsapi.AddRecipeReviewPhoto413ApplicationProblemPlusJSONResponse{ContentTooLargeApplicationProblemPlusJSONResponse: p.contentTooLarge()}, nil
		case http.StatusUnsupportedMediaType:
			return reviewsapi.AddRecipeReviewPhoto415ApplicationProblemPlusJSONResponse{UnsupportedMediaTypeApplicationProblemPlusJSONResponse: p.unsupportedMediaType()}, nil
		case http.StatusUnprocessableEntity:
			return reviewsapi.AddRecipeReviewPhoto422ApplicationProblemPlusJSONResponse{InvalidImageApplicationProblemPlusJSONResponse: p.invalidImage()}, nil
		case http.StatusTooManyRequests:
			return reviewsapi.AddRecipeReviewPhoto429ApplicationProblemPlusJSONResponse{TooManyRequestsApplicationProblemPlusJSONResponse: p.tooManyRequests()}, nil
		}
		return nil, p.unexpected()
	}
	location := recipeSubject.photoPath(req.RecipeId, req.ReviewId, saved.Id)
	return reviewsapi.AddRecipeReviewPhoto201JSONResponse{
		Body:    toPhoto(recipeSubject, saved, req.RecipeId),
		Headers: reviewsapi.AddRecipeReviewPhoto201ResponseHeaders{Location: &location},
	}, nil
}

// UpdateRecipeReviewPhoto changes only the alt text. The image never
// changes.
func (s *Server) UpdateRecipeReviewPhoto(ctx context.Context, req reviewsapi.UpdateRecipeReviewPhotoRequestObject) (reviewsapi.UpdateRecipeReviewPhotoResponseObject, error) {
	saved, p, err := s.updatePhoto(recipeSubject, req.RecipeId, req.ReviewId, req.PhotoId, stateFrom(ctx).user, req.Body)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusNotFound:
			return reviewsapi.UpdateRecipeReviewPhoto404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusForbidden:
			return reviewsapi.UpdateRecipeReviewPhoto403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.UpdateRecipeReviewPhoto200JSONResponse(toPhoto(recipeSubject, saved, req.RecipeId)), nil
}

// DeleteRecipeReviewPhoto lets only the creator of the review delete a
// photo.
func (s *Server) DeleteRecipeReviewPhoto(ctx context.Context, req reviewsapi.DeleteRecipeReviewPhotoRequestObject) (reviewsapi.DeleteRecipeReviewPhotoResponseObject, error) {
	p, err := s.deletePhoto(recipeSubject, req.RecipeId, req.ReviewId, req.PhotoId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusNotFound:
			return reviewsapi.DeleteRecipeReviewPhoto404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusForbidden:
			return reviewsapi.DeleteRecipeReviewPhoto403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.DeleteRecipeReviewPhoto204Response{}, nil
}

// GetRecipeReviewPhotoFile sends the stored JPEG of the variant.
func (s *Server) GetRecipeReviewPhotoFile(ctx context.Context, req reviewsapi.GetRecipeReviewPhotoFileRequestObject) (reviewsapi.GetRecipeReviewPhotoFileResponseObject, error) {
	file, detail, err := s.photoFile(ctx, recipeSubject, req.RecipeId, req.ReviewId, req.PhotoId, req.Variant)
	if err != nil {
		return nil, err
	}
	if detail != "" {
		return reviewsapi.GetRecipeReviewPhotoFile404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}, nil
	}
	return file, nil
}

func (f photoFileResponse) VisitGetRecipeReviewPhotoFileResponse(w http.ResponseWriter) error {
	return f.VisitGetRestaurantReviewPhotoFileResponse(w)
}

func (s *Server) ListRecipePhotos(_ context.Context, req reviewsapi.ListRecipePhotosRequestObject) (reviewsapi.ListRecipePhotosResponseObject, error) {
	rows, next, p, err := s.galleryPage(recipeSubject, req.RecipeId, req.Params.Cursor, req.Params.Limit)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusBadRequest:
			return reviewsapi.ListRecipePhotos400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: p.badRequest()}, nil
		case http.StatusNotFound:
			return reviewsapi.ListRecipePhotos404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	items := make([]reviewsapi.RecipePhoto, len(rows))
	for i, r := range rows {
		items[i] = r.recipePhoto()
	}
	return reviewsapi.ListRecipePhotos200JSONResponse{Items: items, NextCursor: next}, nil
}

func (r galleryRow) recipePhoto() reviewsapi.RecipePhoto {
	p := photoOf(recipeSubject, r.photoRow, r.Parent)
	return reviewsapi.RecipePhoto{
		Id: p.Id, Alt: p.Alt, Width: p.Width, Height: p.Height, Urls: p.Urls, Created: p.Created,
		Review:  r.Review,
		Creator: reviewsapi.Creator{Id: r.Creator, Name: r.CreatorName},
	}
}

// RecipePhotoFilePath returns the path of the GetRecipeReviewPhotoFile
// operation.
func RecipePhotoFilePath(recipeID, reviewID, photoID string, variant reviewsapi.PhotoVariant) string {
	return recipeSubject.photoFilePath(recipeID, reviewID, photoID, variant)
}

func (s *Server) ListListRecipes(ctx context.Context, req reviewsapi.ListListRecipesRequestObject) (reviewsapi.ListListRecipesResponseObject, error) {
	records, next, p, err := s.listItemPage(ctx, recipeSubject, req.ListId, req.Params.Cursor, req.Params.Limit)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusBadRequest:
			return reviewsapi.ListListRecipes400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: p.badRequest()}, nil
		case http.StatusNotFound:
			return reviewsapi.ListListRecipes404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	items, err := s.recipeSummaries(stateFrom(ctx).user, records...)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ListListRecipes200JSONResponse{Items: items, NextCursor: next}, nil
}

// AddListRecipe gives 204 also for a recipe that the list has.
func (s *Server) AddListRecipe(ctx context.Context, req reviewsapi.AddListRecipeRequestObject) (reviewsapi.AddListRecipeResponseObject, error) {
	p, err := s.addListItem(recipeSubject, req.ListRef, req.RecipeId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusForbidden:
			return reviewsapi.AddListRecipe403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		case http.StatusNotFound:
			return reviewsapi.AddListRecipe404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusConflict:
			return reviewsapi.AddListRecipe409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: p.conflict()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.AddListRecipe204Response{}, nil
}

func (s *Server) RemoveListRecipe(ctx context.Context, req reviewsapi.RemoveListRecipeRequestObject) (reviewsapi.RemoveListRecipeResponseObject, error) {
	p, err := s.removeListItem(recipeSubject, req.ListRef, req.RecipeId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusForbidden:
			return reviewsapi.RemoveListRecipe403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		case http.StatusNotFound:
			return reviewsapi.RemoveListRecipe404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.RemoveListRecipe204Response{}, nil
}
