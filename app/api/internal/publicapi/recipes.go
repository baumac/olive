package publicapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/baumac/reviews-app/app/api/internal/subjects"
	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

const recipeNotFound = "The recipe does not exist."

var recipeSubject = newSubject(subject{
	Tables:       subjects.Recipes,
	noun:         "recipe",
	notFound:     recipeNotFound,
	search:       []scoreColumn{{"name", 10}, {"tags", 4}, {"ingredients", 2}, {"description", 1}},
	reviewSearch: []scoreColumn{{"title", 10}, {"recipe_name", 2}, {"feedback", 1}},
})

var recipeSorts = map[string]sortField{
	"name":       textSort("recipes", "name"),
	"overall":    recipeSubject.overallSort,
	"relevance":  relevanceSort,
	"total_time": totalTimeSort,
	"created":    columnSort("recipes", "created"),
	"updated":    columnSort("recipes", "updated"),
}

// totalTimeColumn is NULL for a recipe without a total time. The column
// stores 0.
const totalTimeColumn = "NULLIF([[recipes.total_time]], 0)"

// totalTimeSort puts the recipes without a total time last in both
// directions.
func totalTimeSort(desc bool) []sortKey {
	return []sortKey{
		{expr: "(" + column("recipes", "total_time") + " = 0)"},
		{expr: column("recipes", "total_time"), desc: desc},
	}
}

// ListRecipes uses keyset pagination like ListRestaurants.
func (s *Server) ListRecipes(_ context.Context, req reviewsapi.ListRecipesRequestObject) (reviewsapi.ListRecipesResponseObject, error) {
	p := req.Params
	bad := func(detail string) (reviewsapi.ListRecipesResponseObject, error) {
		return reviewsapi.ListRecipes400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(detail)}, nil
	}

	limit := limitOf(p.Limit)
	search := recipeSubject.searchPage()
	if detail := search.parse(p.Search); detail != "" {
		return bad(detail)
	}
	sortSpec, detail := search.sortWithSearch(p.Sort)
	if detail != "" {
		return bad(detail)
	}
	keys, sort, detail := parseSort(sortSpec, "recipes", recipeSorts)
	if detail != "" {
		return bad(detail)
	}
	if detail := overallDetail(p.Overall); detail != "" {
		return bad(detail)
	}
	p.Limit, p.Cursor, p.Sort = nil, nil, &sort
	query, err := fingerprint("recipes", p)
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
		w.joinOverall(recipeSubject)
	}
	w.compare(recipeSubject.overallColumn(), averageComparisons(p.Overall))
	w.compare(totalTimeColumn, totalTimeComparisons(p.TotalTime))
	w.tagFilters(recipeSubject, p.TagsHas, p.TagsHasAny, p.TagsHasAll)
	w.idFilters("recipes", "creator", p.CreatorEq, p.CreatorIn)
	w.compare(column("recipes", "created"), timestampComparisons(p.Created))
	w.compare(column("recipes", "updated"), timestampComparisons(p.Updated))
	if after != nil {
		w.keyset(keys, after)
	}

	records, next, err := s.page("recipes", w, keys, limit, query)
	if err != nil {
		return nil, err
	}
	items, err := s.recipesOf(records...)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ListRecipes200JSONResponse{Items: items, NextCursor: next}, nil
}

func totalTimeComparisons(r *reviewsapi.TotalTimeRange) []comparison {
	if r == nil {
		return nil
	}
	return comparisons(r.Lt, r.Lte, r.Eq, r.Gte, r.Gt)
}

