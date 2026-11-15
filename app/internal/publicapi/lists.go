package publicapi

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
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
	return propertyConflict("name", "Another list of the user has this name.")
}

// itemCounts gives no entry for a list without restaurants.
func (s *Server) itemCounts(lists ...*core.Record) (map[string]int, error) {
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
	err := s.app.DB().Select("list", "COUNT(*) AS [[count]]").From("list_items").
		Where(dbx.In("list", ids...)).GroupBy("list").All(&rows)
	for _, r := range rows {
		out[r.List] = r.Count
	}
	return out, err
}

// listsWith returns the lists of ids that have the restaurant.
func (s *Server) listsWith(restaurantID string, lists []*core.Record) (map[string]bool, error) {
	out := map[string]bool{}
	if len(lists) == 0 {
		return out, nil
	}
	ids := make([]any, len(lists))
	for i, l := range lists {
		ids[i] = l.Id
	}
	var found []string
	err := s.app.DB().Select("list").From("list_items").
		Where(dbx.HashExp{"restaurant": restaurantID}).AndWhere(dbx.In("list", ids...)).Column(&found)
	for _, id := range found {
		out[id] = true
	}
	return out, err
}

// favoriteIDs returns the restaurants of records that the favorites list
// of the user has. A nil user has none.
func (s *Server) favoriteIDs(user *core.Record, records ...*core.Record) (map[string]bool, error) {
	out := map[string]bool{}
	if user == nil || len(records) == 0 {
		return out, nil
	}
	var found []string
	err := s.favoritesQuery(user.Id, records...).Column(&found)
	for _, id := range found {
		out[id] = true
	}
	return out, err
}

