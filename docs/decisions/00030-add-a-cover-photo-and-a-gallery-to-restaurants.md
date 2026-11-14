---
description: Show a cover photo on the restaurant cards and the restaurant page, and a gallery of all review photos on the restaurant page.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00008 shows each photo only in its review.
The restaurant cards and the top of the restaurant page have no photo.
A person who looks for a restaurant sees only text until they read the reviews.

The reviews already have photos. A cover and a gallery can use them. They need no new upload and no new storage.

Expected outcome:
- each restaurant card with a photo shows a cover photo
- the restaurant page starts with the cover photo
- the restaurant page has a "Photos" section with the photos of all its reviews
- the creator of a restaurant does not choose its photos. Decision 00008 gives the reason.

# Scope
[scope]: #scope

In scope:
- the `cover` property of `Restaurant` and the `RestaurantPhoto` schema
- the `listRestaurantPhotos` operation
- the `CoverPhoto` component of the design system and the screen changes in `app/ui`
- the tests

Out of scope:
- photos of restaurants that the creator of the restaurant uploads. Decision 00008 gives the reason.
- a cover that a person selects, such as the creator of the restaurant or a superuser
- the cover in the link preview tags of decision 00013 and the JSON-LD of decision 00021. `og:image` needs an absolute URL and a check of the `Cross-Origin-Resource-Policy` of the photo files. It is in the backlog.
- a cover on the review cards of `/my-reviews` and the reviewer pages. The review cards already show their photos.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

The **gallery** of a restaurant is the list of the photos of all its reviews.
It shows the photos of the best review first:
1. the review with the highest overall score first
2. for an equal score, the newer review first
3. the photos of one review in the order of the review, the oldest first

The **cover** of a restaurant is the first photo of its gallery.
A restaurant without photos has the cover `null`.

```http
GET /v1/restaurants/k3x9m2p7q1w8z4a
```

```json
{
  "id": "k3x9m2p7q1w8z4a",
  "name": "Luigi's Trattoria",
  "cover": {
    "id": "p4h7j1l5n9q2s6v",
    "alt": "A plate of carbonara on a wooden table",
    "width": 1536,
    "height": 2048,
    "urls": {
      "full": "/v1/restaurants/k3x9m2p7q1w8z4a/reviews/r1b8c5e2g9i6k3m/photos/p4h7j1l5n9q2s6v/full",
      "thumb": "/v1/restaurants/k3x9m2p7q1w8z4a/reviews/r1b8c5e2g9i6k3m/photos/p4h7j1l5n9q2s6v/thumb"
    },
    "created": "2024-11-06T19:04:30.811Z",
    "review": "r1b8c5e2g9i6k3m",
    "creator": { "id": "u5n1c8r3t6y0b2e", "name": "Ana" }
  }
}
```

```http
GET /v1/restaurants/k3x9m2p7q1w8z4a/photos?limit=10
```

The response is a page of `RestaurantPhoto` items with `next_cursor`, like the other lists.

In the UI:
- a restaurant card shows the `thumb` of the cover above the name. A card without a cover shows the `muted` placeholder with the utensils icon. The cards of a row then line up.
- the restaurant page starts with the `full` image of the cover. A select of the cover opens the photo viewer.
- the "Photos" section of the restaurant page comes before "Reviews". It shows the gallery as a `PhotoGrid` with "Load more photos".
- the photo viewer of the gallery shows "From the review by Ana" with a link to the review

# Implementation Details
[implementation-details]: #implementation-details

## API shape

- `RestaurantPhoto` is an `allOf` of `Photo` and two more properties. Both SDK generators make one flat type of it.
  - `review`: the ID of the review of the photo. A client then links to the review without a parse of `urls`.
  - `creator`: the `Creator` of the review. It is never `null`. A delete of a user also deletes the reviews of the user.
