---
description: Let the creator of a review add photos to it through the Public API.
status: implemented
---

# Motivation
[motivation]: #motivation

A review has only text and scores.
A photo shows the dishes, the room, and the menu better than text.
The backlog has the item "Allow adding photos to reviews".
A restaurant gets no photos of its own.

Photos are the first files that users upload.
Decision 00005 removed the `avatar` field of `users`, so no collection has a file field now.
It also records that a later file field needs a check of the Content Security Policy (CSP) risk.
Photos can also contain personal data, such as the GPS position of the camera.

Expected outcome:
- the creator of a review can add, describe, and delete photos of the review
- review responses list their photos
- the app removes all metadata from each photo before it stores it
- the server never serves a file in the form that the client sent
- the CSP of the UI stays strict
- the API serves no photo of a deleted review or user. PocketBase deletes their files. See [Deletion](#deletion) for the rare files that stay.

# Scope
[scope]: #scope

In scope:
- the photo operations and schemas in `specs/api/public-openapi.yaml`
- the PocketBase collection and file fields of photos
- the limits on size, count, dimensions, and type
- the validation and the processing of each upload
- the authorization of the photo operations
- how the app serves the files and how clients cache them
- the CSP change of the UI
- the rate limit of uploads
- the deletion of photos
- the privacy rules for photo data
- the new components in `specs/design-system` and the screen changes in `app/ui`
- the tests

Out of scope:
- photos of restaurants. Only reviews have photos. See [Alternatives](#alternatives).
- photos of the reviews on the restaurant itself, such as a cover photo on the restaurant page or the restaurant cards. Each photo shows only in its review.
- a limit on the total photo storage of a user. A user can write many reviews, so the count limit does not cap it. The upload rate limit makes the growth slow.
- a backup plan for `pb_data/storage`. The photos make it much larger than the database. The backlog has this item.
- a job that finds photo files without a record and deletes them. See [Deletion](#deletion). Decision 00009 adds it.
- photos of users, such as profile pictures
- video, animated images, and other file types
- moderation, reports of bad photos, and automatic checks of the content. PocketBase superusers delete bad photos in the dashboard.
- a separate host for files and S3 storage. The [Alternatives](#alternatives) list them.
- a CDN and its effects on photos. A later decision covers them:
  - a deleted photo stays in the CDN until the app purges it
  - the CDN needs the cache and security headers of [Serving and caching](#serving-and-caching)
  - the CDN host can need a change of `img-src` and `Cross-Origin-Resource-Policy`
- a change to the order of photos
- full-text search in the alt text

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

New concepts:
- **Photo**: one image of a review. It has an ID, an alt text, its size in pixels, and two image files.
- **Alt text**: a short text that describes the photo for screen readers. It is optional.
- **Variant**: one image file of a photo. `full` has at most 2048 pixels on its long side. `thumb` has at most 480 pixels on its long side.
- **Upload**: the request body of the add operation. It is the raw bytes of one JPEG, PNG, or WebP image.

## Operations

The photos have the base path `/restaurants/{restaurantId}/reviews/{reviewId}/photos`:
- `POST .../photos` adds one photo
- `PATCH .../photos/{photoId}` changes the alt text
- `DELETE .../photos/{photoId}` deletes one photo
- `GET .../photos/{photoId}/{variant}` gets the image file of one variant

The spec has no list operation for photos.
The `Review` schema gets a `photos` property with all photos of the review.
A review has at most 5 photos, so the list is short.
The `Restaurant` schema does not change.

## Example

A client adds a photo to a review:

```http
POST /v1/restaurants/k3x9m2p7q1w8z4a/reviews/r1b8c5e2g9i6k3m/photos
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: image/jpeg
Content-Length: 3481920

<the bytes of the image>
```

The response is `201 Created` with a `Location` header and the photo:

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
  "created": "2024-11-06 19:04:30.811Z"
}
```

The camera sent a picture of 4032 by 3024 pixels with the EXIF orientation "rotate 90".
The app turned the pixels, so `width` and `height` show the photo upright.
It made the long side 2048 pixels and removed the EXIF data.

The client then sets the alt text:

```http
PATCH /v1/restaurants/k3x9m2p7q1w8z4a/reviews/r1b8c5e2g9i6k3m/photos/p4h7j1l5n9q2s6v
Content-Type: application/merge-patch+json

{ "alt": "A plate of carbonara on a wooden table" }
```

The review now has the photo:

```json
{
  "id": "r1b8c5e2g9i6k3m",
  "title": "Great food, a bit expensive",
  "photos": [
    { "id": "p4h7j1l5n9q2s6v", "alt": "A plate of carbonara on a wooden table", "width": 1536, "height": 2048, "urls": { "full": "...", "thumb": "..." }, "created": "2024-11-06 19:04:30.811Z" }
  ]
}
```

The UI shows `urls.thumb` in an `<img>` with `width`, `height`, and `alt`.
A click on the thumbnail opens `urls.full` in a dialog.

# Implementation Details
[implementation-details]: #implementation-details

## API shape

The spec comes first.
`make generate` then makes the Go server interface and the TypeScript client, as decision 00003 says.

Schemas:
- `Photo`: `id`, `alt`, `width`, `height`, `urls`, and `created`. All are read-only except `alt`.
  - `alt` is `null` or a string of 1 to 255 characters. It uses the hidden character pattern of `Title`.
  - `width` and `height` are the pixels of the `full` variant, from 1 to 2048
  - `urls.full` and `urls.thumb` are paths on the same origin. Clients use them as given and do not build them.
  - a photo has no `updated`. The image of a photo never changes. A change of `alt` is not a new photo.
  - a photo has no `creator`. The creator of the review is the creator of each of its photos.
- `PhotoPatch`: `alt` only, with `additionalProperties: false`
- `Review.photos`: a required array of `Photo`, ordered by `created` and then `id`, with `maxItems` 5
- `PhotoVariant`: a path parameter with the values `full` and `thumb`

Add operation:
- the request body has the one content type `image/*` with `type: string` and `format: binary`. The handler accepts only `image/jpeg`, `image/png`, and `image/webp`.
- the spec has one content type, not three. With three binary types, oapi-codegen makes a request object with three `Body` fields, which does not compile. `typescript-fetch` sends the first type for all images.
- the spec has no `multipart/form-data` body. A raw body has one content type, and kin-openapi does not need to decode the parts. The alt text comes with a second request.
- a body of a different content type gets `415`
- the response is `201` with the `Photo` and a `Location` header to the photo path, which has the `PATCH` and `DELETE` operations
- a `422` uses the `InvalidImage` response. A `413` uses the `ContentTooLarge` response.

File operation:
- the `200` response has the content type `image/jpeg` with `type: string` and `contentMediaType: image/jpeg`
- it has the headers `Cache-Control`, `ETag`, `Content-Security-Policy`, and `X-Content-Type-Options`
- it allows `HEAD`, `If-None-Match`, and `Range`. `http.ServeContent` handles them. The spec has the `206`, `304`, and `416` responses for them.
- the security is `{}` with `bearerAuth`, like the other reads. An `<img>` sends no `Authorization` header.

A photo that exists but belongs to a different review gets `404`, like a review of a different restaurant.

The SDK clients:
- the TypeScript client of `typescript-fetch` sends a `Blob` body for a binary request body. The UI passes the `File` from the file input.
- the TypeScript client sets `Content-Type: image/*`. `addPhoto` in `app/ui/src/lib/photos.ts` changes it to the type of the file with an init override function. The override keeps the `Authorization` header.
- the Go strict server gives the handler an `io.Reader` for a binary body and the `Content-Type` of the request
- a test of the TypeScript SDK checks that the add operation sends the `Content-Type` of the override and the bearer token

Decision 00001 changes:
- the reviews section gets the `photos` property
- the error table gets `413 Content Too Large` for an upload over the limit
- the `409 Conflict` row gets "a photo over the count limit"
- the `429` row gets "too many uploads"

## Storage

A new collection `review_photos` keeps the photos.
It has these fields:
- `review`: a required relation to `reviews` with cascade delete
- `alt`: an optional text field with a maximum of 255 characters
- `width` and `height`: required number fields
- `full` and `thumb`: file fields with `MaxSelect` 1, the MIME type `image/jpeg`, and `Protected: true`
- `created`: the PocketBase autodate field
- an index on `(review, created, id)`

Rules:
- the API rules are empty, like the other collections. Only superusers use it through the PocketBase API.
- `Protected: true` stops the public download of the files under `/api/files`. A protected file needs a file token and the view rule. The view rule is `null`, so only superusers pass it.
- clients then use only `/v1`, as decision 00001 says. The CSP note in decision 00005 stays true for `/api/files`.
- the app names the files `full.jpg` and `thumb.jpg`. PocketBase adds a random suffix. The name of the file on the device of the user never reaches the server storage.
- the files are in `pb_data/storage`. The PocketBase filesystem API reads them, so a later S3 setting works without code changes.

Why a separate collection and not a file field on `reviews`:
- each photo needs its own ID, alt text, and size
- a file field with more than one file has no stable ID for each file. A delete uses the stored file name, which PocketBase makes.
- an upload changes only the photo record. It does not change `updated` of the review. A sort on `updated` then shows only text and score changes.
- the search hooks of `internal/searchindex` do not run for a new photo

## Limits

| Limit | Value | Status |
| --- | --- | --- |
| Upload size | 10 MB | `413` |
| Upload types | `image/jpeg`, `image/png`, `image/webp` | `415` |
| Pixels of the upload | at most 25 megapixels, at most 8192 on each side, at least 1 on each side | `422` |
| Photos of a review | 5 | `409` |
| Uploads of a user | 30 per hour | `429` |
| Long side of `full` | 2048 pixels | none. The app makes it smaller. |
| Long side of `thumb` | 480 pixels | none. The app makes it smaller. |

- 5 photos for each review and 30 uploads per user per hour are the values of the first release. Real use can change them later.
- the body limit of `prepare` stays 256 KB for JSON. The add operation uses its own limit of 10 MB.
- the `/v1` route replaces the PocketBase body limit. See [PocketBase body limit](#pocketbase-body-limit).
- GIF, HEIC, AVIF, SVG, and TIFF get `415`
- SVG can contain scripts. It never becomes a photo.
- iOS Safari changes HEIC photos to JPEG when the file input does not accept HEIC. The UI file input accepts only the three types.
- an animated PNG keeps only its first frame. The Go WebP decoder does not read animations, so an animated WebP can get `422`.

## PocketBase body limit

A probe test on PocketBase v0.40.6 gave these results:

| Request | Result |
| --- | --- |
| 12 MB with `Content-Length` to a route without its own limit | the handler reads all of it |
| 33 MB with `Content-Length` to a route without its own limit | PocketBase sends `413` before the handler |
| 33 MB without `Content-Length` to a route without its own limit | the handler runs. The read fails after 32 MB. |
| 12 MB with `Content-Length` to a route with `apis.BodyLimit(11 << 20)` | PocketBase sends `413` before the handler |
| 40 MB to a route with `apis.BodyLimit(0)` | the handler reads all of it |
| 12 MB of JSON to `/v1/users` | the Public API sends its Problem Details `413` |
| 33 MB of JSON to `/v1/users`, before `limitBody` of decision 00004 | PocketBase sends `413` in its own JSON format, not Problem Details |

- `apis.NewRouter` binds `apis.BodyLimit(apis.DefaultMaxBodySize)` to all routes. The value is 32 MB.
- a `BodyLimit` on a route replaces it, because both use the ID `pbBodyLimit`
- a 10 MB upload therefore reaches the handler in v0.40.6
- a later version with a lower default can stop uploads with a PocketBase error

The `/v1` route now has the `limitBody` middleware of decision 00004:
- it replaces the PocketBase limit. Each `413` of `/v1` is Problem Details, also for a body over 32 MB.
- it limits each `/v1` body to 10 MB, the size of the largest upload. `readBody` keeps 256 KB for the JSON bodies.
- `TestBodyLimit` sends 33 MB bodies. A route test for photos sends a 10 MB upload. They fail if a PocketBase update changes the result.

## Validation

The handler of the add operation runs these checks in order:

| Order | Check | Status |
| --- | --- | --- |
| 1 | the bearer token | `401` |
| 2 | the rate limit of uploads | `429` |
| 3 | the path parameters against the spec | `400` |
| 4 | the content type is one of the three types | `415` |
| 5 | `Content-Length` is at most 10 MB, if the client sends it | `413` |
| 6 | the restaurant and the review exist | `404` |
| 7 | the user is the creator of the review | `403` |
| 8 | the review has fewer than 5 photos | `409` |
| 9 | the body is at most 10 MB. `http.MaxBytesReader` stops the read. | `413` |
| 10 | the first bytes match the content type | `422` |
| 11 | `image.DecodeConfig` gives a size inside the pixel limits | `422` |
| 12 | the image decodes without an error | `422` |
| 13 | in one transaction: the review and its creator again, the count again, and the save | `404`, `403`, `409` |

- `limitBody` sends the `413` of check 5 before all other checks. It runs as a PocketBase middleware.
- checks 6 to 8 run before the app reads the body. A client that cannot add a photo then does not send 10 MB for nothing.
- check 11 reads only the header of the image. A small file that claims 50,000 by 50,000 pixels never decodes.
- check 13 repeats checks 6 to 8. Two parallel uploads cannot both pass the count, because PocketBase uses one SQLite writer.
- the decode and the encode run before the transaction. They are slow, and the transaction blocks other writes.
- at most 2 uploads decode at the same time. A semaphore holds the others after check 9.
- the semaphore does not cover the read of the body. Two slow clients then cannot block all uploads. The PocketBase read timeout of 5 minutes ends a slow read.
- an upload that waits for the semaphore holds its body of at most 10 MB. The rate limit of each user limits these uploads.
- a decode of 25 megapixels needs up to 100 MB of memory
- a `422` for an upload has a `detail` and no `errors` array. The body has no properties to name.
- the `PATCH` operation uses the JSON checks of decision 00004 and the spec validation of decision 00007
- the response validation of decision 00007 checks the status and the headers of the file operation with `ExcludeResponseBody`. `headerCheckWriter` checks them before it writes them. The image bytes then go to the client without a buffer.
- a file response that fails the check gets `500`, and its headers, such as `Cache-Control`, are dropped
- the parameter check of each operation gives kin-openapi a copy of the request without the body. The security check of kin-openapi reads the full body otherwise.
- a spec operation with a body that is not JSON skips `readBody`. A success response that is not JSON marks the file operation.

## Processing

The app never stores or serves the bytes that the client sent:
1. `imaging.Decode` with `imaging.AutoOrientation(true)` decodes the image. It turns the pixels to match the EXIF orientation.
2. `imaging.Fit` with the Lanczos filter makes `full` and `thumb`. An image that is already small enough keeps its size.
3. `image/jpeg` encodes each variant with quality 85.
4. the PocketBase save stores the two files.

- `github.com/disintegration/imaging` is already in `app/go.sum` through PocketBase. `golang.org/x/image/webp` decodes WebP.
- JPEG is the only output type. The standard library has no WebP encoder.
- a PNG with transparency gets a white background, because JPEG has no alpha channel
- the encoder of `image/jpeg` writes no metadata. The output has no EXIF, GPS, XMP, IPTC, or comment data.
- the output also has no ICC profile. A photo in Display P3, such as from an iPhone, then shows slightly duller colors. This is an accepted risk. A change of each photo to sRGB needs a color management library, which Go does not have.
- a new encode also breaks a file that is valid as an image and as another type, such as an HTML file in a JPEG comment

## Authorization

- only the creator of the review adds, changes, or deletes its photos
- other users get `403 Forbidden`
- anonymous and signed in users read all photos, like reviews
- a restaurant has no photo operations. The creator of a restaurant cannot change the photos of the reviews of other users.

## Serving and caching

The file operation reads the file through the PocketBase filesystem and sends it with `http.ServeContent`.

Headers of each file response:
- `Content-Type: image/jpeg`. The app sets it and does not guess it.
- `Cache-Control: public, max-age=31536000, immutable`. A photo ID never gets other bytes. Only `200`, `206`, and `304` get it. A `416` is not cached.
- `ETag` from the stored file name
- `X-Content-Type-Options: nosniff`. A browser then never runs the file as a script or a style.
- `Content-Security-Policy: default-src 'none'; sandbox`. A file that a user opens in its own tab cannot run code.
- `Content-Disposition: inline; filename="photo.jpg"`
- `Cross-Origin-Resource-Policy: same-origin`. Other sites cannot put the photos in their pages.

- a deleted photo gets `404` at once from the server
- a browser can still show a deleted photo from its cache until the cache ends. Without a CDN, only the browser of each viewer keeps a copy. This is an accepted risk.
- the `photos` of a review list add one query for each page, like `creator` and `restaurant_name` in decision 00004

## Content Security Policy

`img-src 'self'` already allows the files under `/v1`.
The policy of decision 00005 changes in one place:

| Directive | Value | Why |
| --- | --- | --- |
| `img-src` | `'self' blob:` | the upload form shows a preview of the selected file before the upload |

- `URL.createObjectURL` makes the `blob:` URL of the preview. The UI calls `URL.revokeObjectURL` when the preview closes.
- `blob:` is only in `img-src`. `script-src` and `default-src` stay `'self'`. A `blob:` image cannot run code.
- `data:` stays not allowed
- `'self'` trusts scripts and styles of this origin. The photos are on this origin, so the risk of decision 00005 applies.
  - the app sends only JPEG files that it encoded itself
  - `nosniff` stops a browser from running an image as a script or a style
  - the `sandbox` policy stops code in a file that the user opens in its own tab
- `tests/e2e/csp.spec.ts` gets the upload form, the preview, the gallery, and the viewer

## Rate limits

- the add operation gets a rate limit of 30 uploads per user per hour
- the boolean `x-rate-limited` of decision 00004 becomes the object `x-rate-limit`, such as `x-rate-limit: { key: user, name: photo-upload }`
- `name` selects the limit settings. Sign up and sign in both have `{ key: ip, name: sign-in }`. Each operation still has its own count.
- the app refuses to start with an unknown `key` or `name`, or with `key: user` on an operation that does not need a token
- the key is the user ID. Many users of one network then do not share one limit.
- the limiter of `ratelimit.go` gets a second instance with its own window and maximum
- `REVIEWS_PHOTO_UPLOAD_LIMIT` sets the uploads per hour. The end-to-end tests raise it.
- the token check runs before the upload limit, because the key is the user. Anonymous requests get `401` and never count.
- the PATCH and DELETE operations have no new limit
- the file operation has no limit in the app. A reverse proxy can limit it.
- the limit is in memory, like the sign in limit. Decision 00004 has the same open question for more than one server.

## Deletion

| Event | Result |
| --- | --- |
| `DELETE` of a photo | the record and its two files are deleted |
| `DELETE` of a review | its photos are deleted through the cascade of `review` |
| `DELETE` of a restaurant | only a restaurant without reviews can be deleted. It has no photos. |
| delete of a user | the reviews of the user are deleted, and their photos with them |

- PocketBase deletes the files of a record after the delete transaction succeeds. This also applies to a cascade delete.
- the file delete runs in the background, outside the transaction. PocketBase does not try a failed file delete again.
- a failed delete transaction keeps the record and its files
- a Go test checks each row of the table and that `pb_data/storage` has no file of a deleted photo
- a file can stay in storage without a record in these cases:
  - the file delete fails, such as from a disk error or an S3 API limit
  - the server stops before the background file delete ends
  - the save of a new photo fails, and the delete of its new files also fails or does not run
- a file without a record is protected, and `/v1` does not serve it. The daily job of decision 00009 deletes it.
- a job that finds and deletes these files is out of scope. Decision 00009 adds a daily job for it.

## Privacy

- the app removes all metadata, as [Processing](#processing) says. This includes the GPS position, the camera model, the serial number, and the time of the photo.
- the name of the file on the device never reaches the server
- a photo response has no user fields. The review shows its `creator` as decision 00001 says.
- a photo can show people. The UI text of the upload form asks the user to add only photos of food, rooms, and menus. Superusers delete other photos.
- the UI does not upload a file before the user selects "Add photo"
- the privacy text of the site must name photos. The project has no privacy page yet.

## UI and design system

New components in `specs/design-system/components`, with a `README.md` and a `preview.html` each:
- `PhotoUpload`: a file input with the label "Add photo". It accepts JPEG, PNG, and WebP. It shows the preview, an optional alt text `Input`, the "Upload photo" button, and the progress state. It checks the type and the size before the upload and shows the error text of the API after it.
- `PhotoGrid`: thumbnails in a grid with the gap `space-2`, the radius `radius-md`, and `object-cover` in a square. Each thumbnail is a button with the alt text as its accessible name. Without alt text, the name is "Photo 2 of 5".
- `PhotoViewer`: a `Dialog` with the `full` variant, the alt text, "Previous photo" and "Next photo" buttons, and the arrow keys. The creator also gets "Edit description" and "Delete photo".
- `Dialog`: the shadcn-vue dialog on Reka UI. The UI did not have it before. `PhotoViewer` uses it.

Rules for the README of the design system:
- each `<img>` has `width`, `height`, and `loading="lazy"` from the API. The page then does not move while images load.
- a thumbnail shows `Skeleton` until it loads
- a photo that fails to load shows the `muted` background with an image icon from `lucide-vue-next`
- the UI text follows the content rules: "Add photo", "Delete this photo?", "Describe the photo for people who cannot see it"
- an error text says what to do next, such as "Select a JPEG, PNG, or WebP image of at most 10 MB."

Screen changes in `app/ui`:
- each review card on the restaurant page and on `/my-reviews` shows a `PhotoGrid` of its photos. The viewer of a card has no actions.
- the review page shows the photos. Its viewer gives the creator the edit and delete actions.
- the review form gets a "Photos" section after the form. A field of the section is then not part of the review form, and Enter in it does not save the review.
- in the edit mode, the section shows the photos with the actions and uploads each new photo at once
- a new review gets its photos after the first save. The create form keeps the selected files and uploads them after the create succeeds. It tries each photo. A failed upload shows an error toast with the number of failed photos and opens the edit form of the saved review.
- the alt text needs a second request. A failed alt text keeps the added photo and shows "Description not saved". A retry then does not add the photo a second time.
- a `401` on the create opens the sign in page. The queued files stay in the memory of the page and come back with the text draft. A reload of the page loses them.
- the restaurant header of the restaurant page, the restaurant cards, and the restaurant forms do not change. They show no photos of the reviews.
- `src/lib/validation.ts` gets the type and the size checks

## Tests

Go tests in `app/internal/publicapi`:
- each row of the [Validation](#validation) table. The tests make their images in Go, such as a JPEG with an EXIF segment and a PNG header with too many pixels.
- a JPEG with GPS EXIF data: the stored file has no `APP1` segment
- a JPEG with orientation 6: `width` and `height` change places
- a PNG that claims 50,000 by 50,000 pixels: `422` without a decode
- a text file sent as `image/jpeg`: `422`
- two parallel uploads at the last free place: one `201` and one `409`
- the headers of the file operation, `304` for `If-None-Match`, and `404` after a delete
- `/api/files` does not serve the files without a superuser
- a 10 MB upload reaches the handler. `TestBodyLimit` already covers bodies over 32 MB.
- each row of the [Deletion](#deletion) table

Go tests in `app/migrations` check the collection, its rules, and the cascade.

UI tests with Vitest:
- `PhotoUpload` refuses a GIF and a file of 11 MB before the upload
- `PhotoGrid` and `PhotoViewer` pass `axe-core`
- the viewer moves with the arrow keys and gives the focus back to the thumbnail when it closes

End-to-end tests in `tests/e2e/photos.spec.ts`:
- the tests make their records through `@reviews/sdk` and add photos with the SDK or with `setInputFiles`
- the fixture images are in `tests/photos`. They are small files that the tests can keep in git. `gps-rotated.jpg` has a GPS latitude and the orientation "rotate 90".
- a user adds a photo with alt text to a review. The review card shows a thumbnail with that accessible name.
- a user adds a photo on the create form of a review. The saved review shows it.
- a user opens the viewer, moves to the next photo, and closes it with Escape
- a user deletes a photo after the confirm dialog. The file URL then gets `404`.
- another user sees no "Add photo" and no "Delete photo". The API gives `403` to that user.
- an anonymous visitor sees the photos
- a deleted review takes its photos with it
- the downloaded `full` file has no EXIF data
- the `noCspViolations` fixture passes on the upload form with a `blob:` preview
- `rate-limit.spec.ts` gets a test with a low `REVIEWS_PHOTO_UPLOAD_LIMIT`
- `accessibility.spec.ts` gets a review with photos and the open viewer
- the tests run in Chromium, Firefox, WebKit, and the small viewport, like the other specs
- the test server gets `REVIEWS_PHOTO_UPLOAD_LIMIT` with a high value. The parallel tests share users only within one spec.

## Alternatives
[alternatives]: #alternatives

| Alternative | Result |
| --- | --- |
| Photos of restaurants, added by the creator of the restaurant | Not selected. One user would choose the photos of a restaurant for all reviewers. A review photo shows what one user saw on one visit. |
| A cover photo on the restaurant page and the restaurant cards, taken from the reviews | Not selected. Each photo shows only in its review. The API then needs no `Restaurant` property and no list of the photos of all reviews of a restaurant. |
| A file field with more than one file on `reviews` | Not selected. A file has no ID or alt text. An upload changes `updated` of the review. See [Storage](#storage). |
| Serve the files from `/api/files` of PocketBase | Not selected. Clients would use a PocketBase API, which decision 00001 does not allow. PocketBase also sends the stored bytes without the headers of [Serving and caching](#serving-and-caching). |
| PocketBase thumbnails with the `thumb` query parameter | Not selected. They work only under `/api/files`. They also keep the original file, with its metadata. |
| Store the file that the client sent and remove only the EXIF segments | Not selected. Metadata is in many places, such as XMP and PNG text chunks. A new encode removes all of it. It also breaks files that are two types at once. |
| A `multipart/form-data` upload with the file and the alt text | Not selected. The raw body is simpler to validate and to generate. The alt text needs a second request. |
| Upload directly from the browser to S3 with a signed URL | Not selected. The app cannot check or change the file before a client can get it. The project has no S3. |
| A separate host for the files, as decision 00005 says | Not selected now. The new encode, `nosniff`, and the `sandbox` policy remove the risk on the same origin. A separate host needs a second name, TLS, and CORS. It can come with a CDN. |
| Upload first and show the server `thumb` as the preview | Not selected. A new review has no ID until the save, so the create form cannot upload first. The `blob:` preview needs only an `img-src` change. |
| WebP or AVIF output | Not selected. Go has no encoder in the standard library. A cgo library makes the build harder. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