func (s *Server) favoritesQuery(userID string, records ...*core.Record) *dbx.SelectQuery {
	ids := make([]any, len(records))
	for i, r := range records {
		ids[i] = r.Id
	}
	return s.app.DB().Select("list_items.restaurant").From("lists").
		InnerJoin("list_items", dbx.NewExp("[[list_items.list]] = [[lists.id]]")).
		Where(dbx.HashExp{"lists.creator": userID, "lists.kind": kindFavorites}).
		AndWhere(dbx.In("list_items.restaurant", ids...))
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
	counts, err := s.itemCounts(lists...)
	if err != nil {
		return nil, err
	}
	var with map[string]bool
	if req.Params.RestaurantEq != nil {
		if with, err = s.listsWith(*req.Params.RestaurantEq, lists); err != nil {
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
	name := strings.TrimSpace(req.Body.Name)
	var problem *reviewsapi.ConflictApplicationProblemPlusJSONResponse
	var list *core.Record
	err := s.app.RunInTransaction(func(tx core.App) error {
		n, err := tx.CountRecords("lists", dbx.HashExp{"creator": user.Id})
		if err != nil {
			return err
		}
		if n >= maxLists {
			p := conflict("The user has the maximum number of lists.")
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
	counts, err := s.itemCounts(list)
	if err != nil {
		return nil, err
	}
	return reviewsapi.GetList200JSONResponse(toSavedList(list, creator, counts[list.Id])), nil
}

// listChangeForbidden returns a detail for the 403 response if the user
// cannot rename or delete the list.
func listChangeForbidden(list *core.Record, userID string) string {
	if list.GetString("creator") != userID {
		return "Only the creator of the list can change or delete it."
	}
	if list.GetString("kind") == kindFavorites {
		return "The favorites list cannot be renamed or deleted."
	}
	return ""
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
		if detail := listChangeForbidden(list, user.Id); detail != "" {
			response = reviewsapi.UpdateList403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(detail)}
			return nil
		}
		saved = list
		if req.Body.Name == nil {
			return nil
		}
		name := strings.TrimSpace(*req.Body.Name)
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
	counts, err := s.itemCounts(saved)
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
		if detail := listChangeForbidden(list, user.Id); detail != "" {
			response = reviewsapi.DeleteList403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(detail)}
			return nil
		}
		return tx.Delete(list)
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

// listItemKeys put the restaurant that was added last first. The restaurant
// is unique in a list. Its ID is the last sort value.
var listItemKeys = []sortKey{
	{expr: column("list_items", "created"), desc: true},
	{expr: column("list_items", "restaurant"), desc: true, unique: true},
}

func (s *Server) ListListRestaurants(ctx context.Context, req reviewsapi.ListListRestaurantsRequestObject) (reviewsapi.ListListRestaurantsResponseObject, error) {
	query, err := fingerprint("list-restaurants", req.ListId)
	if err != nil {
		return nil, err
	}
	after, detail := decodeCursor(req.Params.Cursor, query, len(listItemKeys))
	if detail != "" {
		return reviewsapi.ListListRestaurants400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(detail)}, nil
	}
	list, _, err := s.publicList(stateFrom(ctx).user, req.ListId)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return reviewsapi.ListListRestaurants404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(listNotFound)}, nil
	}
	w := listItemsWhere(list.Id)
	if after != nil {
		w.keyset(listItemKeys, after)
	}
	ids, next, err := s.pageIDs("list_items", w, listItemKeys, limitOf(req.Params.Limit), query)
	if err != nil {
		return nil, err
	}
	records, err := s.recordsInOrder("restaurants", ids)
	if err != nil {
		return nil, err
	}
	items, err := s.restaurantsOf(stateFrom(ctx).user, records...)
	if err != nil {
		return nil, err
	}
	return reviewsapi.ListListRestaurants200JSONResponse{Items: items, NextCursor: next}, nil
}

func listItemsWhere(listID string) *where {
	w := newWhere()
	w.compare(column("list_items", "list"), []comparison{{"=", listID}})
	return w
}

// listAccess is the list of a listRef, or the problem of the response.
type listAccess struct {
	list      *core.Record
	notFound  *reviewsapi.NotFoundApplicationProblemPlusJSONResponse
	forbidden *reviewsapi.ForbiddenApplicationProblemPlusJSONResponse
}

// ownList gets the list of ref for a change of its restaurants. Only the
// creator of the list can change them.
func ownList(app core.App, ref string, user *core.Record) (listAccess, error) {
	if ref == favoritesRef {
		list, err := favoritesOf(app, user.Id)
		return listAccess{list: list}, err
	}
	list, err := findList(app, ref)
	if err != nil {
		return listAccess{}, err
	}
	if list == nil {
		p := notFound(listNotFound)
		return listAccess{notFound: &p}, nil
	}
	if list.GetString("creator") != user.Id {
		p := forbidden("Only the creator of the list can change its restaurants.")
		return listAccess{forbidden: &p}, nil
	}
	return listAccess{list: list}, nil
}

func findItem(app core.App, listID, restaurantID string) (*core.Record, error) {
	item, err := app.FindFirstRecordByFilter("list_items", "list = {:list} && restaurant = {:restaurant}",
		dbx.Params{"list": listID, "restaurant": restaurantID})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

// AddListRestaurant runs the checks and the save in one transaction. The
// save of the list sets its updated time.
func (s *Server) AddListRestaurant(ctx context.Context, req reviewsapi.AddListRestaurantRequestObject) (reviewsapi.AddListRestaurantResponseObject, error) {
	user := stateFrom(ctx).user
	var response reviewsapi.AddListRestaurantResponseObject = reviewsapi.AddListRestaurant204Response{}
	err := s.app.RunInTransaction(func(tx core.App) error {
		access, err := ownList(tx, req.ListRef, user)
		switch {
		case err != nil:
			return err
		case access.forbidden != nil:
			response = reviewsapi.AddListRestaurant403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: *access.forbidden}
			return nil
		case access.notFound != nil:
			response = reviewsapi.AddListRestaurant404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: *access.notFound}
			return nil
		}
		list := access.list
		if _, err := s.findRestaurant(tx, req.RestaurantId); errors.Is(err, sql.ErrNoRows) {
			response = reviewsapi.AddListRestaurant404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(restaurantNotFound)}
			return nil
		} else if err != nil {
			return err
		}
		item, err := findItem(tx, list.Id, req.RestaurantId)
		if err != nil || item != nil {
			return err
		}
		n, err := tx.CountRecords("list_items", dbx.HashExp{"list": list.Id})
		if err != nil {
			return err
		}
		if n >= maxListItems {
			response = reviewsapi.AddListRestaurant409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict("The list has the maximum number of restaurants.")}
			return nil
		}
		items, err := tx.FindCachedCollectionByNameOrId("list_items")
		if err != nil {
			return err
		}
		item = core.NewRecord(items)
		item.Set("list", list.Id)
		item.Set("restaurant", req.RestaurantId)
		if err := tx.Save(item); err != nil {
			return err
		}
		return tx.Save(list)
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *Server) RemoveListRestaurant(ctx context.Context, req reviewsapi.RemoveListRestaurantRequestObject) (reviewsapi.RemoveListRestaurantResponseObject, error) {
	user := stateFrom(ctx).user
	var response reviewsapi.RemoveListRestaurantResponseObject = reviewsapi.RemoveListRestaurant204Response{}
	err := s.app.RunInTransaction(func(tx core.App) error {
		access, err := ownList(tx, req.ListRef, user)
		switch {
		case err != nil:
			return err
		case access.forbidden != nil:
			response = reviewsapi.RemoveListRestaurant403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: *access.forbidden}
			return nil
		case access.notFound != nil:
			response = reviewsapi.RemoveListRestaurant404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: *access.notFound}
			return nil
		}
		list := access.list
		item, err := findItem(tx, list.Id, req.RestaurantId)
		if err != nil || item == nil {
			return err
		}
		if err := tx.Delete(item); err != nil {
			return err
		}
		return tx.Save(list)
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}
