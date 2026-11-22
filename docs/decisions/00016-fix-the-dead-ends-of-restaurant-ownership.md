---
description: Let the review of the creator not block the rename and the delete of a restaurant, and let reviewers change a restaurant without a creator.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00001 gave a restaurant to its creator. Its rules had four dead ends.

- A creator adds a restaurant and writes the first review. Then the creator sees a typo in the name.
  Each review blocked the rename, also the review of the creator. The name stayed wrong.
- The same creator could not delete the restaurant. The creator first had to delete the own review. The UI did not say this.
- After a superuser deleted the creator, nobody could change the restaurant through the Public API.
  Its description and its tags became stale.
- The UI showed the name field and the delete button to the creator of a restaurant with reviews.
  The API then refused the request with `409 Conflict`.

Expected outcome:
- only the reviews of other users block the rename and the delete
- a restaurant without a creator can be changed by the users who reviewed it
- the UI shows what the user cannot do before the user tries it

# Scope
[scope]: #scope

In scope:
- the rules of `updateRestaurant` and `deleteRestaurant`
- the restaurant page and the restaurant form of the UI
- the tests

Out of scope:
- who can create a restaurant. Each signed in user can, as before.
- changes by users who are not the creator while the creator exists.
  Changes by all users need an edit history and a report flow to undo vandalism. The app has neither, and sign up has no email verification.
  Decision 00051 changes this. Each approved user can change each restaurant at any time. The delete rules stay.
- a moderator role. A moderator needs reports from users, and the backlog has reports under "Won't Do".
- an edit history of restaurants. Decision 00055 adds it.
- a rename after other users wrote reviews. Only a superuser can do it in the dashboard.
- the rules of reviews and photos

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The rules

**Decision 00051 replaces the rules of a change in this section.** Each approved user can change each property at any time. The reviews of other users do not lock a property. The delete rules stay.

| Action | Who | Condition |
| --- | --- | --- |
| Create | each signed in user | none |
| Change `description` and `tags` | the creator. Decision 00051: each approved user. | none |
| Change `name` | the creator. Decision 00051: each approved user. | no other user has a review of the restaurant. Decision 00051: none. |
| Delete | the creator | no other user has a review of the restaurant |

For a restaurant without a creator, "the creator" is each user who has a review of it.

One review of another user blocks the rename and the delete, as before. The reviews of other users are about the restaurant with this name.

A user who cannot change the restaurant gets `403 Forbidden`.
A request that fails the condition gets `409 Conflict`.

## The review of the creator

Ana adds "Luigis" and writes a review of it. She then changes the name to "Luigi's".
The API accepts the change, because the only review is the review of Ana.
The review then shows `restaurant_name` "Luigi's".

Ana deletes the restaurant. The API deletes her review and its photos with it.
The confirm dialog of the UI says "Your review and its photos will be deleted too."

Joe then writes a review of a second restaurant of Ana.
Ana can still change its description and its tags. She cannot change its name or delete it.
The form shows the name field as read-only with the reason. The page shows the reason in place of the delete button. Decision 00051 removes the read-only field.

The creator can still replace the description and the tags after reviews exist.
A restaurant changes over time. Decision 00011 gives the tags to the creator. The locked name keeps the subject of the reviews.
A superuser corrects a misuse in the dashboard.

## A restaurant without a creator

A superuser deletes the user Tom. The restaurants of Tom stay with a `null` creator, as decision 00001 says.

- Joe has a review of "Can Sole" of Tom. Joe can change its description and its tags.
- Joe has the only review. Joe can also change the name and delete the restaurant.
- Eva has no review of it. Eva gets `403 Forbidden`.
- Eva writes a review. Now Joe and Eva can change the description and the tags. Neither can change the name or delete the restaurant.
- "Empty" of Tom has no reviews. Nobody can change it. A user who writes the first review then can.

The restaurant keeps the `null` creator. The page shows "Added by Deleted user".

Alternatives that this decision does not use:
- the creator of the oldest review becomes the owner. The UI then needs a new response property to know the owner.
- the first user who changes the restaurant becomes its `creator`. "Added by" then names a person who did not add it.
- no change, so only superusers change these restaurants. This is the dead end.

