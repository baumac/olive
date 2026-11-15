---
description: Define the OAS specification for the Public API.
status: implemented
---

# Motivation
[motivation]: #motivation

To allow the UI and API to be developed independently, we need to define the OpenAPI Specification (OAS) for the Public API.

The UI also needs users to sign up and sign in.
The Public API needs an auth token for create, update, and delete operations.
The Public API wraps the PocketBase user auth, so clients do not depend on PocketBase.

# Scope
[scope]: #scope

In scope:
- defining one OAS specification in `specs/api/public-openapi.yaml` with two tags:
  - `users`: sign up, sign in, and auth token refresh
  - `restaurants`: restaurants and reviews

Out of scope:
- All APIs provided by PocketBase. Clients use only the Public API.
- Password reset, email verification, and changes to a user after sign up. Decision 00015 added the sign out on the server. Decision 00037 added the other operations.
- Generating client libraries and documentation for the Public API
- Implementation of the API spec
- Reviews of things that are not restaurants, such as books or movies
- Filters and sorts on the category averages and on `review_count` of restaurants. These can be added later without a breaking change.
- Search, such as matching part of a `name`. Filters compare whole values only.
- Preventing or merging duplicate restaurants. Two restaurants can have the same `name`. PocketBase superusers clean up duplicates. Decision 00017 added locations and stops a second restaurant at one place.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

New Concepts:
- **Public API**: the public-facing API that allows users to interact with the review system
- **Review**: scores and feedback for a single restaurant
  - a review must have exactly one restaurant
  - a user can have at most one review per restaurant
  - a review scores five categories: space, service, food, price, and cleanliness
- **Restaurant**: a restaurant that can be reviewed
  - a restaurant has zero to many reviews
- **User**: a person who can sign in
  - a user creates restaurants and reviews
- **Auth token**: a token that the Public API gives to a user at sign up and sign in. Clients send it as a bearer token.

The file `specs/api/public-openapi.yaml` contains the OpenAPI Specification (OAS) for the Public API. In the OAS file,
the endpoints, request/response schemas, and authentication mechanisms are defined.

