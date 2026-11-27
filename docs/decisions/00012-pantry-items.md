---
description: Pantry items are a third kind of subject. A pantry item is a wine, a beer, a cheese, or a chocolate that users taste and review. One kind with a category holds all of them, with the subject pattern of decision 00004 and four tasting scores. Pantry items share the tag records of decision 00011 with restaurants and recipes.
status: proposed
---

# Motivation
[motivation]: #motivation

Decision 00000 wants one app for all of our food experiences, and names wines and cheeses.
The first release has restaurants and recipes. The backlog has the other foods and drinks under "For the Future".
A glass of wine, a bottle of beer, a piece of cheese, or a bar of chocolate is an experience that is not a visit and not a cook.
We want to remember them, score them, and share them like the other experiences.

Expected outcome:
- a user adds a wine, a beer, a cheese, or a chocolate, and writes a review of each tasting
- a person finds them by category, search, tags, filters, and sorts
- pantry items share the tags of restaurants and recipes, and the tag page shows the pantry items of a tag
- the table and the journal show the tastings next to the visits and the cooks
- users put pantry items in their lists and their favorites

# Scope
[scope]: #scope

In scope:
- the pantry item, its category, and its details
- the tasting, its four scores, its photos, and its date
- the operations, the collections, and the tables of the new kind
- the Pantry section, the pantry item page and form, and "I tasted" in "Add review"
- the table and the journal with three kinds, and the new bottom bar
- the tags of the new kind: the tag table, the tag counts, the tag filters, and the tag page
- the link previews, the dashboard checks, the history, the search, the cleanup jobs, and the rate limits of the new kind

