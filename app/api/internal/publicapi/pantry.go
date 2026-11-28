package publicapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/baumac/olive/app/api/internal/history"
	"github.com/baumac/olive/app/api/internal/recordvalues"
	"github.com/baumac/olive/app/api/internal/subjects"
	reviewsapi "github.com/baumac/olive/specs/api/sdk/server/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/pocketbase/core"
)

const pantryItemNotFound = "The pantry item does not exist."

var pantryItemSubject = newSubject(subject{
	Tables:        subjects.PantryItems,
	noun:          "pantry item",
	plural:        "pantry items",
	notFound:      pantryItemNotFound,
	listLimitCode: reviewsapi.ProblemCodeListPantryItemLimit,
	search: []scoreColumn{
		{"name", 10}, {"tags", 4}, {"producer", 3}, {"style", 2}, {"origin", 2}, {"description", 1},
	},
	reviewSearch: []scoreColumn{{"title", 10}, {"pantry_item_name", 2}, {"feedback", 1}},
})

var pantryItemSorts = map[string]sortField{
	"name":      textSort("pantry_items", "name"),
	"overall":   pantryItemSubject.overallSort,
	"relevance": relevanceSort,
	"created":   columnSort("pantry_items", "created"),
	"updated":   columnSort("pantry_items", "updated"),
}

// ListPantryItems uses keyset pagination like ListRestaurants.
func (s *Server) ListPantryItems(ctx context.Context, req reviewsapi.ListPantryItemsRequestObject) (reviewsapi.ListPantryItemsResponseObject, error) {
	p := req.Params
	bad := func(detail string) (reviewsapi.ListPantryItemsResponseObject, error) {
		return reviewsapi.ListPantryItems400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(detail)}, nil
	}

	limit := limitOf(p.Limit)
	search := pantryItemSubject.searchPage()
	if detail := search.parse(p.Search); detail != "" {
		return bad(detail)
	}
	sortSpec, detail := search.sortWithSearch(p.Sort)
	if detail != "" {
		return bad(detail)
	}
	keys, sort, detail := parseSort(sortSpec, "pantry_items", pantryItemSorts)
	if detail != "" {
		return bad(detail)
	}
	if detail := overallDetail(p.Overall); detail != "" {
		return bad(detail)
	}
	p.Limit, p.Cursor, p.Sort = nil, nil, &sort
	query, err := fingerprint("pantry_items", p)
	if err != nil {
		return nil, err
	}
	after, detail := decodeCursor(req.Params.Cursor, query, len(keys))
	if detail != "" {
		return bad(detail)
	}

	w := newWhere()
	search.filter(w)
	if p.Overall != nil || sortsBy(sort, "overall") {
		w.joinOverall(pantryItemSubject)
	}
	w.compare(pantryItemSubject.overallColumn(), averageComparisons(p.Overall))
	w.categoryFilters(p.CategoryEq, p.CategoryIn)
	w.tagFilters(pantryItemSubject, p.TagsHas, p.TagsHasAny, p.TagsHasAll)
	w.idFilters("pantry_items", "creator", p.CreatorEq, p.CreatorIn)
	w.inList(pantryItemSubject, p.ListEq)
	w.compare(column("pantry_items", "created"), timestampComparisons(p.Created))
	w.compare(column("pantry_items", "updated"), timestampComparisons(p.Updated))
	if after != nil {
		w.keyset(keys, after)
	}

	records, next, err := s.page("pantry_items", w, keys, limit, query)
	if err != nil {
		return nil, err
	}
	items, err := s.pantryItems(stateFrom(ctx).user, records...)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ListPantryItems200JSONResponse{Items: items, NextCursor: next}, nil
}

func (w *where) categoryFilters(eq *reviewsapi.PantryItemCategory, in *[]reviewsapi.PantryItemCategory) {
	if eq != nil {
		w.compare(column("pantry_items", "category"), []comparison{{"=", string(*eq)}})
	}
	if in != nil {
		values := make([]string, len(*in))
		for i, c := range *in {
			values[i] = string(c)
		}
		w.in(column("pantry_items", "category"), values)
	}
}

