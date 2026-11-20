package publicapi

import (
	"strings"
	"testing"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
)

func TestProblemUnexpected(t *testing.T) {
	t.Parallel()
	err := conflictProblem(reviewsapi.ProblemCodePhotoLimit, "A sixth photo.").unexpected()
	if err == nil || !strings.Contains(err.Error(), "409") {
		t.Errorf("error %v, want an error with 409", err)
	}
}

func TestProblemBodies(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		got  *problem
		want reviewsapi.Problem
	}{
		{"bad request", badRequestProblem("x"), reviewsapi.Problem(badRequest("x"))},
		{"not found", notFoundProblem("x"), reviewsapi.Problem(notFound("x"))},
		{"not creator", notCreatorProblem("x"), reviewsapi.Problem(notCreator("x"))},
		{"conflict", conflictProblem(reviewsapi.ProblemCodePhotoLimit, "x"), reviewsapi.Problem(conflict(reviewsapi.ProblemCodePhotoLimit, "x"))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, want := c.got.body, c.want
			if got.Status != want.Status || got.Title != want.Title || *got.Detail != *want.Detail ||
				(got.Code == nil) != (want.Code == nil) || (got.Code != nil && *got.Code != *want.Code) {
				t.Errorf("problem %+v, want %+v", got, want)
			}
		})
	}
}
