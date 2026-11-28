package publicapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/baumac/olive/app/api/internal/subjects"
	"github.com/baumac/olive/app/api/internal/tagname"
	reviewsapi "github.com/baumac/olive/specs/api/sdk/server/go"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const tagNotFound = "The tag does not exist."

// maxTags is the maxItems of the Tags schema and the MaxSelect of the tags
// relation.
const maxTags = 20

// tooManyTags gets the 422 problem of more than maxTags unique tags. Only a
// request without the spec validation gets it. The relation keeps the limit.
func tooManyTags(tags []string) *reviewsapi.UnprocessableContentApplicationProblemPlusJSONResponse {
	if len(tagname.List(tags)) <= maxTags {
		return nil
	}
	var errs fieldErrors
	errs.add("tags", fmt.Sprintf("must have at most %d items", maxTags))
	p := unprocessable(errs)
	return &p
}

// patchedTags gets the tags of a patch. Null and a missing value give nil.
func patchedTags(v nullable.Nullable[reviewsapi.Tags]) []string {
	tags, _ := v.Get()
	return tags
}

// createdTags gets the tags of a create body.
func createdTags(v *reviewsapi.Tags) []string {
	if v == nil {
		return nil
	}
	return *v
}

// ListTags lists only the tags with at least one parent of the kinds. A tag
// with the exact name of the search comes first. A tag with the same key
// comes next. These two need a parent of any kind.
func (s *Server) ListTags(_ context.Context, req reviewsapi.ListTagsRequestObject) (reviewsapi.ListTagsResponseObject, error) {
	p := req.Params
	bad := func(detail string) (reviewsapi.ListTagsResponseObject, error) {
		return reviewsapi.ListTags400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(detail)}, nil
	}
	count := tagCount(tagKinds(p.Kind))
	sorts := map[string]sortField{
		"count": func(desc bool) []sortKey {
			return []sortKey{{expr: count, desc: desc}, {expr: column(subjects.TagCollection, "name")}}
		},
		"name": columnSort(subjects.TagCollection, "name"),
	}
	sort := string(reviewsapi.ListTagsParamsSortMinusCount)
	if p.Sort != nil {
		sort = string(*p.Sort)
	}
	keys, _, detail := parseSort(&sort, subjects.TagCollection, sorts)
	if detail != "" {
		return bad(detail)
	}
	w := newWhere()
	inUse := tagInUse(tagKinds(p.Kind))
	var search, isMatch string
	if p.Search != nil {
		search = tagname.SearchKey(*p.Search)
		if exact := tagname.Of(*p.Search); exact != "" {
			// One key ranks the exact name 2 and the same key 1. A cursor then
			// has as many keys as before the key match.
			isExact := "(" + column(subjects.TagCollection, "name") + " = " + w.param(exact) + ")"
			isKey := "(" + column(subjects.TagCollection, "name_key") + " = " + w.param(tagname.Key(exact)) + ")"
			keys = append([]sortKey{{expr: "(" + isExact + " * 2 + " + isKey + ")", desc: true}}, keys...)
			isMatch = "(" + isExact + " OR " + isKey + ")"
			inUse = "(" + inUse + " OR (" + isMatch + " AND " + tagInUse(tagKinds(nil)) + "))"
		}
	}
	limit := limitOf(p.Limit)
	cursor := p.Cursor
	p.Limit, p.Cursor = nil, nil
	query, err := fingerprint(subjects.TagCollection, p, sort)
	if err != nil {
		return nil, err
	}
	after, detail := decodeCursor(cursor, query, len(keys))
	if detail != "" {
		return bad(detail)
	}

	if p.Search != nil && search == "" {
		return reviewsapi.ListTags200JSONResponse{Items: []reviewsapi.TagSummary{}}, nil
	}
	if search != "" {
		like := escapeLike(search)
		nameKey := column(subjects.TagCollection, "name_key")
		// The exact match can miss the prefix. Its last σ became ς.
		w.add("(" + isMatch + " OR " + nameKey + " LIKE " + w.param(like+"%") + ` ESCAPE '\' OR ` + nameKey + " LIKE " + w.param("%-"+like+"%") + ` ESCAPE '\')`)
	}
	w.add(inUse)
	if after != nil {
		w.keyset(keys, after)
	}
	records, next, err := s.page(subjects.TagCollection, w, keys, limit, query)
	if err != nil {
		return nil, err
	}
	items, err := s.tagSummaries(records)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ListTags200JSONResponse{Items: items, NextCursor: next}, nil
}

