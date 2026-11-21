---
description: Each approved user can change each property of each restaurant and recipe. The reviews of other users do not lock a property. Only the creator can delete, as before. This is version 7.1.0 of the spec.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00016 lets only the creator change a restaurant.
Decision 00047 has the same rule for recipes.
Its reason: "Changes by all users need an edit history and a report flow to undo vandalism."
That reason fits an open public site.

Decision 00034 came later. A superuser now approves each user.
The users are friends and family.

The facts of a restaurant change over time. A new phone number, new opening hours, or a new website is common.
Only the creator can fix them. A creator who does not use the app any more freezes them.
The backlog accepts this under "Won't do". Backlog item 2 asks to decide it again.

Backlog item 2 also has this example.
A user adds a restaurant with a typo in its name. Another user reviews it.
Now nobody can fix the name in the UI. Only a superuser can fix it in the dashboard.

The reviews of other users also lock the location, the source, the ingredients, and the steps.
A wrong value then stays until a superuser fixes it.

Expected outcome:
- each approved user can change each restaurant and each recipe
- a wrong name, location, source, ingredient, or step can be fixed in the UI
- two users who change different fields at the same time keep both changes

# Scope
[scope]: #scope

In scope:
- the rules of `updateRestaurant` and `updateRecipe`
- the descriptions of the spec and the new minor version
- the edit button of the restaurant and recipe pages
- the restaurant form and the recipe form
- a patch with only the changed properties
- the tests and the decisions that this decision changes

Out of scope:
- the rules of `deleteRestaurant` and `deleteRecipe`. Only the creator can delete, as before.
  The reviews of other users still block the delete. A delete also deletes the reviews of the user.
- an edit history, or a "Changed by" line on the page
- a report flow. The backlog has reports under "Won't do".
- a check of the `updated` value before a save. Two changes of the same field keep the last one.
- the rules of reviews, photos, and lists
- the dashboard. A superuser can change each field, as before.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The rules

| Action | Who | Condition |
| --- | --- | --- |
| Create | each approved user | none |
| Change each property | each approved user | none |
| Delete | the creator | no other user has a review |

For a restaurant or a recipe without a creator, "the creator" is each user who has a review of it, as in decision 00016.

The duplicate checks of decisions 00017 and 00047 still apply to a change of the location or the source.
A user who is not approved still gets `403` with `not_approved`, as decision 00034 says.

## The reviews of other users

Decisions 00016, 00017, 00047, and 00048 locked the fields that the reviews scored.
A review is about the restaurant with this name at this place, or the recipe with these ingredients.
The lock kept a creator from changing the subject of the reviews of other users.

This decision removes the lock. The group trusts its members.
A wrong value is now more common than a misuse. The lock also kept the wrong values.

Ana adds "Luigis" at a wrong address. Joe and Eva review it.

- Eva changes the name to "Luigi's" and selects the correct place. The API accepts it.
- The reviews show `restaurant_name` "Luigi's". The search of decision 00004 finds them by the new name.
- Ana still cannot delete the restaurant. The page says "Other people have reviewed this restaurant. You cannot delete it."
- Joe and Eva cannot delete it. They are not the creator.

## Examples

Ana adds "Can Solé". Joe reviews it. The restaurant changes its phone number.

- Joe opens "Edit restaurant". Each field can change.
- Joe changes the phone and adds the tag `tapas`. The save works.
- Joe sees no delete button.

Tom adds "Dal" with a source. Tom then leaves, and a superuser deletes him.

- Each approved user can change each property of "Dal".
- Only a user with a review of "Dal" can delete it, while no other user has a review. This is decision 00016.

Ana and Joe open the form of "Can Solé" at the same time.

- Ana changes the phone. Joe changes the opening hours. Both save.
- The form sends only the changed properties. The restaurant then has the phone of Ana and the opening hours of Joe.
- Ana and Joe both change the phone. The last save wins. Each user sees the stored value after the save.

## Vandalism

A user can change or clear each restaurant and recipe. The group accepts this risk.

- Only approved users can change a restaurant or a recipe. A superuser takes the approval back, as decision 00034 says.
- The PocketBase request logs name the user of each `PATCH`. A superuser finds the user there.
- The backups of decision 00022 have the old values. A superuser types them in the dashboard.
- Another approved user can also fix a wrong value in the UI.
- The edit limit of decision 00036 is 300 changes of each operation per user and hour.

# Implementation Details
[implementation-details]: #implementation-details

## Spec

- `updateRestaurant` and `updateRecipe` get the new rule in their descriptions
- the `Conflict` response tells that `other_reviews` is only for a delete
- `deleteRestaurant` and `deleteRecipe` do not change
- no schema change. A client of the old SDK continues to work. The version is 7.1.0.
- `make generate` makes the SDKs again. Only the doc comments change.

## API

- `UpdateRestaurant` and `UpdateRecipe` have no creator check and no count of the reviews of other users
- the place check of `UpdateRestaurant` and the source check of `UpdateRecipe` stay in the transaction of the save
- `changeForbidden` of `restaurants.go` is now `deleteForbidden`. Only `deleteParent` uses it.
- the approval check of decision 00034 runs before the handler
- the `guardRestaurantFields` and recipe guards of the dashboard do not change

## UI

- `parentAccess` in `app/ui/src/lib/parents.ts` returns three values:
  - `canEdit` is true for each signed in user. The router guard of decision 00034 stops a user who waits.
  - `isCreator` is the old `canChange`. The delete needs it.
  - `hasOtherReviews` does not change. It blocks the delete.
- `RestaurantPage.vue` and `RecipePage.vue` show "Edit restaurant" or "Edit recipe" when `canEdit` is true.
  The delete button needs `isCreator` and no reviews of other users, as before.
- `RestaurantFormPage.vue` and `RecipeFormPage.vue` have no read-only fields.
  They show no `403` page for a user who is not the creator.
  The creator of a parent with reviews of other users sees "You cannot delete it" under the buttons.
- `LocationField.vue` loses the `locked` prop. No form uses it.
- Both forms send only the properties that differ from the loaded record.
  A form that is older than a change of another user then does not undo it.

## Data

No migration. The collections and the API rules do not change.

## Other documents

- decision 00016 gets a line in its scope: decision 00051 changes the rules of a change
- decisions 00017, 00038, 00047, and 00048 get the same line at their rules
- the backlog loses item 2 and the "Won't do" line "Only the creator can edit a restaurant"
- backlog item 7 asks for personal tags. Each approved user can now add a tag to a restaurant. The item then goes away.
- `AGENTS.md` has the new rules of a change and a delete

## Tests

- Go tests:
  - another approved user changes each property of a restaurant and a recipe. The answer is `200`.
  - a change of the location, the source, the ingredients, or the steps after a review of another user gets `200`
  - a rename changes `restaurant_name` of the reviews
  - each user can change a parent without a creator
  - the delete rules do not change
- UI unit tests:
  - `parentAccess` for the creator, another user, and a parent without a creator
  - `changedPatch` with no change, one change, and a cleared value
  - the recipe form of another user sends only the changed fields
  - the drafted location of another user goes into the form
- End-to-end tests:
  - `restaurants.spec.ts`: another user changes the name, the location, and the phone. The page has no delete button.
  - `restaurants.spec.ts`: a review that arrives after the form opens does not block a move
  - `recipes.spec.ts`: another user changes the name, the ingredients, and the servings
  - two users change different fields. The restaurant keeps both values.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
