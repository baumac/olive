package reviewsapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
)

const reviewJSON = `{
	"id": "r7d4f0h2j9l5s3v", "restaurant": "k3x9m2p7q1w8z4a", "restaurant_name": "Luigi's",
	"space": 7, "service": 9, "food": 8, "price": 6, "cleanliness": 5, "title": "Best carbonara in town",
	"creator": {"id": "u2g6k0m4o8q1s5w", "name": "Joe"},
	"created": "2024-11-04T01:38:00.431Z", "updated": "2024-11-05T09:12:00.000Z"
}`

// fakeServer embeds the interface. A test implements only the operations
// that it calls.
type fakeServer struct {
	reviewsapi.StrictServerInterface
	listReviews      func(reviewsapi.ListReviewsRequestObject) reviewsapi.ListReviewsResponseObject
	getRestaurant    func(reviewsapi.GetRestaurantRequestObject) reviewsapi.GetRestaurantResponseObject
	createRestaurant func(reviewsapi.CreateRestaurantRequestObject) reviewsapi.CreateRestaurantResponseObject
	updateRestaurant func(reviewsapi.UpdateRestaurantRequestObject) reviewsapi.UpdateRestaurantResponseObject
	signIn           func(reviewsapi.SignInRequestObject) reviewsapi.SignInResponseObject
}

func (f *fakeServer) ListReviews(_ context.Context, r reviewsapi.ListReviewsRequestObject) (reviewsapi.ListReviewsResponseObject, error) {
	return f.listReviews(r), nil
}

func (f *fakeServer) GetRestaurant(_ context.Context, r reviewsapi.GetRestaurantRequestObject) (reviewsapi.GetRestaurantResponseObject, error) {
	return f.getRestaurant(r), nil
}

func (f *fakeServer) CreateRestaurant(_ context.Context, r reviewsapi.CreateRestaurantRequestObject) (reviewsapi.CreateRestaurantResponseObject, error) {
	return f.createRestaurant(r), nil
}

func (f *fakeServer) UpdateRestaurant(_ context.Context, r reviewsapi.UpdateRestaurantRequestObject) (reviewsapi.UpdateRestaurantResponseObject, error) {
	return f.updateRestaurant(r), nil
}

func (f *fakeServer) SignIn(_ context.Context, r reviewsapi.SignInRequestObject) (reviewsapi.SignInResponseObject, error) {
	return f.signIn(r), nil
}

