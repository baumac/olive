---
description: The creator of a review adds up to 5 photos to it. The app checks each upload, encodes it again as JPEG without metadata, and serves only its own files. A restaurant or a recipe shows the photos of its reviews as a cover and a gallery. A daily job deletes the photo files without a record.
status: proposed
---

# Motivation
[motivation]: #motivation

A photo shows the dishes, the room, and the menu better than text.
Decision 00000 has a photo in its first example.

Photos are the only files that users upload. They bring these risks:
- a photo can have personal data, such as the GPS position of the camera
- a file can be valid as an image and as another type, such as HTML. The UI trusts the files of its own origin.
- a large image can use all memory of the server
- a failed file delete can keep a photo that the user deleted

A person who looks for a restaurant or a recipe also wants to see a photo before the reviews.

Expected outcome:
- the creator of a review adds, describes, and deletes its photos
- the app removes all metadata. The server never serves a file in the form that the client sent.
- the Content Security Policy of the UI stays strict
- each subject card and page shows a cover from the review photos. The page has a gallery of all of them.
- the API serves no photo of a deleted review or user. A daily job deletes the files that stay.

# Scope
[scope]: #scope

In scope:
- the photo operations and schemas of both kinds
- the photo collections and their file fields
- the limits, the checks, and the processing of each upload
- the serving and the caching of the files
- the cover and the gallery of a subject
- the delete of photos and the cleanup job
- the photo components of the UI

Out of scope:
- photos that the creator of a subject uploads. Each photo belongs to a review. The backlog has this under "Won't do".
- a cover that a person selects
- photos of users, such as avatars
- video, animated images, and other file types
- moderation and automatic checks of the content. Superusers delete bad photos in the dashboard.
- a limit on the total photo storage of a user
- a CDN, S3 storage, and a separate host for the files
- a change of the order of photos, and a search in the alt text
- the cover in the link previews. Decision 00002 has the previews.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Concepts

- **Photo**: one image of a review. It has an ID, an alt text, its size in pixels, and two image files.
- **Alt text**: a short text that describes the photo for screen readers. It is optional.
- **Variant**: one image file of a photo. `full` has at most 2048 pixels on its long side. `thumb` has at most 480.
- **Gallery**: the photos of all reviews of a subject, the best review first
- **Cover**: the first photo of the gallery. It is `null` for a subject without photos.

## Operations

For the restaurant reviews, under `/restaurants/{restaurantId}/reviews/{reviewId}/photos`:

| Operation | Request |
|---|---|
| `addRestaurantReviewPhoto` | `POST .../photos` with the bytes of one image |
| `updateRestaurantReviewPhoto` | `PATCH .../photos/{photoId}` with the alt text |
| `deleteRestaurantReviewPhoto` | `DELETE .../photos/{photoId}` |
| `getRestaurantReviewPhotoFile` | `GET .../photos/{photoId}/{variant}` |
| `listRestaurantPhotos` | `GET /restaurants/{restaurantId}/photos`, the gallery |

The recipe reviews have the same operations with `Recipe` in their names and `/recipes/{recipeId}` in their paths.

```http
POST /v1/restaurants/k3x9m2p7q1w8z4a/reviews/r1b8c5e2g9i6k3m/photos
Content-Type: image/jpeg

<the bytes of the image>
```

```json
{
  "id": "p4h7j1l5n9q2s6v",
  "alt": null,
  "width": 1536,
  "height": 2048,
  "urls": {
    "full": "/v1/restaurants/k3x9m2p7q1w8z4a/reviews/r1b8c5e2g9i6k3m/photos/p4h7j1l5n9q2s6v/full",
    "thumb": "/v1/restaurants/k3x9m2p7q1w8z4a/reviews/r1b8c5e2g9i6k3m/photos/p4h7j1l5n9q2s6v/thumb"
  },
  "created": "2024-11-06T19:04:30.811Z"
}
```

The camera sent 4032 by 3024 pixels with the EXIF orientation "rotate 90".
The app turned the pixels, so `width` and `height` show the photo upright. It made the long side 2048 pixels and removed the EXIF data.

