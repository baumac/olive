---
description: Restaurants and recipes are the subjects that users review. Both kinds share one pattern in the API, the database, and the UI. Each approved user can change each subject, only the creator can delete one, and each save writes a version that approved users can undo.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00000 wants one app to save each food experience.
The first release has two kinds of subjects: restaurants and recipes.
Wines and cheeses can come later. The backlog has them under "For the Future".

A restaurant needs a place. Without it, two restaurants with one name in different cities look the same, and the maps of decision 00008 have no point.
People who look at a restaurant also want its website, its phone, its opening hours, and how much it costs.

A recipe needs its ingredients and its steps, so a user can cook it again.
The users write the steps in their own words. The source link gives the credit to the original.

Two users can add the same restaurant or recipe. Its reviews then split between two pages.

The users are a small group of friends and family. A superuser approves each of them. See decision 00003.
The facts of a restaurant change over time, such as a new phone number or new opening hours.
A wrong name, place, source, ingredient, or step is more common than a misuse.
If only the creator could fix them, a creator who stops using the app would freeze them.

Each user can then change each subject. The group accepts the risk of vandalism.
A history of each change lets each approved user see who changed what, and undo a wrong change.
A delete stays with the creator. The reviews of other users are about the subject, so they block its delete.

Expected outcome:
- users add restaurants with a location from a place search, and recipes with ingredients and steps
- users find restaurants and recipes by search, tags, filters, and sorts
- the same place cannot get a second restaurant. The same source cannot get a second recipe.
- tags can have the letters and digits of all scripts
- both kinds share the review, photo, rating, search, and list code. A later kind uses the same pattern.
- each approved user can change each restaurant and each recipe
- two users who change different fields at the same time keep both changes
- only the creator deletes a subject, and only while no other user has a review of it
- each change makes a version with the user and the time
- an approved user undoes one change in the UI. A superuser is not necessary.
- an undo is a normal change. The history loses nothing.

# Scope
[scope]: #scope

In scope:
- the `restaurants` and the `recipes` collections and their properties
- the create, read, update, delete, and list operations of both kinds
- the duplicate checks of a place and of a source
- the tags and their tables
- the `subject` pattern of the API, the database, and the UI
- the rules of `updateRestaurant`, `updateRecipe`, `deleteRestaurant`, and `deleteRecipe`
- the subjects without a creator
- the `restaurant_versions` and `recipe_versions` collections and their record hook
- `listRestaurantVersions` and `listRecipeVersions`
- the edit buttons, the history page, and the undo of the UI

Out of scope:
- the reviews and the ratings. Decision 00005 has them.
- the lists and the favorites. Decision 00006 has them.
- the photos, the cover, and the gallery. Decision 00007 has them.
- the place search, the maps, and the distance sort. Decision 00008 has them.
- wines and cheeses in the first release
- a merge of duplicate restaurants or recipes. Superusers do it in the dashboard.
- a duplicate check for recipes without a source, and for restaurants at an address without a place
- opening hours that the app can read, such as "open now"
- amounts and units as separate values of an ingredient, and a scale of the servings
- a recipe import from a URL. A superuser imports old data. See decision 00003.
- the rules of reviews, photos, and lists. Only their creator changes them.
- a history of reviews, photos, and lists
- a moderator role and a report flow. The backlog has reports under "Won't do".
- a check of `updated` before a save. Two changes of the same field keep the last one.
- the restore of a deleted subject. The delete also deletes its versions.
- the name of the superuser of a dashboard change. The version shows "an admin".
- a delete of old versions. The app keeps each version for the life of its subject.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## A restaurant

```json
{
  "id": "k3x9m2p7q1w8z4a",
  "name": "Can Solé",
  "description": "Seafood near the beach",
  "tags": ["seafood", "paella"],
  "location": {
    "latitude": 41.3766,
    "longitude": 2.1893,
    "address": "Carrer de Sant Carles 4, 08003 Barcelona, España",
    "place_id": "osm:N1234567890"
  },
  "website": "https://cansole.example",
  "phone": "+34 932 21 50 12",
  "opening_hours": "Tu-Sa 13:00-16:00, 20:00-23:00\nSu 13:00-16:00",
  "price_level": 3,
  "creator": { "id": "u5n1c8r3t6y0b2e", "name": "Ana" },
  "review_count": 2,
  "own_review_count": 1,
  "rating": { "overall": 7.6, "space": 7.5, "service": 8.5, "food": 9, "price": 5.5 },
  "cover": null,
  "favorite": false,
  "created": "2024-11-04T01:26:55.002Z",
  "updated": "2024-11-04T01:26:55.002Z"
}
```