// CreatePantryItem sets the user of the token as the creator. Two pantry
// items can have one name. See decision 00012.
func (s *Server) CreatePantryItem(ctx context.Context, req reviewsapi.CreatePantryItemRequestObject) (reviewsapi.CreatePantryItemResponseObject, error) {
	user := stateFrom(ctx).user
	b := req.Body
	if p := tooManyTags(createdTags(b.Tags)); p != nil {
		return reviewsapi.CreatePantryItem422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: *p}, nil
	}
	collection, err := s.app.FindCachedCollectionByNameOrId("pantry_items")
	if err != nil {
		return nil, err
	}
	r := core.NewRecord(collection)
	r.Set("name", b.Name)
	r.Set("category", string(b.Category))
	setOptionalText(r, "producer", b.Producer)
	setOptionalText(r, "origin", b.Origin)
	setOptionalText(r, "style", b.Style)
	setOptionalText(r, "description", b.Description)
	r.Set("creator", user.Id)

	err = s.app.RunInTransaction(func(tx core.App) error {
		if err := setTags(tx, r, b.Tags); err != nil {
			return err
		}
		return tx.SaveWithContext(history.WithEditor(ctx, user.Id), r)
	})
	if err != nil {
		return nil, err
	}
	if err := recordvalues.ExpandTags(s.app, r); err != nil {
		return nil, err
	}
	location := pantryItemSubject.path(r.Id)
	return reviewsapi.CreatePantryItem201JSONResponse{
		Body:    toPantryItem(r, map[string]*core.Record{user.Id: user}, nil, nil, viewerData{}),
		Headers: reviewsapi.CreatePantryItem201ResponseHeaders{Location: &location},
	}, nil
}

func (s *Server) GetPantryItem(ctx context.Context, req reviewsapi.GetPantryItemRequestObject) (reviewsapi.GetPantryItemResponseObject, error) {
	r, err := findParent(s.app, pantryItemSubject, req.PantryItemId)
	if errors.Is(err, sql.ErrNoRows) {
		return reviewsapi.GetPantryItem404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(pantryItemNotFound)}, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := s.pantryItems(stateFrom(ctx).user, r)
	if err != nil {
		return nil, err
	}
	return reviewsapi.GetPantryItem200JSONResponse(items[0]), nil
}

// UpdatePantryItem lets each approved user change each property. See
// decision 00004.
func (s *Server) UpdatePantryItem(ctx context.Context, req reviewsapi.UpdatePantryItemRequestObject) (reviewsapi.UpdatePantryItemResponseObject, error) {
	user := stateFrom(ctx).user
	b := req.Body
	if p := tooManyTags(patchedTags(b.Tags)); p != nil {
		return reviewsapi.UpdatePantryItem422ApplicationProblemPlusJSONResponse{UnprocessableContentApplicationProblemPlusJSONResponse: *p}, nil
	}

	var response reviewsapi.UpdatePantryItemResponseObject
	var saved *core.Record
	err := s.app.RunInTransaction(func(tx core.App) error {
		r, err := findParent(tx, pantryItemSubject, req.PantryItemId)
		if errors.Is(err, sql.ErrNoRows) {
			response = reviewsapi.UpdatePantryItem404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(pantryItemNotFound)}
			return nil
		}
		if err != nil {
			return err
		}
		ch := &changes{r: r}
		if b.Name != nil {
			ch.set("name", *b.Name)
		}
		if b.Category != nil {
			ch.set("category", string(*b.Category))
		}
		ch.patchText("producer", b.Producer)
		ch.patchText("origin", b.Origin)
		ch.patchText("style", b.Style)
		ch.patchText("description", b.Description)
		if err := ch.patchTags(tx, b.Tags); err != nil {
			return err
		}
		saved = r
		if !ch.changed {
			return nil
		}
		return tx.SaveWithContext(history.WithEditor(ctx, user.Id), r)
	})
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response, nil
	}
	items, err := s.pantryItems(user, saved)
	if err != nil {
		return nil, err
	}
	return reviewsapi.UpdatePantryItem200JSONResponse(items[0]), nil
}

