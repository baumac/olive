---
description: Each approved user can change each property of each restaurant and recipe. Only the creator can delete one, and the reviews of other users block the delete. Each save writes a version, and approved users undo a change through the edit form.
status: implemented
---

# Motivation
[motivation]: #motivation

The users are a small group of friends and family. A superuser approves each of them. See decision 00003.
The facts of a restaurant change over time, such as a new phone number or new opening hours.
A wrong name, place, source, ingredient, or step is more common than a misuse.
If only the creator could fix them, a creator who stops using the app would freeze them.

Each user can then change each subject. The group accepts the risk of vandalism.
A history of each change lets each approved user see who changed what, and undo a wrong change.
A delete stays with the creator. The reviews of other users are about the subject, so they block its delete.

Expected outcome:
- each approved user can change each restaurant and each recipe
- two users who change different fields at the same time keep both changes
- only the creator deletes a subject, and only while no other user has a review of it
- each change makes a version with the user and the time
- an approved user undoes one change in the UI. A superuser is not necessary.
- an undo is a normal change. The history loses nothing.

# Scope
[scope]: #scope

In scope:
- the rules of `updateRestaurant`, `updateRecipe`, `deleteRestaurant`, and `deleteRecipe`
- the subjects without a creator
- the `restaurant_versions` and `recipe_versions` collections and their record hook
- `listRestaurantVersions` and `listRecipeVersions`
- the edit buttons, the history page, and the undo of the UI

Out of scope:
- the properties of the subjects. Decision 00005 has them.
- the rules of reviews, photos, and lists. Only their creator changes them.
- a history of reviews, photos, and lists
- a moderator role and a report flow. The backlog has reports under "Won't do".
- a check of `updated` before a save. Two changes of the same field keep the last one.
- the restore of a deleted subject. The delete also deletes its versions.
- the name of the superuser of a dashboard change. The version shows "an admin".
- a delete of old versions. The app keeps each version for the life of its subject.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The rules

| Action | Who | Condition |
|---|---|---|
| Create | each approved user | none |
| Change each property | each approved user | none |
| Delete | the creator | no other user has a review |

- for a subject without a creator, "the creator" is each user who has a review of it
- a user who cannot delete gets `403` with `not_creator`. A delete with reviews of other users gets `409` with `other_reviews`.
- the delete also deletes the reviews and the photos of the creator. The confirm dialog says so.
- the duplicate checks of decision 00005 still apply to a change of the location or the source
- a superuser can change each field of each record in the dashboard

Example: Ana adds "Luigis" at a wrong address. Joe and Eva review it.
Eva changes the name to "Luigi's" and selects the correct place. The reviews then show the new name, and the search finds them by it.
Ana cannot delete the restaurant. Joe and Eva cannot delete it either. They are not the creator.

The edit forms send only the properties that differ from the loaded record.
Ana changes the phone while Joe changes the opening hours. Both saves keep their change.

## Vandalism

- only approved users can change a subject. A superuser can take the approval back.
- the history names the editor of each change, and an undo fixes it in the UI
- the edit limit of decision 00004 is 300 changes of each operation per user and hour
- the backups of decision 00013 also have the old values

## The history

Ana adds "Can Solé". This is version 1. Joe changes the phone. This is version 2.
Eva clears the description by mistake and adds the tag `tapas`. This is version 3.
Joe selects "History" on the restaurant page. The page shows the newest version first, with the editor, the time, and the old and new value of each changed property.

- a version from the dashboard or a migration shows "An admin"
- a version of a deleted user shows "Deleted user"
- only approved users can read the history. A wrong private value, such as a phone number, then does not stay public in old versions.

## Undo

Joe selects "Undo this change" on version 3. The edit form opens with `?undo=3`.
The description is "Small tapas bar" again, and the tags are empty again. A note says "Undo of version 3. Check the values, then save."
Joe can change the values. The save makes version 4 with the editor Joe.

- an undo always fixes forward. Version 3 stays, and version 4 has no link to it.
- an undo keeps a property that a newer version changed again. The note names it with the editor and the version.
- an undo uses `updateRestaurant` or `updateRecipe`, so all their checks apply. A place that another restaurant took since then gets `409`.

# Implementation Details
[implementation-details]: #implementation-details

## API

- `UpdateRestaurant` and `UpdateRecipe` have no creator check. The place check and the source check stay in the transaction of the save.
- `deleteForbidden` of `restaurants.go` is the creator check of the delete. A subject with a creator needs no query. A subject without one needs a count of the reviews of the user.
- `countReviewsOfOtherUsers` blocks the delete. `idx_reviews_restaurant_creator` serves both counts.
- the delete removes the reviews of the subject and then the subject, in the transaction of the checks. The parent relation of reviews has no cascade.
- the photo records go with the cascade of decision 00009. PocketBase deletes the files after the commit.
- the triggers and the `searchindex` hooks remove the index rows. A rename updates the parent name in the review search tables.

## Versions

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
- a migration wrote a `baseline` version for each subject that existed. The baseline is not in the list. It is the base of the comparison for the next version.
- `app/api/internal/history` binds `OnRecordCreateExecute` and `OnRecordUpdateExecute` of both collections. It runs the save and the version write in one transaction, for each save.
- `app/api/internal/recordvalues` gives the API values of the fields. The Public API and `Kind.Snapshot` use the same functions.
- a save that changes no property writes no version. A change of only `creator` or `source_key` writes nothing.
- an update of a subject without a version first writes a baseline from the stored record
- a Public API save calls `SaveWithContext` with `history.WithEditor`. A save without it gets the source `admin`. The dashboard of PocketBase 0.40 does not send the request context to the save.
- `TestHistoryHasTheCreateProperties` fails when a create schema has a property that `Kind.Properties` does not have

## Operations

- `GET /restaurants/{restaurantId}/versions` and `GET /recipes/{recipeId}/versions` need a token and an approved user. They have no rate limit.
- the order is `-number`, with `limit` and `cursor`
- a version has `id`, `number`, `created`, `action`, `source`, `editor`, `before`, and `after`
- `before` and `after` have only the changed properties. `before` is empty for a create. Their schemas have the properties of the patch schemas. `location` can also be `null`.
- `versions.go` gets a page of versions and the version before the oldest item. It takes `before` and `after` from the snapshots and `changed`.

## The UI

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
| Only the creator can change a subject | Removed. A creator who left froze wrong values. The group trusts its members. |
| The reviews of other users lock the name, the location, the source, the ingredients, and the steps | Removed. The lock also kept the wrong values. |
| The creator of the oldest review becomes the owner of a subject without a creator | Not selected. The UI would need a new property to know the owner. |
| A revert operation that restores a complete version | Not selected. It also undoes newer changes of other properties, and it needs its own checks. |
| An undo with a link to its version | Not selected. An undo is a new change. |
| Public versions | Not selected. A wrong private value would stay public. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
