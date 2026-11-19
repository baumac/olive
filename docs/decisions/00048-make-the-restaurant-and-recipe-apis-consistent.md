---
description: Fix the differences between the restaurant and the recipe parts of the Public API. The restaurant review names get `restaurant`, a list of recipes has no ingredients and no steps, and a stored source is locked like a stored location. This is version 6.0.0 of the spec.
status: implemented
---

# Motivation
[motivation]: #motivation

A review of `specs/api/public-openapi.yaml` compared the restaurant and the recipe parts after decision 00047.
Most parts agree. The review found these differences:

1. `RecipeReview.overall` has two decimals. Its schema `AverageScore` told "with one decimal".
2. A stored `location` of a restaurant with reviews of other users cannot change. It has the `place_id` of the duplicate check.
   The `source` of a recipe is its duplicate check, but it could change.
   A creator could then give a reviewed recipe the link of a different recipe.
3. `RecipeList` had the complete recipes. A recipe can have 100 ingredients of 200 characters and 50 steps of 2,000 characters.
   A page of 100 recipes could then have about 13 MB. A restaurant of a list has no property of this size.
4. The summaries of `createList`, `getList`, `updateList`, and `deleteList` told "a list of restaurants". A list also has recipes.
5. `addListRecipe` told that the restaurants do not count for the limit. `addListRestaurant` did not tell this about the recipes.
6. `updateRecipe` did not tell how the API compares the sources. The server uses the rule of `createRecipe`.
7. `Problem.property` named only `name` and `location`. The `source_taken` problem has `source`.

The names of the restaurant parts are from the time before recipes:

- the restaurant reviews had no prefix: `Review`, `createReview`, `/reviews`, and `listReviewerReviews`. The recipe reviews have `recipe`.
- `review_count` and `average_overall` of `Reviewer` count only the restaurant reviews. Next to them are `recipe_review_count` and `recipe_average_overall`.
- `item_count` of `SavedList` counts only the restaurants. Next to it is `recipe_count`.

Two rules of the recipes were not good:

- the patterns of `Ingredient` and `Step` had the invisible characters in the file, not `\u` escapes. A person cannot review them, and an editor can remove them.
- a `Step` can start or end with tabs and line breaks. The API removed only the ASCII spaces. A stored step could then start with an empty line.

# Scope
[scope]: #scope

In scope:

- all of the above in the spec, the server, the generated SDKs, the UI, and the tests
- a migration that removes the tabs and the line breaks around the stored steps

Out of scope:

- aliases for the old names and paths. The UI is the only known client, and it changes with the server.
  The version 6.0.0 tells other clients about the break.
  A browser tab with the UI from before the deploy breaks until a reload.
  Its reviewer pages get `404` for the reviews, and its list pages show no restaurant count.
  A reload fixes the tab. Aliases would double the restaurant review operations of the spec for this short time.
- a lock of `source` in the dashboard. The dashboard also does not lock the name of a restaurant.
- new filters of `listRecipes`, such as `servings`

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The new names

| Before | After |
|---|---|
| `GET /reviews`, `listAllReviews` | `GET /restaurant-reviews`, `listAllRestaurantReviews` |
| `GET /reviewers/{reviewerId}/reviews`, `listReviewerReviews` | `GET /reviewers/{reviewerId}/restaurant-reviews`, `listReviewerRestaurantReviews` |
| `createReview`, `listReviews`, `getReview`, `updateReview`, `deleteReview` | `createRestaurantReview`, `listRestaurantReviews`, `getRestaurantReview`, `updateRestaurantReview`, `deleteRestaurantReview` |
| `addReviewPhoto`, `updateReviewPhoto`, `deleteReviewPhoto`, `getReviewPhotoFile` | `addRestaurantReviewPhoto`, `updateRestaurantReviewPhoto`, `deleteRestaurantReviewPhoto`, `getRestaurantReviewPhotoFile` |
| `Review`, `ReviewCreate`, `ReviewPatch`, `ReviewList` | `RestaurantReview`, `RestaurantReviewCreate`, `RestaurantReviewPatch`, `RestaurantReviewList` |
| the parameters `ReviewSort` and `ReviewSearch` | `RestaurantReviewSort` and `RestaurantReviewSearch` |
| `Reviewer.review_count`, `Reviewer.average_overall` | `restaurant_review_count`, `restaurant_average_overall` |
| `SavedList.item_count` | `restaurant_count` |

