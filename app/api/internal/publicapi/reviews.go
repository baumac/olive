package publicapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/pocketbase/core"
)

// reviewInput is a create or patch body. The body schema checked the values
// before.
type reviewInput struct {
	scores   map[string]int
	date     *string
	title    *string
	feedback nullable.Nullable[string]
}

// reviewOverall is a generated column with indexes. It has the value that
// toReview computes. See decision 00005.
func (sub subject) reviewOverall() string {
	return column(sub.Reviews, "overall")
}

// overallFilter uses an overall index only with the overall sort. With a
// different sort, SQLite would read the index and then sort all matches. The
// unary plus keeps the index of the sort.
func (sub subject) overallFilter(sort string) string {
	if sortsBy(sort, "overall") {
		return sub.reviewOverall()
	}
	return "+" + sub.reviewOverall()
}

// reviewDefaultSort puts the newest visit or cook first. See decision 00005.
const reviewDefaultSort = "-date"

func makeReviewSorts(sub subject) map[string]sortField {
	sorts := map[string]sortField{
		"overall":   func(desc bool) []sortKey { return []sortKey{{expr: sub.reviewOverall(), desc: desc}} },
		"title":     textSort(sub.Reviews, "title"),
		"relevance": relevanceSort,
		// The reviews of one day sort by the time that the user wrote them.
		"date": func(desc bool) []sortKey {
			return []sortKey{{expr: column(sub.Reviews, "date"), desc: desc}, {expr: column(sub.Reviews, "created"), desc: desc}}
		},
		"created": columnSort(sub.Reviews, "created"),
		"updated": columnSort(sub.Reviews, "updated"),
	}
	for _, name := range sub.Scores {
		sorts[name] = columnSort(sub.Reviews, name)
	}
	return sorts
}

// ListRestaurantReviews returns 404 when the restaurant does not exist. It does not
// return an empty list.
func (s *Server) ListRestaurantReviews(_ context.Context, req reviewsapi.ListRestaurantReviewsRequestObject) (reviewsapi.ListRestaurantReviewsResponseObject, error) {
	p := req.Params
	all := reviewsapi.ListAllRestaurantReviewsParams{
		Limit: p.Limit, Cursor: p.Cursor, Sort: p.Sort, Search: p.Search, RestaurantEq: &req.RestaurantId,
		Space: p.Space, Service: p.Service, Food: p.Food, Price: p.Price,
		Overall: p.Overall, Created: p.Created, Updated: p.Updated, CreatorEq: p.CreatorEq, CreatorIn: p.CreatorIn,
	}
	// The cursor keeps the fingerprint of the first version of this
	// operation. Open pages can then still load more after a deploy.
	fingerprintOf := func(sort string) []any {
		q := req.Params
		q.Limit, q.Cursor, q.Sort = nil, nil, &sort
		return []any{"reviews", req.RestaurantId, q}
	}
	page, err := s.reviewPage(all, fingerprintOf, func() (*core.Record, error) {
		r, err := s.findRestaurant(s.app, req.RestaurantId)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return r, err
	})
	switch {
	case err != nil:
		return nil, err
	case page.bad != "":
		return reviewsapi.ListRestaurantReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(page.bad)}, nil
	case page.notFound:
		return reviewsapi.ListRestaurantReviews404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(restaurantNotFound)}, nil
	}
	return reviewsapi.ListRestaurantReviews200JSONResponse(page.list), nil
}

// ListAllRestaurantReviews lists the reviews of all restaurants. An unknown
// restaurant in restaurant[eq] gives an empty list, not 404.
func (s *Server) ListAllRestaurantReviews(_ context.Context, req reviewsapi.ListAllRestaurantReviewsRequestObject) (reviewsapi.ListAllRestaurantReviewsResponseObject, error) {
	page, err := s.reviewPage(req.Params, nil, nil)
	if err != nil {
		return nil, err
	}
	if page.bad != "" {
		return reviewsapi.ListAllRestaurantReviews400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(page.bad)}, nil
	}
	return reviewsapi.ListAllRestaurantReviews200JSONResponse(page.list), nil
}

