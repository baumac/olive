---
description: Keep a version of each change of a restaurant or a recipe. Approved users read the history and undo a change through the edit form. This is version 8.2.0 of the spec.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00051 lets each approved user change each property of each restaurant and recipe.
It lists the edit history as out of scope.
Its plan for a wrong change has gaps:

- A superuser finds the user of a `PATCH` in the request logs. The logs do not have the old values.
- The backups of decision 00022 have the old values. A superuser must restore a backup to a second app to read them.
- Another user can fix a wrong value only if that user knows the old value.

A user can clear the steps of a recipe by mistake. Nobody can then type them again from memory.

The backlog asks for a versioned history so that edits can be reverted.

Expected outcome:
- each change of a restaurant or a recipe makes a version with the user and the time
- an approved user sees who changed what and when
- an approved user undoes one change in the UI. A superuser is not necessary.
- an undo is a normal change. It also makes a version. The history loses nothing.

# Scope
[scope]: #scope

In scope:
- the `restaurant_versions` and `recipe_versions` collections and their migration
- a record hook that writes a version for each save of a restaurant or a recipe
- the `listRestaurantVersions` and `listRecipeVersions` operations
- a history page for restaurants and recipes in the UI
- the undo of one change through the edit form
- the tests and the documents that this decision changes

