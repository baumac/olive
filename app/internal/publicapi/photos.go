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
)

const (
	maxPhotosPerReview = 5
	// maxParallelDecodes limits the memory of the decodes. A decode of 25
	// megapixels needs up to 100 MB.
	maxParallelDecodes = 2
	// memoryFullRetryAfter is in seconds.
	memoryFullRetryAfter = 5
	photoNotFound        = "The photo does not exist."
)

// AddReviewPhoto checks the review before it reads the body. A client that
// cannot add a photo then does not send the image for nothing. The
// transaction checks the review and the count again.
func (s *Server) AddReviewPhoto(ctx context.Context, req reviewsapi.AddReviewPhotoRequestObject) (reviewsapi.AddReviewPhotoResponseObject, error) {
	user := stateFrom(ctx).user
	mediaType, _, err := mime.ParseMediaType(req.ContentType)
	if _, ok := uploadTypes[mediaType]; err != nil || !ok {
		return reviewsapi.AddReviewPhoto415ApplicationProblemPlusJSONResponse{UnsupportedMediaTypeApplicationProblemPlusJSONResponse: reviewsapi.UnsupportedMediaTypeApplicationProblemPlusJSONResponse(
			newProblem(http.StatusUnsupportedMediaType, "The content type must be image/jpeg, image/png, or image/webp."))}, nil
	}

	check := func(app core.App) (reviewsapi.AddReviewPhotoResponseObject, error) {
		_, review, detail, err := s.findReview(app, req.RestaurantId, req.ReviewId)
		if err != nil {
			return nil, err
		}
		if detail != "" {
			return reviewsapi.AddReviewPhoto404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}, nil
		}
		if review.GetString("creator") != user.Id {
			return reviewsapi.AddReviewPhoto403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden("Only the creator of the review can add photos.")}, nil
		}
		n, err := app.CountRecords("review_photos", dbx.HashExp{"review": req.ReviewId})
		if err != nil {
			return nil, err
		}
		if n >= maxPhotosPerReview {
			return reviewsapi.AddReviewPhoto409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict("A review can have at most 5 photos.")}, nil
		}
		return nil, nil
	}
	if response, err := check(s.app); response != nil || err != nil {
		return response, err
	}

	photo, response, err := s.readPhoto(ctx, req.Body, mediaType)
	if response != nil || err != nil {
		return response, err
	}
	defer s.photoMemory.give(photo.held)

	var saved *core.Record
	err = s.app.RunInTransaction(func(tx core.App) error {
		if response, err = check(tx); response != nil || err != nil {
			return err
		}
		collection, err := tx.FindCachedCollectionByNameOrId("review_photos")
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
		r.Set("review", req.ReviewId)
		r.Set("width", photo.width)
		r.Set("height", photo.height)
		r.Set("full", full)
		r.Set("thumb", thumb)
		saved = r
		return tx.Save(r)
	})
	if response != nil || err != nil {
		return response, err
	}
	location := photoPath(req.RestaurantId, req.ReviewId, saved.Id)
	return reviewsapi.AddReviewPhoto201JSONResponse{
		Body:    toPhoto(saved, req.RestaurantId),
		Headers: reviewsapi.AddReviewPhoto201ResponseHeaders{Location: &location},
	}, nil
}

// readPhoto reads and processes the body. The decodes semaphore holds it
// only after the read. A slow client then cannot block the other uploads.
// The returned photo holds its memory until the caller gives it back.
func (s *Server) readPhoto(ctx context.Context, body io.Reader, mediaType string) (processedPhoto, reviewsapi.AddReviewPhotoResponseObject, error) {
	data, held, err := s.photoMemory.read(body)
	if errors.Is(err, errPhotoMemoryFull) {
		stateFrom(ctx).giveBackLimit()
		return processedPhoto{}, reviewsapi.AddReviewPhoto429ApplicationProblemPlusJSONResponse{TooManyRequestsApplicationProblemPlusJSONResponse: reviewsapi.TooManyRequestsApplicationProblemPlusJSONResponse{
			Body:    newProblem(http.StatusTooManyRequests, "The server processes too many photos. Try again in a few seconds."),
			Headers: reviewsapi.TooManyRequestsResponseHeaders{RetryAfter: memoryFullRetryAfter},
		}}, nil
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return processedPhoto{}, reviewsapi.AddReviewPhoto413ApplicationProblemPlusJSONResponse{ContentTooLargeApplicationProblemPlusJSONResponse: reviewsapi.ContentTooLargeApplicationProblemPlusJSONResponse(
			newProblem(http.StatusRequestEntityTooLarge, "The image must be at most 10 MB."))}, nil
	}
	if err != nil {
		return processedPhoto{}, reviewsapi.AddReviewPhoto400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest("The request body cannot be read.")}, nil
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
		return processedPhoto{}, reviewsapi.AddReviewPhoto422ApplicationProblemPlusJSONResponse{InvalidImageApplicationProblemPlusJSONResponse: reviewsapi.InvalidImageApplicationProblemPlusJSONResponse(
			newProblem(http.StatusUnprocessableEntity, detail))}, nil
	}
	photo.held = int64(len(photo.full) + len(photo.thumb))
	s.photoMemory.resize(held, photo.held)
	return photo, nil, nil
}