type reviewPageResult struct {
	list     reviewsapi.RestaurantReviewList
	bad      string
	notFound bool
}

// reviewPage gets a page of restaurant reviews. A nil fingerprintOf uses the
// parameters of listAllRestaurantReviews.
func (s *Server) reviewPage(
	p reviewsapi.ListAllRestaurantReviewsParams,
	fingerprintOf func(sort string) []any,
	restaurant func() (*core.Record, error),
) (reviewPageResult, error) {
	if fingerprintOf == nil {
		fingerprintOf = func(sort string) []any {
			q := p
			q.Limit, q.Cursor, q.Sort = nil, nil, &sort
			return []any{"reviews", q}
		}
	}
	page, err := s.reviewsOf(restaurantSubject, reviewQuery{
		limit: p.Limit, cursor: p.Cursor, sort: p.Sort, search: p.Search,
		parentEq: p.RestaurantEq, parentIn: p.RestaurantIn,
		scores: map[string]*reviewsapi.ScoreRange{
			"space": p.Space, "service": p.Service, "food": p.Food, "price": p.Price,
		},
		overall: p.Overall, created: p.Created, updated: p.Updated, creatorEq: p.CreatorEq, creatorIn: p.CreatorIn,
	}, fingerprintOf, restaurant)
	if err != nil || page.bad != "" || page.notFound {
		return reviewPageResult{bad: page.bad, notFound: page.notFound}, err
	}
	items := make([]reviewsapi.RestaurantReview, len(page.records))
	for i, r := range page.records {
		if items[i], err = toReview(r, page.users[r.GetString("creator")], page.names[r.GetString("restaurant")], page.photos[r.Id]); err != nil {
			return reviewPageResult{}, err
		}
	}
	return reviewPageResult{list: reviewsapi.RestaurantReviewList{Items: items, NextCursor: page.next}}, nil
}

// reviewQuery has the parameters of a review list. Each subject has its own
// params types.
type reviewQuery struct {
	limit            *int
	cursor, sort     *string
	search           *string
	parentEq         *string
	parentIn         *[]string
	scores           map[string]*reviewsapi.ScoreRange
	overall          *reviewsapi.AverageRange
	created, updated *reviewsapi.TimestampRange
	creatorEq        *string
	creatorIn        *[]string
}

// reviewRecords is a page of reviews with the data of their responses, or
// the reason for a 400 or a 404.
type reviewRecords struct {
	records  []*core.Record
	users    map[string]*core.Record
	names    map[string]string
	photos   map[string][]reviewsapi.Photo
	next     nullable.Nullable[string]
	bad      string
	notFound bool
}

