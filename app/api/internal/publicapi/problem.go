package publicapi

import (
	"encoding/json"
	"net/http"

	reviewsapi "github.com/baumac/reviews-app/specs/api/sdk/server/go"
)

const problemContentType = "application/problem+json"

func newProblem(status int, detail string) reviewsapi.Problem {
	p := reviewsapi.Problem{Type: "about:blank", Title: http.StatusText(status), Status: status}
	if detail != "" {
		p.Detail = &detail
	}
	return p
}

// writeProblem writes an error response outside the strict handler.
func writeProblem(w http.ResponseWriter, p reviewsapi.Problem) {
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

func badRequest(detail string) reviewsapi.BadRequestApplicationProblemPlusJSONResponse {
	return reviewsapi.BadRequestApplicationProblemPlusJSONResponse(newProblem(http.StatusBadRequest, detail))
}

func forbidden(code reviewsapi.ProblemCode, detail string) reviewsapi.ForbiddenApplicationProblemPlusJSONResponse {
	p := newProblem(http.StatusForbidden, detail)
	p.Code = &code
	return reviewsapi.ForbiddenApplicationProblemPlusJSONResponse(p)
}

// notCreator is the 403 of a user who does not own the resource.
func notCreator(detail string) reviewsapi.ForbiddenApplicationProblemPlusJSONResponse {
	return forbidden(reviewsapi.ProblemCodeNotCreator, detail)
}

func notFound(detail string) reviewsapi.NotFoundApplicationProblemPlusJSONResponse {
	return reviewsapi.NotFoundApplicationProblemPlusJSONResponse(newProblem(http.StatusNotFound, detail))
}

func conflict(code reviewsapi.ProblemCode, detail string) reviewsapi.ConflictApplicationProblemPlusJSONResponse {
	p := newProblem(http.StatusConflict, detail)
	p.Code = &code
	return reviewsapi.ConflictApplicationProblemPlusJSONResponse(p)
}

// propertyConflict names the request property that causes the conflict.
func propertyConflict(code reviewsapi.ProblemCode, property, detail string) reviewsapi.ConflictApplicationProblemPlusJSONResponse {
	p := conflict(code, detail)
	p.Property = &property
	return p
}

// placeConflict names the restaurant that has the place ID of the request.
func placeConflict(restaurantID string) reviewsapi.ConflictApplicationProblemPlusJSONResponse {
	p := propertyConflict(reviewsapi.ProblemCodePlaceTaken, "location", "Another restaurant has this place.")
	p.Restaurant = &restaurantID
	return p
}

func unprocessable(errs fieldErrors) reviewsapi.UnprocessableContentApplicationProblemPlusJSONResponse {
	p := newProblem(http.StatusUnprocessableEntity, "The request body has invalid properties.")
	list := []reviewsapi.ValidationError(errs)
	p.Errors = &list
	return reviewsapi.UnprocessableContentApplicationProblemPlusJSONResponse(p)
}

type fieldErrors []reviewsapi.ValidationError

func (e *fieldErrors) add(property, message string) {
	*e = append(*e, reviewsapi.ValidationError{Property: property, Message: message})
}
