package publicapi

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The spec has the limits in the descriptions of createList and
// addListRestaurant.
const (
	maxLists     = 50
	maxListItems = 500
)

const (
	kindFavorites = "favorites"
	kindCustom    = "custom"
	favoritesName = "Favorites"
	// favoritesRef is the listRef of the favorites list of the user.
	favoritesRef = "favorites"
)

const listNotFound = "The list does not exist."

// registerFavorites gives each new user a favorites list. The hook runs in
// the transaction of the user save. It also runs for a dashboard user.
func registerFavorites(app core.App) {
	app.OnRecordCreateExecute("users").BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		_, err := saveList(e.App, e.Record.Id, favoritesName, kindFavorites)
		return err
	})
}

func saveList(app core.App, creator, name, kind string) (*core.Record, error) {
	lists, err := app.FindCachedCollectionByNameOrId("lists")
	if err != nil {
		return nil, err
	}
	list := core.NewRecord(lists)
	list.Set("creator", creator)
	list.Set("name", name)
	list.Set("kind", kind)
	return list, app.Save(list)
}

// favoritesOf makes the favorites list of a user that has none. A user
// from SQL has none.
func favoritesOf(app core.App, userID string) (*core.Record, error) {
	list, err := app.FindFirstRecordByFilter("lists", "creator = {:creator} && kind = {:kind}",
		dbx.Params{"creator": userID, "kind": kindFavorites})
	if errors.Is(err, sql.ErrNoRows) {
		return saveList(app, userID, favoritesName, kindFavorites)
	}
	return list, err
}