// reviewsOf checks the parameters, then calls parent, then gets the page. A
// nil parent skips the check. This order gives 400 before 404. fingerprintOf
// gets the sort value of the cursor.
func (s *Server) reviewsOf(
	sub subject,
	q reviewQuery,
	fingerprintOf func(sort string) []any,
	parent func() (*core.Record, error),
) (reviewRecords, error) {
	limit := limitOf(q.limit)
	search := sub.reviewSearchPage()
	if detail := search.parse(q.search); detail != "" {
		return reviewRecords{bad: detail}, nil
	}
	sortSpec, detail := search.sortWithSearch(q.sort)
	if detail != "" {
		return reviewRecords{bad: detail}, nil
	}
	if sortSpec == nil {
		sortSpec = new(reviewDefaultSort)
	}
	keys, sort, detail := parseSort(sortSpec, sub.Reviews, sub.reviewSorts)
	if detail != "" {
		return reviewRecords{bad: detail}, nil
	}
	if detail := overallDetail(q.overall); detail != "" {
		return reviewRecords{bad: detail}, nil
	}
	query, err := fingerprint(fingerprintOf(sort)...)
	if err != nil {
		return reviewRecords{}, err
	}
	after, detail := decodeCursor(q.cursor, query, len(keys))
	if detail != "" {
		return reviewRecords{bad: detail}, nil
	}

	w := newWhere()
	search.filter(w)
	w.idFilters(sub.Reviews, sub.Parent, q.parentEq, q.parentIn)
	for _, name := range sub.Scores {
		w.compare(column(sub.Reviews, name), scoreComparisons(q.scores[name]))
	}
	w.compare(sub.overallFilter(sort), averageComparisons(q.overall))
	w.idFilters(sub.Reviews, "creator", q.creatorEq, q.creatorIn)
	w.compare(column(sub.Reviews, "created"), timestampComparisons(q.created))
	w.compare(column(sub.Reviews, "updated"), timestampComparisons(q.updated))
	if after != nil {
		w.keyset(keys, after)
	}

	known := map[string]string{}
	if parent != nil {
		r, err := parent()
		if err != nil {
			return reviewRecords{}, err
		}
		if r == nil {
			return reviewRecords{notFound: true}, nil
		}
		known[r.Id] = r.GetString("name")
	}

	out := reviewRecords{}
	if out.records, out.next, err = s.page(sub.Reviews, w, keys, limit, query); err != nil {
		return reviewRecords{}, err
	}
	if out.users, err = s.creators(out.records...); err != nil {
		return reviewRecords{}, err
	}
	if out.names, err = s.parentNames(sub, known, out.records...); err != nil {
		return reviewRecords{}, err
	}
	if out.photos, err = s.reviewPhotos(sub, out.records...); err != nil {
		return reviewRecords{}, err
	}
	return out, nil
}

// parentNames reads only the parents that are not in known. The list of the
// reviews of one parent then needs no extra query.
func (s *Server) parentNames(sub subject, known map[string]string, reviews ...*core.Record) (map[string]string, error) {
	out := maps.Clone(known)
	ids := map[string]bool{}
	for _, r := range reviews {
		if _, ok := out[r.GetString(sub.Parent)]; !ok {
			ids[r.GetString(sub.Parent)] = true
		}
	}
	if len(ids) == 0 {
		return out, nil
	}
	parents, err := s.app.FindRecordsByIds(sub.Table, slices.Collect(maps.Keys(ids)))
	if err != nil {
		return nil, err
	}
	for _, r := range parents {
		out[r.Id] = r.GetString("name")
	}
	return out, nil
}

