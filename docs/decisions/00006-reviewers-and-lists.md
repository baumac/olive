---
description: Each reviewer has a public journal with the counts, the averages, the favorites, the lists, the subjects that the reviewer added, and the reviews by month. Users make public lists of restaurants and recipes, and a list page reads like a menu. Each user has a favorites list, and a heart on each card adds a subject to it.
status: proposed
---

# Motivation
[motivation]: #motivation

Decision 00000 wants users to read the reviews of others and to share lists of their favorite things.
A reader who trusts a reviewer wants the other reviews of that person, and whether the person gives high or low scores.
A user also wants to save a restaurant or a recipe for later, before a visit or a cook, and to recommend it.

Expected outcome:
- each reviewer has a page at `/reviewers/{id}`. The name on a review card links to it.
- the page shows only public data: the name, the counts, the averages, the added subjects, the lists, and the reviews
- a user makes lists of restaurants and recipes, such as "Date night"
- each user has a favorites list. It always exists.
- a heart on each card adds the subject to the favorites list with one click
- the lists are public, like the reviews
- the restaurants or recipes of a list have all searches, sorts, and filters of their page

# Scope
[scope]: #scope

In scope:
- `getReviewer` and the review lists of one reviewer
- the journal of the UI, for each reviewer and for the signed in user
- the `lists`, `list_items`, and `list_recipes` collections and the operations of the `lists` tag
- the `favorite` property of restaurants and recipes
- the `list[eq]` filter of `listRestaurants` and `listRecipes`
- the heart, the "Save to list" menu, and the list page of the UI

Out of scope:
- a list of all reviewers, and a sort or a filter of reviewers
- a profile that the user edits, such as a bio or an avatar
- the average of each score of a reviewer
- private or unlisted lists. Decision 00000 makes all data public.
- an order that the user sets, and a description of a list
- the number of users who have a subject as a favorite. Decision 00000 puts likes out of scope.
- follows and messages. Decision 00000 puts social features out of scope.
- the reviewers that wait for approval. Decision 00003 hides them.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## A reviewer

```json
{
  "id": "u5n1c8r3t6y0b2e",
  "name": "Ana",
  "restaurant_review_count": 12,
  "restaurant_average_overall": 7.4,
  "restaurant_count": 3,
  "recipe_review_count": 4,
  "recipe_average_overall": 8.1,
  "recipe_count": 1,
  "list_count": 2
}
```

- the profile has no email address and no timestamps
- an average is the average of the `overall` values of the reviews of the user, rounded to one decimal. It is `null` without reviews.
- `restaurant_count` and `recipe_count` count the subjects with the user as their creator. `listRestaurants` with `creator[eq]` lists them.
- an unknown user, a deleted user, and a user who waits for approval get `404`

| Operation | Route |
|---|---|
| `getReviewer` | `GET /reviewers/{reviewerId}` |
| `listReviewerRestaurantReviews` | `GET /reviewers/{reviewerId}/restaurant-reviews` |
| `listReviewerRecipeReviews` | `GET /reviewers/{reviewerId}/recipe-reviews` |
| `listReviewerLists` | `GET /reviewers/{reviewerId}/lists` |

The review lists have the filters, the sorts, and the pages of the lists of all reviews, without the `creator` filters.

## A list

```json
{
  "id": "l7d2f9h4j1k8m3p",
  "name": "Date night",
  "kind": "custom",
  "creator": { "id": "u5n1c8r3t6y0b2e", "name": "Ana" },
  "restaurant_count": 4,
  "recipe_count": 1,
  "created": "2024-11-04T01:26:55.002Z",
  "updated": "2024-11-04T01:26:55.002Z"
}
```

- the kind is `favorites` or `custom`. A user has exactly one `favorites` list with the name "Favorites".
- a rename or a delete of the favorites list gets `403` with `favorites_list`
- a list name is a `Name`. The names of one user are unique, without the case of ASCII letters. A second list with a name gets `409` with `list_name_taken` and `"property": "name"`.
- a user cannot make a list with the name "favorites"
- a user has at most 50 lists. A list has at most 500 restaurants and 500 recipes. The next one gets `409` with `list_limit`, `list_restaurant_limit`, or `list_recipe_limit`.

| Operation | Route | Who |
|---|---|---|
| `listReviewerLists` | `GET /reviewers/{reviewerId}/lists` | all |
| `getList` | `GET /lists/{listId}` | all |
| `listListRestaurants`, `listListRecipes` | `GET /lists/{listId}/restaurants`, `GET /lists/{listId}/recipes` | all |
| `createList` | `POST /lists` | an approved user |
| `updateList`, `deleteList` | `PATCH /lists/{listId}`, `DELETE /lists/{listId}` | the creator |
| `addListRestaurant`, `removeListRestaurant` | `PUT /lists/{listRef}/restaurants/{restaurantId}`, `DELETE` | the creator |
| `addListRecipe`, `removeListRecipe` | `PUT /lists/{listRef}/recipes/{recipeId}`, `DELETE` | the creator |

