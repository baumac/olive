package publicapi

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	maxPhotosPerReview = 5
	// maxParallelDecodes limits the memory of the decodes. A decode of 25
	// megapixels needs up to 100 MB.
	maxParallelDecodes   = 2
	memoryFullRetryAfter = 5 * time.Second
	photoNotFound        = "The photo does not exist."
)

// AddRestaurantReviewPhoto lets only the creator of the review add a photo. A review
// has at most 5 photos.
func (s *Server) AddRestaurantReviewPhoto(ctx context.Context, req reviewsapi.AddRestaurantReviewPhotoRequestObject) (reviewsapi.AddRestaurantReviewPhotoResponseObject, error) {
	saved, p, err := s.addPhoto(ctx, restaurantSubject, req.RestaurantId, req.ReviewId, req.ContentType, req.Body)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusBadRequest:
			return reviewsapi.AddRestaurantReviewPhoto400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: p.badRequest()}, nil
		case http.StatusForbidden:
			return reviewsapi.AddRestaurantReviewPhoto403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		case http.StatusNotFound:
			return reviewsapi.AddRestaurantReviewPhoto404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusConflict:
			return reviewsapi.AddRestaurantReviewPhoto409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: p.conflict()}, nil
		case http.StatusRequestEntityTooLarge:
			return reviewsapi.AddRestaurantReviewPhoto413ApplicationProblemPlusJSONResponse{ContentTooLargeApplicationProblemPlusJSONResponse: p.contentTooLarge()}, nil
		case http.StatusUnsupportedMediaType:
			return reviewsapi.AddRestaurantReviewPhoto415ApplicationProblemPlusJSONResponse{UnsupportedMediaTypeApplicationProblemPlusJSONResponse: p.unsupportedMediaType()}, nil
		case http.StatusUnprocessableEntity:
			return reviewsapi.AddRestaurantReviewPhoto422ApplicationProblemPlusJSONResponse{InvalidImageApplicationProblemPlusJSONResponse: p.invalidImage()}, nil
		case http.StatusTooManyRequests:
			return reviewsapi.AddRestaurantReviewPhoto429ApplicationProblemPlusJSONResponse{TooManyRequestsApplicationProblemPlusJSONResponse: p.tooManyRequests()}, nil
		}
		return nil, p.unexpected()
	}
	location := restaurantSubject.photoPath(req.RestaurantId, req.ReviewId, saved.Id)
	return reviewsapi.AddRestaurantReviewPhoto201JSONResponse{
		Body:    toPhoto(restaurantSubject, saved, req.RestaurantId),
		Headers: reviewsapi.AddRestaurantReviewPhoto201ResponseHeaders{Location: &location},
	}, nil
}

// addPhoto checks the review before it reads the body. A client that cannot
// add a photo then does not send the image for nothing. The transaction
// checks the review and the count again.
func (s *Server) addPhoto(ctx context.Context, sub subject, parentID, reviewID, contentType string, body io.Reader) (*core.Record, *problem, error) {
	user := stateFrom(ctx).user
	mediaType, _, err := mime.ParseMediaType(contentType)
	if _, ok := uploadTypes[mediaType]; err != nil || !ok {
		return nil, statusProblem(http.StatusUnsupportedMediaType, "The content type must be image/jpeg, image/png, or image/webp."), nil
	}

	check := func(app core.App) (*problem, error) {
		_, review, detail, err := s.findReview(app, sub, parentID, reviewID)
		if err != nil {
			return nil, err
		}
		if detail != "" {
			return notFoundProblem(detail), nil
		}
		if review.GetString("creator") != user.Id {
			return notCreatorProblem("Only the creator of the review can add photos."), nil
		}
		n, err := app.CountRecords(sub.Photos, dbx.HashExp{"review": reviewID})
		if err != nil {
			return nil, err
		}
		if n >= maxPhotosPerReview {
			return conflictProblem(reviewsapi.ProblemCodePhotoLimit, "A review can have at most 5 photos."), nil
		}
		return nil, nil
	}
	if p, err := check(s.app); p != nil || err != nil {
		return nil, p, err
	}

	photo, p, err := s.readPhoto(ctx, body, mediaType)
	if p != nil || err != nil {
		return nil, p, err
	}
	defer s.photoMemory.give(photo.held)

	var saved *core.Record
	err = s.app.RunInTransaction(func(tx core.App) error {
		if p, err = check(tx); p != nil || err != nil {
			return err
		}
		collection, err := tx.FindCachedCollectionByNameOrId(sub.Photos)
		if err != nil {
			return err
		}
		full, err := filesystem.NewFileFromBytes(photo.full, "full.jpg")
		if err != nil {
			return err
		}
		thumb, err := filesystem.NewFileFromBytes(photo.thumb, "thumb.jpg")
		if err != nil {
			return err
		}
		r := core.NewRecord(collection)
		r.Set("review", reviewID)
		r.Set("width", photo.width)
		r.Set("height", photo.height)
		r.Set("full", full)
		r.Set("thumb", thumb)
		saved = r
		return tx.Save(r)
	})
	if p != nil || err != nil {
		return nil, p, err
	}
	return saved, nil, nil
}