func (s *Server) GetTag(_ context.Context, req reviewsapi.GetTagRequestObject) (reviewsapi.GetTagResponseObject, error) {
	notFoundResponse := reviewsapi.GetTag404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(tagNotFound)}
	r, err := s.app.FindFirstRecordByData(subjects.TagCollection, "name", req.TagName)
	if errors.Is(err, sql.ErrNoRows) {
		return notFoundResponse, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := s.tagSummaries([]*core.Record{r})
	if err != nil {
		return nil, err
	}
	if items[0].RestaurantCount == 0 && items[0].RecipeCount == 0 && items[0].PantryItemCount == 0 {
		return notFoundResponse, nil
	}
	return reviewsapi.GetTag200JSONResponse(items[0]), nil
}

// tagKindSubjects has the subject of each value of the kind parameter.
var tagKindSubjects = map[reviewsapi.ListTagsParamsKind]subject{
	reviewsapi.ListTagsParamsKindRestaurants: restaurantSubject,
	reviewsapi.ListTagsParamsKindRecipes:     recipeSubject,
	reviewsapi.ListTagsParamsKindPantry:      pantryItemSubject,
}

// tagKinds returns the subjects of kind. A nil or unknown kind gets all
// subjects. Only a request without the spec validation has an unknown kind.
func tagKinds(kind *reviewsapi.ListTagsParamsKind) []subject {
	if kind != nil {
		if sub, ok := tagKindSubjects[*kind]; ok {
			return []subject{sub}
		}
	}
	return []subject{restaurantSubject, recipeSubject, pantryItemSubject}
}

// tagCount is the SQL of the number of parents of the kinds with a tag of
// the tags table.
func tagCount(kinds []subject) string {
	parts := make([]string, len(kinds))
	for i, k := range kinds {
		parts[i] = tagCountOf(k)
	}
	return "(" + strings.Join(parts, " + ") + ")"
}

