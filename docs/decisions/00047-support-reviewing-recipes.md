---
description: Let users add recipes with ingredients and steps, review them with their own scores, and save them to lists. Recipes get their own collections, operations, and pages next to the restaurants. The review code of the API works for both kinds.
status: proposed
---

# Motivation
[motivation]: #motivation

The backlog has "Support reviewing cheeses, wines, and recipes".
This decision does the recipes.
The users cook at home and want to tell each other which recipes work.
Today the app can only review restaurants.

The expected outcome:

- A user adds a recipe with a name, a description, tags, the ingredients, the steps, and an optional link to the source.
- Other users review it with scores that fit a recipe, a title, feedback, and photos.
- Users save recipes to their favorites and to their other lists.
- Recipes have the search, sorts, filters, overall score, reviewer pages, link previews, and structured data of restaurants.
- Cheeses and wines can later use the same pattern with little new code.

# Scope
[scope]: #scope

In scope:

- The `recipes`, `recipe_reviews`, `recipe_review_photos`, and `list_recipes` collections and their indexes, triggers, and search tables.
- The operations of the new `recipes` tag and the recipe operations of the `lists` tag in `specs/api/public-openapi.yaml`, and the generated SDKs.
- A refactor of `app/api/internal/publicapi` so the review, photo, gallery, rating, and search code works for both kinds.
- The recipe counts and the recipe average of the reviewer profile.
- The title, link preview, JSON-LD, and sitemap of recipe and recipe review pages.
- The recipe pages of the UI, the recipes on list pages, and the recipe reviews on the "My reviews" and reviewer pages.
- The Go tests and the end-to-end tests of all of the above.

Out of scope:

- Cheeses and wines. They stay in the backlog.
- Amounts and units as separate values of an ingredient. An ingredient is one line of text, for example `2 cans of chickpeas`.
- A scale of the servings that changes the amounts.
- A recipe import from a URL, for example from the schema.org `Recipe` of a web page.
- A shared "all reviews" feed of restaurants and recipes.
- A duplicate check for recipes without a source link.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

The header of the UI has two sections: "Restaurants" and "Recipes".
`/` still opens the restaurants.

Ana opens "Recipes" and selects "Add recipe".
She types "Shakshuka", a short description, and the tags `breakfast` and `eggs`.
She types one ingredient on each line and one step in each step field.
A recipe needs at least one ingredient and one step.
She also gives the link of the blog post, the total time in minutes, and the number of servings.
These three are optional.

The users write the steps in their own words. The source link gives the credit to the original.

If another recipe has the same source link, the API returns `409 Conflict` with the code `source_taken`.
The UI then links to the recipe that exists.
Two recipes without a link can have the same name.

Ben opens the recipe and selects "Write a review".
A recipe review scores four things from 1 to 10:

| Score          | Question                                                       |
| -------------- | -------------------------------------------------------------- |
| `taste`        | How good is the result?                                        |
| `ease`         | How easy is it to make? A high score is easy.                  |
| `instructions` | Are the steps clear and do the times and amounts work?         |
| `value`        | Is the result worth the cost of the ingredients and the time?  |

The API computes the `overall` of the review as the average of the four scores.
The recipe gets a `rating` with the average of each score, like a restaurant.
The `overall` sort uses the same Bayesian average as restaurants. See decision 00029.

Ben adds two photos of his result.
The recipe page shows a cover photo and a gallery of all review photos. See decision 00030.

Ben selects the heart on the recipe card. The recipe goes into his favorites list.
On the recipe page, he also saves it to his list "Weeknight dinners".
A list page shows its restaurants and its recipes in two tabs.

The page of Ben shows his restaurant reviews and his recipe reviews in two tabs.
"My reviews" also has these two tabs.

A shared link of a recipe has its own title and preview.
The page has a schema.org `Recipe` with the ingredients, the steps, and an `aggregateRating`.