// readPhoto reads and processes the body. The decodes semaphore holds it
// only after the read. A slow client then cannot block the other uploads.
// The returned photo holds its memory until the caller gives it back.
func (s *Server) readPhoto(ctx context.Context, body io.Reader, mediaType string) (processedPhoto, *problem, error) {
	data, held, err := s.photoMemory.read(body)
	if errors.Is(err, errPhotoMemoryFull) {
		stateFrom(ctx).giveBackLimit()
		return processedPhoto{}, tooManyRequestsProblem(memoryFullRetryAfter, "The server processes too many photos. Try again in a few seconds."), nil
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return processedPhoto{}, statusProblem(http.StatusRequestEntityTooLarge, "The image must be at most 10 MB."), nil
	}
	if err != nil {
		return processedPhoto{}, badRequestProblem("The request body cannot be read."), nil
	}
	select {
	case s.decodes <- struct{}{}:
	case <-ctx.Done():
		s.photoMemory.give(held)
		return processedPhoto{}, nil, ctx.Err()
	}
	photo, detail, err := processPhoto(data, mediaType)
	<-s.decodes
	if err != nil || detail != "" {
		s.photoMemory.give(held)
	}
	if err != nil {
		return processedPhoto{}, nil, err
	}
	if detail != "" {
		return processedPhoto{}, statusProblem(http.StatusUnprocessableEntity, detail), nil
	}
	photo.held = int64(len(photo.full) + len(photo.thumb))
	s.photoMemory.resize(held, photo.held)
	return photo, nil, nil
}

// UpdateRestaurantReviewPhoto changes only the alt text. The image never changes.
func (s *Server) UpdateRestaurantReviewPhoto(ctx context.Context, req reviewsapi.UpdateRestaurantReviewPhotoRequestObject) (reviewsapi.UpdateRestaurantReviewPhotoResponseObject, error) {
	saved, p, err := s.updatePhoto(restaurantSubject, req.RestaurantId, req.ReviewId, req.PhotoId, stateFrom(ctx).user, req.Body)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusNotFound:
			return reviewsapi.UpdateRestaurantReviewPhoto404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusForbidden:
			return reviewsapi.UpdateRestaurantReviewPhoto403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.UpdateRestaurantReviewPhoto200JSONResponse(toPhoto(restaurantSubject, saved, req.RestaurantId)), nil
}

// updatePhoto changes the alt text. The problem is a 403 or a 404.
func (s *Server) updatePhoto(sub subject, parentID, reviewID, photoID string, user *core.Record, body *reviewsapi.PhotoPatch) (saved *core.Record, p *problem, err error) {
	err = s.app.RunInTransaction(func(tx core.App) error {
		review, photo, detail, err := s.findPhoto(tx, sub, parentID, reviewID, photoID)
		if err != nil {
			return err
		}
		if detail != "" {
			p = notFoundProblem(detail)
			return nil
		}
		if review.GetString("creator") != user.Id {
			p = notCreatorProblem("Only the creator of the review can change its photos.")
			return nil
		}
		saved = photo
		ch := &changes{r: photo}
		ch.patchText("alt", nullable.Nullable[string](body.Alt))
		if !ch.changed {
			return nil
		}
		return tx.Save(photo)
	})
	return saved, p, err
}