- `listRef` is a list ID or the word `favorites`, the favorites list of the user of the token. A heart then needs no list ID.
- `favorites` works only on the add and remove operations. `listReviewerLists` gives the ID of the favorites list for the read operations.
- `PUT` and `DELETE` of an item give `204` also when nothing changes. A client can send them again after a network error.
- an added or removed item changes `updated` of the list
- `listReviewerLists` gives all lists in one response. The favorites list is first, then the other lists, the newest first.
- with `restaurant[eq]` or `recipe[eq]`, each list has `has_restaurant` or `has_recipe`. The parameter does not filter the lists. The "Save to list" menu uses it for its checkboxes.
- `listListRestaurants` and `listListRecipes` give pages, the last added first

## Favorites

- each `Restaurant` and `Recipe` has `favorite`. It is `true` when the favorites list of the user of the token has it. It is `false` without a token.
- all operations that give a restaurant or a recipe set it

## The list filter

```
GET /v1/restaurants?list[eq]=l7d2f9h4j1k8m3p&sort=distance&near[latitude]=41.38&near[longitude]=2.17
```

- `list[eq]` of `listRestaurants` and `listRecipes` gives only the subjects of one list
- it works with each other parameter, such as `search`, `within`, `tags[has_any]`, and `overall[gte]`
- an ID without a list gives an empty page. A value that is not a record ID gets `400`.
- the word `favorites` is not a value. A response with it would change with the token.

## The UI

The journal:
- `/reviewers/{id}` is the journal of a reviewer. "You" in the navigation opens the journal of the signed in user. It replaces "My reviews", and `/my-reviews` redirects to it.
- the top has the avatar with the first letter, the name, the counts, such as "12 visits · 4 cooks · 3 lists", and the averages, such as "Average score 7.4 for restaurants, 8.1 for recipes"
- then "Favorites" as a row of cards with "See all", and the other lists with their counts
- then "Restaurants added" and "Recipes added". Each shows 3 cards and "Load more restaurants" or "Load more recipes". They are above the entries, so a "Load more" of the entries does not move them.
- then the entries by the month of their `date`. Each month is a grid of tiles with the day, the first photo, the overall, and the name. A tile without a photo shows the name and the overall on `accent`.
- a `KindSwitch` with "All", "Restaurants", and "Recipes" selects the kinds of the entries. `?kind=recipes` selects the recipes on the journal.
- with "All", the entries are always the newest date first. The sort control and the score filters show only for one kind.
- a section without items is not shown. A reviewer without reviews has "No reviews yet".

Lists:
- each card and each row of a subject has a heart button. The subject page has a "Favorite" button.
- the subject page has a "Save to list" menu with a checkbox for each list and a field for a new list
- `/lists/{id}` shows the list as a menu: "A list by Ana", the name, the counts, and the date of the last change. Then it has a section for the restaurants and a section for the recipes.
- each item of the menu has the name, a dotted leader, the overall, and a line with the address or the total time and the tags. Each section has the highest overall first.
- each section shows 100 items, then "Load more restaurants" or "Load more recipes"
- "Menu" and "Map" select the view. "Map" opens `/restaurants?list=<id>&view=map`. "Sort and filter" opens `/restaurants?list=<id>`.
- the creator of a custom list sees "Rename list", "Delete list", and "Remove" next to each item
- the restaurants page and the recipes page have a "List" filter with the lists of the user. A list of the URL from another user is also an option, such as "Date night by Joe".
- the journal of the signed in user has "New list"
- a signed out user who selects a heart or "Save to list" goes to the sign in page
- a signed out visitor sees the note of decision 00009 at the end of the journal and the list page

# Implementation Details
[implementation-details]: #implementation-details

## Reviewers

- `findReviewer` returns no user for a deleted user or a user without approval, except for the user of the token. The reviewer and list read operations use it.
- `reviewerStats` reads the counts and the averages in one statement. The counts and the averages then come from the same reviews.
- the averages are `ROUND(AVG(overall), 1)` of the generated `overall` columns of decision 00005. The statement finds the reviews through the creator indexes.
- the cursor fingerprint of a reviewer list has its own prefix and the user ID. A cursor of one list does not work in another.
- the `webui` package gives `/reviewers/{id}` its own page meta with `og:type` `profile`. See decision 00002.