# Implementation Details
[implementation-details]: #implementation-details

## The data model

Recipes get their own collections. They do not share the `restaurants`, `reviews`, and `list_items` collections.

| Collection             | Fields                                                                                          |
| ---------------------- | ----------------------------------------------------------------------------------------------- |
| `recipes`              | `name`, `description`, `tags`, `ingredients`, `steps`, `source`, `source_key`, `total_time`, `servings`, `creator`, `created`, `updated` |
| `recipe_reviews`       | `recipe`, `creator`, `taste`, `ease`, `instructions`, `value`, `title`, `feedback`, `created`, `updated` |
| `recipe_review_photos` | The fields of `review_photos`, with a `review` relation to `recipe_reviews`                     |
| `list_recipes`         | `list`, `recipe`, `created`                                                                     |

- `ingredients` and `steps` are JSON fields with an array of strings.
- `creator` of `recipes` has no cascade delete. It becomes empty when the user is deleted, like restaurants.
- `creator` and `recipe` of `recipe_reviews` cascade delete, like `reviews`.
- `list` and `recipe` of `list_recipes` cascade delete, like `list_items`.
  `list_recipes` has the three indexes of `list_items` with `recipe` in the place of `restaurant`.
- `source_key` is the normalized `source` URL. It has a unique index for the rows with a value. The API makes it.
  The normalization makes the scheme and the host lowercase and removes the fragment and a final `/`.
- One review per user and recipe. A second review gets `409` with `review_exists`.

The migration also makes, like the migrations of restaurants:

- `recipe_overall` with the plain and the Bayesian average, and its triggers. See decisions 00012 and 00029.
  The average divides by 4, not 5. `rankingAverage` and `scoreSum` of the migrations need a parameter for this.
- `recipe_tags` and its triggers.
- The generated `overall` column of `recipe_reviews` and its indexes. See decision 00031.
- `recipes_fts` with the name, the description, the tags, and the ingredients. A search for `chickpeas` then finds the recipe.
- `recipe_reviews_fts`. See decision 00004.
- The indexes on `creator, created` and on `recipe, created` that the list operations need.

The rule of AGENTS.md about the drop of a column with triggers applies to these tables also.

### Alternative: one generic collection with a `kind`

One `items` collection with a `kind` of `restaurant` or `recipe` would share more code.
This decision does not do this:

- The details are different for each kind. Restaurants have a location, opening hours, and a price level. Recipes have ingredients, steps, a source, and a time.
- The scores are different for each kind. One `reviews` table would have many empty score columns.
- The triggers and the generated `overall` column would need a `CASE` on the kind.
- The paths and the schemas of restaurants would change. This is a new major version of the API.

The refactor below gives most of the reuse without these costs.

### Alternative: a `recipe` relation in `list_items`

`list_items` could get an optional `recipe` relation next to `restaurant`.
PocketBase stores an empty relation as `''`, not as `NULL`.
The unique index on `list, restaurant` then needs a partial index, and each item needs a check for one relation.
The restaurant queries of the lists would also need a new condition.
`list_recipes` has none of these problems.

## The API

The spec gets the `recipes` tag and version 5.4.0. All changes add operations or properties. No change breaks a client.