- `alt` is `null` or a `Title` of 1 to 255 characters. A second request sets it.
- `urls` are paths on the same origin. Clients use them as given and do not build them.
- a photo has no `updated`, because its image never changes. It has no `creator`, because the creator of the review is its creator.
- `photos` of a review has all its photos, the oldest first. A review has at most 5.
- only the creator of the review adds, changes, or deletes its photos. Other users get `403`. All visitors read the photos.
- a photo of a different review than the path gets `404`

## Limits

| Limit | Value | Status |
|---|---|---|
| upload size | 10 MiB | `413` |
| upload types | `image/jpeg`, `image/png`, `image/webp` | `415` |
| pixels of the upload | at most 25 megapixels, at most 8192 on each side | `422` |
| photos of a review | 5 | `409` with `photo_limit` |
| uploads of a user | 30 per hour. See decision 00003. | `429` |
| memory of all uploads | 64 MB of bodies and processed photos | `429` with `Retry-After: 5` |

- GIF, HEIC, AVIF, SVG, and TIFF get `415`. SVG can contain scripts.
- iOS Safari changes HEIC photos to JPEG when the file input does not accept HEIC
- an animated PNG keeps only its first frame. The Go WebP decoder does not read animations, so an animated WebP can get `422`.

## Gallery and cover

The gallery order:
1. the review with the highest overall first
2. for an equal overall, the newer review first
3. the photos of one review, the oldest first

- each `Restaurant` and `Recipe` has `cover`. It is a `RestaurantPhoto` or a `RecipePhoto`, with the `review` ID and the `creator` of the review.
- `listRestaurantPhotos` and `listRecipePhotos` give pages of the gallery with `limit` and `cursor`
- a change of a score or a new photo can change the cover

## The UI

- an entry of the table shows the first photo of the review with the `srcset` of the `hero` size of `CoverPhoto`. A phone with a dense screen then loads `full`.
- a tile of the journal shows the `thumb` of the first photo of the review. See decisions 00009 and 00006.
- the review page starts with the `full` variant of the first photo. Its "Photos" section has a `PhotoGrid` of all photos. A click opens the `full` variant in `PhotoViewer`.
- the viewer has "Previous photo", "Next photo", and the arrow keys. On the review page, the creator also gets "Edit description" and "Delete photo".
- the review form has a "Photos" section after the form. Enter in it does not save the review.
- a new review gets its photos after the first save. A failed upload shows a toast and opens the edit form of the saved review.
- a failed alt text keeps the added photo and shows "Description not saved"
- the upload form asks the user to add only photos of food, rooms, and menus
- a subject card shows the `thumb` of the cover. A card without a cover shows the `muted` placeholder with the utensils icon.
- the subject page starts with the `full` cover. Its "Photos" section comes before "Every visit" or "Every cook", with "Load more photos".
- the viewer of the gallery shows "From the review by Ana" with a link to the review

# Implementation Details
[implementation-details]: #implementation-details

## The spec

- the add operation has one content type, `image/*`, with `format: binary`. The handler accepts only the three types.
  With three binary types, oapi-codegen would make three `Body` fields, which does not compile.
- the body is raw bytes, not `multipart/form-data`. A raw body has one content type, and kin-openapi does not decode parts.
- the TypeScript client sends `Content-Type: image/*`. `addPhoto` of `app/ui/src/lib/photos.ts` changes it to the type of the file with an init override. A test of the SDK checks it.
- the file operation has `HEAD`, `If-None-Match`, and `Range`, with the `206`, `304`, and `416` responses. Its security is `{}` with a token, like the other reads. An `<img>` sends no `Authorization` header.
- `RestaurantPhoto` is an `allOf` of `Photo`, `review`, and `creator`. Both generators make one flat type.

## Collections

| Field | `review_photos` and `recipe_review_photos` |
|---|---|
| `review` | a required relation to the reviews with cascade delete |
| `alt` | optional text, at most 255 characters |
| `width`, `height` | required numbers |
| `full`, `thumb` | file fields with `MaxSelect` 1, the MIME type `image/jpeg`, and `Protected: true` |
| `created` | autodate |

- an index on `(review, created, id)` serves the photos of a review and the gallery
- `Protected: true` stops the download under `/api/files`. The view rule is `null`, so only superusers pass it.
- the app names the files `full.jpg` and `thumb.jpg`. PocketBase adds a random suffix. The file name of the device never reaches the server.
- the files are in `pb_data/storage`. The PocketBase filesystem API reads them, so a later S3 setting needs no code change.
- a photo is its own record, not a file field of the review. Each photo needs an ID and an alt text, and an upload does not change `updated` of the review.