// CreateRecipe sets the user of the token as the creator. The source check
// and the save run in one transaction.
func (s *Server) CreateRecipe(ctx context.Context, req reviewsapi.CreateRecipeRequestObject) (reviewsapi.CreateRecipeResponseObject, error) {
	user := stateFrom(ctx).user
	b := req.Body
	collection, err := s.app.FindCachedCollectionByNameOrId("recipes")
	if err != nil {
		return nil, err
	}
	r := core.NewRecord(collection)
	r.Set("name", b.Name)
	setOptionalText(r, "description", b.Description)
	if b.Tags != nil {
		r.Set("tags", *b.Tags)
	}
	r.Set("ingredients", b.Ingredients)
	r.Set("steps", b.Steps)
	setOptionalText(r, "source", b.Source)
	setOptionalNumber(r, "total_time", b.TotalTime)
	setOptionalNumber(r, "servings", b.Servings)
	r.Set("creator", user.Id)

	conflict, err := s.saveRecipe(r)
	if err != nil {
		return nil, err
	}
	if conflict != nil {
		return reviewsapi.CreateRecipe409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: *conflict}, nil
	}
	location := recipeSubject.path(r.Id)
	return reviewsapi.CreateRecipe201JSONResponse{
		Body:    toRecipe(r, map[string]*core.Record{user.Id: user}, nil, nil),
		Headers: reviewsapi.CreateRecipe201ResponseHeaders{Location: &location},
	}, nil
}

// saveRecipe returns the conflict of a source that another recipe has. It
// then does not save r. The unique index also blocks a second recipe with
// the source.
func (s *Server) saveRecipe(r *core.Record) (*reviewsapi.ConflictApplicationProblemPlusJSONResponse, error) {
	var duplicate string
	err := s.app.RunInTransaction(func(tx core.App) (err error) {
		if duplicate, err = recipeWithSource(tx, r); err != nil || duplicate != "" {
			return err
		}
		return tx.Save(r)
	})
	taken := isSourceTaken(err)
	if taken {
		duplicate, err = takenSource(s.app, r), nil
	}
	if err != nil {
		return nil, err
	}
	if duplicate == "" && !taken {
		return nil, nil
	}
	p := sourceConflict(duplicate)
	return &p, nil
}

// isSourceTaken is true for the error of the unique index of source_key. A
// recipe with the source then came after the check of recipeWithSource.
func isSourceTaken(err error) bool {
	var invalid validation.Errors
	return errors.As(err, &invalid) && isNotUnique(invalid, "source_key")
}

// takenSource returns the ID of the recipe with the source of r after a
// unique index error. It returns an empty string when the lookup fails. The
// other recipe can change its source again before the lookup.
func takenSource(app core.App, r *core.Record) string {
	id, _ := recipeWithSource(app, r)
	return id
}

// sourceConflict names the recipe with the source of the request. An empty
// recipeID gives a problem without a recipe.
func sourceConflict(recipeID string) reviewsapi.ConflictApplicationProblemPlusJSONResponse {
	p := propertyConflict(reviewsapi.ProblemCodeSourceTaken, "source", "Another recipe has this source.")
	if recipeID != "" {
		p.Recipe = &recipeID
	}
	return p
}

// sourceKey is the source without the case of the scheme and the host,
// without the fragment, and without a final slash. Two links to one page
// then get one key. The key is never longer than the source.
func sourceKey(source string) string {
	source, _, _ = strings.Cut(source, "#")
	if scheme, rest, ok := strings.Cut(source, "://"); ok {
		end := strings.IndexAny(rest, "/?")
		if end < 0 {
			end = len(rest)
		}
		source = asciiLower(scheme) + "://" + asciiLower(rest[:end]) + rest[end:]
	}
	return strings.TrimSuffix(source, "/")
}

// asciiLower changes only the letters A to Z. strings.ToLower can change a
// letter to two runes.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

// registerSourceKey sets source_key on each save of a recipe. The validate
// hook gives a duplicate the error of the unique index. The execute hooks
// also run for a save without validation.
func registerSourceKey(app core.App) {
	set := func(e *core.RecordEvent) error {
		e.Record.Set("source_key", sourceKey(e.Record.GetString("source")))
		return e.Next()
	}
	app.OnRecordValidate("recipes").BindFunc(set)
	app.OnRecordCreateExecute("recipes").BindFunc(set)
	app.OnRecordUpdateExecute("recipes").BindFunc(set)
}