| Path                                                               | Operations                                                             |
| ------------------------------------------------------------------ | ---------------------------------------------------------------------- |
| `/recipes`                                                         | `createRecipe`, `listRecipes`                                          |
| `/recipes/{recipeId}`                                              | `getRecipe`, `updateRecipe`, `deleteRecipe`                            |
| `/recipes/{recipeId}/reviews`                                      | `createRecipeReview`, `listRecipeReviews`                              |
| `/recipes/{recipeId}/reviews/{reviewId}`                           | `getRecipeReview`, `updateRecipeReview`, `deleteRecipeReview`          |
| `/recipes/{recipeId}/reviews/{reviewId}/photos`                    | `addRecipeReviewPhoto`                                                 |
| `/recipes/{recipeId}/reviews/{reviewId}/photos/{photoId}`          | `updateRecipeReviewPhoto`, `deleteRecipeReviewPhoto`                   |
| `/recipes/{recipeId}/reviews/{reviewId}/photos/{photoId}/{variant}`| `getRecipeReviewPhotoFile`                                             |
| `/recipes/{recipeId}/photos`                                       | `listRecipePhotos`                                                     |
| `/recipe-reviews`                                                  | `listAllRecipeReviews`                                                 |
| `/reviewers/{reviewerId}/recipe-reviews`                           | `listReviewerRecipeReviews`                                            |
| `/lists/{listId}/recipes`                                          | `listListRecipes`                                                      |
| `/lists/{listRef}/recipes/{recipeId}`                              | `addListRecipe`, `removeListRecipe`                                    |

New schemas: `Recipe`, `RecipeCreate`, `RecipePatch`, `RecipeList`, `RecipeRating`, `RecipeReview`, `RecipeReviewCreate`, `RecipeReviewPatch`, `RecipeReviewList`, `RecipePhoto`, `RecipePhotoList`, `TasteScore`, `EaseScore`, `InstructionsScore`, `ValueScore`, `Ingredient`, `Ingredients`, `Step`, `Steps`, `TotalTime`, and `Servings`.
`Photo`, `PhotoPatch`, `Score`, `Tags`, `Name`, `Title`, `LongText`, and `Website` stay shared.

The recipe properties:

| Property      | Schema                                                                                         | Create   |
| ------------- | ---------------------------------------------------------------------------------------------- | -------- |
| `name`        | `Name`                                                                                         | required |
| `description` | `LongText`                                                                                     | optional |
| `tags`        | `Tags`                                                                                         | optional |
| `ingredients` | `Ingredients`: 1 to 100 items. An `Ingredient` has 1 to 200 characters, no line breaks, and `x-trim: true`. | required |
| `steps`       | `Steps`: 1 to 50 items. A `Step` has 1 to 2,000 characters, the rules of `LongText`, and `x-trim: true`.   | required |
| `source`      | `Website`                                                                                      | optional |
| `total_time`  | `TotalTime`: an integer of minutes from 1 to 10,080                                            | optional |
| `servings`    | `Servings`: an integer from 1 to 100                                                           | optional |

`RecipePatch` cannot clear `ingredients` and `steps` with `null`. A patch replaces the complete array.
`Recipe` also has `creator`, `review_count`, `rating`, `cover`, `favorite`, `created`, and `updated`, like `Restaurant`.

Rules that the new operations follow:

- Each write operation has an `x-rate-limit` and a `429` response. Creates use `create`, photo uploads use `photo-upload`, and the others use `edit`. See decision 00036.
- Each operation with a token needs an approved user and has a `403` response. See decision 00034.
- Only the creator changes or deletes a recipe. Reviews of other users block the delete with `other_reviews`.
  Reviewers can change a recipe without a creator. See decision 00016.
- `name`, `title`, the ingredients, the steps, and the alt texts have `x-trim: true`. `Tag` keeps `x-nfc: true`. See decisions 00040 and 00041.
- The `Problem` code enum gets `source_taken` and `list_recipe_limit`.
  The `source_taken` problem has the ID of the other recipe in `recipe`.
  `list_recipe_limit` is a recipe above the 500 recipes of a list.
- `listRecipes` has the sorts, the search, and the filters of `listRestaurants` without the location and the price level. It has `list[eq]` like decision 00043. It adds a `total_time` filter.
- The review list operations have the review sorts and filters with the four recipe scores.

`Reviewer` gets `recipe_count`, `recipe_review_count`, and `recipe_average_overall`.
`review_count`, `average_overall`, and `restaurant_count` keep their meaning. They count only restaurants.

