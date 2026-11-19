---
description: Let users make public lists of restaurants. Each user has a favorites list that cannot be renamed or deleted. Each restaurant card has a heart that adds the restaurant to it.
status: implemented
---

# Motivation
[motivation]: #motivation

A user could not save a restaurant for later.
The only record of interest in a restaurant was a review, and a review needs a visit.
A reader of a reviewer page could see what the reviewer scored, but not what the reviewer recommends.

Expected outcome:
- a user can make lists of restaurants, such as "Date night"
- each user has a favorites list. It always exists.
- a heart on each restaurant card adds the restaurant to the favorites list with one click
- the lists are public, like the reviews of a user

# Scope
[scope]: #scope

In scope:
- the `lists` and `list_items` collections and the migration that gives each user a favorites list
- the `lists` tag of the API with eight operations
- the `favorite` property of `Restaurant` and the `list_count` property of `Reviewer`
- the heart, the "Save to list" menu, the list page, and the "Lists" section of the reviewer page
- the link preview, the structured data, and the sitemap entry of a list page. See decisions 00013 and 00021.

Out of scope:
- private or unlisted lists. All lists are public, the favorites list also.
- an order that the user sets, and a description of a list
- the number of users who have a restaurant as a favorite
- a search, a filter, or a sort of the restaurants of a list
- a heart on the map pins. The card of a selected pin has the heart.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Lists

A list has a name, a kind, a creator, and restaurants.

```json
{
  "id": "l7d2f9h4j1k8m3p",
  "name": "Date night",
  "kind": "custom",
  "creator": { "id": "u5n1c8r3t6y0b2e", "name": "Ana" },
  "item_count": 4,
  "created": "2024-11-04T01:26:55.002Z",
  "updated": "2024-11-04T01:26:55.002Z"
}
```

The kind is `favorites` or `custom`. A user has exactly one `favorites` list with the name "Favorites".
The API refuses a rename or a delete of it with `403`.

| Operation | Route | Who |
|---|---|---|
| `listReviewerLists` | `GET /v1/reviewers/{reviewerId}/lists` | all |
| `getList` | `GET /v1/lists/{listId}` | all |
| `listListRestaurants` | `GET /v1/lists/{listId}/restaurants` | all |
| `createList` | `POST /v1/lists` | a signed in user |
| `updateList` | `PATCH /v1/lists/{listId}` | the creator |
| `deleteList` | `DELETE /v1/lists/{listId}` | the creator |
| `addListRestaurant` | `PUT /v1/lists/{listRef}/restaurants/{restaurantId}` | the creator |
| `removeListRestaurant` | `DELETE /v1/lists/{listRef}/restaurants/{restaurantId}` | the creator |

`listRef` is a list ID or the word `favorites`. `favorites` is the favorites list of the user of the token.
A heart then needs no list ID.

`PUT` and `DELETE` of a restaurant give `204` also when nothing changes.
A client can send them again after a network error.
A restaurant that is added or removed changes `updated` of the list. The sitemap uses this time.

`listReviewerLists` gives all lists of a user in one response. The favorites list is first. The other lists follow, the newest first.
With `restaurant[eq]`, each list has `has_restaurant`. The "Save to list" menu uses it for its checkboxes.
The parameter does not filter the lists. It has its own description in the spec, `ListHasRestaurant`.

`listListRestaurants` gives pages of `Restaurant`. The restaurant that was added last is first.

## Limits and conflicts

- A user has at most 50 lists. A list has at most 500 restaurants. The next one gets `409 Conflict`.
- The names of the lists of one user are unique. The comparison ignores the case of ASCII letters.
  A second list with a name gets `409 Conflict` with `property` `name`.
- A user cannot make a list with the name "favorites". The favorites list has this name.
- The API removes the spaces at the start and the end of a name. " Date night" is then the name "Date night".

## Favorite flag

Each `Restaurant` has `favorite`. It is `true` if the favorites list of the user of the token has the restaurant.
It is `false` for a request without a token.
All operations that give a `Restaurant` set it: the list, get, create, and update operations of restaurants, and `listListRestaurants`.

## UI

- Each restaurant card has a round heart button at the top right of the cover. The restaurant page has a "Favorite" button with the heart.
- The restaurant page has a "Save to list" menu. It has a checkbox for each list of the user and a field for a new list.
- `/lists/{id}` shows a list: the name, the creator with a link, and the restaurant cards.
  The creator of a custom list sees "Rename list", "Delete list", and a "Remove" button below each card.