| Property | Rules |
|---|---|
| `name` | required. A `Name` of 1 to 255 characters. |
| `location` | required on create. `latitude` and `longitude` in degrees, an `address` of 1 to 255 characters, and an optional `place_id`. |
| `description` | optional. A `LongText` of at most 10,000 characters. |
| `tags` | optional. See [Tags](#tags). |
| `website` | optional. An `http` or `https` URL of at most 2,048 characters, without spaces and hidden characters. |
| `phone` | optional. 3 to 20 digits. Spaces, parentheses, points, slashes, and hyphens can separate them. A `+` can come first. At most 40 characters. |
| `opening_hours` | optional. A text of at most 500 characters with line breaks. |
| `price_level` | optional. 1 is inexpensive, 2 moderate, 3 expensive, and 4 very expensive. |

- `place_id` is the ID of the place at the geocoder. It starts with the source, such as `osm`.
- an update replaces the complete `location`. `null` is not allowed, so a location cannot be removed.
- `null` removes a stored detail, as for `description`
- the price level is not the `price` score of a review. The score tells if the price was fair. The level tells how much a meal costs. The API has no currency.
- a property without a value is not in the response
- `creator` is `null` after the user is deleted
- `review_count`, `own_review_count`, and `rating` come from the reviews. See decision 00005.
- `cover` is the first photo of the gallery. See decision 00007.
- `favorite` tells if the restaurant is in the favorites list of the user of the token. See decision 00006.

## A recipe

| Property | Rules |
|---|---|
| `name` | required. A `Name`. |
| `ingredients` | required. 1 to 100 items. An `Ingredient` has 1 to 200 characters and no line breaks, such as `2 cans of chickpeas`. |
| `steps` | required. 1 to 50 items. A `Step` has 1 to 2,000 characters with the rules of `LongText`. |
| `description` | optional. A `LongText`. |
| `tags` | optional. See [Tags](#tags). |
| `source` | optional. A `Website`, such as the link of a blog post. |
| `total_time` | optional. An integer of minutes from 1 to 10,080. |
| `servings` | optional. An integer from 1 to 100. |

- a patch replaces the complete `ingredients` or `steps`. `null` cannot clear them.
- `Recipe` also has `creator`, `review_count`, `own_review_count`, `rating`, `cover`, `favorite`, `created`, and `updated`, like `Restaurant`
- `listRecipes` and `listListRecipes` return `RecipeSummary` items. A summary has all properties of `Recipe` except `ingredients` and `steps`.
  A recipe can have 100 ingredients and 50 long steps, so a page of complete recipes could have about 13 MB.
- `getRecipe`, `createRecipe`, and `updateRecipe` return the complete `Recipe`

## Operations

| Path | Restaurants | Recipes |
|---|---|---|
| `/restaurants`, `/recipes` | `createRestaurant`, `listRestaurants` | `createRecipe`, `listRecipes` |
| `/restaurants/{restaurantId}`, `/recipes/{recipeId}` | `getRestaurant`, `updateRestaurant`, `deleteRestaurant` | `getRecipe`, `updateRecipe`, `deleteRecipe` |

The list operations:

| | `listRestaurants` | `listRecipes` |
|---|---|---|
| sorts | `name`, `overall`, `relevance`, `distance`, `created`, `updated` | `name`, `overall`, `relevance`, `total_time`, `created`, `updated` |
| filters | `overall`, `price_level`, `near`, `within`, `tags`, `created`, `updated`, `creator`, `list` | `overall`, `total_time`, `tags`, `created`, `updated`, `creator`, `list` |
| search | `name` 10, `tags` 4, `address` 2, `description` 1 | `name` 10, `tags` 4, `ingredients` 2, `description` 1 |

- the default sort of the API is `-created`. With `search`, it is `relevance`. The UI has its own default. See decision 00008.
- decision 00001 has the rules of the filters, the search, and the sorts
- decision 00005 has the `overall` sort and filter. Decision 00008 has `near`, `within`, and `distance`. Decision 00006 has `list[eq]`.
- the `total_time` sort puts recipes without a `total_time` last in both directions
- a restaurant without a `price_level` matches no `price_level` filter, like a restaurant without reviews and the `overall` filter

## Tags

- restaurants and recipes have tags. Reviews have no tags. See decision 00005.
- `tags` has at most 20 unique items
- an empty list is no tags. The API stores and returns it as `null`, in a create, a patch, and a dashboard save. A patch from `null` to `[]` changes nothing and makes no version.
- a tag has 1 to 63 characters. It has words with single hyphens between them.
- each word starts with a lowercase letter, a letter without case, or a digit of any script. Combining marks, such as the vowel signs of Devanagari, can follow.
- the API changes each tag to Unicode normalization form C first. `cafe` with a separate accent U+0301 becomes `café`.

| Tag | Result |
|---|---|
| `café`, `2-für-1`, `ψάρι`, `ラーメン`, `चाय` | valid |
| `café` and `cafe` + U+0301 in one request | `422`. The two values are the same tag. |
| `Café`, `Ψάρι` | `422`. Uppercase letters are not allowed. |
| `pasta` + U+FE0F, U+3164 | `422`. A default ignorable code point is invisible. |
| `dim` + U+00A0 + `sum` | `422`. A no-break space is not a letter. |

- the tag filters change their values in the same way. A filter with an invalid tag gets `400`.
- the search finds `café` with `cafe`. The tag filters compare whole tags, so `tags[has]=cafe` does not find `café`.
- the UI suggests a valid form for an invalid tag, such as `crème-brûlée` for `Crème Brûlée`, or `istanbul` for `İstanbul`

## One restaurant for one place

A place with a `place_id` is a restaurant, a cafe, a bar, or a similar place of OpenStreetMap.
An address, a building, a street, or a city has no `place_id`. One building can have many restaurants.

- only one restaurant can have a `place_id`. A create or an update with the `place_id` of a different restaurant gets `409` with `place_taken`.
- the problem has the ID of the other restaurant in `restaurant` and `"property": "location"`. The UI shows a link to it.
- after the user selects a location, the form shows the restaurants at most 75 meters from it under "Is it one of these?". The user can still save.

The hard rule does not stop each duplicate. A user can select the address instead of the restaurant, or type the location.
The soft rule shows the duplicate in these cases.
A rule on the name and the distance would refuse real restaurants, such as two bars in one food court.

## One recipe for one source

- only one recipe can have a source. A create or an update with the source of a different recipe gets `409` with `source_taken`.
- the problem has the ID of the other recipe in `recipe` and `"property": "source"`. The UI links to it.
- the API compares the sources without the case of the scheme and the host, without the fragment, and without a final `/`. `https://DAL.example/tarka/` is the source `https://dal.example/tarka`.
- two recipes without a source can have the same name

## The UI

- decision 00009 has the navigation and the route of `/`. `/restaurants` and `/recipes` open the lists of the subjects.
- the restaurant form has "Name", "Location", "Description", "Tags", and a "Details" group with "Website", "Phone", "Opening hours", and "Price level"
  - "Location" has the place search of decision 00008. The selected place fills the empty name and the empty details.
  - "Website" gets `https://` for a value without a scheme, such as `cansole.example`
- the restaurant page shows the address with a pin icon, then the website as a link with the host name, the phone as a `tel:` link, the opening hours with their line breaks, and the price level as a word
- the recipe form has a text area for the ingredients with one ingredient on each line. The steps are a list of text areas that the user adds, removes, and moves.
- the recipe page shows the ingredients as a list and the steps as a numbered list. The total time shows in hours and minutes.
- the "Filters" popover of the restaurants page has a "Price level" select with ranges, such as "Moderate or less"

## Editing and history

### The rules

| Action | Who | Condition |
|---|---|---|
| Create | each approved user | none |
| Change each property | each approved user | none |
| Delete | the creator | no other user has a review |

- for a subject without a creator, "the creator" is each user who has a review of it
- a user who cannot delete gets `403` with `not_creator`. A delete with reviews of other users gets `409` with `other_reviews`.
- the delete also deletes the reviews and the photos of the creator. The confirm dialog says so.
- the duplicate checks of [One restaurant for one place](#one-restaurant-for-one-place) and [One recipe for one source](#one-recipe-for-one-source) still apply to a change of the location or the source
- a superuser can change each field of each record in the dashboard

Example: Ana adds "Luigis" at a wrong address. Joe and Eva review it.
Eva changes the name to "Luigi's" and selects the correct place. The reviews then show the new name, and the search finds them by it.
Ana cannot delete the restaurant. Joe and Eva cannot delete it either. They are not the creator.

The edit forms send only the properties that differ from the loaded record.
Ana changes the phone while Joe changes the opening hours. Both saves keep their change.

### Vandalism

- only approved users can change a subject. A superuser can take the approval back.
- the history names the editor of each change, and an undo fixes it in the UI
- the edit limit of decision 00003 is 300 changes of each operation per user and hour
- the backups of decision 00010 also have the old values

### The history

Ana adds "Can Solé". This is version 1. Joe changes the phone. This is version 2.
Eva clears the description by mistake and adds the tag `tapas`. This is version 3.
Joe selects "History" on the restaurant page. The page shows the newest version first, with the editor, the time, and the old and new value of each changed property.

- a version from the dashboard or a migration shows "An admin"
- a version of a deleted user shows "Deleted user"
- only approved users can read the history. A wrong private value, such as a phone number, then does not stay public in old versions.

### Undo

Joe selects "Undo this change" on version 3. The edit form opens with `?undo=3`.
The description is "Small tapas bar" again, and the tags are empty again. A note says "Undo of version 3. Check the values, then save."
Joe can change the values. The save makes version 4 with the editor Joe.

- an undo always fixes forward. Version 3 stays, and version 4 has no link to it.
- an undo keeps a property that a newer version changed again. The note names it with the editor and the version.
- an undo uses `updateRestaurant` or `updateRecipe`, so all their checks apply. A place that another restaurant took since then gets `409`.

# Implementation Details
[implementation-details]: #implementation-details

## Collections

| Collection | Fields |
|---|---|
| `restaurants` | `name`, `description`, `tags`, `latitude`, `longitude`, `address`, `place_id`, `website`, `phone`, `opening_hours`, `price_level`, `creator`, `created`, `updated` |
| `recipes` | `name`, `description`, `tags`, `ingredients`, `steps`, `source`, `source_key`, `total_time`, `servings`, `creator`, `created`, `updated` |

- `creator` is an optional relation without a cascade delete. PocketBase removes the user ID in the delete transaction of the user. A test checks this behavior.
- each restaurant has a location. `address` is a required field. `latitude` and `longitude` are not required fields, because a required number field refuses 0. 0 is a valid coordinate.
- `idx_restaurants_place_id` is a unique index on `place_id` where it is not empty. It also stops a duplicate from the dashboard.
- `idx_restaurants_latitude` on `(latitude, longitude)` serves the `near` and `within` filters
- `price_level` is 0 for "not set". PocketBase makes each number column `NUMERIC DEFAULT 0 NOT NULL`.
  `priceLevelColumn` is `NULLIF([[restaurants.price_level]], 0)`, so a comparison matches no restaurant without a level.
  The column has no index. It has only four values.
- the text fields of the details have the patterns of the spec. The dashboard then cannot save a value that the API refuses.
- `ingredients` and `steps` are JSON fields with an array of strings
- `source_key` is the normalized source. It has a unique index for the rows with a value. A record hook sets it on each save, so a save in the dashboard also gets the duplicate check.
- both collections have indexes for the `created` sort, the `name` sort, and the `creator` filter, such as `idx_restaurants_created`, `idx_restaurants_name`, and `idx_restaurants_creator_created`
- each create and update checks the place or the source and saves in one transaction. See decision 00002.

## Tags

- `restaurant_tags` and `recipe_tags` have one row for each tag of a subject. Their primary key is `(tag, restaurant)` or `(tag, recipe)`, with an index on the subject.
- SQLite triggers keep these tables equal to the records. A subject that is made, changed, or deleted changes its rows.
- `tags[has_any]=a&tags[has_any]=b` adds `restaurants.id IN (SELECT restaurant FROM restaurant_tags WHERE tag IN ('a', 'b'))`. `tags[has]` and each `tags[has_all]` value add one such condition.
- the tables keep only the strings of a JSON array. A tag that occurs two times has one row.
- a migration that drops `tags` must drop the triggers first. SQLite refuses the drop.
- the `Tag` pattern is `^[\p{Ll}\p{Lm}\p{Lo}\p{Nd}][\p{Ll}\p{Lm}\p{Lo}\p{Mn}\p{Mc}\p{Nd}]*(-[\p{Ll}\p{Lm}\p{Lo}\p{Nd}][\p{Ll}\p{Lm}\p{Lo}\p{Mn}\p{Mc}\p{Nd}]*)*$`. Go `regexp` and JavaScript with the `u` flag both read it.
- `\p{Lu}` and `\p{Lt}` are not in the classes, so the pattern refuses uppercase letters. `\p{Lm}` is in them for the `ー` of `ラーメン`.
- `Tag` has `format: tag`. `checkTagFormat` of `openapi.go` refuses the code points with the property `Default_Ignorable_Code_Point`. Go has no table of this property, so it uses `Other_Default_Ignorable_Code_Point` and `Variation_Selector`. The pattern refuses the rest, which are in `Cf`.
- the validation comes after the normalization. `uniqueItems` and `maxLength` then count the changed tags.
- `normalizeQuery` changes the query values of a schema with `x-nfc`. The validation and the handler read the changed query.
- the UI uses `\p{Default_Ignorable_Code_Point}`. `maxLength` counts code points in the UI too. `suggestTag` removes the combining dot that `toLowerCase` adds to `i` for `İ`.

## The subject pattern

`app/api/internal/subjects` has the table names of each kind in `subjects.Tables`. `subjects.All` lists the kinds.
`subject` of `app/api/internal/publicapi/subject.go` adds the values that only the Public API uses.

| Field | `restaurantSubject` | `recipeSubject` |
|---|---|---|
| `Table`, `Parent` | `restaurants`, `restaurant` | `recipes`, `recipe` |
| `Reviews`, `Photos` | `reviews`, `review_photos` | `recipe_reviews`, `recipe_review_photos` |
| `ListItems` | `list_items` | `list_recipes` |
| `Overall`, `Tags` | `restaurant_overall`, `restaurant_tags` | `recipe_overall`, `recipe_tags` |
| `FTS`, `ReviewsFTS` | `restaurants_fts`, `reviews_fts` | `recipes_fts`, `recipe_reviews_fts` |
| `scores` | `space`, `service`, `food`, `price` | `taste`, `ease`, `instructions`, `value` |
| `search`, `reviewSearch` | the column weights of the search | the column weights of the search |

- the review, photo, gallery, rating, search, tag, and list code takes a `subject`
- the shared code returns a `problem` for a 4xx result. Each strict server method changes its request, calls the shared code, and changes the result to its own response type.
- a status without a response of the operation is an error and gives `500`
- `app/api/internal/searchindex` adds its hooks for each kind of `subjects.All`
- the location, the places, the maps, and the price level stay restaurant code. The ingredients, the steps, the source, and the times stay recipe code.
- `withDetails` copies the fields of `RecipeSummary` into `Recipe` by their names. A new summary field then also reaches `getRecipe`.
- a new kind needs an entry in `subjects.All`, its collections, its tables, its triggers, its search tables, and its spec operations

## Dashboard saves

- `restaurantguard.go` and `recipeguard.go` give a save in the dashboard the checks of `RestaurantCreate` and `RecipeCreate`
- the guard runs on the PocketBase create and update requests. A scripted import with a superuser token gets the same checks.

## The UI

- `lib/reviewKinds.ts` has the score fields, the paths, and the API calls of each kind. The review form, the review page, the review list, and the gallery read them and name no score.
- `ParentCard.vue` has the parts of a card that both kinds share. `RestaurantCard.vue` adds the location. `RecipeCard.vue` adds the total time.
- `lib/restaurantDetails.ts` has the website change, the `tel:` link, and the names of the price levels
- `RestaurantDetails` shows the details. The website link has `rel="nofollow ugc noopener noreferrer"`, because a user typed the URL.
- `priceLevelRanges` of `restaurantFilters.ts` has the options of the price level select. `useListQuery` ignores a link value that is not an option.
- the form trims the phone and the opening hours before it checks them. A draft keeps the details.

## Editing and history

### API

- `UpdateRestaurant` and `UpdateRecipe` have no creator check. The place check and the source check stay in the transaction of the save.
- `deleteForbidden` of `restaurants.go` is the creator check of the delete. A subject with a creator needs no query. A subject without one needs a count of the reviews of the user.
- `countReviewsOfOtherUsers` blocks the delete. `idx_reviews_restaurant_creator` serves both counts.
- the delete removes the reviews of the subject and then the subject, in the transaction of the checks. The parent relation of reviews has no cascade.
- the photo records go with the cascade of decision 00007. PocketBase deletes the files after the commit.
- the triggers and the `searchindex` hooks remove the index rows. A rename updates the parent name in the review search tables.

### Versions

| Field | `restaurant_versions` and `recipe_versions` |
|---|---|
| parent | `restaurant` or `recipe`, a required relation with cascade delete |
| `number` | starts at 1 for each parent. A unique index on the parent and `number`. |
| `action` | `baseline`, `create`, or `update` |
| `source` | `user` or `admin` |
| `editor` | an optional relation to `users` without cascade |
| `changed` | a JSON array of the changed property names |
| `snapshot` | a JSON object with all properties after the save, in the shape of `RestaurantCreate` or `RecipeCreate` |
| `created` | autodate |

- the API rules are `null`. Only superusers use the collections in the dashboard.
- a migration that drops or renames a property of `restaurants` or `recipes` also changes the `snapshot` of their versions
- `app/api/internal/history` binds `OnRecordCreateExecute` and `OnRecordUpdateExecute` of both collections. It runs the save and the version write in one transaction, for each save.
- `app/api/internal/recordvalues` gives the API values of the fields. The Public API and `Kind.Snapshot` use the same functions.
- a save that changes no property writes no version. A change of only `creator` or `source_key` writes nothing.
- an update of a subject without a version first writes a baseline from the stored record. The baseline is not in the list. It is the base of the comparison for the next version.
- a Public API save calls `SaveWithContext` with `history.WithEditor`. A save without it gets the source `admin`. The dashboard of PocketBase 0.40 does not send the request context to the save.
- `TestHistoryHasTheCreateProperties` fails when a create schema has a property that `Kind.Properties` does not have

### Operations

- `GET /restaurants/{restaurantId}/versions` and `GET /recipes/{recipeId}/versions` need a token and an approved user. They have no rate limit.
- the order is `-number`, with `limit` and `cursor`
- a version has `id`, `number`, `created`, `action`, `source`, `editor`, `before`, and `after`
- `before` and `after` have only the changed properties. `before` is empty for a create. Their schemas have the properties of the patch schemas.
- `versions.go` gets a page of versions and the version before the oldest item. It takes `before` and `after` from the snapshots and `changed`.

### The UI

- `parentAccess` of `app/ui/src/lib/parents.ts` returns `canEdit`, `isCreator`, and `hasOtherReviews`. The delete needs `isCreator` and no reviews of other users.
- another user has a review when `review_count` minus `own_review_count` is more than 0
- the restaurant and recipe pages show "Edit" and "History" for each signed in user. The router guard of decision 00003 stops a user who waits.
- the creator of a subject with reviews of other users sees "You cannot delete it" in place of the delete button
- `changedPatch` makes a patch with only the changed properties
- `HistoryPage.vue` has the routes `/restaurants/:restaurantId/history` and `/recipes/:recipeId/history`
- `undoChange` and `findUndo` of `lib/history.ts` fill the form. A `null` value of `name`, `location`, `ingredients`, or `steps` stays out, because a patch cannot clear them.
- `UndoAlert.vue` shows the undo note, the kept properties, or "This change cannot be undone here"
- a draft after a new sign in wins over the undo

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| One `items` collection with a `kind` | Not selected. The details, the scores, the triggers, and the paths differ for each kind. |
| A `recipe` relation in `list_items` | Not selected. PocketBase stores an empty relation as `''`. The unique index and each query would need a condition for the kind. |
| Tags from the `json_each` of each record | Removed. SQLite cannot index the items of a JSON array, so a tag filter read all subjects. |
| Tags of only ASCII letters and digits | Removed. Users could not add `café` or `ラーメン`. |
| Tags on reviews | Removed. They repeated the job of the tags of the subject and made the review form longer. |
| A restaurant without a location | Removed. Two restaurants with one name looked the same, and the maps had no point. |
| A duplicate rule on the name and the distance | Not selected. It would refuse real restaurants, such as two bars in one food court. |
| The details from a second request to OpenStreetMap | Not selected. It is a second service with its own terms. The place search fills them when the geocoder has them. |
| Complete recipes in the list responses | Removed. A page of 100 recipes could have about 13 MB. |
| Wines and cheeses in the first release | Not selected. The backlog has them under "For the Future". |
| Only the creator can change a subject | Removed. A creator who left froze wrong values. The group trusts its members. |
| The reviews of other users lock the name, the location, the source, the ingredients, and the steps | Removed. The lock also kept the wrong values. |
| The creator of the oldest review becomes the owner of a subject without a creator | Not selected. The UI would need a new property to know the owner. |
| A revert operation that restores a complete version | Not selected. It also undoes newer changes of other properties, and it needs its own checks. |
| An undo with a link to its version | Not selected. An undo is a new change. |
| Public versions | Not selected. A wrong private value would stay public. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