## Checks of an upload

| Order | Check | Status |
|---|---|---|
| 1 | the token and the approval | `401`, `403` |
| 2 | the upload limit of the user | `429` |
| 3 | the path parameters | `400` |
| 4 | the content type | `415` |
| 5 | `Content-Length` is at most 10 MiB | `413` |
| 6 | the subject and the review exist | `404` |
| 7 | the user is the creator of the review | `403` |
| 8 | the review has fewer than 5 photos | `409` |
| 9 | the body is at most 10 MiB. `http.MaxBytesReader` stops the read. | `413` |
| 10 | the memory limit has space for the body, at each step of the read | `429` |
| 11 | the first bytes match the content type | `422` |
| 12 | `image.DecodeConfig` gives a size inside the pixel limits | `422` |
| 13 | the image decodes | `422` |
| 14 | in one transaction: the review, its creator, the count, and the save | `404`, `403`, `409` |

- checks 6 to 8 run before the app reads the body. A client that cannot add a photo then does not send 10 MiB for nothing.
- check 12 reads only the header. A small file that claims 50,000 by 50,000 pixels never decodes.
- check 14 repeats checks 6 to 8. PocketBase has one SQLite writer, so two parallel uploads cannot both pass the count.
- the decode and the encode run before the transaction. They are slow, and the transaction blocks other writes.
- at most 2 uploads decode at the same time. A decode of 25 megapixels needs up to 100 MB, so the semaphore limits it to 200 MB.
- `photoMemory` of `photomemory.go` counts the bytes in memory. The read buffer starts at 64 KB and doubles. A slow client holds only the bytes that it sent.
- check 10 does not wait. It stops the read and sends `429`. The `429` gives back the count of check 2.
- `Options.PhotoMemoryLimit` sets the limit. It has no environment variable.
- the UI shows "Try again in a few seconds" for a `429` with a `Retry-After` of at most 10 seconds
- a `422` of an upload has a `detail` and no `errors` array
- the response validation checks the status and the headers of the file operation without the body. `headerCheckWriter` checks them before it writes them.

## Processing

1. `image.Decode` decodes JPEG and PNG. `golang.org/x/image/webp` decodes WebP.
2. `fit` makes `full` and `thumb` smaller with `draw.CatmullRom` of `golang.org/x/image/draw`. It does not make a small image larger.
3. `orient` turns the pixels after the fit, by the orientation that `exifOrientation` of `photoorient.go` reads from a JPEG file
4. `image/jpeg` encodes each variant with quality 85
5. the save stores the two files

- `exifOrientation` reads only the first IFD of the EXIF segment and checks each offset. A file that it cannot read has the orientation 1.
- the encoder writes no metadata. The output has no EXIF, GPS, XMP, IPTC, or comment data.
- the output also has no ICC profile. A photo in Display P3 then shows slightly duller colors. Go has no color management library.
- a PNG with transparency gets a white background. JPEG has no alpha channel.
- the new encode also breaks a file that is valid as an image and as another type
- the resize uses one CPU core. An image of 6000 by 4000 pixels takes about 0.6 seconds.

## Serving

The file operation reads the file through the PocketBase filesystem and sends it with `http.ServeContent`. Each response has:
- `Content-Type: image/jpeg`. The app sets it and does not guess it.
- `Cache-Control: public, max-age=31536000, immutable`. A photo ID never gets other bytes. A `416` is not cached.
- `ETag` from the stored file name
- `X-Content-Type-Options: nosniff`. A browser never runs the file as a script or a style.
- `Content-Security-Policy: default-src 'none'; sandbox`. A file in its own tab cannot run code.
- `Content-Disposition: inline; filename="photo.jpg"`
- `Cross-Origin-Resource-Policy: same-origin`. Other sites cannot put the photos in their pages.

- a deleted photo gets `404` at once. A browser can still show it from its cache. Without a CDN, only the browser of each viewer keeps a copy.
- `img-src` of the UI policy has `blob:` for the preview of a selected file. See decision 00002. The UI calls `URL.revokeObjectURL` when the preview closes.

## Gallery queries

| Key | Direction |
|---|---|
| the overall of the review | descending |
| `created` of the review | descending |
| `id` of the review | descending |
| `created` of the photo | ascending |
| `id` of the photo | ascending |