The paths under `/restaurants/{restaurantId}/reviews` stay. They already name the restaurant. The photo `urls` then also stay.
`review_count` of `Restaurant` and `Recipe` stays. It counts the reviews of one restaurant or recipe.
`ReviewId` and `ReviewOverallFilter` stay. Both kinds use them.

## A list of recipes

`listRecipes` and `listListRecipes` return `RecipeSummary` items.
A summary has all properties of `Recipe` except `ingredients` and `steps`.
`getRecipe`, `createRecipe`, and `updateRecipe` return the complete `Recipe`.
The search of `listRecipes` still finds the words of the ingredients.

## The source of a reviewed recipe

Ana adds "Dal" with the source `https://dal.example/tarka`. Joe reviews it.

- Ana changes the source to `https://dal.example/other`. The API returns `409` with `other_reviews` and the `property` `source`.
- Ana clears the source with `null`. The API returns the same problem.
- Ana changes the source to `https://DAL.example/tarka/`. The API compares the sources without the case of the host and without a final `/`. The change is allowed.
- A recipe without a source can always get one, like a restaurant without a location.

The recipe form makes a stored source read only when other people reviewed the recipe.
Its note then names the source with the name, the ingredients, and the steps.
After an `other_reviews` conflict, the form gets the recipe again.
A locked source field then shows the stored source, not the text of the user. The next save does not send it.

## Steps

The API removes the ASCII spaces, tabs, and line breaks at the start and the end of each step.
`"\n\t Mix.\nBake. \r\n"` becomes `"Mix.\nBake."`. The line break between the two sentences stays.

# Implementation Details
[implementation-details]: #implementation-details

- `make generate` changes the SDKs. The server methods and the UI calls follow the new names.
  The `KindInfo` methods of `reviewKinds.ts` keep their names. They are the same for both kinds.
- The cursors keep their fingerprints. A page that was open during the deploy can still load more.
- `Recipe` is an `allOf` of `RecipeSummary` and the ingredients and the steps, like `RestaurantPhoto`.
  `recipeSummaries` in `recipes.go` makes the summaries. `recipeOf` adds the ingredients and the steps for one recipe.
  The UI types the list items and the cards with `RecipeSummary`. `ParentSummary` of `reviewKinds.ts` is `Restaurant | RecipeSummary`.
- `withDetails` copies the fields of `RecipeSummary` into `Recipe` by their names. A new summary field then also reaches `getRecipe`.
- `UpdateRecipe` puts `source` in the frozen properties when the stored source is not empty and the source keys differ.
  The source key is the value of decision 00047 that the duplicate check uses.
- `trimWhitespace` of `body.go` removes `" \t\n\r"`. It was `trimSpaces`. The patterns of `Name`, `Title`, `Address`, `AltText`, and `Ingredient` do not allow tabs and line breaks.
  Only `Step` then changes. A no-break space still stays. See decision 00040.
- The migration `1794268820_trim_stored_steps.go` trims each stored step with `json_each` and `json_group_array` with `ORDER BY key`. The `updated` time stays.
  It changes only an array of text. A save without validation can store other values. The migration keeps them as they are, and invalid JSON does not stop it.
  The steps are not in the search tables.
- `AverageScore` tells that each property gives its number of decimals. The descriptions of `overall` already tell one or two decimals.
- The literal characters of the two patterns became the `\u` escapes of the other patterns. The set of characters is the same.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