`SavedList` gets `recipe_count`. `item_count` keeps its meaning. It counts only restaurants.
`listReviewerLists` gets a `recipe[eq]` filter. It then gives `has_recipe`, like `has_restaurant`.

The `info.description` of the spec says "restaurants and recipes".

## The refactor of the review code

Most review code of `app/api/internal/publicapi` has the restaurant in its names and its SQL.
A first change moves it onto a small descriptor of a kind:

```go
// subject is a kind of thing that users review.
type subject struct {
	collection   string   // restaurants
	reviews      string   // reviews
	photos       string   // review_photos
	listItems    string   // list_items
	parentField  string   // restaurant
	scores       []string // space, service, food, price, cleanliness
	overallTable string   // restaurant_overall
	tagsTable    string   // restaurant_tags
	searchTable  string   // restaurants_fts
	pathPrefix   string   // /v1/restaurants
}
```

These parts then take a `subject`:

- `reviewPage`, `reviewInput`, `findReview`, `scoreComparisons`, and the names of `restaurantNames` in `reviews.go`.
- `ratings`, `overallSort`, and `joinOverall` in `ratings.go`.
- The photo upload, the photo files, `PhotoFilePath`, `reviewPhotos`, the gallery, and the covers in `photos.go` and `gallery.go`.
- `searchPage` in `search.go` and the hooks of `app/api/internal/searchindex`.
- `guardReviewOverall` in `reviewschema.go`.
- `restaurantChangeForbidden` and `countReviewsOfOtherUsers` in `restaurants.go`.
- `itemCounts`, `listsWith`, `favoriteIDs`, `inList`, `findItem`, and the add and remove code in `lists.go`.

The strict server methods stay thin. Each one changes its request object, calls the shared code with its subject, and changes the result to its response object.
The restaurant behavior does not change. The Go tests of today prove this before the recipe code starts.

The restaurant details stay restaurant code: the location, `place_id`, the places, the maps, and the price level.

## The other server parts

- `restaurantguard.go` gets a recipe guard. A recipe save in the dashboard gets the checks of `RecipeCreate`. See decision 00016.
- `photocleanup` also checks the folder of `recipe_review_photos`. See decision 00009.
- `usercleanup` also keeps a user with recipes or recipe reviews. See decision 00034.
- `reviewers.go` counts the recipes and the recipe reviews, and computes the recipe average.
- `webui/pagemeta.go` writes the title and the preview of `/recipes/{id}` and `/recipes/{id}/reviews/{id}`. See decision 00013.
  `/recipes/new` does not match a recipe. The code has the same rule for `/restaurants/new`.
- `webui/structureddata.go` writes a schema.org `Recipe` with `recipeIngredient`, `recipeInstructions` as `HowToStep` items, `aggregateRating`, `recipeYield`, `totalTime` as an ISO 8601 duration, and `isBasedOn` for the source.
  A recipe review page writes a `Review` with an `itemReviewed` of type `Recipe`. See decision 00021.
- `webui/sitemap.go` adds the recipes and the recipe reviews.

## The UI