- `Restaurant.cover` is a required `RestaurantPhoto` or `null`. Each operation that returns a `Restaurant` sets it. A new restaurant has `null`.
- `listRestaurantPhotos` has the path `/restaurants/{restaurantId}/photos` with `limit` and `cursor`. It has no sort and no filters.
- an unknown restaurant gets `404`, like `listReviews`
- the version of the spec becomes 2.2.0. The new property and the new operation do not break clients.

## Order and queries

The sort keys of the gallery are, in this order:

| Key | Direction |
| --- | --- |
| the overall of the review, as `reviewOverall` of `reviews.go` | descending |
| `reviews.created` | descending |
| `reviews.id` | descending |
| `review_photos.created` | ascending |
| `review_photos.id` | ascending |

- the gallery uses the keyset cursor of decision 00004 with these keys. The keys have two directions, so the cursor condition is a list of `OR` terms.
- the query joins `reviews` to `review_photos` and keeps the reviews of the restaurant. An index of `reviews` that starts with `restaurant` finds the reviews. The index `idx_review_photos_review_created` finds their photos.
- the sort uses a temporary B-tree. A restaurant has few photos, so this is an accepted cost. A review has at most 5 photos.
- the covers of a page of restaurants come from one query with `ROW_NUMBER() OVER (PARTITION BY restaurant ...)` and the same order. The query joins `reviews` and `users`. It selects all fields of the response. A page then has one more query.
- a page of the gallery has the sort query of decision 00004 and one query for the fields of its photos. This query also joins `reviews` and `users`.
- a change of a score or a new photo can change the cover. Each response computes it again. The cover needs no trigger and no stored column.

## UI and design system

- the new `CoverPhoto` component in `specs/design-system/components/CoverPhoto` and `app/ui/src/components/ui/cover-photo` shows one photo in a box with a fixed aspect ratio and `object-cover`
  - it has the sizes `card` with the `thumb` image and `hero` with the `full` image
  - `hero` has a `srcset` with `thumb` and `full`. A narrow screen then loads the `thumb`.
  - it shows `Skeleton` until the image loads and the image-off icon on `muted` when it fails, like `PhotoGrid`
  - without a photo, it shows the utensils icon on `muted`
  - its `<img>` has `alt=""` in a card. The name of the card link is the name of the restaurant.
- the cover of the restaurant page is a button. Its accessible name is "Open the cover photo" and the alt text of the photo.
- `PhotoViewer` gets a `caption` slot. The gallery fills it with "From the review by Ana".
- the restaurant page loads the gallery only when the restaurant has a cover
- the gallery uses `PagedList` with its `items` slot. `PagedList` then shows the loading, error, and "Load more photos" states.
- a select of the cover opens the cover in the loaded gallery. When the gallery does not have it, the viewer shows only the cover. A review can change between the two requests.
- a change of a photo in a review card does not reload the cover or the gallery. A reload of the page shows the change.

## Tests

- Go tests of the order, the cursor, `404`, the cover of each `Restaurant` operation, and a restaurant without photos
- a query plan test checks that the gallery query uses the two indexes
- Vitest tests of `CoverPhoto` and the caption slot of `PhotoViewer`
- end-to-end tests: a card shows the cover, the restaurant page shows the cover and the gallery, the viewer links to the review, and the CSP and accessibility checks pass

## Alternatives
[alternatives]: #alternatives

| Alternative | Result |
| --- | --- |
| The newest photo as the cover | Not selected. Each upload then changes the cover. A photo of a bad visit can become the cover. |
| A cover that the creator of the restaurant selects | Not selected. One user would choose for all reviewers. Decision 00008 gives the reason. |
| A stored cover column with triggers | Not selected. The order needs the scores of the reviews. A trigger on scores and photos is more complex than one query for each page. |
| The gallery newest first | Not selected. The cover is then not the first photo of the gallery. |
| `Restaurant.cover` as a `Photo` | Not selected. The viewer then cannot link to the review without a parse of `urls`. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
