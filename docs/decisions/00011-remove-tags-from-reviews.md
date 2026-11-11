---
description: Remove the tags of reviews from the Public API, the database, the search index, and the UI. Restaurants keep their tags.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00001 gives tags to restaurants and to reviews.
Tags help people find a restaurant, such as all `italian` restaurants.
The tags of a review repeat this job. They also make the review form longer.

Expected outcome:
- a review has scores, a title, feedback, and photos. It has no tags.
- restaurants keep their tags, their tag filters, and their tag search

# Scope
[scope]: #scope

In scope:
- the `Review`, `ReviewCreate`, and `ReviewPatch` schemas
- the tag filters of `listReviews` and `listAllReviews`
- the full-text search of reviews and its `relevance` sort
- a migration that removes the `tags` field of `reviews` and the `tags` column of `reviews_fts`
- the review form, the review page, the review card, and the review filters of the UI
- the tests

Out of scope:
- the tags of restaurants. Nothing about them changes.
- a copy of the review tags to the tags of their restaurant. The restaurant creator owns these tags.
- a version 2 of the API. The API has no deployment and no clients other than the UI.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

A review has no `tags` property:

```json
{
  "id": "r1b8c5e2g9i6k3m",
  "restaurant": "k3x9m2p7q1w8z4a",
  "restaurant_name": "Luigi's Trattoria",
  "space": 7, "service": 9, "food": 8, "price": 6, "cleanliness": 8,
  "title": "Great food, a bit expensive",
  "feedback": "Friendly staff and great pasta",
  "creator": { "id": "u5n1c8r3t6y0b2e", "name": "Ana" },
  "photos": [],
  "created": "2024-11-06 19:02:11.120Z",
  "updated": "2024-11-07 09:12:00.000Z"
}
```

These requests now get `400 Bad Request`:
- `POST /v1/restaurants/{restaurantId}/reviews` with `tags` in the body. The body has an unknown property.
- `PATCH /v1/restaurants/{restaurantId}/reviews/{reviewId}` with `tags` in the body
- `GET /v1/reviews?tags[has]=pasta`. The query parameter is not supported.

The review search looks in `title`, `restaurant_name`, and `feedback`.
The `relevance` sort counts a match in `title` most, then `restaurant_name`, then `feedback`.

The UI has no tag field in the review form and no tag filter in the review lists.
A review card and a review page show no tag badges.

# Implementation Details
[implementation-details]: #implementation-details

Spec:
- `Review`, `ReviewCreate`, and `ReviewPatch` lose `tags`
- `listReviews` and `listAllReviews` lose `TagsHas`, `TagsHasAny`, and `TagsHasAll`. `listRestaurants` keeps them.
- `ReviewSearch` does not name `tags`
- the SDKs are generated again. See decision 00003.

Server:
- `reviewInput` has no tags. A create or patch never writes the `tags` field.
- `toReview` sets no tags
- `reviewSorts` and the review filters do not change. Only the tag filters go.
- the `relevance` weights of reviews are `title` 10, `restaurant_name` 2, and `feedback` 1
- `searchindex` writes the `tags` column only for restaurants. A change of a review re-indexes it only after a change of `title`, `feedback`, or `restaurant`.

Cursors:
- the cursor fingerprint marshals the list parameters to JSON
- the tag parameters have `omitempty`. A request without them had no tag keys in the fingerprint.
- a cursor of a list without tag filters therefore stays valid after the change
- a cursor of a list with tag filters cannot occur after the change. That request gets `400`.

Migration `1794268807_remove_review_tags.go`:
- up drops `reviews_fts`, makes it again without `tags`, and fills it from `reviews` and `restaurants`
- up then removes the `tags` field of `reviews`. PocketBase drops the column and its values.
- down adds an empty `tags` JSON field to `reviews`. It makes `reviews_fts` again with an empty `tags` column.
- down cannot restore the deleted tags. There is no deployment, so no stored tags are lost.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