## Collections

| Collection | Fields | Indexes |
|---|---|---|
| `lists` | `creator`, `name`, `kind`, `created`, `updated` | `idx_lists_creator_created` on `(creator, created, id)`. Unique `idx_lists_creator_name` on `(creator, name COLLATE NOCASE)`. Unique `idx_lists_favorites` on `(creator)` where `kind = 'favorites'`. |
| `list_items` | `list`, `restaurant`, `created` | unique `(list, restaurant)`, `(list, created, restaurant)`, and `(restaurant, list)` |
| `list_recipes` | `list`, `recipe`, `created` | the same three indexes with `recipe` |

- the relations cascade. A deleted user loses the lists. A deleted list loses its items. A deleted subject leaves all lists.
- the collections have no API rules
- the item collections are separate. A `recipe` relation in `list_items` would need a partial index, because PocketBase stores an empty relation as `''`.

## Server

- `lists.go` has the operations. Each write runs its checks and the save in one transaction, so two requests cannot pass a limit together.
- `registerFavorites` binds `OnRecordCreateExecute` of `users`. It saves the favorites list in the transaction of the user, also for a user from the dashboard.
- a user from SQL has no favorites list. `favoritesOf` makes it at the first `PUT` or `DELETE` with `favorites`.
- `ownList` gives `404` for an unknown list and `403` with `not_creator` for a list of another user. The lists are public, so `403` tells nothing new.
- `listItemPage` reads one page of the `(list, created, subject)` index. The subject ID is the last key.
- `viewerDataOf` gets the favorites and the own review counts of a page in two queries. A request without a token runs none.
- `listCounts` counts the restaurants and the recipes of the lists in one statement
- `inList` adds an inner join on the item table with the list ID in the `ON` term. SQLite then starts from the unique index and reads at most 500 rows.
- with `list[eq]`, the `overall` sort sorts the rows of the list in a temporary B-tree. A list has at most 500 items.
- `TestRestaurantListsUseIndexes` checks the plans of the list filter for the `-created`, `-overall`, and `name` sorts

## The UI

- `FavoriteButton` changes the heart before the response. A failed request changes it back and shows a toast. `aria-pressed` tells the state.
- the card is one link with a layer above it. The heart has `z-10`, so it is above this layer.
- `SaveToListMenu` loads the lists when it opens. It does not show the favorites list, because the heart changes that list.
- "Create" in the menu makes the list and then adds the subject. If the add fails, the new list stays in the menu with an empty checkbox. The checkbox is the retry.
- `listFailureText` of `lib/lists.ts` has the texts of the list codes
- `useListFilter` gets the lists of the user with `ownLists` and the list of the URL with `getList`. `ownLists` loads them one time for each page load. Each change of a list calls `forgetOwnLists`.
- the option of the URL list exists before `getList` answers, with the label "A list". The first page then has the filter.
- the map view of a list filter moves to the box of the restaurants of the list. A list filter ignores the distance of the preferred location.
- `ReviewerAdded` loads `listRestaurants` or `listRecipes` with `creator[eq]`, `-created`, and a limit of 3. A reviewer with a count of 0 causes no request.
- the journal loads its entries with `useEntries` of decision 00009 from `listReviewerRestaurantReviews` and `listReviewerRecipeReviews`
- the menu of a list loads `listRestaurants` and `listRecipes` with `list[eq]` and `sort=-overall`

## Corner cases

- a user deletes a subject while the reviewer page is open. The count in the header is one more than the list until a reload.
- the subjects of a list can be more than the count for a moment after another tab adds one
- a signed out visitor gets `favorite` `false` for each subject, also in the favorites list of another user
- a limit response uses `409`. The body of such a request is valid, so `422` does not fit.

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| Lists only in the reviews, without a list to save a subject | Removed. A user could not save a subject before a visit. |
| A `recipe` relation in `list_items` | Not selected. The unique index and each query would need a condition for the kind. |
| Only `limit` and `cursor` for the subjects of a list | Removed. A user could not show the favorites on a map, sort them by distance, or search them. |
| `list[in]`, and `favorites` as the value of `list[eq]` | Not selected. A response with `favorites` would change with the token. |
| A reviewer page with only the reviews | Removed. A reader could not see the scores in general or what the reviewer added. |
| Tabs for the kinds on the reviewer page | Not selected. `KindSwitch` is links, so each kind has its own URL. |
| "My reviews" apart from the reviewer page | Removed. Both pages showed the same reviews. The journal is one page for both. |
| The cards of a list in a grid | Removed. A shared list reads better as a menu with the scores in one column. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