// tagInUse is the SQL condition that a parent of the kinds has the tag. An
// EXISTS stops at the first row of the tag table.
func tagInUse(kinds []subject) string {
	parts := make([]string, len(kinds))
	for i, k := range kinds {
		parts[i] = "EXISTS (SELECT 1 FROM {{" + k.Tags + "}} WHERE [[tag]] = " + column(subjects.TagCollection, "id") + ")"
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

func tagCountOf(k subject) string {
	return "(SELECT COUNT(*) FROM {{" + k.Tags + "}} WHERE [[tag]] = " + column(subjects.TagCollection, "id") + ")"
}

// tagSummaries keeps the order of records.
func (s *Server) tagSummaries(records []*core.Record) ([]reviewsapi.TagSummary, error) {
	items := make([]reviewsapi.TagSummary, len(records))
	if len(records) == 0 {
		return items, nil
	}
	ids := make([]any, len(records))
	for i, r := range records {
		ids[i] = r.Id
	}
	var rows []struct {
		ID          string `db:"id"`
		Restaurants int    `db:"restaurants"`
		Recipes     int    `db:"recipes"`
		PantryItems int    `db:"pantry_items"`
	}
	err := s.app.DB().Select("id", tagCountOf(restaurantSubject)+" AS [[restaurants]]", tagCountOf(recipeSubject)+" AS [[recipes]]",
		tagCountOf(pantryItemSubject)+" AS [[pantry_items]]").
		From(subjects.TagCollection).Where(dbx.In("id", ids...)).All(&rows)
	if err != nil {
		return nil, err
	}
	for i, r := range records {
		items[i] = reviewsapi.TagSummary{Name: r.GetString("name"), Created: apiTimestamp(r.GetDateTime("created"))}
		for _, row := range rows {
			if row.ID == r.Id {
				items[i].RestaurantCount, items[i].RecipeCount, items[i].PantryItemCount = row.Restaurants, row.Recipes, row.PantryItems
			}
		}
	}
	return items, nil
}

// resolveTags gets the IDs of the tags of names in their order. It makes a
// tag for each missing name. The spec validation already checked the names.
func resolveTags(app core.App, names []string) ([]string, error) {
	names = tagname.List(names)
	values := make([]any, len(names))
	for i, n := range names {
		values[i] = n
	}
	var rows []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	if err := app.DB().Select("id", "name").From(subjects.TagCollection).Where(dbx.In("name", values...)).All(&rows); err != nil {
		return nil, err
	}
	byName := make(map[string]string, len(rows))
	for _, row := range rows {
		byName[row.Name] = row.ID
	}
	ids := make([]string, len(names))
	for i, name := range names {
		id, ok := byName[name]
		if !ok {
			c, err := app.FindCachedCollectionByNameOrId(subjects.TagCollection)
			if err != nil {
				return nil, err
			}
			r := core.NewRecord(c)
			r.Set("name", name)
			if err := app.SaveNoValidate(r); err != nil {
				return nil, err
			}
			id = r.Id
			byName[name] = id
		}
		ids[i] = id
	}
	return ids, nil
}

// registerTags adds the record hooks of decision 00011.
func registerTags(app core.App) {
	setKey := func(e *core.RecordEvent) error {
		name := tagname.Of(e.Record.GetString("name"))
		e.Record.Set("name", name)
		e.Record.Set("name_key", tagname.Key(name))
		return e.Next()
	}
	app.OnRecordValidate(subjects.TagCollection).BindFunc(setKey)
	app.OnRecordCreateExecute(subjects.TagCollection).BindFunc(setKey)
	app.OnRecordUpdateExecute(subjects.TagCollection).BindFunc(setKey)
	tagGuard.register(app)

	parents := make([]string, len(subjects.All))
	for i, k := range subjects.All {
		parents[i] = k.Table
	}
	app.OnRecordUpdateExecute(parents...).BindFunc(func(e *core.RecordEvent) error {
		old := e.Record.Original().GetStringSlice("tags")
		return removeUnusedAfter(e, old, func() []string { return e.Record.GetStringSlice("tags") })
	})
	app.OnRecordDeleteExecute(parents...).BindFunc(func(e *core.RecordEvent) error {
		return removeUnusedAfter(e, e.Record.GetStringSlice("tags"), func() []string { return nil })
	})
}

// removeUnusedAfter runs the save or the delete of e. In the same
// transaction, it deletes the tags of old that no parent has now.
func removeUnusedAfter(e *core.RecordEvent, old []string, kept func() []string) error {
	app := e.App
	defer func() { e.App = app }()
	return app.RunInTransaction(func(tx core.App) error {
		e.App = tx
		if err := e.Next(); err != nil {
			return err
		}
		now := kept()
		var removed []any
		for _, id := range old {
			if !slices.Contains(now, id) {
				removed = append(removed, id)
			}
		}
		if len(removed) == 0 {
			return nil
		}
		unused := make([]string, len(subjects.All))
		for i, k := range subjects.All {
			unused[i] = "NOT EXISTS (SELECT 1 FROM {{" + k.Tags + "}} WHERE [[tag]] = " + column(subjects.TagCollection, "id") + ")"
		}
		// A delete of a tag in the dashboard deletes the row before it saves
		// the parents. This delete then finds no row of that tag.
		_, err := tx.DB().Delete(subjects.TagCollection, dbx.And(dbx.In("id", removed...), dbx.NewExp(strings.Join(unused, " AND ")))).Execute()
		return err
	})
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

var tagGuard = fieldGuard{
	collection: subjects.TagCollection,
	noun:       "tag",
	schema:     tagRecordSchema,
	properties: func(r *core.Record, _ *openapi3.Schema) (map[string]json.RawMessage, error) {
		name, err := json.Marshal(r.GetString("name"))
		if err != nil {
			return nil, err
		}
		return map[string]json.RawMessage{"name": name}, nil
	},
	set: setText,
}

// tagRecordSchema has the Tag schema of the spec as the name property.
var tagRecordSchema = sync.OnceValues(func() (*openapi3.Schema, error) {
	doc, err := embeddedSpec()
	if err != nil {
		return nil, err
	}
	ref := doc.Components.Schemas["Tag"]
	if ref == nil || ref.Value == nil {
		return nil, errors.New("the spec has no Tag schema")
	}
	return &openapi3.Schema{Type: &openapi3.Types{openapi3.TypeObject}, Properties: openapi3.Schemas{"name": ref}}, nil
})
