package publicapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/baumac/olive/app/api/internal/subjects"
	reviewsapi "github.com/baumac/olive/specs/api/sdk/server/go"
)

// subject has the tables of one kind of thing that users review. The review,
// photo, rating, search, and list code takes a subject. See decision 00004.
type subject struct {
	subjects.Tables
	noun string
	// plural is the plural noun of the problem details.
	plural        string
	notFound      string
	listLimitCode reviewsapi.ProblemCode
	// The search scores depend only on the row. bm25 also depends on all
	// other rows. A write then moves the bm25 ranks under an open cursor.
	search       []scoreColumn
	reviewSearch []scoreColumn

	// newSubject sets these fields. The callers must not change them.
	reviewSorts    map[string]sortField
	galleryKeys    []sortKey
	galleryColumns []string
	listItemKeys   []sortKey
}

var restaurantSubject = newSubject(subject{
	Tables:        subjects.Restaurants,
	noun:          "restaurant",
	plural:        "restaurants",
	notFound:      restaurantNotFound,
	listLimitCode: reviewsapi.ProblemCodeListRestaurantLimit,
	search:        []scoreColumn{{"name", 10}, {"tags", 4}, {"address", 2}, {"description", 1}},
	reviewSearch:  []scoreColumn{{"title", 10}, {"restaurant_name", 2}, {"feedback", 1}},
})

func newSubject(sub subject) subject {
	sub.reviewSorts = makeReviewSorts(sub)
	sub.galleryKeys = makeGalleryKeys(sub)
	sub.galleryColumns = makeGalleryColumns(sub)
	sub.listItemKeys = makeListItemKeys(sub)
	return sub
}

// problem is a 4xx result of the code that all subjects share. Each
// operation changes it to its own response type.
type problem struct {
	body reviewsapi.Problem
	// retryAfter is the Retry-After header of a 429 in seconds.
	retryAfter int
}

func (p *problem) status() int {
	return p.body.Status
}

// unexpected is the error of a problem status that the operation has no
// response for.
func (p *problem) unexpected() error {
	return fmt.Errorf("the operation has no %d response for the problem %q", p.status(), p.body.Title)
}

func statusProblem(status int, detail string) *problem {
	return &problem{body: newProblem(status, detail)}
}

func codeProblem(status int, code reviewsapi.ProblemCode, detail string) *problem {
	p := statusProblem(status, detail)
	p.body.Code = &code
	return p
}

func badRequestProblem(detail string) *problem {
	return statusProblem(http.StatusBadRequest, detail)
}

func notFoundProblem(detail string) *problem {
	return statusProblem(http.StatusNotFound, detail)
}

func notCreatorProblem(detail string) *problem {
	return codeProblem(http.StatusForbidden, reviewsapi.ProblemCodeNotCreator, detail)
}

func conflictProblem(code reviewsapi.ProblemCode, detail string) *problem {
	return codeProblem(http.StatusConflict, code, detail)
}

func tooManyRequestsProblem(wait time.Duration, detail string) *problem {
	r := tooManyRequests(wait, detail)
	return &problem{body: r.Body, retryAfter: r.Headers.RetryAfter}
}

func (p *problem) badRequest() reviewsapi.BadRequestApplicationProblemPlusJSONResponse {
	return reviewsapi.BadRequestApplicationProblemPlusJSONResponse(p.body)
}

func (p *problem) forbidden() reviewsapi.ForbiddenApplicationProblemPlusJSONResponse {
	return reviewsapi.ForbiddenApplicationProblemPlusJSONResponse(p.body)
}

func (p *problem) notFound() reviewsapi.NotFoundApplicationProblemPlusJSONResponse {
	return reviewsapi.NotFoundApplicationProblemPlusJSONResponse(p.body)
}

func (p *problem) conflict() reviewsapi.ConflictApplicationProblemPlusJSONResponse {
	return reviewsapi.ConflictApplicationProblemPlusJSONResponse(p.body)
}

func (p *problem) contentTooLarge() reviewsapi.ContentTooLargeApplicationProblemPlusJSONResponse {
	return reviewsapi.ContentTooLargeApplicationProblemPlusJSONResponse(p.body)
}

func (p *problem) unsupportedMediaType() reviewsapi.UnsupportedMediaTypeApplicationProblemPlusJSONResponse {
	return reviewsapi.UnsupportedMediaTypeApplicationProblemPlusJSONResponse(p.body)
}

func (p *problem) invalidImage() reviewsapi.InvalidImageApplicationProblemPlusJSONResponse {
	return reviewsapi.InvalidImageApplicationProblemPlusJSONResponse(p.body)
}

func (p *problem) tooManyRequests() reviewsapi.TooManyRequestsApplicationProblemPlusJSONResponse {
	return reviewsapi.TooManyRequestsApplicationProblemPlusJSONResponse{
		Body: p.body, Headers: reviewsapi.TooManyRequestsResponseHeaders{RetryAfter: p.retryAfter},
	}
}