The OAS uses tags to separate the two sets of operations:
- `users`: the operations of the [Users API](#users-api)
- `restaurants`: the operations of the [Restaurants API](#restaurants-api) and the [Reviews API](#reviews-api)

The paths of the Public API are relative to the versioned base path `/v1`.
Paths use the OAS path template syntax, for example `{restaurantId}`.

## Restaurants API

A restaurant can be reviewed and has zero to many reviews.
The restaurants API exists on the base path `/restaurants`.

### Properties

The properties of a restaurant are as follows:
- required properties:
    - `name`: string with a length of 1 to 255 characters. It must contain at least one character that is not ASCII whitespace.
    - `location`: the point and the address of the restaurant. Decision 00017 added it. It is `null` for a restaurant from before that decision.
- optional properties:
    - `description`: string with a maximum length of 10,000 characters
    - `tags`: see [Tags](#tags)
- read-only properties maintained by the API:
    - `created`: timestamp
    - `updated`: timestamp
    - `creator`: the user who created the restaurant, see [Creator](#creator). It is `null` after that user is deleted. See [User Deletion](#user-deletion).
    - `review_count`: the number of reviews of the restaurant
    - `rating`: the average scores of the reviews, see [Rating](#rating). It is `null` for a restaurant without reviews.
    - `cover`: the first photo of the gallery of the restaurant. It is `null` for a restaurant without photos. Decision 00030 has the gallery.
    - `id`: PocketBase record ID

Example:
```json
{
  "id": "k3x9m2p7q1w8z4a",
  "name": "Luigi's Trattoria",
  "description": "A family-run Italian restaurant",
  "tags": ["italian", "pasta"],
  "creator": {
    "id": "u5n1c8r3t6y0b2e",
    "name": "Ana"
  },
  "review_count": 2,
  "rating": {
    "overall": 7.6,
    "space": 7.5,
    "service": 8.5,
    "food": 9,
    "price": 5.5,
    "cleanliness": 7.5
  },
  "created": "2024-11-04 01:26:55.002Z",
  "updated": "2024-11-04 01:26:55.002Z"
}
```

### Rating

`rating` has the average of each review score over all reviews of the restaurant.
`overall` is the average of the five averages. Each review then counts the same in each category.
The API computes the averages before it rounds. It rounds each value to one decimal.
Every response has the current values. A new, changed, or deleted review changes them at once.

### Operations

The operations of the restaurants API are as follows:
- `POST /restaurants` to create a single restaurant
- `GET /restaurants` to list restaurants
    - results are paginated, see [Pagination](#pagination)
    - results are sortable by the following properties ascending and descending, see [Sorting](#sorting):
      - `name`
      - `overall`: the `rating.overall` value. Restaurants without reviews come last in both directions.
      - `created`
      - `updated`
    - the default sort is `-created`
    - results are not filterable by `name` or `description`. The `search` parameter finds words in them, see [Search](#search).
    - results are filterable by the following properties and operators, see [Filtering](#filtering):
        - `overall`: `lt`, `lte`, `eq`, `gte`, `gt`. It compares the rounded `rating.overall` value. Restaurants without reviews match no `overall` filter.
        - `tags`: `has`, `has_any`, `has_all`
        - `created`: `lt`, `lte`, `eq`, `gte`, `gt`
        - `updated`: `lt`, `lte`, `eq`, `gte`, `gt`
        - `creator`: `eq`, `in`
- `GET /restaurants/{restaurantId}` to retrieve a single restaurant
- `PATCH /restaurants/{restaurantId}` to update a single restaurant
    - `description` and `tags` can always be changed
    - `name` can be changed only while no other user has a review of the restaurant. Otherwise the API returns `409 Conflict`. Decision 00016 changed this rule.
    - a `name` value that is equal to the stored value is not a change. It does not cause `409 Conflict`.
    - this prevents one user from changing what the reviews of other users are about
- `DELETE /restaurants/{restaurantId}` to delete a single restaurant
    - a restaurant that has a review of another user cannot be deleted. The API returns `409 Conflict`.
    - this prevents one user from deleting the reviews of other users
    - the review of the user is deleted with the restaurant. Decision 00016 changed this rule.

## Reviews API

A review is a set of scores and associated feedback for a single restaurant.
The reviews API exists on the base path `/restaurants/{restaurantId}/reviews`.
`GET /reviews` lists the reviews of all restaurants.

A review has one schema.
It scores the space, service, food, price, and cleanliness of the restaurant.

The first version had two review kinds with a `kind` discriminator.
A `basic` review had one `rating` from 1 to 5. A `joc_de_cartes` review had the four scores.
One kind is simpler for clients, filters, and sorts, so the API dropped `kind`, `rating`, and `basic`.
This change breaks `/v1` clients. The only client is the UI in this repository, and it changes with the API.
The migration gives each old `basic` review two times its `rating` in each of the four scores.

`cleanliness` came later as a fifth score.
The `space` score covered cleanliness before, so the migration gives existing reviews their `space` value as `cleanliness`.

### Properties

- required properties:
    - `space`: decoration, comfort, atmosphere, and kitchen setup
    - `service`: professionalism, speed, friendliness, and attentiveness of the waitstaff
    - `food`: presentation, flavor, ingredient quality, and culinary execution
    - `price`: value for money, and whether the final bill matches the quality provided
    - `cleanliness`: cleanliness of the dining room, the tableware, and the restrooms
    - `title`: string with a length of 1 to 255 characters. It must contain at least one character that is not ASCII whitespace.
- optional properties:
    - `feedback`: string with a maximum length of 10,000 characters
    - `tags`: see [Tags](#tags)
- read-only properties maintained by the API:
    - `restaurant`: id of the reviewed restaurant. The API sets it from the request path.
    - `restaurant_name`: the `name` of the reviewed restaurant. It is the current name. A rename of the restaurant changes it. A list of reviews from many restaurants can then show the restaurant without more requests.
    - `created`: timestamp
    - `updated`: timestamp
    - `creator`: the user who created the review, see [Creator](#creator)
    - `photos`: the photos of the review, at most 5. Decision 00008 has the photo operations.
    - `id`: PocketBase record ID

Each score is an integer with a minimum value of 1 and a maximum value of 10.
The API does not calculate a total score.
A total can be added later without a breaking change.

Example:
```json
{
  "id": "r1b8c5e2g9i6k3m",
  "restaurant": "k3x9m2p7q1w8z4a",
  "restaurant_name": "Luigi's Trattoria",
  "space": 7,
  "service": 9,
  "food": 8,
  "price": 6,
  "cleanliness": 8,
  "title": "Great food, a bit expensive",
  "tags": ["date-night"],
  "creator": {
    "id": "u5n1c8r3t6y0b2e",
    "name": "Ana"
  },
  "created": "2024-11-06 19:02:11.120Z",
  "updated": "2024-11-06 19:02:11.120Z"
}
```

### Operations

The operations of the reviews API are as follows:
- `POST /restaurants/{restaurantId}/reviews` to create a single review for a restaurant
  - if the user already has a review for the restaurant, the API returns `409 Conflict`
- `GET /restaurants/{restaurantId}/reviews` to list reviews for a restaurant
  - results are paginated, see [Pagination](#pagination)
  - results are sortable by the following properties ascending and descending, see [Sorting](#sorting):
    - `space`
    - `service`
    - `food`
    - `price`
    - `cleanliness`
    - `title`
    - `created`
    - `updated`
  - the default sort is `-created`
  - results are not filterable by the following properties:
    - `title`
    - `feedback`
  - results are filterable by the following properties and operators, see [Filtering](#filtering):
    - `space`: `lt`, `lte`, `eq`, `gte`, `gt`
    - `service`: `lt`, `lte`, `eq`, `gte`, `gt`
    - `food`: `lt`, `lte`, `eq`, `gte`, `gt`
    - `price`: `lt`, `lte`, `eq`, `gte`, `gt`
    - `cleanliness`: `lt`, `lte`, `eq`, `gte`, `gt`
    - `tags`: `has`, `has_any`, `has_all`
    - `created`: `lt`, `lte`, `eq`, `gte`, `gt`
    - `updated`: `lt`, `lte`, `eq`, `gte`, `gt`
    - `creator`: `eq`, `in`
- `GET /restaurants/{restaurantId}/reviews/{reviewId}` to retrieve a single review for a restaurant
- `PATCH /restaurants/{restaurantId}/reviews/{reviewId}` to update a single review for a restaurant
  - the scores, `title`, `feedback`, and `tags` can be changed
- `DELETE /restaurants/{restaurantId}/reviews/{reviewId}` to delete a single review for a restaurant

- `GET /reviews` to list the reviews of all restaurants
  - it has the pages, the sorts, and the filters of `GET /restaurants/{restaurantId}/reviews`
  - it also has these filters, see [Filtering](#filtering):
    - `restaurant`: `eq`, `in`
  - `creator[eq]` gives the reviews of one user, such as "My reviews" in the UI
  - an unknown restaurant in `restaurant[eq]` gives an empty list, not `404 Not Found`

A review that exists but belongs to a different restaurant is treated as not found. The API returns `404 Not Found`.

If the restaurant in the path does not exist, all reviews operations return `404 Not Found`.
This includes the list operation. It does not return an empty list.

## Users API

The users API exists on the base path `/users`.
It lets a user sign up, sign in, and refresh an auth token.

### Properties

The properties of a user are as follows:
- required properties to sign up:
    - `email`: email address with a maximum length of 255 characters. It must be unique.
    - `password`: string with a length of 8 to 71 characters. Its UTF-8 encoding must be at most 72 bytes.
    - `name`: string with 1 to 255 characters and at least one character that is not ASCII whitespace. Other users see it as the [Creator](#creator) `name`.
- read-only properties:
    - `id`: user ID
    - `created`: timestamp
    - `updated`: timestamp

Only the user of the auth token gets the user object.
Other users see only the [Creator](#creator) fields.
Passwords are never in a response.

Example:
```json
{
  "id": "u5n1c8r3t6y0b2e",
  "email": "ana@example.com",
  "name": "Ana",
  "created": "2024-11-01 08:12:40.517Z",
  "updated": "2024-11-01 08:12:40.517Z"
}
```

### Operations

The operations of the users API are as follows:
- `POST /users` to sign up a user
    - it does not require authentication
    - the response is `201 Created` with an auth token. The user is signed in after sign up.
    - the response has no `Location` header. The API has no operation to read a user.
    - an `email` that another user has causes `409 Conflict`
- `POST /users/tokens` to sign in a user
    - it does not require authentication
    - the request body has the `email` and `password` properties
    - the response is `200 OK` with an auth token
    - a wrong email address or password causes `401 Unauthorized`. The response does not say which one is wrong.
- sign up and sign in ignore the `Authorization` header. A stored token that is not valid does not block them.
- sign up and sign in have a rate limit. Too many requests cause `429 Too Many Requests` with a `Retry-After` header.
- sign in also has a limit of wrong passwords per user. See decision 00015.
- `DELETE /users/tokens` to sign out a user on all devices. See decision 00015.
- `POST /users/tokens/refresh` to get a new auth token
    - it requires a valid auth token that is not expired
    - the response is `200 OK` with a new auth token
    - clients also use it to check a stored token

The auth token response has this format:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires": "2024-11-08 08:12:40.517Z",
  "user": { "id": "u5n1c8r3t6y0b2e", "email": "ana@example.com", "name": "Ana", "created": "...", "updated": "..." }
}
```

Clients must not parse `token`.
`expires` is the time when the token stops being valid.

A typical client flow is as follows:
1. sign up the user, or sign in the user. Store the auth token.
2. send the auth token as a bearer token to the Public API
3. refresh the auth token before it expires. If the refresh fails, sign in again.

# Implementation Details
[implementation-details]: #implementation-details

## Authentication
Create, update, and delete endpoints require authentication using a bearer token in the `Authorization` header.
The client gets the token from the [Users API](#users-api).
The token is a PocketBase auth token for a record in the `users` auth collection.
Requests to these endpoints without a valid token get `401 Unauthorized`.

Read endpoints (`GET`) do not require authentication.
Anonymous users can read all restaurants and reviews.
A read request with an invalid token gets `401 Unauthorized`.
The API does not treat it as an anonymous request.

Sign up and sign in ignore the `Authorization` header.

## Authorization
- anonymous and authenticated users can read all restaurants and reviews
- any authenticated user can create restaurants and reviews
- only the creator of a restaurant or review can update or delete it. Other users get `403 Forbidden`.
- a restaurant without a creator can be updated or deleted by each user who has a review of it. See decision 00016.
- PocketBase superusers manage all records through the PocketBase API, not the Public API

## Text that hides characters

Control characters, direction marks and overrides (such as U+202E and U+061C), and invisible characters cause `422` in `name` and `title`.
The invisible characters are the zero-width spaces, the soft hyphen, the invisible operators, and the Hangul fillers.
A name with them can look like the name of another record, or show as blank space.
`description` and `feedback` allow line breaks and tabs, but not the other characters.
The zero-width joiners stay allowed, because emoji and some scripts use them.

## Token lifetime

Tokens are valid until they expire or until the user signs out.
- each sign in and each refresh makes a new token. The tokens before it stay valid.
- `DELETE /users/tokens` signs out the user. It makes all tokens of the user invalid. Decision 00015 added it.
- a token that leaks works until it expires or until the user signs out

## Account Enumeration

Sign up shows whether an email address has an account.
An email that another user has causes `409 Conflict`.
This is an accepted risk.
An account shows only that a person reviews restaurants.

These rules limit the risk:
- sign up and sign in have a rate limit. It makes checks of many email addresses slow.
- sign in returns the same `401 Unauthorized` for an unknown email address and a wrong password
- sign in takes the same time for an unknown email address and a wrong password. The API checks a password hash also when no user has the email address.

Email verification can remove this risk.
Sign up then gives the same response for all email addresses, and the API sends the result by email.
The backlog has this change.


When a user is deleted:
- the reviews of that user are deleted
- the restaurants of that user stay. Their `creator` becomes `null`.
- a restaurant with a `null` creator can be changed or deleted by each user who has a review of it. See decision 00016.

Restaurants stay because the reviews of other users can refer to them.

## Common Formats

- Timestamps use the PocketBase format: UTC with millisecond precision and a space between the date and the time, for example `2024-11-04 01:26:55.002Z`.
  - this is not RFC 3339, so the OAS schema uses a `string` with a `pattern` instead of `format: date-time`
  - decision 00023 changes the timestamps to RFC 3339
  - in query parameters the space must be URL-encoded, for example `2024-11-04%2001:26:55.002Z`
  - the `created` and `updated` properties use the default PocketBase autodate fields with the same names
- IDs use the default PocketBase record ID format: 15 characters that match the pattern `^[a-z0-9]+$`, for example `k3x9m2p7q1w8z4a`.
  - PocketBase generates IDs at random when it creates a record
  - IDs are not ordered by time
- String lengths count Unicode code points, not bytes. This matches JSON Schema `maxLength`.

### Creator

The `creator` property is an object with the public fields of a user:
- `id`: the PocketBase record ID of the user
- `name`: the `name` field of the user in the PocketBase `users` collection. It has the same rules as the user `name`.

The API does not return other user fields, such as `email`.
The `creator` filter compares the user `id`, for example `creator[eq]=u2g6k0m4o8q1s5w`.
The `creator` filter cannot select restaurants with a `null` creator.
PocketBase superusers find these restaurants through the PocketBase API.

### Tags

`tags` is an optional array of strings with these rules:
- at most 20 items
- each item has a length of 1 to 63 characters
- each item matches the pattern `^[a-z0-9]+(-[a-z0-9]+)*$`
- items are unique

The API does not change tag values. Invalid tags cause `422 Unprocessable Content`.

## Request Bodies

- request bodies with properties that the schema does not define cause `400 Bad Request`
- read-only properties in the request body cause `400 Bad Request`
- property values that break a schema rule cause `422 Unprocessable Content`

## Create operations

- the request content type is `application/json`. Other content types cause `415 Unsupported Media Type`.
- the response is `201 Created` with the created resource in the body
- the `Location` header contains the URL of the created resource
- sign up is different. Its body is an auth token, and it has no `Location` header.

## Update operations

PATCH requests use JSON Merge Patch (RFC 7396) with the content type `application/merge-patch+json`.
Other content types cause `415 Unsupported Media Type`.
- properties not in the request body stay unchanged
- `null` clears an optional property
- `null` on a required property causes `422 Unprocessable Content`
- an empty body `{}` is a valid patch. It changes nothing and keeps `updated`.
- the response is `200 OK` with the updated resource in the body

## Delete operations

The response is `204 No Content` with an empty body.

## List operations

List endpoints have the following implementation details:
- results will be filterable using query parameters
- results are paginated using cursor pagination

### Filtering

A filter is a query parameter in the format `{property}[{operator}]={value}`.
All filters in a request must match. They are combined with a logical AND.

| Operator | Meaning | Example |
| --- | --- | --- |
| `eq` | equal to | `food[eq]=5` |
| `lt` | less than | `food[lt]=3` |
| `lte` | less than or equal to | `created[lte]=2024-11-04%2000:00:00.000Z` |
| `gt` | greater than | `food[gt]=3` |
| `gte` | greater than or equal to | `updated[gte]=2024-11-04%2000:00:00.000Z` |
| `in` | equal to any of the values | `creator[in]=u5n1c8r3t6y0b2e&creator[in]=u2g6k0m4o8q1s5w` |
| `has` | the array contains the value | `tags[has]=italian` |
| `has_any` | the array contains at least one of the values | `tags[has_any]=italian&tags[has_any]=pasta` |
| `has_all` | the array contains all of the values | `tags[has_all]=italian&tags[has_all]=pasta` |

Operators that take more than one value use a repeated query parameter.
An unknown property, an unknown operator, or an operator that is not allowed for a property causes `400 Bad Request`.

The tag operators compare whole tags.
For example, `tags[has]=italian` does not match the tag `south-italian`.

### Search

`GET /restaurants`, `GET /restaurants/{restaurantId}/reviews`, and `GET /reviews` take a `search` query parameter for full-text search.
- restaurants: `name`, `description`, `tags`, and the address of the location. Decision 00017 added the address.
- reviews: `title`, `feedback`, `tags`, and `restaurant_name`
- an item matches when it has all words of the search. Each word of two or more characters also matches longer words that start with it. For example, `carb` matches "carbonara". A one-character word matches only that word.
- the search ignores case and accents in all languages. For example, `cafe` matches "Café".
- the search splits words at spaces and punctuation. For example, `date-night` matches the tag `date-night` and the text "date night".
- the search is 1 to 255 characters with at most 10 words. A search without a letter or a digit causes `400 Bad Request`.
- characters such as `"`, `*`, and `OR` have no special meaning. The search is plain text.
- `search` combines with all filters. All of them must match.

The `relevance` sort puts the best matches first.
The score of an item adds a weight for each property that has all words of the search:
- restaurants: `name` 10, `tags` 4, `description` 1
- reviews: `title` 10, `tags` 4, `restaurant_name` 2, `feedback` 1

An item that has the words only across properties has the score 0.
The score depends only on the item. A change of another item does not move it, so pages of a `relevance` sort stay stable.
With `search` and no `sort`, the sort is `relevance`.
A `relevance` sort without `search` causes `400 Bad Request`.

### Sorting

The `sort` query parameter is a comma-separated list of properties.
A `-` prefix sorts a property in descending order.
For example, `sort=-food,title` sorts by food from high to low, then by title from A to Z.

The `name` and `title` sorts ignore case for the ASCII letters `A` to `Z`.
For example, "apple" comes before "Zebra".
These sorts do not ignore case for other letters, and they do not use language-specific rules.
For example, "Ärger" and "ärger" do not sort as equal.

The API always adds `id` as the last sort key.
IDs are unique.
This gives a stable order when other sort values are equal.

### Pagination

- the `limit` query parameter sets the page size. The default is 20 and the maximum is 100. A value outside 1 to 100 causes `400 Bad Request`.
- the `cursor` query parameter gets the next page. Clients must not parse or build cursor values.
- a cursor is valid only with the same filters and sort that made it. A cursor with different filters or sort causes `400 Bad Request`.

The response body has this format:
```json
{
  "items": [],
  "next_cursor": "c2Fk9xQe7LmT2vBn"
}
```

`next_cursor` is `null` on the last page.

The cursor does not give a snapshot of the results.
An item that changes during paging can move to a different position in the sort order.
A client can then miss the item or get it two times, mostly with a sort on `updated`.

## Errors

The Public API error responses use RFC 9457 Problem Details with the content type `application/problem+json`.
Validation errors add an `errors` array with one item for each invalid property.
The `property` of an array item uses a zero-based index, for example `tags[2]` for the third tag.

Example:
```json
{
  "type": "about:blank",
  "title": "Unprocessable Content",
  "status": 422,
  "detail": "The request body has invalid properties.",
  "errors": [
    { "property": "food", "message": "must be between 1 and 10" }
  ]
}
```

| Status | Use |
| --- | --- |
| `400 Bad Request` | malformed request, invalid query parameters, or unknown or read-only properties in the body |
| `401 Unauthorized` | missing bearer token on a create, update, delete, or token refresh request, an invalid bearer token on any request, or a wrong email or password on sign in |
| `403 Forbidden` | the user is not the creator of the resource. For a restaurant without a creator, the user has no review of it. |
| `404 Not Found` | the resource or the restaurant in the path does not exist, or the review belongs to a different restaurant |
| `409 Conflict` | sign up with an email that another user has, duplicate review for a restaurant, delete of a restaurant that has a review of another user, change of `name` or `location` on a restaurant that has a review of another user, a place that another restaurant has, or a photo over the count limit |
| `413 Content Too Large` | a request body over the size limit, such as a photo over 10 MB |
| `415 Unsupported Media Type` | wrong request content type |
| `422 Unprocessable Content` | the request body has invalid property values |
| `429 Too Many Requests` | too many sign up, sign in, photo upload, or place search requests in a short time |
| `503 Service Unavailable` | the place search of decision 00017 is off or its geocoder does not answer |

## OAS Mapping

`Review`, `ReviewCreate`, and `ReviewPatch` are plain object schemas.
The create and patch schemas set `additionalProperties: false`.

The `users` and `restaurants` operations share the security scheme, the error responses, and the common schemas.
The sign in `401` uses its own `InvalidCredentials` response.
Its `WWW-Authenticate` header is `Bearer`, the scheme of the Public API.

## PocketBase Mapping

The spec uses the default PocketBase `id`, `created`, and `updated` fields without changes.
The `creator` and `restaurant` properties are PocketBase relation fields with the same names.
The API expands the `creator` relation and returns only the user `id` and `name`.

The PocketBase collections need this additional setup:
- the API rules of `restaurants` and `reviews` are empty. Only superusers can use these collections through the PocketBase API.
  - all user access goes through the Public API
  - without this, users can skip the Public API rules, such as creator-only changes and the `409 Conflict` checks
- `tags` is a `json` field. PocketBase has no field type for a list of free-form strings.
  - the API validates the [Tags](#tags) rules
  - tag filters use SQLite `json_each` to compare whole items. The PocketBase `~` operator matches substrings of the JSON text, so tag filters do not use it.
- the list operations order by `name COLLATE NOCASE` for the restaurant `name` sort and by `title COLLATE NOCASE` for the review `title` sort
  - the sort direction and the `id` tie-breaker come from the list query
  - SQLite `NOCASE` ignores case only for ASCII letters
  - `restaurants` has an index on `(name COLLATE NOCASE, id)`. `reviews` has an index on `(title COLLATE NOCASE, id)`. Without them, these sorts read the full table.
- `space`, `service`, `food`, `price`, and `cleanliness` are required `number` fields from 1 to 10
- the `restaurant` relation on reviews is required and does not cascade on delete
  - the API checks for reviews before it deletes a restaurant and returns `409 Conflict`
- the `creator` relation on reviews is required and cascades on delete
- the `creator` relation on restaurants is optional and does not cascade on delete
  - PocketBase removes the user ID from `creator` in the same transaction as the delete of the user
  - a hook outside that transaction could clear `creator` and then fail to delete the user. A test checks the PocketBase behavior.
- the reviews collection has a unique index on the `restaurant` and `creator` fields
  - two requests at the same time can both pass the duplicate check of the API
  - the API changes the unique index error to `409 Conflict`, not `500 Internal Server Error`
- the [Users API](#users-api) wraps the PocketBase password auth of the `users` collection
  - password auth is on. `email` is the only identity field.
  - the unique index on `email` uses `COLLATE NOCASE`. Email uniqueness and sign in then ignore ASCII case.
  - sign up creates the record and then makes an auth token for it. Decision 00004 has the details.
  - sign in uses the PocketBase password auth. A wrong email or password causes `401 Unauthorized`, not the PocketBase `400`.
  - refresh uses the PocketBase auth refresh
  - `expires` comes from the `exp` claim of the token
  - the API changes PocketBase errors to Problem Details. A PocketBase `validation_not_unique` error on `email` causes `409 Conflict`.
  - `name` is a required text field with a maximum length of 255 characters
  - the auth token duration is the PocketBase default
- the API rules of `users` are empty. Only superusers can use this collection through the PocketBase API.
  - without this, users can skip the Users API rules, such as the required `name`
- the API uses custom auth middleware for the [Authentication](#authentication) rules
  - by default, PocketBase treats a request with an invalid token as anonymous
  - the custom middleware returns `401 Unauthorized` for an invalid token on all requests

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