// CreateRestaurantReview makes a review of one visit. A user can write many
// reviews of one restaurant. See decision 00005.
func (s *Server) CreateRestaurantReview(ctx context.Context, req reviewsapi.CreateRestaurantReviewRequestObject) (reviewsapi.CreateRestaurantReviewResponseObject, error) {
	user := stateFrom(ctx).user
	saved, restaurant, p, err := s.createReview(restaurantSubject, req.RestaurantId, user, createInput(req.Body))
	if err != nil {
		return nil, err
	}
	if p != nil {
		if p.status() == http.StatusNotFound {
			return reviewsapi.CreateRestaurantReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	out, err := toReview(saved, user, restaurant.GetString("name"), nil)
	if err != nil {
		return nil, err
	}
	location := restaurantSubject.reviewPath(req.RestaurantId, saved.Id)
	return reviewsapi.CreateRestaurantReview201JSONResponse{
		Body:    out,
		Headers: reviewsapi.CreateRestaurantReview201ResponseHeaders{Location: &location},
	}, nil
}

// createReview reads the parent and saves the review in one transaction. The
// problem is a 404.
func (s *Server) createReview(sub subject, parentID string, user *core.Record, in reviewInput) (review, parent *core.Record, p *problem, err error) {
	err = s.app.RunInTransaction(func(tx core.App) error {
		var err error
		if parent, err = findParent(tx, sub, parentID); errors.Is(err, sql.ErrNoRows) {
			p = notFoundProblem(sub.notFound)
			return nil
		} else if err != nil {
			return err
		}

		collection, err := tx.FindCachedCollectionByNameOrId(sub.Reviews)
		if err != nil {
			return err
		}
		r := core.NewRecord(collection)
		r.Set(sub.Parent, parentID)
		r.Set("creator", user.Id)
		in.apply(r)
		review = r
		return tx.Save(r)
	})
	switch {
	case err != nil:
		return nil, nil, nil, err
	case p != nil:
		return nil, nil, p, nil
	}
	return review, parent, nil, nil
}

// GetRestaurantReview returns 404 for a review of a different restaurant.
func (s *Server) GetRestaurantReview(_ context.Context, req reviewsapi.GetRestaurantReviewRequestObject) (reviewsapi.GetRestaurantReviewResponseObject, error) {
	restaurant, r, detail, err := s.findReview(s.app, restaurantSubject, req.RestaurantId, req.ReviewId)
	if err != nil {
		return nil, err
	}
	if detail != "" {
		return reviewsapi.GetRestaurantReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(detail)}, nil
	}
	users, err := s.creators(r)
	if err != nil {
		return nil, err
	}
	photos, err := s.reviewPhotos(restaurantSubject, r)
	if err != nil {
		return nil, err
	}
	out, err := toReview(r, users[r.GetString("creator")], restaurant.GetString("name"), photos[r.Id])
	if err != nil {
		return nil, err
	}
	return reviewsapi.GetRestaurantReview200JSONResponse(out), nil
}

// UpdateRestaurantReview lets only the creator change a review.
func (s *Server) UpdateRestaurantReview(ctx context.Context, req reviewsapi.UpdateRestaurantReviewRequestObject) (reviewsapi.UpdateRestaurantReviewResponseObject, error) {
	user := stateFrom(ctx).user
	saved, restaurant, p, err := s.updateReview(restaurantSubject, req.RestaurantId, req.ReviewId, user, patchInput(req.Body))
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusNotFound:
			return reviewsapi.UpdateRestaurantReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusForbidden:
			return reviewsapi.UpdateRestaurantReview403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		}
		return nil, p.unexpected()
	}
	photos, err := s.reviewPhotos(restaurantSubject, saved)
	if err != nil {
		return nil, err
	}
	out, err := toReview(saved, user, restaurant.GetString("name"), photos[saved.Id])
	if err != nil {
		return nil, err
	}
	return reviewsapi.UpdateRestaurantReview200JSONResponse(out), nil
}

// updateReview lets only the creator change a review. The problem is a 403
// or a 404.
func (s *Server) updateReview(sub subject, parentID, reviewID string, user *core.Record, in reviewInput) (review, parent *core.Record, p *problem, err error) {
	err = s.app.RunInTransaction(func(tx core.App) error {
		var detail string
		var err error
		parent, review, detail, err = s.findReview(tx, sub, parentID, reviewID)
		if err != nil {
			return err
		}
		if detail != "" {
			p = notFoundProblem(detail)
			return nil
		}
		if review.GetString("creator") != user.Id {
			p = notCreatorProblem("Only the creator of the review can change it.")
			return nil
		}
		if !in.apply(review) {
			return nil
		}
		return tx.Save(review)
	})
	return review, parent, p, err
}