// UpdateReviewPhoto changes only the alt text. The image never changes.
func (s *Server) UpdateReviewPhoto(ctx context.Context, req reviewsapi.UpdateReviewPhotoRequestObject) (reviewsapi.UpdateReviewPhotoResponseObject, error) {
	user := stateFrom(ctx).user
	var response reviewsapi.UpdateReviewPhotoResponseObject
	var saved *core.Record
	err := s.app.RunInTransaction(func(tx core.App) error {
		review, photo, detail, err := s.findPhoto(tx, req.RestaurantId, req.ReviewId, req.PhotoId)
		if err != nil {
			return err
		}
		if detail != "" {
			response = reviewsapi.UpdateReviewPhoto404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}
			return nil
		}
		if review.GetString("creator") != user.Id {
			response = reviewsapi.UpdateReviewPhoto403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden("Only the creator of the review can change its photos.")}
			return nil
		}
		saved = photo
		ch := &changes{r: photo}
		ch.patchText("alt", nullable.Nullable[string](req.Body.Alt))
		if !ch.changed {
			return nil
		}
		return tx.Save(photo)
	})
	if response != nil || err != nil {
		return response, err
	}
	return reviewsapi.UpdateReviewPhoto200JSONResponse(toPhoto(saved, req.RestaurantId)), nil
}

// DeleteReviewPhoto deletes the record. PocketBase deletes its files after
// the transaction.
func (s *Server) DeleteReviewPhoto(ctx context.Context, req reviewsapi.DeleteReviewPhotoRequestObject) (reviewsapi.DeleteReviewPhotoResponseObject, error) {
	user := stateFrom(ctx).user
	var response reviewsapi.DeleteReviewPhotoResponseObject = reviewsapi.DeleteReviewPhoto204Response{}
	err := s.app.RunInTransaction(func(tx core.App) error {
		review, photo, detail, err := s.findPhoto(tx, req.RestaurantId, req.ReviewId, req.PhotoId)
		if err != nil {
			return err
		}
		if detail != "" {
			response = reviewsapi.DeleteReviewPhoto404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}
			return nil
		}
		if review.GetString("creator") != user.Id {
			response = reviewsapi.DeleteReviewPhoto403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden("Only the creator of the review can delete its photos.")}
			return nil
		}
		return tx.Delete(photo)
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetReviewPhotoFile sends the stored JPEG. The files are protected.
// /api/files does not send them to clients.
func (s *Server) GetReviewPhotoFile(ctx context.Context, req reviewsapi.GetReviewPhotoFileRequestObject) (reviewsapi.GetReviewPhotoFileResponseObject, error) {
	_, photo, detail, err := s.findPhoto(s.app, req.RestaurantId, req.ReviewId, req.PhotoId)
	if err != nil {
		return nil, err
	}
	if detail != "" {
		return reviewsapi.GetReviewPhotoFile404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}, nil
	}
	name := photo.GetString(string(req.Variant))
	if name == "" {
		return nil, errors.New("photo " + photo.Id + " has no " + string(req.Variant) + " file")
	}
	return photoFileResponse{app: s.app, request: stateFrom(ctx).request, key: photo.BaseFilesPath() + "/" + name, name: name}, nil
}

// photoFileResponse sends a file with http.ServeContent. It handles HEAD,
// Range, and If-None-Match.
type photoFileResponse struct {
	app     core.App
	request *http.Request
	key     string
	name    string
}

func (f photoFileResponse) VisitGetReviewPhotoFileResponse(w http.ResponseWriter) error {
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
func (s *Server) findPhoto(app core.App, restaurantID, reviewID, photoID string) (review, photo *core.Record, detail string, err error) {
	_, review, detail, err = s.findReview(app, restaurantID, reviewID)
	if err != nil || detail != "" {
		return nil, nil, detail, err
	}
	photo, err = app.FindRecordById("review_photos", photoID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && photo.GetString("review") != reviewID) {
		return nil, nil, photoNotFound, nil
	}
	if err != nil {
		return nil, nil, "", err
	}
	return review, photo, "", nil
}

// reviewPhotos gets the photos of all reviews in one query, like creators.
func (s *Server) reviewPhotos(reviews ...*core.Record) (map[string][]reviewsapi.Photo, error) {
	out := map[string][]reviewsapi.Photo{}
	if len(reviews) == 0 {
		return out, nil
	}
	ids := make([]any, len(reviews))
	restaurants := map[string]string{}
	for i, r := range reviews {
		ids[i] = r.Id
		restaurants[r.Id] = r.GetString("restaurant")
	}
	var photos []*core.Record
	err := s.app.RecordQuery("review_photos").AndWhere(dbx.In("review", ids...)).OrderBy("created ASC", "id ASC").All(&photos)
	if err != nil {
		return nil, err
	}
	for _, p := range photos {
		review := p.GetString("review")
		out[review] = append(out[review], toPhoto(p, restaurants[review]))
	}
	return out, nil
}

func toPhoto(r *core.Record, restaurantID string) reviewsapi.Photo {
	reviewID := r.GetString("review")
	return reviewsapi.Photo{
		Id:     r.Id,
		Alt:    nullableText(r.GetString("alt")),
		Width:  r.GetInt("width"),
		Height: r.GetInt("height"),
		Urls: reviewsapi.PhotoUrls{
			Full:  PhotoFilePath(restaurantID, reviewID, r.Id, reviewsapi.Full),
			Thumb: PhotoFilePath(restaurantID, reviewID, r.Id, reviewsapi.Thumb),
		},
		Created: r.GetDateTime("created").String(),
	}
}

func nullableText(v string) nullable.Nullable[string] {
	if v == "" {
		return nullable.NewNullNullable[string]()
	}
	return nullable.NewNullableWithValue(v)
}

// PhotoFilePath returns the path of the GetReviewPhotoFile operation.
func PhotoFilePath(restaurantID, reviewID, photoID string, variant reviewsapi.PhotoVariant) string {
	return photoPath(restaurantID, reviewID, photoID) + "/" + string(variant)
}

func photoPath(restaurantID, reviewID, photoID string) string {
	return basePath + "/restaurants/" + restaurantID + "/reviews/" + reviewID + "/photos/" + photoID
}