// DeleteRestaurantReviewPhoto lets only the creator of the review delete a photo.
func (s *Server) DeleteRestaurantReviewPhoto(ctx context.Context, req reviewsapi.DeleteRestaurantReviewPhotoRequestObject) (reviewsapi.DeleteRestaurantReviewPhotoResponseObject, error) {
	p, err := s.deletePhoto(restaurantSubject, req.RestaurantId, req.ReviewId, req.PhotoId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusNotFound:
			return reviewsapi.DeleteRestaurantReviewPhoto404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusForbidden:
			return reviewsapi.DeleteRestaurantReviewPhoto403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.DeleteRestaurantReviewPhoto204Response{}, nil
}

// deletePhoto deletes the record. PocketBase deletes its files after the
// transaction. The problem is a 403 or a 404.
func (s *Server) deletePhoto(sub subject, parentID, reviewID, photoID string, user *core.Record) (p *problem, err error) {
	err = s.app.RunInTransaction(func(tx core.App) error {
		review, photo, detail, err := s.findPhoto(tx, sub, parentID, reviewID, photoID)
		if err != nil {
			return err
		}
		if detail != "" {
			p = notFoundProblem(detail)
			return nil
		}
		if review.GetString("creator") != user.Id {
			p = notCreatorProblem("Only the creator of the review can delete its photos.")
			return nil
		}
		return tx.Delete(photo)
	})
	return p, err
}

// GetRestaurantReviewPhotoFile sends the stored JPEG of the variant.
func (s *Server) GetRestaurantReviewPhotoFile(ctx context.Context, req reviewsapi.GetRestaurantReviewPhotoFileRequestObject) (reviewsapi.GetRestaurantReviewPhotoFileResponseObject, error) {
	file, detail, err := s.photoFile(ctx, restaurantSubject, req.RestaurantId, req.ReviewId, req.PhotoId, req.Variant)
	if err != nil {
		return nil, err
	}
	if detail != "" {
		return reviewsapi.GetRestaurantReviewPhotoFile404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}, nil
	}
	return file, nil
}

// photoFile gets the stored JPEG. The files are protected. /api/files does
// not send them to clients. The detail is for a 404.
func (s *Server) photoFile(ctx context.Context, sub subject, parentID, reviewID, photoID string, variant reviewsapi.PhotoVariant) (photoFileResponse, string, error) {
	_, photo, detail, err := s.findPhoto(s.app, sub, parentID, reviewID, photoID)
	if err != nil || detail != "" {
		return photoFileResponse{}, detail, err
	}
	name := photo.GetString(string(variant))
	if name == "" {
		return photoFileResponse{}, "", errors.New("photo " + photo.Id + " has no " + string(variant) + " file")
	}
	return photoFileResponse{app: s.app, request: stateFrom(ctx).request, key: photo.BaseFilesPath() + "/" + name, name: name}, "", nil
}

// photoFileResponse sends a file with http.ServeContent. It handles HEAD,
// Range, and If-None-Match.
type photoFileResponse struct {
	app     core.App
	request *http.Request
	key     string
	name    string
}

func (f photoFileResponse) VisitGetRestaurantReviewPhotoFileResponse(w http.ResponseWriter) error {
	fsys, err := f.app.NewFilesystem()
	if err != nil {
		return err
	}
	defer fsys.Close()
	file, err := fsys.GetReader(f.key)
	if err != nil {
		return err
	}
	defer file.Close()

	h := w.Header()
	h.Set("Content-Type", "image/jpeg")
	h.Set("ETag", `"`+f.name+`"`)
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cross-Origin-Resource-Policy", "same-origin")
	// A zero time leaves out Last-Modified. The ETag is enough.
	http.ServeContent(&fileHeadersWriter{ResponseWriter: w}, f.request, "", time.Time{}, file)
	return nil
}

// fileHeadersWriter adds the cache headers only to a response with the file
// or a part of it. An error, such as 416, then is not cached for a year.
type fileHeadersWriter struct {
	http.ResponseWriter
	wrote bool
}