// findList returns a nil record and no error when the list does not exist.
func findList(app core.App, id string) (*core.Record, error) {
	list, err := app.FindRecordById("lists", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return list, err
}

// listNameTaken ignores the case of ASCII letters, the same as the unique
// index of the names.
func listNameTaken(app core.App, creator, name, exceptID string) (bool, error) {
	var id string
	err := app.DB().Select("id").From("lists").
		Where(dbx.NewExp("[[creator]] = {:creator} AND [[name]] = {:name} COLLATE NOCASE AND [[id]] != {:id}",
			dbx.Params{"creator": creator, "name": name, "id": exceptID})).
		Limit(1).Row(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func nameTaken() reviewsapi.ConflictApplicationProblemPlusJSONResponse {
	return propertyConflict(reviewsapi.ProblemCodeListNameTaken, "name", "Another list of the user has this name.")
}

// itemCounts gives no entry for a list without items.
func (s *Server) itemCounts(sub subject, lists ...*core.Record) (map[string]int, error) {
	out := map[string]int{}
	if len(lists) == 0 {
		return out, nil
	}
	ids := make([]any, len(lists))
	for i, l := range lists {
		ids[i] = l.Id
	}
	var rows []struct {
		List  string `db:"list"`
		Count int    `db:"count"`
	}
	err := s.app.DB().Select("list", "COUNT(*) AS [[count]]").From(sub.ListItems).
		Where(dbx.In("list", ids...)).GroupBy("list").All(&rows)
	for _, r := range rows {
		out[r.List] = r.Count
	}
	return out, err
}

// listsWith returns the lists of ids that have the parent.
func (s *Server) listsWith(sub subject, parentID string, lists []*core.Record) (map[string]bool, error) {
	out := map[string]bool{}
	if len(lists) == 0 {
		return out, nil
	}
	ids := make([]any, len(lists))
	for i, l := range lists {
		ids[i] = l.Id
	}
	var found []string
	err := s.app.DB().Select("list").From(sub.ListItems).
		Where(dbx.HashExp{sub.Parent: parentID}).AndWhere(dbx.In("list", ids...)).Column(&found)
	for _, id := range found {
		out[id] = true
	}
	return out, err
}

// favoriteIDs returns the parents of records that the favorites list of the
// user has. A nil user has none.
func (s *Server) favoriteIDs(sub subject, user *core.Record, records ...*core.Record) (map[string]bool, error) {
	out := map[string]bool{}
	if user == nil || len(records) == 0 {
		return out, nil
	}
	var found []string
	err := s.favoritesQuery(sub, user.Id, records...).Column(&found)
	for _, id := range found {
		out[id] = true
	}
	return out, err
}

func (s *Server) favoritesQuery(sub subject, userID string, records ...*core.Record) *dbx.SelectQuery {
	ids := make([]any, len(records))
	for i, r := range records {
		ids[i] = r.Id
	}
	parent := sub.ListItems + "." + sub.Parent
	return s.app.DB().Select(parent).From("lists").
		InnerJoin(sub.ListItems, dbx.NewExp(column(sub.ListItems, "list")+" = [[lists.id]]")).
		Where(dbx.HashExp{"lists.creator": userID, "lists.kind": kindFavorites}).
		AndWhere(dbx.In(parent, ids...))
}

func toSavedList(list, creator *core.Record, count int) reviewsapi.SavedList {
	return reviewsapi.SavedList{
		Id: list.Id, Name: list.GetString("name"), Kind: reviewsapi.SavedListKind(list.GetString("kind")),
		Creator: toCreator(creator), ItemCount: count,
		Created: apiTimestamp(list.GetDateTime("created")), Updated: apiTimestamp(list.GetDateTime("updated")),
	}
}

// publicList returns the list and its creator. It returns a nil list when
// the list does not exist or viewer cannot see its creator.
func (s *Server) publicList(viewer *core.Record, id string) (list, creator *core.Record, err error) {
	list, err = findList(s.app, id)
	if err != nil || list == nil {
		return nil, nil, err
	}
	creator, err = s.findReviewer(viewer, list.GetString("creator"))
	if err != nil || creator == nil {
		return nil, nil, err
	}
	return list, creator, nil
}

// ListReviewerLists puts the favorites list first. The other lists follow,
// the newest first.
func (s *Server) ListReviewerLists(ctx context.Context, req reviewsapi.ListReviewerListsRequestObject) (reviewsapi.ListReviewerListsResponseObject, error) {
	user, err := s.findReviewer(stateFrom(ctx).user, req.ReviewerId)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return reviewsapi.ListReviewerLists404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(reviewerNotFound)}, nil
	}
	lists, err := s.app.FindAllRecords("lists", dbx.HashExp{"creator": user.Id})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(lists, func(a, b *core.Record) int {
		return cmp.Or(
			-cmp.Compare(boolInt(a.GetString("kind") == kindFavorites), boolInt(b.GetString("kind") == kindFavorites)),
			-cmp.Compare(a.GetString("created"), b.GetString("created")),
			-cmp.Compare(a.Id, b.Id),
		)
	})
	counts, err := s.itemCounts(restaurantSubject, lists...)
	if err != nil {
		return nil, err
	}
	var with map[string]bool
	if req.Params.RestaurantEq != nil {
		if with, err = s.listsWith(restaurantSubject, *req.Params.RestaurantEq, lists); err != nil {
			return nil, err
		}
	}
	items := make([]reviewsapi.SavedList, len(lists))
	for i, l := range lists {
		items[i] = toSavedList(l, user, counts[l.Id])
		if with != nil {
			has := with[l.Id]
			items[i].HasRestaurant = &has
		}
	}
	return reviewsapi.ListReviewerLists200JSONResponse{Items: items}, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// CreateList runs the limit check, the name check, and the save in one
// transaction. It saves the name without the spaces at its start and end.
func (s *Server) CreateList(ctx context.Context, req reviewsapi.CreateListRequestObject) (reviewsapi.CreateListResponseObject, error) {
	user := stateFrom(ctx).user
	name := req.Body.Name
	var problem *reviewsapi.ConflictApplicationProblemPlusJSONResponse
	var list *core.Record
	err := s.app.RunInTransaction(func(tx core.App) error {
		n, err := tx.CountRecords("lists", dbx.HashExp{"creator": user.Id})
		if err != nil {
			return err
		}
		if n >= maxLists {
			p := conflict(reviewsapi.ProblemCodeListLimit, "The user has the maximum number of lists.")
			problem = &p
			return nil
		}
		taken, err := listNameTaken(tx, user.Id, name, "")
		if err != nil {
			return err
		}
		if taken {
			p := nameTaken()
			problem = &p
			return nil
		}
		list, err = saveList(tx, user.Id, name, kindCustom)
		return err
	})
	if err != nil {
		return nil, err
	}
	if problem != nil {
		return reviewsapi.CreateList409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: *problem}, nil
	}
	location := basePath + "/lists/" + list.Id
	return reviewsapi.CreateList201JSONResponse{
		Body:    toSavedList(list, user, 0),
		Headers: reviewsapi.CreateList201ResponseHeaders{Location: &location},
	}, nil
}

func (s *Server) GetList(ctx context.Context, req reviewsapi.GetListRequestObject) (reviewsapi.GetListResponseObject, error) {
	list, creator, err := s.publicList(stateFrom(ctx).user, req.ListId)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return reviewsapi.GetList404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(listNotFound)}, nil
	}
	counts, err := s.itemCounts(restaurantSubject, list)
	if err != nil {
		return nil, err
	}
	return reviewsapi.GetList200JSONResponse(toSavedList(list, creator, counts[list.Id])), nil
}

// listChangeForbidden returns the 403 problem if the user cannot rename or
// delete the list.
func listChangeForbidden(list *core.Record, userID string) *reviewsapi.ForbiddenApplicationProblemPlusJSONResponse {
	if list.GetString("creator") != userID {
		p := notCreator("Only the creator of the list can change or delete it.")
		return &p
	}
	if list.GetString("kind") == kindFavorites {
		p := forbidden(reviewsapi.ProblemCodeFavoritesList, "The favorites list cannot be renamed or deleted.")
		return &p
	}
	return nil
}

