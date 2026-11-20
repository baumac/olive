---
description: A review is one visit to a restaurant or one cook of a recipe. A user can write many reviews of one subject. Each review has a date. This is version 7.0.0 of the spec.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00001 allows one review for each user and restaurant. Decision 00047 has the same rule for recipes.
A second review gets `409` with `review_exists`. A review also has no date.

Decision 00000 wants to "remember and share all of our food experiences".
A second dinner at a restaurant or a second cook of a recipe is a new experience.
Now the user must change the old review. The old scores, text, and date are then lost.
The app cannot show how a restaurant changed, or how often we cooked a recipe.

Expected outcome:
- a review is one visit or one cook
- a user can write many reviews of one restaurant or recipe
- each review has the date of the visit or the cook
- the review lists sort by this date

# Scope
[scope]: #scope

In scope:
- the removal of the one-review rule and of the `review_exists` code
- the `date` property of restaurant reviews and recipe reviews
- the `date` sort of the review lists
- the migration and its down migration
- the review form, the review card, the review page, and the restaurant and recipe pages
- the tests

Out of scope:
- a `date` filter of the review lists
- a count of the visits or the cooks of one user, such as "You visited 3 times"
- one review with a list of dated visits. This decision does not use it.
- a change of the Bayesian ranking of decision 00029. Backlog item 3 has it.
- the link preview tags of decision 00013

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## A review is one visit or one cook

Ana eats at Luigi's Trattoria in March and in November. She writes two reviews.
Each review has its own scores, title, feedback, photos, and date.
The March review stays the same after the November review.

```
POST /v1/restaurants/k3x9m2p7q1w8z4a/reviews
{ "date": "2026-11-14", "space": 7, "service": 9, "food": 8, "price": 6, "cleanliness": 8, "title": "Still great" }
```

The API returns `201` for each review. It never returns `409` for a create of a review.

## The date

`date` is the date of the visit or the cook. It has the form `YYYY-MM-DD`. It has no time and no time zone.
Both kinds use the name `date`, the same as decision 00048 wants.

- A create needs `date`. A patch can change it.
- The date must exist. `2026-02-30` gets `422`.
- The date must be `1900-01-01` or later.
- The date must not be after the current date in UTC+14. That is the latest time zone. A user anywhere can then use the local date of today.

The UI shows the date with a label for each kind:

| Kind | Form label | Card and review page |
|---|---|---|
| restaurant | Visited on | Visited on 14 Nov 2026 |
| recipe | Cooked on | Cooked on 14 Nov 2026 |

The form starts with the local date of today. The date input does not allow a later date.

## The sort

The review lists have a `date` sort. The default sort is `-date`, the newest visit or cook first.
The reviews of one day then sort by `created`. Two reviews of one day show the last written first.
`created` stays. It is the time that the user wrote the review.
The sort control has "Newest visit" or "Newest cook", "Highest rated", and "Lowest rated".
A link can also select "Oldest visit" or "Oldest cook", "Newest written", and "Oldest written".

## The pages

The restaurant and the recipe pages always have "Write a review". It opens a new review.
The own reviews are in the review list, with their edit buttons.
The delete text of a restaurant or a recipe tells "Your 3 reviews and their photos will be deleted too."
The edit of a restaurant or a recipe counts all own reviews. The reviews of other users lock the name, like decision 00016.
A restaurant and a recipe have `own_review_count`, the number of reviews of the user of the token. It is like `favorite`.
The UI uses it for the delete text and for the lock. It needs no other request.

## The counts and the averages

Each review counts one time:
- `review_count` of a restaurant or a recipe is the number of visits or cooks
- `rating` is the average over all reviews
- the Bayesian ranking of decision 00029 counts each review
- `restaurant_review_count`, `recipe_review_count`, and the averages of `Reviewer` count each review

A person who visits a restaurant ten times has ten reviews in the average.
The database triggers of decision 00012 do not change.

# Implementation Details
[implementation-details]: #implementation-details

## Spec

- `RestaurantReview`, `RecipeReview`, and their `Create` and `Patch` schemas get `date`. Its schema is `ReviewDate`.
  It is required in `RestaurantReview`, `RecipeReview`, and the `Create` schemas.
- `ReviewDate` is a string with the pattern `^\d{4}-\d{2}-\d{2}$` and the format `review-date`.
  It does not use `format: date`. The TypeScript generator changes `format: date` to a `Date` in UTC.
  The API checks the `review-date` format. The response validation does not check it.
- `RestaurantReviewSort` and `RecipeReviewSort` get `date`. Their default is `-date`.
  The `date` sort has the keys `date`, `created`, and `id`.
  `parseSort` skips a key that an earlier property already has. `-date,created` then has no second `created` key.