// DeleteRestaurantReview lets only the creator delete a review.
func (s *Server) DeleteRestaurantReview(ctx context.Context, req reviewsapi.DeleteRestaurantReviewRequestObject) (reviewsapi.DeleteRestaurantReviewResponseObject, error) {
	p, err := s.deleteReview(restaurantSubject, req.RestaurantId, req.ReviewId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusNotFound:
			return reviewsapi.DeleteRestaurantReview404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusForbidden:
			return reviewsapi.DeleteRestaurantReview403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.DeleteRestaurantReview204Response{}, nil
}

// deleteReview lets only the creator delete a review. The problem is a 403
// or a 404.
func (s *Server) deleteReview(sub subject, parentID, reviewID string, user *core.Record) (p *problem, err error) {
	err = s.app.RunInTransaction(func(tx core.App) error {
		_, r, detail, err := s.findReview(tx, sub, parentID, reviewID)
		if err != nil {
			return err
		}
		if detail != "" {
			p = notFoundProblem(detail)
			return nil
		}
		if r.GetString("creator") != user.Id {
			p = notCreatorProblem("Only the creator of the review can delete it.")
			return nil
		}
		return tx.Delete(r)
	})
	return p, err
}

func findParent(app core.App, sub subject, id string) (*core.Record, error) {
	return app.FindRecordById(sub.Table, id)
}

// findReview returns a detail for the 404 response if the parent or the
// review does not exist. A review of a different parent does not exist.
func (s *Server) findReview(app core.App, sub subject, parentID, reviewID string) (parent, review *core.Record, detail string, err error) {
	parent, err = findParent(app, sub, parentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, sub.notFound, nil
	} else if err != nil {
		return nil, nil, "", err
	}
	review, err = app.FindRecordById(sub.Reviews, reviewID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && review.GetString(sub.Parent) != parentID) {
		return nil, nil, "The review does not exist.", nil
	}
	if err != nil {
		return nil, nil, "", err
	}
	return parent, review, "", nil
}

func (sub subject) path(parentID string) string {
	return basePath + "/" + sub.Table + "/" + parentID
}

func (sub subject) reviewPath(parentID, reviewID string) string {
	return sub.path(parentID) + "/reviews/" + reviewID
}

func createInput(body *reviewsapi.RestaurantReviewCreate) reviewInput {
	return reviewInput{
		scores: map[string]int{"space": body.Space, "service": body.Service, "food": body.Food, "price": body.Price},
		date:   &body.Date, title: &body.Title, feedback: optional(body.Feedback),
	}
}

func patchInput(body *reviewsapi.RestaurantReviewPatch) reviewInput {
	in := reviewInput{scores: map[string]int{}, date: body.Date, title: body.Title, feedback: body.Feedback}
	setScore(in.scores, "space", body.Space)
	setScore(in.scores, "service", body.Service)
	setScore(in.scores, "food", body.Food)
	setScore(in.scores, "price", body.Price)
	return in
}

func setScore(scores map[string]int, name string, v *int) {
	if v != nil {
		scores[name] = *v
	}
}

func optional[T any](v *T) nullable.Nullable[T] {
	if v == nil {
		return nil
	}
	return nullable.NewNullableWithValue(*v)
}

// apply returns true if a field gets a different value.
func (in reviewInput) apply(r *core.Record) bool {
	ch := &changes{r: r}
	for name, v := range in.scores {
		ch.set(name, float64(v))
	}
	if in.date != nil {
		ch.set("date", *in.date)
	}
	if in.title != nil {
		ch.set("title", *in.title)
	}
	ch.patchText("feedback", in.feedback)
	return ch.changed
}

// toReview gives an empty photos array for nil photos. The spec requires
// the array.
func toReview(r *core.Record, creator *core.Record, restaurantName string, photos []reviewsapi.Photo) (reviewsapi.RestaurantReview, error) {
	if creator == nil {
		return reviewsapi.RestaurantReview{}, fmt.Errorf("review %s has no creator", r.Id)
	}
	if restaurantName == "" {
		return reviewsapi.RestaurantReview{}, fmt.Errorf("review %s has no restaurant", r.Id)
	}
	if photos == nil {
		photos = []reviewsapi.Photo{}
	}
	space, service, food, price := r.GetInt("space"), r.GetInt("service"), r.GetInt("food"), r.GetInt("price")
	// TestOverallColumnEqualsResponse keeps the overall column equal to this.
	return reviewsapi.RestaurantReview{
		Id: r.Id, Restaurant: r.GetString("restaurant"), RestaurantName: restaurantName,
		Date: r.GetString("date"), Overall: float64(space+service+food+price) / 4,
		Space: space, Service: service, Food: food, Price: price,
		Title: r.GetString("title"), Feedback: optionalText(r, "feedback"),
		Creator: toCreator(creator), Photos: photos, Created: apiTimestamp(r.GetDateTime("created")), Updated: apiTimestamp(r.GetDateTime("updated")),
	}, nil
}

func scoreComparisons(r *reviewsapi.ScoreRange) []comparison {
	if r == nil {
		return nil
	}
	return comparisons(r.Lt, r.Lte, r.Eq, r.Gte, r.Gt)
}