func (s *Server) UpdateList(ctx context.Context, req reviewsapi.UpdateListRequestObject) (reviewsapi.UpdateListResponseObject, error) {
	user := stateFrom(ctx).user
	var response reviewsapi.UpdateListResponseObject
	var saved *core.Record
	err := s.app.RunInTransaction(func(tx core.App) error {
		list, err := findList(tx, req.ListId)
		if err != nil {
			return err
		}
		if list == nil {
			response = reviewsapi.UpdateList404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(listNotFound)}
			return nil
		}
		if p := listChangeForbidden(list, user.Id); p != nil {
			response = reviewsapi.UpdateList403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: *p}
			return nil
		}
		saved = list
		if req.Body.Name == nil {
			return nil
		}
		name := *req.Body.Name
		if name == list.GetString("name") {
			return nil
		}
		taken, err := listNameTaken(tx, user.Id, name, list.Id)
		if err != nil {
			return err
		}
		if taken {
			response = reviewsapi.UpdateList409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: nameTaken()}
			return nil
		}
		list.Set("name", name)
		return tx.Save(list)
	})
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response, nil
	}
	counts, err := s.itemCounts(restaurantSubject, saved)
	if err != nil {
		return nil, err
	}
	return reviewsapi.UpdateList200JSONResponse(toSavedList(saved, user, counts[saved.Id])), nil
}

// DeleteList also deletes the items of the list. The restaurants stay.
func (s *Server) DeleteList(ctx context.Context, req reviewsapi.DeleteListRequestObject) (reviewsapi.DeleteListResponseObject, error) {
	user := stateFrom(ctx).user
	var response reviewsapi.DeleteListResponseObject = reviewsapi.DeleteList204Response{}
	err := s.app.RunInTransaction(func(tx core.App) error {
		list, err := findList(tx, req.ListId)
		if err != nil {
			return err
		}
		if list == nil {
			response = reviewsapi.DeleteList404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(listNotFound)}
			return nil
		}
		if p := listChangeForbidden(list, user.Id); p != nil {
			response = reviewsapi.DeleteList403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: *p}
			return nil
		}
		return tx.Delete(list)
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

// makeListItemKeys puts the item that was added last first. The parent is
// unique in a list. Its ID is the last sort value.
func makeListItemKeys(sub subject) []sortKey {
	return []sortKey{
		{expr: column(sub.ListItems, "created"), desc: true},
		{expr: column(sub.ListItems, sub.Parent), desc: true, unique: true},
	}
}

func (s *Server) ListListRestaurants(ctx context.Context, req reviewsapi.ListListRestaurantsRequestObject) (reviewsapi.ListListRestaurantsResponseObject, error) {
	records, next, p, err := s.listItemPage(ctx, restaurantSubject, req.ListId, req.Params.Cursor, req.Params.Limit)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusBadRequest:
			return reviewsapi.ListListRestaurants400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: p.badRequest()}, nil
		case http.StatusNotFound:
			return reviewsapi.ListListRestaurants404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	items, err := s.restaurantsOf(stateFrom(ctx).user, records...)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ListListRestaurants200JSONResponse{Items: items, NextCursor: next}, nil
}

// listItemPage gets one page of the parents of a list. The problem is a 400
// or a 404.
func (s *Server) listItemPage(ctx context.Context, sub subject, listID string, cursor *string, limit *int) ([]*core.Record, nullable.Nullable[string], *problem, error) {
	keys := sub.listItemKeys
	query, err := fingerprint("list-"+sub.Table, listID)
	if err != nil {
		return nil, nil, nil, err
	}
	after, detail := decodeCursor(cursor, query, len(keys))
	if detail != "" {
		return nil, nil, badRequestProblem(detail), nil
	}
	list, _, err := s.publicList(stateFrom(ctx).user, listID)
	if err != nil {
		return nil, nil, nil, err
	}
	if list == nil {
		return nil, nil, notFoundProblem(listNotFound), nil
	}
	w := listItemsWhere(sub, list.Id)
	if after != nil {
		w.keyset(keys, after)
	}
	ids, next, err := s.pageIDs(sub.ListItems, w, keys, limitOf(limit), query)
	if err != nil {
		return nil, nil, nil, err
	}
	records, err := s.recordsInOrder(sub.Table, ids)
	return records, next, nil, err
}

// inList keeps the parents of the list. The join lets SQLite start from the
// unique index on the list and the parent for a small list.
func (w *where) inList(sub subject, listID *string) {
	if listID == nil {
		return
	}
	w.joins = append(w.joins, join{
		inner: true,
		table: "{{" + sub.ListItems + "}}",
		on: dbx.NewExp(column(sub.ListItems, sub.Parent) + " = " + column(sub.Table, "id") + " AND " +
			column(sub.ListItems, "list") + " = " + w.param(*listID)),
	})
}