- `createRestaurantReview` and `createRecipeReview` lose the `409` response and the text about one review.
- The `code` enum of `Problem` loses `review_exists`.
- The descriptions of `review_count` and of the counts of `Reviewer` tell that each review counts.
- `Restaurant` and `RecipeSummary` get the required `own_review_count`. It is `0` for a request without a token.
- The version is 7.0.0. A required property and a removed enum value break old clients.
- `make generate` makes the SDKs again.

## Migration

`1794268821_allow_many_reviews.go` changes `reviews` and `recipe_reviews`:

- It drops `idx_reviews_restaurant_creator` and `idx_recipe_reviews_recipe_creator`.
  It adds them again without `UNIQUE`. The lookup of the reviews of one user and one subject uses them.
- It adds the required text field `date` with the pattern `^\d{4}-\d{2}-\d{2}$`.
- It sets `date` of each old review to the date of `created` in UTC with `substr(created, 1, 10)`.
- It adds the indexes of the `date` sort:
  - `(restaurant, date, created, id)`, `(creator, date, created, id)`, and `(date, created, id)` on `reviews`
  - `(recipe, date, created, id)`, `(creator, date, created, id)`, and `(date, created, id)` on `recipe_reviews`
- `date` is not in the triggers of decision 00012 and not in the generated `overall` column of decision 00031.

The down migration deletes reviews:

- For each user and subject, it keeps one review. It keeps the latest `date`, then the latest `created`, then the largest `id`.
- It deletes the other reviews with `app.Delete`. The cascade of decision 00008 deletes their photos. PocketBase deletes the files.
- It deletes the rows of the deleted reviews in `reviews_fts` and `recipe_reviews_fts`. It does not depend on the hooks of `searchindex`.
- The triggers of decision 00012 change the overall tables after each delete.
- It then removes `date` and the `date` indexes, and adds the unique indexes again.
- A restore of a backup is the only way to get the deleted reviews back. See decision 00022.

## Public API

- `createReview` of `reviews.go` loses the count of the own reviews and the `isNotUnique` case.
  The transaction still reads the parent. A missing parent still gets `404`.
- The review input, the patch, and `toReview` have `date`.
- `checkReviewDate` of `reviewdate.go` checks the `review-date` format for both kinds.
  A wrong date gets `422` with the property `date`.
- `reviewDateGuard` checks `date` on each save of `reviews` and `recipe_reviews` through the PocketBase API.
  A save in the dashboard then gets the same check, like decision 00016.
- `makeReviewSorts` adds `date`. Both kinds then have it.
- A review list without `sort` uses `-date`. The lists of restaurants and recipes keep `-created`.
  A cursor that the old default `-created` made keeps `-created`. An open page can then load more after the deploy.
- `viewerDataOf` gets the favorites and the own review counts of a page in two queries.
- The reviewer stats query can use the creator index of the date or of the overall. Both read the same rows.

## UI

- `reviewKinds.ts` has the date label of each kind: "Visited on" and "Cooked on".
- `ReviewFormPage.vue`:
  - It does not open the existing review of the user. It always makes a new review.
  - It has no `review_exists` alert.
  - It has a date input with the label of the kind. It starts with the local date of today. Its `max` is the local date of today.
  - The draft of the form keeps the date.
  - An error of the API for `date` shows at the input.
- The UI shows `date` without a change of time zone.
  `new Date('2026-11-14')` is midnight UTC. West of UTC, it shows the day before.
  `format.ts` gets a function that reads the year, the month, and the day of the string.
- `ReviewDate.vue` shows "Visited on 14 Nov 2026" or "Cooked on 14 Nov 2026".
  `ReviewCard.vue` shows it in the place of `DateText`. It keeps the "edited" mark.
  `ReviewPage.vue` shows it, and "Written" with `DateText`.
- `parents.ts`:
  - `findOwnReviewId` is gone. `parentAccess` uses `ownReviewCount` of the parent.
  - `hasOtherReviews` is true when `reviewCount` is larger than `ownReviewCount`.
- `RestaurantPage.vue` and `RecipePage.vue`:
  - The delete text uses `ownReviewCount`.
  - The page loads the parent again after a change of the signed in user.
  - The write link is always "Write a review" to the new review form.
- `ReviewFormPage.vue` gets the local date again on each focus of the date input and on each submit.
- `ReviewList.vue` has the default sort `-date`. `reviewFilters.ts` has the sort options of the kind.
  The old URLs with `sort=-created` and `sort=created` still work.

## Tests

- Go tests:
  - a second review of one user and subject gets `201`, for both kinds
  - each date rule, for the API and for a save in the dashboard
  - the `date` sort and its cursor
  - the query plans of the `date` sort use the new indexes
  - the migration sets `date` from `created`
  - the down migration keeps the latest review, deletes the photos and the FTS5 rows of the others, and restores the unique index
  - `subject_internal_test.go` uses another code than `review_exists`
- End-to-end tests:
  - a user writes two reviews of one restaurant with different dates. The list shows both, the newest date first.
  - the same for a recipe
  - the axe test of the review form opens the new form and the edit form
  - `api` helpers send the date of today in UTC without a `date`

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