Out of scope:
- a new write operation for a revert. The undo uses `updateRestaurant` and `updateRecipe`. See [Why no revert operation](#why-no-revert-operation).
- the restore of a deleted restaurant or recipe. The delete also deletes its versions.
  Only the creator can delete, and the reviews of other users block it. See decision 00016.
- a history of reviews, photos, and lists. Only their creator can change them.
- the name of the superuser of a dashboard change. The version shows "an admin".
- a delete of old versions. The app keeps each version for the life of its parent.
- a link from an undo to the version that it undoes. An undo is a new change.
- a check of the `updated` value before a save. The last save still wins, as decision 00051 says.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## New concepts

- **Version**: the state of a restaurant or a recipe after one save. It has a number, the editor, the time, and the changed properties.
- **Editor**: the user of the change. Each version names one editor or "an admin".
- **Baseline**: the state of each restaurant and recipe when the history starts. The migration makes it. It has no editor and no change.
- **Undo**: a change that sets the properties of one version back to their values before that version.

## The history page

Ana adds "Can Solé" with the phone `931 000 000`. This is version 1.
Joe changes the phone to `931 111 111`. This is version 2.
Eva clears the description by mistake and adds the tag `tapas`. This is version 3.

Joe opens "Can Solé" and selects "History". The page shows the newest version first:

| Version | Editor | Changes |
| --- | --- | --- |
| 3 | Eva, 2 minutes ago | Description: "Small tapas bar" → none. Tags: none → `tapas`. |
| 2 | Joe, yesterday | Phone: `931 000 000` → `931 111 111` |
| 1 | Ana, last week | Added the restaurant |

A version from the dashboard or a migration shows "An admin".
A version of a deleted user shows "Deleted user", as "Added by" does.

## Undo one change

Joe selects "Undo this change" on version 3.
The edit form opens with the current values. The description is "Small tapas bar" again. The tags are empty again.
A note on the form says "Undo of version 3. Check the values, then save."
Joe can keep the tag. He changes the tags back to `tapas` and saves.
The form sends only the description. Version 4 has the editor Joe and the change of the description.

An undo always fixes forward. Version 3 stays in the history. Version 4 has no link to version 3.
The history page shows version 4 as a normal change of Joe.

An undo does not change a property that a newer version changed again.

- Ana changes the description to "Tapas and vermouth". This is version 4.
- Joe then selects "Undo this change" on version 3.
- The form keeps "Tapas and vermouth". The note says "Ana changed the description again in version 4. The form keeps the current value."
- The tags go back to none.

## A conflict

Ana moves "Can Solé" to a wrong place. This is version 5.
Tom adds a second restaurant at the old place. Joe then undoes version 5.
The old place ID now belongs to the restaurant of Tom.
The save gets `409 Conflict` with the code of decision 00017, as each other move does.
The form shows the error as before.

## Why no revert operation
[why-no-revert-operation]: #why-no-revert-operation

The undo fills the edit form. It does not call a new operation.

- The checks of `updateRestaurant` and `updateRecipe` apply. These are the spec validation, `x-trim`, `x-nfc`, the place check, the source check, the approval, and the edit limit.
- The user sees each value before the save and can change it.
- The form sends only the changed properties, as decision 00051 says. A change of another property at the same time stays.
- The spec gets no new write operation, no new `409` reason, and no new `x-rate-limit`.

An operation that restores a complete version was the alternative.
It also undoes each newer change of other properties. One wrong change then can cost the good changes after it.

# Implementation Details
[implementation-details]: #implementation-details

## Data

The migration `1794268824_add_versions.go` makes two collections with the same fields:

| Field | `restaurant_versions` | `recipe_versions` |
| --- | --- | --- |
| parent | `restaurant`, relation, required, cascade delete | `recipe`, relation, required, cascade delete |
| `number` | number, starts at 1 for each parent | the same |
| `action` | select: `baseline`, `create`, `update` | the same |
| `source` | select: `user`, `admin` | the same |
| `editor` | relation to `users`, optional, no cascade | the same |
| `changed` | JSON array of the changed property names | the same |
| `snapshot` | JSON object with all properties after the save | the same |
| `created` | autodate | the same |

- `snapshot` has the shape of `RestaurantCreate` or `RecipeCreate`. A property without a value is `null`.
  The `location` is one object, as in the body. The name `values` is an SQL keyword. The field does not use it.
- a unique index on the parent and `number` stops two versions with one number
- the list, view, create, update, and delete rules are `null`. Only superusers can use the collections in the dashboard.
- a deleted user leaves the version with an empty `editor`. PocketBase clears an optional relation without cascade.
- the migration writes a `baseline` version for each existing restaurant and recipe with SQL.
  Its `created` is the `updated` of the record. Its `source` is `admin`. Its `changed` is empty.
- `subjects.Tables` has the `Versions` table name. See decision 00047.

## Record hook

The package `app/api/internal/history` has the hook. It binds `OnRecordCreateExecute` and `OnRecordUpdateExecute` of `restaurants` and `recipes`, as `searchindex` does.
`main.go` and the `publicapi` tests call `history.Register`.

- the hook runs the save and the version write in one transaction. A failed version write stops the save.
- it runs for each save. These are the Public API, the dashboard, a migration, and the CLI.
- `Kind.Snapshot` gets the snapshot of a record. `Kind.Changed` compares two snapshots by their JSON values.
- `app/api/internal/recordvalues` gets the API values of the fields. The Public API and `Kind.Snapshot` use the same functions.
- `TestHistoryHasTheCreateProperties` fails when `RestaurantCreate` or `RecipeCreate` has a property that `Kind.Properties` does not have.
  A save that changes no property writes no version. A change of only `creator` or `source_key` then writes nothing.
- an update of a parent without a version first writes a baseline from the stored record. A migration that inserts a parent with SQL makes such a parent.
- the editor comes from the context of the save. `history.WithEditor(ctx, userID)` sets it.
  `CreateRestaurant`, `UpdateRestaurant`, `CreateRecipe`, and `UpdateRecipe` call `tx.SaveWithContext` with it.
- a save without an editor gets the source `admin`.
  The dashboard of PocketBase 0.40 does not send the request context to the save.
- `number` is the number of the newest version plus 1. PocketBase uses one SQLite writer. Two saves cannot get the same number.
- `TestMigrationBaselineIsTheSnapshot` compares the SQL of the migration with `Kind.Snapshot`. A difference makes a version without a change.

## Spec

The version is 8.2.0. A client of the old SDK continues to work.

- `GET /restaurants/{restaurantId}/versions` is `listRestaurantVersions` in the `restaurants` tag
- `GET /recipes/{recipeId}/versions` is `listRecipeVersions` in the `recipes` tag
- both need a token and an approved user, as decision 00034 says. They have `400`, `401`, `403`, and `404` responses.
  A visitor without a token cannot read them. A wrong private value, such as a phone number, then does not stay public.
- they have no `x-rate-limit`. They are read operations.
- the order is `-number`. The page has `limit` and `cursor`, as the other lists.
- the baseline is not in the list. It is only the base of the comparison for the next version.
- `RestaurantVersion` has:
  - `id`, `number`, and `created`
  - `action`: `create` or `update`. The schema is `VersionAction`.
  - `source`: `user` or `admin`. The schema is `VersionSource`.
  - `editor`: a `Creator` or `null`
  - `before`: the changed properties with their old values. It is empty for a create.
  - `after`: the changed properties with their new values
- `before` and `after` are `RestaurantVersionValues`. It has the properties of `RestaurantPatch`.
  Its `location` can also be `null`. A restaurant from before decision 00017 has no location.
- `RecipeVersion` and `RecipeVersionValues` have the same shape with the properties of `RecipePatch`
- `RestaurantVersionList` and `RecipeVersionList` have `items` and `next_cursor`

## API

- `app/api/internal/publicapi/versions.go` has both handlers. They take a `subject`, as the gallery does.
- the handler gets the page of versions and the version before the oldest item. It then takes `before` and `after` from the snapshots and `changed`.
- the approval check of decision 00034 comes from the spec. The operations need no line in `server.go`.

## UI

- `app/ui/src/lib/reviewKinds.ts` has the `listVersions` call of each kind
- `app/ui/src/pages/HistoryPage.vue` shows the versions of restaurants and recipes.
  The routes are `/restaurants/:restaurantId/history` and `/recipes/:recipeId/history`. They need a sign in and the approval.
- each version shows its number, the editor, the time, and the old and new value of each changed property. A create shows "Added the restaurant." or "Added the recipe."
- `RestaurantPage.vue` and `RecipePage.vue` show a "History" link next to "Edit" for each signed in user
- `app/ui/src/lib/history.ts` has `undoChange` and `findUndo`:
  - each property of `before` goes into the form
  - a property that a newer version changed keeps the current value. The form names it with the editor and the version.
  - a `null` value of `name`, `location`, `ingredients`, or `steps` stays out. A patch cannot clear them.
- "Undo this change" opens the edit page with `?undo=NUMBER`. It is only on an `update` version.
- `RestaurantFormPage.vue` and `RecipeFormPage.vue` load the versions from the newest down to `NUMBER`. They then fill the fields.
  A draft of `lib/drafts.ts` after a new sign in still wins over the undo.
- `app/ui/src/components/app/UndoAlert.vue` shows "Undo of version NUMBER" and the kept properties.
  A version that does not load, or a create, shows "This change cannot be undone here". The form then has the current values.
  A version without a property that can go back shows "Nothing to undo in version NUMBER".
- `changedPatch` of decision 00051 makes the patch. The form sends only the undone properties.
- the history shows a list of ingredients or steps as the old list and the new list. A line diff is out of scope.

## Other documents

- decision 00051 gets a line at "an edit history": decision 00055 adds it
- decision 00016 gets the same line at "an edit history of restaurants"
- the backlog loses the item "Restaurants and recipes should have a versioned history so that edits can be reverted"
- `AGENTS.md` has the `history` package, the operations, and the migration rule of the versions

## Tests

- `app/api/internal/history/history_test.go`:
  - a create writes version 1 with the source `user`, the editor of the context, and the set properties
  - an update writes only the changed properties
  - a save without a change, or with only a `creator` change, writes nothing
  - a save without an editor gets `admin`
  - a failed version write stops the save
  - the delete of a user clears `editor`. The delete of a parent deletes its versions.
  - a parent without a version gets a baseline before its change
  - the baseline of the migration is equal to `Kind.Snapshot` for full and empty restaurants and recipes
- `app/api/internal/publicapi/versions_test.go`:
  - the order, the pagination, the cursor check, `before`, and `after` of both lists
  - an undo is a new version. A dashboard change has the source `admin`.
  - a deleted user has no editor
  - the baseline is not in the list. The first version after it compares with it.
  - `401` without a token, `403` for a user who waits, and `404` for an unknown parent
- `app/ui/test/history.test.ts`:
  - `undoChange` with a change, a change that a newer version changed, a `null` location, and a create
  - `findUndo` loads the pages down to the version
  - the history page with the editor, "An admin", the create, and the undo link. It passes the axe check.
  - the restaurant form and the recipe form send only the undone values. A failed load shows the alert.
- `tests/e2e/history.spec.ts`:
  - Eva clears the description. Joe opens the history, undoes the change, and saves. The restaurant has the old description.
  - an undo keeps a property that a newer version changed
  - a user undoes a change of the steps of a recipe
  - a signed out visitor sees no "History" link. The history route opens the sign in page.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.

These questions had answers before the implementation:

- Only approved users can read the history. Each visitor can read `getRestaurant`, but not the old values.
- The app never deletes an old version. The edit limit of decision 00036 allows 300 changes per user and hour. The group is small.
- A dashboard change does not name its superuser. The version shows "An admin".
- An undo has no link to its version. It always fixes forward with a new change.