// recipeWithSource returns the ID of a different recipe with the source key
// of r. It returns an empty string if no recipe has it.
func recipeWithSource(app core.App, r *core.Record) (string, error) {
	key := sourceKey(r.GetString("source"))
	if key == "" {
		return "", nil
	}
	var id string
	err := app.DB().Select("id").From("recipes").
		Where(dbx.HashExp{"source_key": key}).AndWhere(dbx.Not(dbx.HashExp{"id": r.Id})).
		Limit(1).Row(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (s *Server) GetRecipe(_ context.Context, req reviewsapi.GetRecipeRequestObject) (reviewsapi.GetRecipeResponseObject, error) {
	r, err := findParent(s.app, recipeSubject, req.RecipeId)
	if errors.Is(err, sql.ErrNoRows) {
		return reviewsapi.GetRecipe404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(recipeNotFound)}, nil
	}
	if err != nil {
		return nil, err
	}
	out, err := s.recipesOf(r)
	if err != nil {
		return nil, err
	}
	return reviewsapi.GetRecipe200JSONResponse(out[0]), nil
}

// UpdateRecipe runs the review check and the save in one transaction. The
// name, the ingredients, and the steps are what the reviews of other users
// scored. They then cannot change.
func (s *Server) UpdateRecipe(ctx context.Context, req reviewsapi.UpdateRecipeRequestObject) (reviewsapi.UpdateRecipeResponseObject, error) {
	user := stateFrom(ctx).user
	b := req.Body

	var response reviewsapi.UpdateRecipeResponseObject
	var saved, current *core.Record
	err := s.app.RunInTransaction(func(tx core.App) error {
		r, err := findParent(tx, recipeSubject, req.RecipeId)
		current = r
		if errors.Is(err, sql.ErrNoRows) {
			response = reviewsapi.UpdateRecipe404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(recipeNotFound)}
			return nil
		}
		if err != nil {
			return err
		}
		if detail, err := changeForbidden(tx, recipeSubject, r, user.Id); err != nil {
			return err
		} else if detail != "" {
			response = reviewsapi.UpdateRecipe403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: notCreator(detail)}
			return nil
		}

		ch := &changes{r: r}
		var frozen []string
		if b.Name != nil && *b.Name != r.GetString("name") {
			frozen = append(frozen, "name")
			ch.set("name", *b.Name)
		}
		for _, list := range []struct {
			field string
			v     *[]string
		}{{"ingredients", b.Ingredients}, {"steps", b.Steps}} {
			if list.v != nil && !slices.Equal(*list.v, listOf(r, list.field)) {
				frozen = append(frozen, list.field)
				r.Set(list.field, *list.v)
				ch.changed = true
			}
		}
		ch.patchText("description", b.Description)
		ch.patchTags(b.Tags)
		oldSource := r.GetString("source")
		ch.patchText("source", b.Source)
		ch.patchNumber("total_time", b.TotalTime)
		ch.patchNumber("servings", b.Servings)

		if len(frozen) > 0 {
			n, err := countReviewsOfOtherUsers(tx, recipeSubject, r.Id, user.Id)
			if err != nil {
				return err
			}
			if n > 0 {
				property := frozen[0]
				response = reviewsapi.UpdateRecipe409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: propertyConflict(
					reviewsapi.ProblemCodeOtherReviews, property, "The "+property+" cannot change because the recipe has reviews of other users.")}
				return nil
			}
		}
		if r.GetString("source") != oldSource {
			duplicate, err := recipeWithSource(tx, r)
			if err != nil {
				return err
			}
			if duplicate != "" {
				response = reviewsapi.UpdateRecipe409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: sourceConflict(duplicate)}
				return nil
			}
		}
		saved = r
		if !ch.changed {
			return nil
		}
		return tx.Save(r)
	})
	if isSourceTaken(err) {
		return reviewsapi.UpdateRecipe409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: sourceConflict(takenSource(s.app, current))}, nil
	}
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response, nil
	}
	out, err := s.recipesOf(saved)
	if err != nil {
		return nil, err
	}
	return reviewsapi.UpdateRecipe200JSONResponse(out[0]), nil
}