# Implementation Details
[implementation-details]: #implementation-details

## Spec

- only the descriptions of `updateRestaurant`, `deleteRestaurant`, `restaurant_name`, and the `Forbidden` and `Conflict` responses change
- no schema changes. The statuses stay the same. A client of the old SDK continues to work.

## API

- `restaurantChangeForbidden` in `app/internal/publicapi/restaurants.go` is check 10 of decision 00004 for restaurants
  - a restaurant with a creator needs no query
  - a restaurant without a creator needs one count of the reviews of the user
- `countReviewsOfOtherUsers` is check 11. `idx_reviews_restaurant_creator` serves both counts.
- `DeleteRestaurant` deletes the reviews of the restaurant and then the restaurant, in the transaction of the checks.
  After check 11, only the review of the user can exist.
  `reviews.restaurant` has no cascade, so the API deletes the review itself.
- the delete of the review deletes its photo records by the cascade of decision 00008. PocketBase deletes the files after the commit, as for `deleteReview`.
- the triggers of decision 00012 and the hooks of `searchindex` remove the index rows of the review and the restaurant
- a rename of a restaurant with a review changes `restaurant_name` in `reviews_fts`. The `searchindex` hook did this before for a rename in the dashboard.
- PocketBase uses one SQLite writer. A review that another user adds at the same time cannot pass between the check and the save.
- a superuser can change each field of each restaurant in the dashboard. The ownership checks do not apply.
- `guardRestaurantFields` in `app/api/internal/publicapi/restaurantguard.go` gives these saves the checks of the `RestaurantCreate` schema.
  It applies `x-nfc` before the checks and `x-trim` after them, as for a body. The responses then do not fail with a value from the dashboard.
  It uses the record request hooks. Saves of the Public API, migrations, and the CLI do not run it.
  An update checks only the changed properties. An old invalid value then does not block a change of another field.
  A new restaurant needs an address. A latitude, longitude, or place ID without an address gets an error of the address.
  The dashboard shows each error under its field. An error of the location names the `address`, `latitude`, `longitude`, or `place_id` field.
- the field patterns of decision 00038 stay for the saves outside the PocketBase API. `TestRestaurantFieldPatterns` fails when they differ from the spec.

## UI

- `restaurantAccess` in `app/ui/src/lib/restaurants.ts` has the rules. It needs the restaurant, the user, and whether the user has a review.
- another user has a review when `review_count` minus the review of the user is more than 0
- `RestaurantPage` already gets the review of the user for the "Write a review" button. `RestaurantFormPage` gets it at the same time as the restaurant.
- this request can fail. The UI then counts as if the user has a review, so it does not lock more than the API. The API decides.
  A restaurant without a creator needs the request. The form then shows the error page with "Try again".
- the name field is read-only when other users have reviews. A `FormDescription` gives the reason. Focus can still reach the field. Decision 00051 removes the read-only field.
  `Input` shows a read-only field with the `muted` fill. The form does not send the `name` then.
- the page shows "Other people have reviewed this restaurant. You cannot delete it." in place of the delete button
- the page can be older than a new review of another user. The API then answers `409`, and the UI shows the error as before.

## Tests

- `TestRestaurantOwnership` in `app/internal/publicapi/publicapi_test.go` has the cases of the guide-level explanation. It also checks the index tables after a delete and the search by the new name after a rename.
- `TestDashboardRestaurantEdit` in `app/api/internal/publicapi/restaurantguard_test.go` changes a restaurant of another user with a superuser token. It checks the trim, the NFC tags, the refused values, and a change next to an old invalid value.
- `TestPhotosOfDeletedReviewsRestaurantsAndUsers` checks the photo files after the delete of a restaurant
- `app/ui/test/lib.test.ts` tests `restaurantAccess`
- `tests/e2e/restaurants.spec.ts` tests the locked name, the missing delete button, the `409` of a late review, and the rename and delete with the own review
- `tests/e2e/csp.spec.ts` adds a review of another user after the page loads. It checks the error toast of the `409`.
- the end-to-end tests cannot delete a user. Only the Go tests have the restaurant without a creator.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