- The reviewer page has a "Lists" section with a link for each list. The page of the signed in user also has a "Create list" form.
- The account menu has "My lists". It opens the reviewer page of the user.
- A signed out user who selects a heart or "Save to list" goes to the sign in page.

# Implementation Details
[implementation-details]: #implementation-details

## Data

The migration `1794268813_add_lists.go` makes two collections.

`lists`: `creator`, `name`, `kind`, `created`, `updated`.
- `idx_lists_creator_created` on `(creator, created, id)`
- `idx_lists_creator_name`, unique, on `(creator, name COLLATE NOCASE)`
- `idx_lists_favorites`, unique, on `(creator)` where `kind = 'favorites'`

`list_items`: `list`, `restaurant`, `created`.
- `idx_list_items_list_restaurant`, unique, on `(list, restaurant)`
- `idx_list_items_list_created` on `(list, created, restaurant)`. `listListRestaurants` reads one page of it.
- `idx_list_items_restaurant` on `(restaurant, list)`

The relations have cascade delete. A deleted user loses the lists. A deleted list loses its items. A deleted restaurant leaves all lists.
The collections have no API rules. Only the Public API and superusers can read or write them.

The migration inserts a favorites list for each user that exists.

## Favorites list of a new user

`registerFavorites` binds a hook to the create of a `users` record. It saves the favorites list in the transaction of the user.
The hook also runs for a user that a superuser makes in the dashboard.
A user from SQL has no favorites list. `favoritesOf` makes it at the first `PUT` or `DELETE` with `favorites`.

## Server

- `app/internal/publicapi/lists.go` has the operations.
- The write operations run their checks and the save in one transaction. SQLite then has one writer, so two requests cannot pass a limit together.
- `listNameTaken` compares with `COLLATE NOCASE`, the same as the unique index.
- `ownList` gives `404` for an unknown list and `403` for a list of another user. The lists are public, so `403` tells nothing new.
- `listListRestaurants` uses `pageIDs` on `list_items`. The sort keys are `created` and `restaurant`, the two last columns of the index.
  The restaurant ID is the last key, so the page query gives the restaurant IDs. The cursor fingerprint has the prefix `list-restaurants` and the list ID.
- `restaurantsOf` makes the `Restaurant` values of a page. It gets the favorites of the user with one statement for the page.
  The statement uses `idx_lists_favorites` and `idx_list_items_list_restaurant`. A request without a token runs no statement.
- `GetReviewer` counts the lists in the statement of decision 00032.
- `noRoute` knows `PUT`. A wrong method on a list path gets `405`.

## Web UI server

- `/lists/{id}` gets the title "<name> by <creator>" and a description with the number of restaurants.
- Its structured data is a schema.org `ItemList` with `name`, `numberOfItems`, and `url`.
- `/sitemap.xml` has the lists that have at least one restaurant. An empty list is not in it.

## UI

- `FavoriteButton` changes the heart before the response. A failed request changes it back and shows a toast.
  Its accessible name is "Favorite <name>" in the two states. `aria-pressed` tells the state.
- The restaurant card is one link with a layer above the card. The heart has `z-10`, so it is above this layer.
- `FavoriteButton` sets its state again when it gets a different restaurant. The map view uses one card for all pins.
- `SaveToListMenu` loads the lists when it opens. It does not show the favorites list. The heart changes that list.
- "Create" in the menu makes the list and then adds the restaurant. If the add fails, the new list stays in the menu with an empty checkbox. The checkbox is the retry.
- `listFailureText` in `app/ui/src/lib/lists.ts` has the texts for a failed list request. The menu, the list page, and the reviewer page use it.
- `ListPage` removes a card from the favorites list of the user when the user selects its heart.
- `specs/design-system` has the rules of `FavoriteButton` and `SaveToListMenu`.

## Corner cases

- Another user deletes a restaurant of a list. The cascade removes the item. `item_count` is then one less.
- A user opens a list of a deleted user. The list is deleted with the user, so the page shows "We cannot find this page".
- A list request for a signed out visitor has `favorite` `false` for each restaurant, also in the favorites list of another user.
- The restaurants of a list can be more than `item_count` for a moment after another tab adds one. A reload makes them equal.
- The limit responses use `409`. The body of such a request is valid, so `422` does not fit.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- The favorites list is public. A user who wants a private list has no option. The backlog has private lists under "Won't do".
- A list name is public text without moderation, the same as a review title.