Out of scope:
- one kind for each category. See [Alternatives](#alternatives).
- other categories, such as spirits, cider, coffee, or tea. A later change adds a value to `PantryItemCategory`.
- the vintage of a wine and the alcohol content of a drink. A tag can hold them, such as `2019` or `abv-5-5`.
- tags for each category. All kinds share the tags of decision 00011.
- a barcode scan
- a link from a tasting to the restaurant where it happened
- scales other than 1 to 10, such as the 100 points of wine critics
- the price and the shop of a purchase

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The concepts

- **Pantry item**: a wine, a beer, a cheese, or a chocolate that users taste. It has a name and a category. A producer, an origin, a style, a description, and tags are optional.
- **Category**: `wine`, `beer`, `cheese`, or `chocolate`. Each pantry item has one.
- **Tasting**: a review of a pantry item. One tasting is one occasion. Its date reads "Tasted on 14 Nov 2026".
- **Pantry**: the section of the UI that lists the pantry items, next to Restaurants and Recipes.

Examples:
- Ana opens a bottle of Riesling at home. She selects "Add review", then "I tasted". The search finds no Riesling of that producer. She selects "Add a new pantry item", enters the name, the category "Wine", the producer, the origin "Mosel, Germany", and the style "Riesling", and saves. The form of the tasting opens with the new wine. She selects "Today", four scores, and a title.
- Marco buys a blue cheese at a market. He adds it with the category "Cheese" and the style "Blue", and writes a tasting. Priya tastes the same cheese a week later and adds her own tasting to it.
- Marco types `Raw Milk` in the tags. The tag input suggests the existing `raw-milk`, which a restaurant already has. The tag page of `raw-milk` then shows the restaurant and the cheese.
- Jules opens the Pantry, selects the category "Beer", and sorts by the highest overall.
- Ana saves a chocolate to her favorites. Her journal shows "12 visits · 4 cooks · 3 tastings · 2 lists".

## The scores

Each score is an integer from 1 to 10, as in decision 00005. All categories have the same four scores.

| Score | Question |
|---|---|
| `taste` | how good is the flavor? |
| `aroma` | how good is the smell? |
| `texture` | how good is the feel in the mouth? This is the body of a wine or a beer, the paste of a cheese, and the snap and melt of a chocolate. |
| `value` | is it worth its price? A high score is good. |

- the overall of a tasting is the average of its four scores, as in decision 00005
- the rating of a pantry item counts each reviewer one time, as for restaurants and recipes
- recipes also have `taste` and `value`. The names are the same, but each kind has its own reviews and tables.

## The UI

- the Pantry is at `/pantry`. It has the search, the filters, the sorts, and the list of `PantryItemCard`, like the Restaurants and Recipes pages. A row of chips selects the category.
- `PantryItemCard` is `ParentCard` with the category, the producer, and the origin
- the page of a pantry item is at `/pantry/{id}`. It has the name, the category, the details, the score block, the photos, "Every tasting", and "Add a tasting".
- the form is at `/pantry/new` and `/pantry/{id}/edit`. It has "Name", "Category", "Producer", "Origin", "Style", "Description", and "Tags".
- the form shows "Is it one of these?" with the pantry items of the same name, like the restaurant form shows the restaurants near a place
- "Add review" asks "What did you have?" with the choices "I ate out", "I cooked", and "I tasted". `/reviews/new?kind=pantry` starts with "I tasted".
- "Add a new pantry item" below the search of "Add review" opens the form with `?review=new`. After the save, the UI goes back to the review, as for restaurants and recipes. See decision 00005.
- the table and the journal merge the reviews of the three kinds. `KindSwitch` has "All", "Ate out", "Cooked", and "Tasted" on the table, and "All", "Restaurants", "Recipes", and "Pantry" on a journal. `?kind=pantry` selects the pantry items.
- below `md`, `BottomNav` has "Table", "Restaurants", "Add review", "Recipes", and "Pantry". "You" is in `UserMenu` of `AppHeader`. From `md`, `AppHeader` has the sections "Table", "Restaurants", "Recipes", and "Pantry".
- the tag input of the pantry item form suggests the most used tags of pantry items, like the restaurant form. See decision 00011.
- the tag page at `/tags/{name}` gets a third section, "Pantry", after Restaurants and Recipes. Its `KindSwitch` gets "Pantry".
- each tag on a pantry item page links to the tag page
- the Pantry section uses the lucide icon `shopping-basket`. Wine uses `wine` and beer uses `beer`. Lucide has no icon for cheese or chocolate, so they use `shopping-basket`.

# Implementation Details
[implementation-details]: #implementation-details

## A pantry item

```json
{
  "id": "p2k7m4q9s1v6x3z",
  "name": "Wehlener Sonnenuhr Kabinett",
  "category": "wine",
  "producer": "Weingut Example",
  "origin": "Mosel, Germany",
  "style": "Riesling",
  "description": "Off-dry, with green apple and slate.",
  "tags": ["off-dry", "white"],
  "creator": { "id": "u5n1c8r3t6y0b2e", "name": "Ana" },
  "review_count": 2,
  "own_review_count": 1,
  "rating": { "overall": 8.1, "taste": 8.5, "aroma": 8, "texture": 7.5, "value": 8.5 },
  "cover": null,
  "favorite": false,
  "created": "2026-11-23T18:20:11.410Z",
  "updated": "2026-11-23T18:20:11.410Z"
}
```

| Property | Rules |
|---|---|
| `name` | required. A `Name`. |
| `category` | required. `PantryItemCategory`: `wine`, `beer`, `cheese`, or `chocolate`. |
| `producer` | optional. A `Name`, such as the winery, the brewery, the dairy, or the chocolate maker. |
| `origin` | optional. A `Name`, such as "Mosel, Germany". |
| `style` | optional. A `Name`, such as the grape, the beer style, the type of cheese, or the cocoa content. |
| `description` | optional. A `LongText`. |
| `tags` | optional. Free text that the API changes to tag names. See decisions 00004 and 00011. |

- `PantryItem` also has `creator`, `review_count`, `own_review_count`, `rating`, `cover`, `favorite`, `created`, and `updated`, like `Restaurant`
- `null` removes a stored detail, as for `description`. `category` cannot be `null`.
- a pantry item is small. `listPantryItems` and `listListPantryItems` return complete `PantryItem` items, so the kind has no summary schema like `RecipeSummary`.
- there is no unique rule. Two wines of one producer can have one name and different vintages. The form shows the pantry items of the same name before the save.

## Operations

The tag `pantry` has 21 operations, one for each recipe operation of decision 00004:

| Recipes | Pantry |
|---|---|
| `createRecipe`, `listRecipes` | `createPantryItem`, `listPantryItems` |
| `getRecipe`, `updateRecipe`, `deleteRecipe` | `getPantryItem`, `updatePantryItem`, `deletePantryItem` |
| `listRecipeVersions`, `listRecipePhotos` | `listPantryItemVersions`, `listPantryItemPhotos` |
| `createRecipeReview`, `listRecipeReviews`, `listAllRecipeReviews`, `listReviewerRecipeReviews` | `createPantryItemReview`, `listPantryItemReviews`, `listAllPantryItemReviews`, `listReviewerPantryItemReviews` |
| `getRecipeReview`, `updateRecipeReview`, `deleteRecipeReview` | `getPantryItemReview`, `updatePantryItemReview`, `deletePantryItemReview` |
| `addRecipeReviewPhoto`, `updateRecipeReviewPhoto`, `deleteRecipeReviewPhoto`, `getRecipeReviewPhotoFile` | `addPantryItemReviewPhoto`, `updatePantryItemReviewPhoto`, `deletePantryItemReviewPhoto`, `getPantryItemReviewPhotoFile` |
| `listListRecipes`, `addListRecipe`, `removeListRecipe` | `listListPantryItems`, `addListPantryItem`, `removeListPantryItem` |

- the paths start with `/pantry` and `/pantry/{pantryItemId}`, such as `/pantry/{pantryItemId}/reviews`
- the schemas are `PantryItem`, `PantryItemCreate`, `PantryItemPatch`, `PantryItemCategory`, `PantryItemRating`, `PantryItemReview`, `PantryItemReviewCreate`, and `PantryItemReviewPatch`
- each write operation has an `x-rate-limit`, and each operation with a token has the `403` of decision 00003
- `listPantryItems`:

| | `listPantryItems` |
|---|---|
| sorts | `name`, `overall`, `relevance`, `created`, `updated` |
| filters | `category[eq]`, `category[in]`, `overall`, `tags`, `created`, `updated`, `creator`, `list` |
| search | `name` 10, `tags` 4, `producer` 3, `style` 2, `origin` 2, `description` 1 |

- the filters `tags[has]`, `tags[has_any]`, and `tags[has_all]` reuse the parameters of `listRecipes`. They have `x-tag`, like the bodies of `createPantryItem` and `updatePantryItem`.
- the tag `tags` of decision 00011 gets the new kind. `TagSummary` gets `pantry_item_count`. `kind` of `listTags` gets the value `pantry`.
- the `-count` sort of `listTags` uses the sum of the three counts. `getTag` gets `404` only for a tag without a restaurant, a recipe, or a pantry item.
- `Reviewer` gets `pantry_item_review_count`, `pantry_item_average_overall`, and `pantry_item_count`. `SavedList` gets `pantry_item_count`.
- a review of a pantry item has `pantry_item`, `pantry_item_name`, and the four scores, like a recipe review has `recipe` and `recipe_name`

## The database

A new migration makes the collections, the tables, the triggers, and the indexes. `1794096000_init.go`, `1795737600_tags.go`, and `1796342400_tag_names.go` stay as they are. See decision 00002.

- the migration adds `pantryItemKind` to `migrations/subjects.go` and calls its `create`
- `pantryItemKind` does not join `subjectKinds`. The three older migrations loop over `subjectKinds`. On a new database, they run before the pantry collections exist.
- `newPantryItems` makes `tags` a relation to the `tags` collection with a maximum of 20, like the relation of decision 00011.
- `newParent` gets only the `users` collection. The migration finds the `tags` collection first and gives it to `newPantryItems` in a closure. The signature of `newParent` stays.
- `subjectKind` has one `searchColumn` today. It becomes a list, so the FTS5 row of a pantry item has `producer`, `style`, and `origin`. The restaurant and recipe tables keep their columns.

| Collection | Fields |
|---|---|
| `pantry_items` | `name`, `category`, `producer`, `origin`, `style`, `description`, `tags`, `creator`, `created`, `updated`. `tags` is a relation to `tags`. |
| `pantry_item_reviews` | `pantry_item`, `creator`, `date`, `taste`, `aroma`, `texture`, `value`, `title`, `feedback`, `created`, `updated` |
| `pantry_item_review_photos` | the fields of `recipe_review_photos` with `review` to `pantry_item_reviews` |
| `list_pantry_items` | `list`, `pantry_item`, `created` |
| `pantry_item_versions` | the fields of `recipe_versions` with `pantry_item` |

- `category` is a select field with the four values. The spec and the collection have the same values.
- `pantry_item_reviews` has the generated `overall` column and its indexes, like `recipe_reviews`. See decision 00005.
- `pantry_item_overall`, `pantry_item_tags`, `pantry_items_fts`, and `pantry_item_reviews_fts` have the same columns and triggers as the recipe tables, with the pantry scores
- `pantry_item_tags` has one row for each tag ID of a pantry item. `insertTags` of `subjects.go` already reads the IDs of a relation.

The subject pattern of decision 00004 gets a third column:

| Field | `pantryItemSubject` |
|---|---|
| `Table`, `Parent` | `pantry_items`, `pantry_item` |
| `Reviews`, `Photos` | `pantry_item_reviews`, `pantry_item_review_photos` |
| `ListItems` | `list_pantry_items` |
| `Overall`, `Tags` | `pantry_item_overall`, `pantry_item_tags` |
| `FTS`, `ReviewsFTS` | `pantry_items_fts`, `pantry_item_reviews_fts` |
| `scores` | `taste`, `aroma`, `texture`, `value` |

## The server

- `subjects.PantryItems` joins `subjects.All`. Its `IndexedFields` are `name`, `description`, `producer`, `style`, and `origin`. The search index then adds its hooks.
- the tag code of decision 00011 loops over `subjects.All` for the unused tags and for the search rows of a renamed tag. It then covers pantry items.
- `writeRenames` of `history` loops over the `kinds` of the history package, not over `subjects.All`. `history.PantryItems` must join `kinds`.
- `publicapi` has `pantryItemSubject`, the handlers of the 21 operations through the shared subject code, and `pantryguard.go` for the dashboard saves
- `createPantryItem` and `updatePantryItem` call `resolveTags` in their transaction, like the recipe handlers. A new tag adds no count to the create limit.
- `pantryguard.go` checks the tags of a dashboard save as IDs of existing tags, like `recipeguard.go`
- `tagKinds` and `tagCount` of `tags.go` get the pantry kind. `ListTags` then counts and filters it.
- `tagKinds` matches each value of `kind` exactly. Its `default` branch gives the restaurants today, so `pantry` would get them.
- the category stays pantry code, like the location stays restaurant code
- `history.PantryItems` writes a version of each save. Its properties are the seven properties of the table above.
- the snapshot has the tag names from `recordvalues.Tags`. A rename of a tag in the dashboard writes a version with the source `admin` for each pantry item with the tag.
- `usercleanup` counts the pantry items, the tastings, and the list items of a user. `photocleanup` knows the photo files of the tastings.
- `webui` writes the page meta of `/pantry/{id}` and of its reviews. Without a description, the description is `Tastings of NAME on Olive.` or `A tasting of NAME on Olive.` See decision 00002.
- `tagPage` of `pagemeta.go` also counts the pantry items. The link preview of a tag page names the number of restaurants, recipes, and pantry items.
- the create limit and the edit limit of decision 00003 count the pantry operations, each with its own count

## The UI

- `lib/reviewKinds.ts` gets the kind `pantryItem` with its score fields, its paths, and its API calls. The occasion is "tasting", and the date label is "Tasted on".
- the review form, the review page, the review list, the gallery, and the history page read the kind, as for recipes
- `useEntries` merges three sources on the table and the journal
- `lib/sections.ts` gets the Pantry. `BottomNav` loses "You".
- the reviewer page counts the tastings and lists the pantry items that the reviewer added
- `FavoriteButton` and `SaveToListMenu` take a pantry item
- `TagKind` of `lib/tags.ts` gets `pantry`. `tagSuggester`, `tagCountText`, and `countOf` read `pantryItemCount`. `countOf` gives the recipe count for each kind except `restaurants` today. It matches each kind exactly. The text of a tag with three kinds is "4 restaurants, 1 recipe, and 2 pantry items".
- the pantry item form and the tag filter of the Pantry use `TagInput` with `tagSuggester('pantry')`. The filter has `onlyUsed`.
- `TagPage.vue` gets a section for pantry items with `listPantryItems`, `tags[has]`, and `sort=-overall`. It loads when `KindSwitch` shows it.
- `showRestaurants` and `showRecipes` of `TagPage.vue` use `kind !== 'recipe'` and `kind !== 'restaurant'` today. Each section shows for no kind or for its own kind only.
- `specs/design-system` gets `PantryItemCard`, the category chips, the icons, and the new `BottomNav`

## Changes to other decisions

When this decision is implemented, these decisions change with it:

| Decision | Change |
|---|---|
| 00000 | wines and cheeses are no longer out of scope |
| 00002 | the page meta of the pantry pages |
| 00003 | the rate limits of the pantry operations |
| 00004 | the third column of the subject pattern. The Alternatives row "Wines and cheeses in the first release" points here. |
| 00005 | the tasting scores, "Tasted on", and "I tasted" in "Add review" |
| 00006 | the pantry counts of reviewers and lists, and pantry items in lists |
| 00007 | the photos of tastings |
| 00009 | the Pantry section, the bottom bar, and `KindSwitch` |
| 00011 | `pantry_item_count`, the `pantry` kind of `listTags`, the third section of the tag page, and the tag link preview. The section "Product reviews" goes. The tags work is done. |

The backlog item of the products points here until then.

## Work order

1. the spec, the generated SDKs, the migration, and the server. One pull request.
2. the UI and the design system. A second pull request.

Each pull request has its Go, unit, and end-to-end tests. A test that searches or filters uses a name or tag from `uniqueTag`.

The tests of the first pull request check a new database and a database from before the migration. They also check the tag counts of `listTags`, the tag filters of `listPantryItems`, the unused tags after a pantry delete, and a tag rename with a pantry item.
The migration tests that loop over `subjectKinds` also check `pantryItemKind`. A test list of all kinds holds `subjectKinds` and `pantryItemKind`.
The e2e tests of the second pull request check the pantry section of the tag page and the tag suggestions of the pantry item form.

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| One kind for each category, such as wines, beers, and cheeses | Not selected. Each kind needs 21 operations, five collections, four tables, and its pages. A new category would need all of them again. |
| Two kinds, drinks and foods | Not selected. It is twice the work, and the four scores fit both. |
| The API name `products` | Not selected. It sounds commercial, and a homemade cheese is not a product. |
| The API name `foods` | Not selected. Restaurant reviews have a `food` score. |
| The API name `specialties` | Not selected. The UI writes British English, where the word is "speciality". |
| The API name `items` | Not selected. Each page response has `items`, and `list_items` holds restaurants. |
| The section names "Tastings", "Food & drink", and "Cellar" | Not selected. "Tastings" names the reviews, "Food & drink" is long for the bottom bar, and "Cellar" does not fit cheese. |
| A vintage of a wine | Not selected. It applies to one category. A tag can hold the year. |
| The alcohol content of a drink | Not selected. It applies to two categories. A tag can hold it. |
| An "Other" category | Not selected. A new category is one more value. |
| The category as a tag, such as `wine` | Not selected. A tag is free text. The filters and the icons need a fixed value. |
| Tags of their own for pantry items | Not selected. Decision 00011 shares one tag record for all kinds, so `raw-milk` finds a restaurant and a cheese. |
| The scores taste, aroma, look, and value | Not selected. The look tells little about a cheese or a chocolate. |
| The scores taste, aroma, finish, and value | Not selected. The finish is clear for wine and less clear for cheese. |
| Score names for each category | Not selected. The UI, the docs, and the filters would need four sets of names. |
| Six slots in the bottom bar | Not selected. Each slot is about 62px wide at 375px. |
| One "Explore" slot for the three lists | Not selected. Each list needs one more tap. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