func serve(t *testing.T, f *fakeServer, method, target, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := reviewsapi.HandlerWithOptions(
		reviewsapi.NewStrictHandler(f, nil),
		reviewsapi.StdHTTPServerOptions{BaseURL: "/v1"},
	)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestReviewReadsScores(t *testing.T) {
	var r reviewsapi.Review
	if err := json.Unmarshal([]byte(reviewJSON), &r); err != nil {
		t.Fatal(err)
	}
	if r.Space != 7 || r.Service != 9 || r.Food != 8 || r.Price != 6 || r.Cleanliness != 5 || r.RestaurantName != "Luigi's" {
		t.Errorf("got %+v", r)
	}
}

func TestMergePatchSeparatesMissingAndNull(t *testing.T) {
	f := &fakeServer{}
	var got reviewsapi.RestaurantPatch
	f.updateRestaurant = func(r reviewsapi.UpdateRestaurantRequestObject) reviewsapi.UpdateRestaurantResponseObject {
		got = *r.Body
		return reviewsapi.UpdateRestaurant200JSONResponse{}
	}
	rec := serve(t, f, http.MethodPatch, "/v1/restaurants/k3x9m2p7q1w8z4a",
		"application/merge-patch+json", `{"description": null}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if !got.Description.IsNull() {
		t.Error("description: want null")
	}
	if got.Tags.IsSpecified() {
		t.Error("tags: want missing")
	}
	if got.Name != nil {
		t.Error("name: want missing")
	}
}

func TestDeepObjectAndExplodedFilters(t *testing.T) {
	f := &fakeServer{}
	var got reviewsapi.ListReviewsParams
	f.listReviews = func(r reviewsapi.ListReviewsRequestObject) reviewsapi.ListReviewsResponseObject {
		got = r.Params
		return reviewsapi.ListReviews200JSONResponse{Items: []reviewsapi.Review{}}
	}
	rec := serve(t, f, http.MethodGet,
		"/v1/restaurants/k3x9m2p7q1w8z4a/reviews?food[gte]=4&food[lt]=5"+
			"&creator[in]=u5n1c8r3t6y0b2e&creator[in]=u2g6k0m4o8q1s5w"+
			"&created[gte]=2024-11-01T00:00:00.000Z", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if got.Food == nil || *got.Food.Gte != 4 || *got.Food.Lt != 5 || got.Food.Eq != nil {
		t.Errorf("food: got %+v", got.Food)
	}
	if got.CreatorIn == nil || strings.Join(*got.CreatorIn, ",") != "u5n1c8r3t6y0b2e,u2g6k0m4o8q1s5w" {
		t.Errorf("creator[in]: got %v", got.CreatorIn)
	}
	if got.Created == nil || *got.Created.Gte != "2024-11-01T00:00:00.000Z" {
		t.Errorf("created: got %+v", got.Created)
	}
}

func TestNextCursorNullOnLastPage(t *testing.T) {
	f := &fakeServer{}
	f.listReviews = func(reviewsapi.ListReviewsRequestObject) reviewsapi.ListReviewsResponseObject {
		return reviewsapi.ListReviews200JSONResponse{
			Items:      []reviewsapi.Review{},
			NextCursor: nullable.NewNullNullable[reviewsapi.NextCursor](),
		}
	}
	rec := serve(t, f, http.MethodGet, "/v1/restaurants/k3x9m2p7q1w8z4a/reviews", "", "")
	if want := `{"items":[],"next_cursor":null}`; strings.TrimSpace(rec.Body.String()) != want {
		t.Errorf("got %s, want %s", rec.Body, want)
	}
}

func TestProblemResponse(t *testing.T) {
	f := &fakeServer{}
	f.getRestaurant = func(reviewsapi.GetRestaurantRequestObject) reviewsapi.GetRestaurantResponseObject {
		return reviewsapi.GetRestaurant404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: reviewsapi.NotFoundApplicationProblemPlusJSONResponse{
				Type: "about:blank", Title: "Not Found", Status: 404,
			},
		}
	}
	rec := serve(t, f, http.MethodGet, "/v1/restaurants/k3x9m2p7q1w8z4a", "", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("content type %q", ct)
	}
	var p reviewsapi.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || p.Title != "Not Found" {
		t.Errorf("body %s: %v", rec.Body, err)
	}
}

func TestResponseHeaders(t *testing.T) {
	f := &fakeServer{}
	location := "/v1/restaurants/k3x9m2p7q1w8z4a"
	f.createRestaurant = func(reviewsapi.CreateRestaurantRequestObject) reviewsapi.CreateRestaurantResponseObject {
		return reviewsapi.CreateRestaurant201JSONResponse{
			Headers: reviewsapi.CreateRestaurant201ResponseHeaders{Location: &location},
		}
	}
	f.signIn = func(reviewsapi.SignInRequestObject) reviewsapi.SignInResponseObject {
		return reviewsapi.SignIn429ApplicationProblemPlusJSONResponse{
			TooManyRequestsApplicationProblemPlusJSONResponse: reviewsapi.TooManyRequestsApplicationProblemPlusJSONResponse{
				Headers: reviewsapi.TooManyRequestsResponseHeaders{RetryAfter: 60},
			},
		}
	}
	rec := serve(t, f, http.MethodPost, "/v1/restaurants", "application/json", `{"name": "Luigi's"}`)
	if rec.Code != http.StatusCreated || rec.Header().Get("Location") != location {
		t.Errorf("create: status %d, Location %q", rec.Code, rec.Header().Get("Location"))
	}
	rec = serve(t, f, http.MethodPost, "/v1/users/tokens", "application/json",
		`{"email": "ana@example.com", "password": "correct-horse-battery"}`)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "60" {
		t.Errorf("sign in: status %d, Retry-After %q", rec.Code, rec.Header().Get("Retry-After"))
	}
}