- the keys have two directions, so the cursor condition is a list of `OR` terms
- the query joins the reviews of the subject to their photos. It sorts in a temporary B-tree. A subject has few photos.
- the covers of a page come from one query with `ROW_NUMBER() OVER (PARTITION BY parent ...)` and the same order
- each response computes the cover again. It needs no trigger and no stored column.
- a query plan test checks that the gallery query uses the indexes

## Delete

| Event | Result |
|---|---|
| a delete of a photo | the record and its two files are deleted |
| a delete of a review | its photos go with the cascade |
| a delete of a subject | the reviews of the creator and their photos go. See decision 00004. |
| a delete of a user | the reviews of the user and their photos go |

- PocketBase deletes the files after the delete transaction, in the background. It does not try a failed file delete again.
- a file can stay without a record after a failed file delete, a stop of the server, or a failed save of a new photo. `/v1` does not serve it.
- the job `photo-cleanup` of `app/api/internal/photocleanup` runs each day at 03:17 UTC
  - it lists the files of both photo collections with the PocketBase filesystem API and groups them by record ID
  - it deletes each folder without a record whose newest file is at least 1 hour old. The age protects an upload in progress.
  - it checks the record IDs in batches of 500. A failed folder logs an error, and the job continues.
  - superusers see the job under Settings, Crons in the dashboard and can start it there
- the job lists each file, also of records that exist. The filesystem API cannot list only the folders.

## The UI components

- `PhotoUpload` is a file input with "Add photo". It accepts the three types and checks the type and the size before the upload.
- `PhotoGrid` shows square thumbnails. Each one is a button with the alt text as its name. Without alt text, the name is "Photo 2 of 5".
- `PhotoViewer` is a `Dialog` with the `full` variant. It has a `caption` slot. The focus goes back to the thumbnail when it closes.
- `CoverPhoto` has the sizes `card` with `thumb` and `hero` with `full`. `hero` has a `srcset`, so a narrow screen loads the `thumb`. Its `<img>` has `alt=""` in a card.
- each `<img>` has `width`, `height`, and `loading="lazy"`. A thumbnail shows `Skeleton` until it loads. A failed load shows an image icon on `muted`.
- `PhotoGallery.vue` shows the gallery of both kinds with `PagedList`. The page loads it only when the subject has a cover.

## Tests

- the Go tests make their images in Go, such as a JPEG with GPS EXIF data, which then has no `APP1` segment
- the end-to-end tests use the images in `tests/photos`. `gps-rotated.jpg` has a GPS latitude and the orientation "rotate 90".
- the end-to-end tests check that the downloaded `full` file has no EXIF data, and that the upload form passes the CSP fixture

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| Photos that the creator of a subject uploads | Not selected. One user would choose the photos for all reviewers. |
| A file field with many files on the review | Not selected. A file has no ID or alt text, and an upload changes `updated` of the review. |
| Files from `/api/files` of PocketBase | Not selected. Clients would use a PocketBase API, and PocketBase sends the stored bytes without the headers above. |
| PocketBase thumbnails | Not selected. They keep the original file with its metadata. |
| Store the upload and remove only the EXIF segments | Not selected. Metadata is in many places. A new encode removes all of it. |
| A `multipart/form-data` upload with the alt text | Not selected. The raw body is simpler to validate and to generate. |
| An upload to S3 with a signed URL | Not selected. The app could not check the file first. |
| A separate host for the files | Not selected now. The new encode, `nosniff`, and `sandbox` remove the risk on one origin. |
| A queue or a wait for free memory | Not selected. A waiting upload holds its body, so the memory has no limit. |
| `503` for a full memory limit | Not selected. `429` already has `Retry-After`. |
| WebP or AVIF output | Not selected. Go has no encoder in the standard library. |
| `disintegration/imaging` for the decode and the resize | Removed. It had no release since 2019, and the uploads are untrusted data. |
| `libvips` through cgo | Not selected. The build would need a C compiler and a C library. |
| The newest photo as the cover | Not selected. A photo of a bad visit could become the cover. |
| A stored cover column with triggers | Not selected. The order needs the scores, so one query for each page is simpler. |
| A retry of each failed file delete in a hook, or a queue table | Not selected. A crash leaves no failure to retry. The scan finds all cases without new state. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
