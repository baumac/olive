---
description: Remove the cleanliness score of restaurant reviews from the Public API, the database, and the UI. The overall of a restaurant review is the average of four scores. This is version 8.0.0 of the spec.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00001 gives a restaurant review five scores: space, service, food, price, and cleanliness.
`cleanliness` came later as the fifth score. Before it, the `space` score covered cleanliness.

The `space` score still covers cleanliness. A clean or a dirty dining room is part of the space.
The `cleanliness` score therefore repeats a part of `space`. A reviewer scores the same thing two times.
It also counts two times in `overall`.

This decision removes the `cleanliness` score.
The review form then has four scores, like the recipe review form of decision 00047.

Expected outcome:
- a restaurant review has the scores `space`, `service`, `food`, and `price`
- `overall` of a restaurant review is the average of these four scores
- the restaurant rating, the overall sort, and the reviewer averages use the four scores

# Scope
[scope]: #scope

In scope:
- the `RestaurantReview`, `RestaurantReviewCreate`, `RestaurantReviewPatch`, and `RestaurantRating` schemas
- the `cleanliness` filter and the `cleanliness` sort of the restaurant review lists
- a migration that removes the `cleanliness` field and computes the overall values again
- the review form, the score lines, the review filters, and the review sorts of the UI
- the design system, the tests, and the decisions that this decision changes

Out of scope:
- the recipe reviews. They keep their four scores.
- the ranking prior of decision 00029. It stays 5 reviews of 5.5.
- a copy of the old `cleanliness` values to another score
- a `/v2` of the API. The app has no deployment yet. The only client is the UI.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

A restaurant review has four scores:

```json
{
  "id": "r1b8c5e2g9i6k3m",
  "restaurant": "k3x9m2p7q1w8z4a",
  "restaurant_name": "Luigi's Trattoria",
  "date": "2024-11-06",
  "overall": 7.5,
  "space": 7, "service": 9, "food": 8, "price": 6,
  "title": "Great food, a bit expensive"
}
```

`overall` is `(7 + 9 + 8 + 6) / 4 = 7.5`.
The value is exact with two decimals, as for recipe reviews. A review of 7, 9, 8, and 7 has `overall` 7.75.
The UI shows one decimal, so it shows 7.8.

`rating` of a restaurant has `overall`, `space`, `service`, `food`, and `price`.
`rating.overall` is the average of the `overall` values of the reviews, rounded to one decimal.

These requests now get `400 Bad Request`:
- `POST /v1/restaurants/{restaurantId}/reviews` with `cleanliness` in the body. The body has an unknown property.
- `PATCH /v1/restaurants/{restaurantId}/reviews/{reviewId}` with `cleanliness` in the body
- `GET /v1/restaurant-reviews?cleanliness[gte]=7`. The query parameter is not supported.
- `GET /v1/restaurant-reviews?sort=cleanliness`

The UI has no "Cleanliness" score in the review form and on the review pages.
The review filters have no "Cleanliness" filter. The review sort has no "Cleanest" option.
The help text of the review form says "It shows after you select each score."

## Existing reviews

Ana has a review of Luigi's with 7, 9, 8, 6, and a cleanliness of 3. Its `overall` was 6.6.
After the migration, the review has 7, 9, 8, and 6. Its `overall` is 7.5.
The `rating.overall` and the ranking of Luigi's change with it.

The migration deletes the stored `cleanliness` values.
The down migration cannot restore them. It gives each review its `space` value as `cleanliness`, as migration `1794268804` did.

# Implementation Details
[implementation-details]: #implementation-details

## Spec

- version 8.0.0. The removal of a property breaks a client that reads it.
- `RestaurantReview`, `RestaurantReviewCreate`, `RestaurantReviewPatch`, and `RestaurantRating` lose `cleanliness`
- `CleanlinessScore` and the `CleanlinessFilter` parameter go
- the description of `SpaceScore` is "Decoration, comfort, atmosphere, cleanliness, and kitchen setup."
  The help text of the space score in the UI is a copy of it.
- the `RestaurantReviewSort` pattern loses `cleanliness`
- the `overall` of `RestaurantReview` is "the average of the four scores" and "exact with two decimals"
- the SDKs are generated again. See decision 00003.

## Migration

Migration `1794268822_remove_review_cleanliness.go`:
- up drops the triggers of decision 00012 that use the scores. These are `restaurant_index_insert` and the three `review_overall` triggers.
  It also drops the two sort indexes of `restaurant_overall`, as `dropOverallSort` of migration `1794268810` does.
- up drops the generated `overall` column of decision 00031 and its three indexes. SQLite refuses to drop `cleanliness` while they use it.
- up then removes the `cleanliness` field of `reviews`. PocketBase drops the column and its values.
- up makes the `overall` column again as `(space + service + food + price) / 4.0`, with the same three indexes
- up computes `overall` and `ranking` of each `restaurant_overall` row again. `makeOverallSort` makes the triggers and the sort indexes again with the four scores.
- down does the same steps with five scores. It adds `cleanliness` with the `space` value of each review, then makes it required.

`reviewScores` of `migrations/scores.go` makes the SQL of the overall column, the overall, and the ranking from a list of scores.
The restaurant migrations before `1794268822` use `fiveRestaurantScores`. The recipe migrations use `recipeScores`.

## Public API

- `subjects.Restaurants.Scores` has `space`, `service`, `food`, and `price`. The rating query, the score sorts, and the schema guard use it.
- `toReview` computes `overall` from four scores. `TestOverallColumnEqualsResponse` keeps it equal to the column.
- the cursor fingerprint marshals the list parameters to JSON. `cleanliness` had `omitempty`.
  A cursor of a list without the `cleanliness` filter therefore stays valid.

## UI

- `scoreFields` of `lib/reviews.ts` has four restaurant scores
- `reviewFilters.ts` has no `cleanliness` filter and no "Cleanest" sort
- the restaurant list text names the four scores

## Tests

- the migration tests check up and down. The tests of earlier migrations stop before `1794268822` when they check `cleanliness`.
- the API tests check that `cleanliness` in a body, a filter, or a sort gets `400`
- the UI tests and the end-to-end tests use four scores

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
