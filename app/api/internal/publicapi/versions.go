package publicapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/baumac/reviews-app/app/api/internal/history"
	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// versionItem has the JSON shape of a RestaurantVersion or a RecipeVersion.
// Before and After have only the changed properties.
type versionItem struct {
	ID      string              `json:"id"`
	Number  int                 `json:"number"`
	Action  string              `json:"action"`
	Source  string              `json:"source"`
	Editor  *reviewsapi.Creator `json:"editor"`
	Before  map[string]any      `json:"before"`
	After   map[string]any      `json:"after"`
	Created string              `json:"created"`
}

// remarshal changes the version items to the item type of the response.
func remarshal[T any](items []versionItem) ([]T, error) {
	b, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	var out []T
	return out, json.Unmarshal(b, &out)
}

func (s *Server) ListRestaurantVersions(_ context.Context, req reviewsapi.ListRestaurantVersionsRequestObject) (reviewsapi.ListRestaurantVersionsResponseObject, error) {
	versions, next, p, err := s.versionPage(restaurantSubject, req.RestaurantId, req.Params.Cursor, req.Params.Limit)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusBadRequest:
			return reviewsapi.ListRestaurantVersions400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: p.badRequest()}, nil
		case http.StatusNotFound:
			return reviewsapi.ListRestaurantVersions404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	items, err := remarshal[reviewsapi.RestaurantVersion](versions)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ListRestaurantVersions200JSONResponse{Items: items, NextCursor: next}, nil
}

func (s *Server) ListRecipeVersions(_ context.Context, req reviewsapi.ListRecipeVersionsRequestObject) (reviewsapi.ListRecipeVersionsResponseObject, error) {
	versions, next, p, err := s.versionPage(recipeSubject, req.RecipeId, req.Params.Cursor, req.Params.Limit)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusBadRequest:
			return reviewsapi.ListRecipeVersions400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: p.badRequest()}, nil
		case http.StatusNotFound:
			return reviewsapi.ListRecipeVersions404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	items, err := remarshal[reviewsapi.RecipeVersion](versions)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ListRecipeVersions200JSONResponse{Items: items, NextCursor: next}, nil
}

// versionPage gets one page of the versions of a parent, the newest first.
// The baseline is not on a page. The problem is a 400 or a 404.
func (s *Server) versionPage(sub subject, parentID string, cursor *string, limit *int) ([]versionItem, nullable.Nullable[string], *problem, error) {
	keys := []sortKey{
		{expr: column(sub.Versions, "number"), desc: true},
		{expr: column(sub.Versions, "id"), desc: true, unique: true},
	}
	query, err := fingerprint(sub.Parent+"_versions", parentID)
	if err != nil {
		return nil, nil, nil, err
	}
	after, detail := decodeCursor(cursor, query, len(keys))
	if detail != "" {
		return nil, nil, badRequestProblem(detail), nil
	}
	_, err = findParent(s.app, sub, parentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, notFoundProblem(sub.notFound), nil
	}
	if err != nil {
		return nil, nil, nil, err
	}

	w := newWhere()
	w.add(column(sub.Versions, sub.Parent) + " = " + w.param(parentID))
	w.add(column(sub.Versions, "action") + " != " + w.param(history.ActionBaseline))
	if after != nil {
		w.keyset(keys, after)
	}
	records, next, err := s.page(sub.Versions, w, keys, limitOf(limit), query)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(records) == 0 {
		return []versionItem{}, next, nil, nil
	}
	older, err := s.versionBefore(sub, parentID, records[len(records)-1].GetInt("number"))
	if err != nil {
		return nil, nil, nil, err
	}
	editors, err := s.usersOf("editor", records)
	if err != nil {
		return nil, nil, nil, err
	}

	items := make([]versionItem, len(records))
	for i, r := range records {
		prev := older
		if i+1 < len(records) {
			prev = records[i+1]
		}
		items[i], err = toVersionItem(r, prev, editors)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return items, next, nil, nil
}

// versionBefore gets the newest version of a parent with a number below
// number. It returns nil for the first version.
func (s *Server) versionBefore(sub subject, parentID string, number int) (*core.Record, error) {
	r := &core.Record{}
	err := s.app.RecordQuery(sub.Versions).
		AndWhere(dbx.HashExp{sub.Parent: parentID}).
		AndWhere(dbx.NewExp("[[number]] < {:number}", dbx.Params{"number": number})).
		OrderBy("number DESC").Limit(1).One(r)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// toVersionItem takes before and after from the snapshots of prev and v.
// The stored changed field names the properties. A create has an empty
// before.
func toVersionItem(v, prev *core.Record, editors map[string]*core.Record) (versionItem, error) {
	item := versionItem{
		ID: v.Id, Number: v.GetInt("number"), Action: v.GetString("action"), Source: v.GetString("source"),
		Before: map[string]any{}, After: map[string]any{}, Created: apiTimestamp(v.GetDateTime("created")),
	}
	if u, ok := editors[v.GetString("editor")]; ok {
		editor := toCreator(u)
		item.Editor = &editor
	}
	var snapshot, old map[string]any
	if err := v.UnmarshalJSONField("snapshot", &snapshot); err != nil {
		return item, err
	}
	if prev != nil && item.Action != history.ActionCreate {
		if err := prev.UnmarshalJSONField("snapshot", &old); err != nil {
			return item, err
		}
	}
	var changed []string
	if err := v.UnmarshalJSONField("changed", &changed); err != nil {
		return item, err
	}
	for _, name := range changed {
		item.After[name] = snapshot[name]
		if old != nil {
			item.Before[name] = old[name]
		}
	}
	return item, nil
}