// DeleteRecipe also deletes the review of the user.
func (s *Server) DeleteRecipe(ctx context.Context, req reviewsapi.DeleteRecipeRequestObject) (reviewsapi.DeleteRecipeResponseObject, error) {
	p, err := s.deleteParent(recipeSubject, req.RecipeId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusForbidden:
			return reviewsapi.DeleteRecipe403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		case http.StatusNotFound:
			return reviewsapi.DeleteRecipe404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusConflict:
			return reviewsapi.DeleteRecipe409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: p.conflict()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.DeleteRecipe204Response{}, nil
}

type recipeRating struct {
	Recipe       string  `db:"recipe"`
	Count        int     `db:"count"`
	Overall      float64 `db:"overall"`
	Taste        float64 `db:"taste"`
	Ease         float64 `db:"ease"`
	Instructions float64 `db:"instructions"`
	Value        float64 `db:"value"`
}

func (r recipeRating) toAPI() reviewsapi.RecipeRating {
	return reviewsapi.RecipeRating{Overall: r.Overall, Taste: r.Taste, Ease: r.Ease, Instructions: r.Instructions, Value: r.Value}
}

// recipesOf gets the creators, the ratings, and the covers of records.
func (s *Server) recipesOf(records ...*core.Record) ([]reviewsapi.Recipe, error) {
	users, err := s.creators(records...)
	if err != nil {
		return nil, err
	}
	ratings, err := ratingsOf(s, recipeSubject, records, func(r recipeRating) string { return r.Recipe })
	if err != nil {
		return nil, err
	}
	rows, err := s.coverRows(recipeSubject, records...)
	if err != nil {
		return nil, err
	}
	covers := make(map[string]reviewsapi.RecipePhoto, len(rows))
	for id, r := range rows {
		covers[id] = r.recipePhoto()
	}
	items := make([]reviewsapi.Recipe, len(records))
	for i, r := range records {
		items[i] = toRecipe(r, users, ratings, covers)
	}
	return items, nil
}

// toRecipe gives a recipe that ratings does not list no reviews.
func toRecipe(r *core.Record, users map[string]*core.Record, ratings map[string]recipeRating, covers map[string]reviewsapi.RecipePhoto) reviewsapi.Recipe {
	out := reviewsapi.Recipe{
		Id:          r.Id,
		Name:        r.GetString("name"),
		Description: optionalText(r, "description"),
		Tags:        tagsOf(r),
		Ingredients: listOf(r, "ingredients"),
		Steps:       listOf(r, "steps"),
		Source:      optionalText(r, "source"),
		TotalTime:   optionalNumber(r, "total_time"),
		Servings:    optionalNumber(r, "servings"),
		Creator:     nullable.NewNullNullable[reviewsapi.Creator](),
		Rating:      nullable.NewNullNullable[reviewsapi.RecipeRating](),
		Cover:       nullable.NewNullNullable[reviewsapi.RecipePhoto](),
		Created:     apiTimestamp(r.GetDateTime("created")),
		Updated:     apiTimestamp(r.GetDateTime("updated")),
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

// listOf returns an empty array for a field that is not a JSON array of
// strings. The spec requires the array.
func listOf(r *core.Record, field string) []string {
	var items []string
	if err := r.UnmarshalJSONField(field, &items); err != nil || items == nil {
		return []string{}
	}
	return items
}

// optionalNumber returns nil for the stored 0 of a number that is not set.
func optionalNumber(r *core.Record, field string) *int {
	if v := r.GetInt(field); v != 0 {
		return &v
	}
	return nil
}

func setOptionalNumber(r *core.Record, field string, v *int) {
	if v != nil {
		r.Set(field, float64(*v))
	}
}
