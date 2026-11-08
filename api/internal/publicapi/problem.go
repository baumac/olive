package publicapi

import (
	"encoding/json"
	"net/http"

	reviewsapi "github.com/baumac/reviews-app/sdk/go"
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

func forbidden(detail string) reviewsapi.ForbiddenApplicationProblemPlusJSONResponse {
	return reviewsapi.ForbiddenApplicationProblemPlusJSONResponse(newProblem(http.StatusForbidden, detail))
}

func notFound(detail string) reviewsapi.NotFoundApplicationProblemPlusJSONResponse {
	return reviewsapi.NotFoundApplicationProblemPlusJSONResponse(newProblem(http.StatusNotFound, detail))
}

func conflict(detail string) reviewsapi.ConflictApplicationProblemPlusJSONResponse {
	return reviewsapi.ConflictApplicationProblemPlusJSONResponse(newProblem(http.StatusConflict, detail))
}

func unprocessable(errs fieldErrors) reviewsapi.UnprocessableContentApplicationProblemPlusJSONResponse {
	p := newProblem(http.StatusUnprocessableEntity, "The request body has invalid properties.")
	list := []reviewsapi.ValidationError(errs)
	p.Errors = &list
	return reviewsapi.UnprocessableContentApplicationProblemPlusJSONResponse(p)
}
