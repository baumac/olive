package publicapi_test

import (
	"net/http"
	"testing"
)

const unknownID = "aaaaaaaaaaaaaaa"

func TestChangesOfMissingRecords(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	bo := e.signUp("bo@example.com", "Bo")
	restaurantID := e.createRestaurant(ana.Token, `{"name":"Cafe"}`)["id"].(string)
	reviewID := e.createReview(bo.Token, restaurantID, `{"date":"2024-11-06","space":7,"service":9,"food":8,"price":6,"title":"T"}`)["id"].(string)
	reviews := "/v1/restaurants/" + restaurantID + "/reviews/"

	for _, req := range []request{
		{method: http.MethodPatch, path: "/v1/restaurants/" + unknownID, body: `{"name":"New"}`},
		{method: http.MethodDelete, path: "/v1/restaurants/" + unknownID},
		{method: http.MethodPatch, path: reviews + unknownID, body: `{"title":"New"}`},
		{method: http.MethodDelete, path: reviews + unknownID},
		{method: http.MethodPut, path: "/v1/lists/" + unknownID + "/restaurants/" + restaurantID},
		{method: http.MethodDelete, path: "/v1/lists/" + unknownID + "/restaurants/" + restaurantID},
	} {
		req.token = ana.Token
		e.expect(req, http.StatusNotFound)
	}

	e.expectCode(request{method: http.MethodDelete, path: reviews + reviewID, token: ana.Token}, http.StatusForbidden, "not_creator")
	e.expect(request{method: http.MethodGet, path: reviews + reviewID}, http.StatusOK)
}