func (f *fileHeadersWriter) WriteHeader(status int) {
	if !f.wrote && (status == http.StatusOK || status == http.StatusPartialContent || status == http.StatusNotModified) {
		f.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		if status != http.StatusNotModified {
			f.Header().Set("Content-Disposition", `inline; filename="photo.jpg"`)
		}
	}
	f.wrote = true
	f.ResponseWriter.WriteHeader(status)
}

func (f *fileHeadersWriter) Write(p []byte) (int, error) {
	if !f.wrote {
		f.WriteHeader(http.StatusOK)
	}
	return f.ResponseWriter.Write(p)
}

// findPhoto returns a detail for the 404 response. A photo of a different
// review does not exist.
func (s *Server) findPhoto(app core.App, sub subject, parentID, reviewID, photoID string) (review, photo *core.Record, detail string, err error) {
	_, review, detail, err = s.findReview(app, sub, parentID, reviewID)
	if err != nil || detail != "" {
		return nil, nil, detail, err
	}
	photo, err = app.FindRecordById(sub.Photos, photoID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && photo.GetString("review") != reviewID) {
		return nil, nil, photoNotFound, nil
	}
	if err != nil {
		return nil, nil, "", err
	}
	return review, photo, "", nil
}

// reviewPhotos gets the photos of all reviews in one query, like creators.
func (s *Server) reviewPhotos(sub subject, reviews ...*core.Record) (map[string][]reviewsapi.Photo, error) {
	out := map[string][]reviewsapi.Photo{}
	if len(reviews) == 0 {
		return out, nil
	}
	ids := make([]any, len(reviews))
	parents := map[string]string{}
	for i, r := range reviews {
		ids[i] = r.Id
		parents[r.Id] = r.GetString(sub.Parent)
	}
	var photos []*core.Record
	err := s.app.RecordQuery(sub.Photos).AndWhere(dbx.In("review", ids...)).OrderBy("created ASC", "id ASC").All(&photos)
	if err != nil {
		return nil, err
	}
	for _, p := range photos {
		review := p.GetString("review")
		out[review] = append(out[review], toPhoto(sub, p, parents[review]))
	}
	return out, nil
}

func toPhoto(sub subject, r *core.Record, parentID string) reviewsapi.Photo {
	return photoOf(sub, photoRow{
		ID: r.Id, Review: r.GetString("review"), Alt: r.GetString("alt"),
		Width: r.GetInt("width"), Height: r.GetInt("height"), Created: r.GetDateTime("created"),
	}, parentID)
}

// photoRow has the stored fields of a photo that a response shows.
type photoRow struct {
	ID      string         `db:"id"`
	Review  string         `db:"review"`
	Alt     string         `db:"alt"`
	Width   int            `db:"width"`
	Height  int            `db:"height"`
	Created types.DateTime `db:"created"`
}

func photoOf(sub subject, p photoRow, parentID string) reviewsapi.Photo {
	return reviewsapi.Photo{
		Id:     p.ID,
		Alt:    nullableText(p.Alt),
		Width:  p.Width,
		Height: p.Height,
		Urls: reviewsapi.PhotoUrls{
			Full:  sub.photoFilePath(parentID, p.Review, p.ID, reviewsapi.Full),
			Thumb: sub.photoFilePath(parentID, p.Review, p.ID, reviewsapi.Thumb),
		},
		Created: apiTimestamp(p.Created),
	}
}

func nullableText(v string) nullable.Nullable[string] {
	if v == "" {
		return nullable.NewNullNullable[string]()
	}
	return nullable.NewNullableWithValue(v)
}

// PhotoFilePath returns the path of the GetRestaurantReviewPhotoFile operation.
func PhotoFilePath(restaurantID, reviewID, photoID string, variant reviewsapi.PhotoVariant) string {
	return restaurantSubject.photoFilePath(restaurantID, reviewID, photoID, variant)
}

func (sub subject) photoFilePath(parentID, reviewID, photoID string, variant reviewsapi.PhotoVariant) string {
	return sub.photoPath(parentID, reviewID, photoID) + "/" + string(variant)
}

func (sub subject) photoPath(parentID, reviewID, photoID string) string {
	return sub.reviewPath(parentID, reviewID) + "/photos/" + photoID
}