func listItemsWhere(sub subject, listID string) *where {
	w := newWhere()
	w.compare(column(sub.ListItems, "list"), []comparison{{"=", listID}})
	return w
}

// ownList gets the list of ref for a change of its items. Only the creator
// of the list can change them. The problem is a 403 or a 404.
func ownList(app core.App, sub subject, ref string, user *core.Record) (*core.Record, *problem, error) {
	if ref == favoritesRef {
		list, err := favoritesOf(app, user.Id)
		return list, nil, err
	}
	list, err := findList(app, ref)
	if err != nil {
		return nil, nil, err
	}
	if list == nil {
		return nil, notFoundProblem(listNotFound), nil
	}
	if list.GetString("creator") != user.Id {
		return nil, notCreatorProblem("Only the creator of the list can change its " + sub.Table + "."), nil
	}
	return list, nil, nil
}

func findItem(app core.App, sub subject, listID, parentID string) (*core.Record, error) {
	item, err := app.FindFirstRecordByFilter(sub.ListItems, "list = {:list} && "+sub.Parent+" = {:parent}",
		dbx.Params{"list": listID, "parent": parentID})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

// AddListRestaurant gives 204 also for a restaurant that the list has.
func (s *Server) AddListRestaurant(ctx context.Context, req reviewsapi.AddListRestaurantRequestObject) (reviewsapi.AddListRestaurantResponseObject, error) {
	p, err := s.addListItem(restaurantSubject, req.ListRef, req.RestaurantId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusForbidden:
			return reviewsapi.AddListRestaurant403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		case http.StatusNotFound:
			return reviewsapi.AddListRestaurant404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		case http.StatusConflict:
			return reviewsapi.AddListRestaurant409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: p.conflict()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.AddListRestaurant204Response{}, nil
}

// addListItem runs the checks and the save in one transaction. The save of
// the list sets its updated time. The problem is a 403, a 404, or a 409.
func (s *Server) addListItem(sub subject, ref, parentID string, user *core.Record) (p *problem, err error) {
	err = s.app.RunInTransaction(func(tx core.App) error {
		var list *core.Record
		var err error
		if list, p, err = ownList(tx, sub, ref, user); p != nil || err != nil {
			return err
		}
		if _, err := findParent(tx, sub, parentID); errors.Is(err, sql.ErrNoRows) {
			p = notFoundProblem(sub.notFound)
			return nil
		} else if err != nil {
			return err
		}
		item, err := findItem(tx, sub, list.Id, parentID)
		if err != nil || item != nil {
			return err
		}
		n, err := tx.CountRecords(sub.ListItems, dbx.HashExp{"list": list.Id})
		if err != nil {
			return err
		}
		if n >= maxListItems {
			p = conflictProblem(sub.listLimitCode, "The list has the maximum number of "+sub.Table+".")
			return nil
		}
		items, err := tx.FindCachedCollectionByNameOrId(sub.ListItems)
		if err != nil {
			return err
		}
		item = core.NewRecord(items)
		item.Set("list", list.Id)
		item.Set(sub.Parent, parentID)
		if err := tx.Save(item); err != nil {
			return err
		}
		return tx.Save(list)
	})
	return p, err
}

func (s *Server) RemoveListRestaurant(ctx context.Context, req reviewsapi.RemoveListRestaurantRequestObject) (reviewsapi.RemoveListRestaurantResponseObject, error) {
	p, err := s.removeListItem(restaurantSubject, req.ListRef, req.RestaurantId, stateFrom(ctx).user)
	if err != nil {
		return nil, err
	}
	if p != nil {
		switch p.status() {
		case http.StatusForbidden:
			return reviewsapi.RemoveListRestaurant403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: p.forbidden()}, nil
		case http.StatusNotFound:
			return reviewsapi.RemoveListRestaurant404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: p.notFound()}, nil
		}
		return nil, p.unexpected()
	}
	return reviewsapi.RemoveListRestaurant204Response{}, nil
}

// removeListItem gives no problem for a parent that the list does not have.
// The problem is a 403 or a 404.
func (s *Server) removeListItem(sub subject, ref, parentID string, user *core.Record) (p *problem, err error) {
	err = s.app.RunInTransaction(func(tx core.App) error {
		var list *core.Record
		var err error
		if list, p, err = ownList(tx, sub, ref, user); p != nil || err != nil {
			return err
		}
		item, err := findItem(tx, sub, list.Id, parentID)
		if err != nil || item == nil {
			return err
		}
		if err := tx.Delete(item); err != nil {
			return err
		}
		return tx.Save(list)
	})
	return p, err
}