// DeletePantryItem also deletes the reviews of the user.
func (s *Server) DeletePantryItem(ctx context.Context, req reviewsapi.DeletePantryItemRequestObject) (reviewsapi.DeletePantryItemResponseObject, error) {
	p, err := s.deleteParent(pantryItemSubject, req.PantryItemId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusForbidden:
			return reviewsapi.DeletePantryItem403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		case http.StatusNotFound:
			return reviewsapi.DeletePantryItem404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusConflict:
			return reviewsapi.DeletePantryItem409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: p.conflict()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.DeletePantryItem204Response{}, nil
}

type pantryItemRating struct {
	PantryItem string  `db:"pantry_item"`
	Count      int     `db:"count"`
	Overall    float64 `db:"overall"`
	Taste      float64 `db:"taste"`
	Aroma      float64 `db:"aroma"`
	Texture    float64 `db:"texture"`
	Value      float64 `db:"value"`
}

func (r pantryItemRating) toAPI() reviewsapi.PantryItemRating {
	return reviewsapi.PantryItemRating{Overall: r.Overall, Taste: r.Taste, Aroma: r.Aroma, Texture: r.Texture, Value: r.Value}
}

// pantryItems gets the creators, the ratings, the covers, the favorites, and
// the own reviews of records. A nil user has none of the last two.
func (s *Server) pantryItems(user *core.Record, records ...*core.Record) ([]reviewsapi.PantryItem, error) {
	if err := recordvalues.ExpandTags(s.app, records...); err != nil {
		return nil, err
	}
	users, err := s.creators(records...)
	if err != nil {
		return nil, err
	}
	ratings, err := ratingsOf(s, pantryItemSubject, records, func(r pantryItemRating) string { return r.PantryItem })
	if err != nil {
		return nil, err
	}
	rows, err := s.coverRows(pantryItemSubject, records...)
	if err != nil {
		return nil, err
	}
	covers := make(map[string]reviewsapi.PantryItemPhoto, len(rows))
	for id, r := range rows {
		covers[id] = r.pantryItemPhoto()
	}
	viewer, err := s.viewerDataOf(pantryItemSubject, user, records...)
	if err != nil {
		return nil, err
	}
	items := make([]reviewsapi.PantryItem, len(records))
	for i, r := range records {
		items[i] = toPantryItem(r, users, ratings, covers, viewer)
	}
	return items, nil
}

// toPantryItem gives a pantry item that ratings does not list no reviews.
func toPantryItem(r *core.Record, users map[string]*core.Record, ratings map[string]pantryItemRating, covers map[string]reviewsapi.PantryItemPhoto, viewer viewerData) reviewsapi.PantryItem {
	out := reviewsapi.PantryItem{
		Id:             r.Id,
		Name:           r.GetString("name"),
		Category:       reviewsapi.PantryItemCategory(r.GetString("category")),
		Producer:       optionalText(r, "producer"),
		Origin:         optionalText(r, "origin"),
		Style:          optionalText(r, "style"),
		Description:    optionalText(r, "description"),
		Tags:           tagsOf(r),
		Creator:        nullable.NewNullNullable[reviewsapi.Creator](),
		Rating:         nullable.NewNullNullable[reviewsapi.PantryItemRating](),
		Cover:          nullable.NewNullNullable[reviewsapi.PantryItemPhoto](),
		Favorite:       viewer.favorites[r.Id],
		OwnReviewCount: viewer.ownReviews[r.Id],
		Created:        apiTimestamp(r.GetDateTime("created")),
		Updated:        apiTimestamp(r.GetDateTime("updated")),
	}
	if u, ok := users[r.GetString("creator")]; ok {
		out.Creator = nullable.NewNullableWithValue(toCreator(u))
	}
	if rating, ok := ratings[r.Id]; ok {
		out.ReviewCount = rating.Count
		out.Rating = nullable.NewNullableWithValue(rating.toAPI())
	}
	if cover, ok := covers[r.Id]; ok {
		out.Cover = nullable.NewNullableWithValue(cover)
	}
	return out
}
