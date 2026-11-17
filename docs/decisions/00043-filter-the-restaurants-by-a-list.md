---
description: listRestaurants gets a `list[eq]` filter. The restaurants of a list then have all searches, sorts, and filters of the restaurants page, the map and the distance sort also.
status: implemented
---

# Motivation
[motivation]: #motivation

`listListRestaurants` of decision 00033 has only `limit` and `cursor`.
A user could not show the favorites on a map, sort them by distance, or search them.
`listRestaurants` already has all of this. It only had no filter for a list.

Expected outcome:
- `listRestaurants` with `list[eq]` gives only the restaurants of one list
- all other parameters of `listRestaurants` work with it
- the list page links the restaurants page with this filter, as a list and as a map

# Scope
[scope]: #scope

In scope:
- the `list[eq]` parameter of `listRestaurants`. Version 5.2.0 of the spec has it.
- the "List" filter of the restaurants page and the `list` key of its URL
- the "Sort and filter" and "Show on map" links of the list page

Out of scope:
- a sort by the time that a restaurant was added to the list. `listListRestaurants` keeps this order.
- `list[in]`, and the word `favorites` as the value of `list[eq]`. A response with `favorites` would change with the token.
- a 404 for an unknown list. A filter selects no restaurant, the same as `creator[eq]`.
- a change of `listListRestaurants`. It stays for the list page and for other clients.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

```
GET /v1/restaurants?list[eq]=l7d2f9h4j1k8m3p&sort=distance&near[latitude]=41.38&near[longitude]=2.17
```

This request gives the restaurants of the list, the nearest first.
`list[eq]` works with each other parameter, such as `search`, `within`, `tags[has_any]`, and `overall[gte]`.
All lists are public. The filter needs no token.
An ID without a list gives an empty page. A value that is not a record ID gets `400`.

The cursor fingerprint has all parameters, so a cursor of one list is not valid for another list.

## UI

- The list page has "Sort and filter" and "Show on map" when the list has restaurants.
  They open `/restaurants?list=<id>` and `/restaurants?list=<id>&view=map`.
- The restaurants page has a "List" filter. Its options are the lists of the signed in user.
  A list of the URL that is not a list of the user is also an option, with the name of its creator, such as "Date night by Joe".
- The chip is "List: Favorites". Its remove button gives all restaurants again.
  A list that does not exist has the chip "List: Unknown list".
- With a preferred location, the default sort is "Nearest". The favorites are then sorted by distance.
- A list filter ignores the distance of the preferred location. A favorite in another city then stays in the list.
- The map view of a list filter moves to the box of the restaurants of the list.
- With the favorites list of the user as the filter, a card leaves the page when the user removes its heart.

# Implementation Details
[implementation-details]: #implementation-details

- `inList` of `lists.go` adds an inner join on `list_items` with the list ID in the `ON` term.
  A join lets SQLite start from `idx_list_items_list_restaurant` and read at most 500 rows.
  `TestRestaurantListsUseIndexes` checks this plan for the `-created`, `-overall`, and `name` sorts.
- With `list[eq]`, the `overall` sort sorts the rows of the list in a temporary B-tree. A list has at most 500 restaurants.
- `useListFilter` of `app/ui/src/composables` makes the filter definition and gives the ID of the favorites list.
  - It gets the lists of the user with `ownLists` of `app/ui/src/lib/lists.ts` and the list of the URL with `getList`.
  - `ownLists` gets the lists one time per page load. Each create, rename, and delete of a list calls `forgetOwnLists`.
    A list that another tab makes is not an option until a reload.
  - A failed `getList` without a `404` runs again after 2 and 10 seconds.
  - The option of the URL list exists before `getList` answers, with the label "A list". `useListQuery` then keeps the filter of the URL. The first page has the filter.
  - A malformed ID in the URL is not an option. The page then ignores it.
- `RestaurantsPage` watches the JSON of the filters, not the filters object. A new definition after the lists load then starts no second request.
- `RestaurantsPage` gets up to 100 restaurants of the list for the map box. `MapView` has the `fit` prop for this box.
  The box of a list across the 180th meridian goes around the earth the long way.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