- `AppHeader.vue` gets the links "Restaurants" and "Recipes". The design system of `AppHeader` gets the same change.
- New routes and pages: `/recipes` with `RecipesPage.vue`, `/recipes/new` and `/recipes/:recipeId/edit` with `RecipeFormPage.vue`, and `/recipes/:recipeId` with `RecipePage.vue`.
- `RecipeFormPage.vue` has a text area for the ingredients with one ingredient on each line. The steps are a list of text areas that the user can add, remove, and move.
- `RecipePage.vue` shows the ingredients as a list and the steps as a numbered list.
- `/recipes/:recipeId/reviews/new`, `/recipes/:recipeId/reviews/:reviewId`, and its `/edit` use `ReviewFormPage.vue` and `ReviewPage.vue` with a `kind` prop.
- `lib/reviews.ts` gets the score set, the labels, and the help texts of each kind. `ReviewCard.vue`, `ScoreSummary.vue`, `ReviewList.vue`, and `ReviewPhotos.vue` read the score set and do not name a score.
- `lib/drafts.ts` gets a key for each kind. A draft of a restaurant review and a recipe review do not mix.
- `RecipeCard.vue` is like `RestaurantCard.vue` with the `FavoriteButton`, and without the location and the price level.
- `SaveToListMenu.vue`, `FavoriteButton.vue`, and `lib/lists.ts` take a kind.
- `ListPage.vue` gets tabs for the restaurants and the recipes of the list. `ReviewerLists.vue` shows both counts.
- `RestaurantGallery.vue` becomes a gallery for both kinds.
- `MyReviewsPage.vue` and `ReviewerPage.vue` get tabs for restaurant reviews and recipe reviews. `ReviewerPage.vue` also lists the recipes of the user and shows the recipe average.
- The total time shows in hours and minutes with the rules of `specs/design-system/formatting.md`.

## The tests

- Go: the tests of the refactor are the tests of today. They must pass without a change.
- Go: new tests for each recipe operation, the ingredient and step limits, the `source_taken` conflict, the owner rules, the recipe guard, the triggers of `recipe_overall` and `recipe_tags`, the search index, both cleanup jobs, the reviewer counts, the page meta, the JSON-LD, and the sitemap.
- Go: new tests for the recipe list operations, `list_recipe_limit`, `has_recipe`, `favorite` of `Recipe`, and `listRecipes` with `list[eq]`.
- End-to-end: `tests/e2e/recipes.spec.ts` adds, changes, searches, filters, and deletes a recipe. It also writes, changes, and deletes a recipe review with a photo.
  The searches use `uniqueTag`.
- End-to-end: `lists.spec.ts` adds a recipe to the favorites and to a custom list, and opens the recipes tab of the list page.
- End-to-end: `accessibility.spec.ts` gets a test for the recipes page, the recipe page, and the recipe form. `link-previews.spec.ts` gets a recipe page.

## The steps

Each step is one pull request. Each step ends with `make lint test-app` and the tests of its part.

1. **Refactor the review and list code.** Add the `subject` descriptor and move restaurants onto it. No change of behavior, the spec, or the UI.
   Check with `make test-app`, `make test-race`, and `make test-e2e PROJECT=chromium`.
2. **Add the recipe API.** Change the spec to 5.4.0. Run `make lint-spec` and `make generate`. Add the migration, the handlers, the hooks, the cleanup changes, the webui parts, and the Go tests.
   Update AGENTS.md: the `recipes` tag, the new collections in `app/api/migrations`, and the new tables.
   Check with `make check-generate`, `make test-app`, and `make test-coverage`.
3. **Add the recipes to the lists API.** Add `list_recipes`, the list operations of recipes, `favorite`, `recipe_count`, `has_recipe`, and `list[eq]` of `listRecipes`, with their Go tests.
   Check with `make check-generate`, `make test-app`, and `make test-coverage`.
4. **Add the recipe UI.** Add the pages, the components, the list changes, the design system changes, and the end-to-end tests.
   Check with `make lint-ui lint-eslint test-ui` and `make test-e2e PROJECT=chromium E2E_FILE=e2e/recipes.spec.ts`. Also run `E2E_FILE=e2e/lists.spec.ts`. CI runs all browsers.
5. **Close the decision.** Set the status of this decision to `implemented`. Change the backlog item to "Support reviewing cheeses and wines".

Steps 2 and 3 can be in production without step 4. The UI then has no link to the new operations.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None. The answers of 2026-11-18:

- A recipe needs its ingredients and its steps. The source link is optional.
- A recipe review scores `taste`, `ease`, `instructions`, and `value`.
- The reviewer profile and the lists keep the restaurant counts and get separate recipe counts.
- Recipes are in lists and have the favorites heart in the first version.
- The only duplicate check is the unique source link.
