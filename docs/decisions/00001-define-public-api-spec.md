---
description: Define OAS specification for the Public API.
status: implemented
---

# Motivation
[motivation]: #motivation

To allow the UI and API to be developed independently, we need to define the OpenAPI Specification (OAS) for the Public API.

# Scope
[scope]: #scope

In scope:
- defining the Public API OAS specification in `specs/public-openapi.yaml`
- APIs for review subjects
- APIs for reviews

Out of scope:
- APIs provided by PocketBase, including the endpoints that issue auth tokens
- Generating client libraries and documentation for the Public API
- Implementation of the API spec
- Rating aggregates on subjects, such as average rating and review count. These can be added later without a breaking change.
- Search, such as matching part of a `title`. Filters compare whole values only.
- Preventing or merging duplicate subjects. Two subjects can have the same `title` and `kind`. PocketBase superusers clean up duplicates.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

New Concepts:
- **Public API**: the public-facing API that allows users to interact with the review system
- **Review**: rating and feedback for a single subject
  - a review must have exactly one subject
  - a user can have at most one review per subject
- **Subject**: a thing that can be reviewed
  - a subject has zero to many reviews

The file `specs/public-openapi.yaml` contains the OpenAPI Specification (OAS) for the Public API. In the OAS file,
the endpoints, request/response schemas, and authentication mechanisms are defined.

All paths in this document are relative to the versioned base path `/v1`.
Paths use the OAS path template syntax, for example `{subjectId}`.

## Subjects API

A subject is a thing that can be reviewed and has zero to many reviews.
This can be anything ranging from books to movies to restaurants.
The subjects API exists on the base path `/subjects`.

### Properties

The properties of a subject are as follows:
- required properties:
    - `title`: string with a length of 1 to 255 characters. It must contain at least one character that is not ASCII whitespace.
    - `kind`: enum of "book", "movie", or "restaurant"
- optional properties:
    - `description`: string with a maximum length of 10,000 characters
    - `tags`: see [Tags](#tags)
- read-only properties maintained by the API:
    - `created`: timestamp
    - `updated`: timestamp
    - `creator`: the user who created the subject, see [Creator](#creator). It is `null` after that user is deleted. See [User Deletion](#user-deletion).
    - `id`: PocketBase record ID

New `kind` values can be added in a later release of `/v1`. Clients must handle unknown `kind` values without failing,
for example by showing them as "other". Adding a `kind` value is not a breaking change.

Example:
```json
{
  "id": "k3x9m2p7q1w8z4a",
  "title": "The Great Gatsby",
  "kind": "book",
  "description": "A novel by F. Scott Fitzgerald",
  "tags": ["classic", "fiction"],
  "creator": {
    "id": "u5n1c8r3t6y0b2e",
    "name": "Ana"
  },
  "created": "2024-11-04 01:26:55.002Z",
  "updated": "2024-11-04 01:26:55.002Z"
}
```

### Operations

The operations of the subjects API are as follows:
- `POST /subjects` to create a single subject
- `GET /subjects` to list subjects
    - results are paginated, see [Pagination](#pagination)
    - results are sortable by the following properties ascending and descending, see [Sorting](#sorting):
      - `title`
      - `created`
      - `updated`
    - the default sort is `-created`
    - results are not filterable by the following properties:
        - `title`
        - `description`
    - results are filterable by the following properties and operators, see [Filtering](#filtering):
        - `kind`: `eq`, `in`
        - `tags`: `has`, `has_any`, `has_all`
        - `created`: `lt`, `lte`, `eq`, `gte`, `gt`
        - `updated`: `lt`, `lte`, `eq`, `gte`, `gt`
        - `creator`: `eq`, `in`
- `GET /subjects/{subjectId}` to retrieve a single subject
- `PATCH /subjects/{subjectId}` to update a single subject
    - `description` and `tags` can always be changed
    - `title` and `kind` can be changed only while the subject has no reviews. Otherwise the API returns `409 Conflict`.
    - a `title` or `kind` value that is equal to the stored value is not a change. It does not cause `409 Conflict`.
    - this prevents one user from changing what the reviews of other users are about
- `DELETE /subjects/{subjectId}` to delete a single subject
    - a subject that has reviews cannot be deleted. The API returns `409 Conflict`.
    - this prevents one user from deleting the reviews of other users

## Reviews API

A review is a rating and associated feedback for a single subject.
The reviews API exists on the base path `/subjects/{subjectId}/reviews`.

### Properties

The properties of a review are as follows:
- required properties:
    - `rating`: integer with a minimum value of 1 and a maximum value of 5
    - `title`: string with a length of 1 to 255 characters. It must contain at least one character that is not ASCII whitespace.
- optional properties:
    - `feedback`: string with a maximum length of 10,000 characters
    - `tags`: see [Tags](#tags)
- read-only properties maintained by the API:
    - `subject`: id of the reviewed subject. The API sets it from the request path.
    - `created`: timestamp
    - `updated`: timestamp
    - `creator`: the user who created the review, see [Creator](#creator)
    - `id`: PocketBase record ID

Example:
```json
{
  "id": "r7d4f0h2j9l5s3v",
  "subject": "k3x9m2p7q1w8z4a",
  "rating": 5,
  "title": "Joe's Great Gatsby Review",
  "feedback": "Great novel",
  "tags": ["favorites"],
  "creator": {
    "id": "u2g6k0m4o8q1s5w",
    "name": "Joe"
  },
  "created": "2024-11-04 01:38:00.431Z",
  "updated": "2024-11-05 09:12:00.000Z"
}
```

### Operations

The operations of the reviews API are as follows:
- `POST /subjects/{subjectId}/reviews` to create a single review for a subject
  - if the user already has a review for the subject, the API returns `409 Conflict`
- `GET /subjects/{subjectId}/reviews` to list reviews for a subject
  - results are paginated, see [Pagination](#pagination)
  - results are sortable by the following properties ascending and descending, see [Sorting](#sorting):
    - `rating`
    - `title`
    - `created`
    - `updated`
  - the default sort is `-created`
  - results are not filterable by the following properties:
    - `title`
    - `feedback`
  - results are filterable by the following properties and operators, see [Filtering](#filtering):
    - `rating`: `lt`, `lte`, `eq`, `gte`, `gt`
    - `tags`: `has`, `has_any`, `has_all`
    - `created`: `lt`, `lte`, `eq`, `gte`, `gt`
    - `updated`: `lt`, `lte`, `eq`, `gte`, `gt`
    - `creator`: `eq`, `in`
- `GET /subjects/{subjectId}/reviews/{reviewId}` to retrieve a single review for a subject
- `PATCH /subjects/{subjectId}/reviews/{reviewId}` to update a single review for a subject
  - `rating`, `title`, `feedback`, and `tags` can be changed
- `DELETE /subjects/{subjectId}/reviews/{reviewId}` to delete a single review for a subject

A review that exists but belongs to a different subject is treated as not found. The API returns `404 Not Found`.

If the subject in the path does not exist, all reviews operations return `404 Not Found`.
This includes the list operation. It does not return an empty list.

# Implementation Details
[implementation-details]: #implementation-details

## Authentication
Create, update, and delete endpoints require authentication using a bearer token in the `Authorization` header.
The token is a PocketBase auth token for a record in the `users` auth collection.
Requests to these endpoints without a valid token get `401 Unauthorized`.

Read endpoints (`GET`) do not require authentication.
Anonymous users can read all subjects and reviews.
A read request with an invalid token gets `401 Unauthorized`.
The API does not treat it as an anonymous request.

## Authorization
- anonymous and authenticated users can read all subjects and reviews
- any authenticated user can create subjects and reviews
- only the creator of a subject or review can update or delete it. Other users get `403 Forbidden`.
- PocketBase superusers manage all records through the PocketBase API, not the Public API

## User Deletion

When a user is deleted:
- the reviews of that user are deleted
- the subjects of that user stay. Their `creator` becomes `null`.
- a subject with a `null` creator can be changed or deleted only by PocketBase superusers

Subjects stay because the reviews of other users can refer to them.

## Common Formats

- Timestamps use the PocketBase format: UTC with millisecond precision and a space between the date and the time, for example `2024-11-04 01:26:55.002Z`.
  - this is not RFC 3339, so the OAS schema uses a `string` with a `pattern` instead of `format: date-time`
  - in query parameters the space must be URL-encoded, for example `2024-11-04%2001:26:55.002Z`
  - the `created` and `updated` properties use the default PocketBase autodate fields with the same names
- IDs use the default PocketBase record ID format: 15 characters that match the pattern `^[a-z0-9]+$`, for example `k3x9m2p7q1w8z4a`.
  - PocketBase generates IDs at random when it creates a record
  - IDs are not ordered by time
- String lengths count Unicode code points, not bytes. This matches JSON Schema `maxLength`.

### Creator

The `creator` property is an object with the public fields of a user:
- `id`: the PocketBase record ID of the user
- `name`: the `name` field of the user in the PocketBase `users` collection. It can be an empty string.

The API does not return other user fields, such as `email`.
The `creator` filter compares the user `id`, for example `creator[eq]=u2g6k0m4o8q1s5w`.
The `creator` filter cannot select subjects with a `null` creator.
PocketBase superusers find these subjects through the PocketBase API.

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

## Update operations

PATCH requests use JSON Merge Patch (RFC 7396) with the content type `application/merge-patch+json`.
Other content types cause `415 Unsupported Media Type`.
- properties not in the request body stay unchanged
- `null` clears an optional property
- `null` on a required property causes `422 Unprocessable Content`
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
| `eq` | equal to | `kind[eq]=book` |
| `lt` | less than | `rating[lt]=3` |
| `lte` | less than or equal to | `created[lte]=2024-11-04%2000:00:00.000Z` |
| `gt` | greater than | `rating[gt]=3` |
| `gte` | greater than or equal to | `updated[gte]=2024-11-04%2000:00:00.000Z` |
| `in` | equal to any of the values | `kind[in]=book&kind[in]=movie` |
| `has` | the array contains the value | `tags[has]=classic` |
| `has_any` | the array contains at least one of the values | `tags[has_any]=classic&tags[has_any]=fiction` |
| `has_all` | the array contains all of the values | `tags[has_all]=classic&tags[has_all]=fiction` |

Operators that take more than one value use a repeated query parameter.
An unknown property, an unknown operator, or an operator that is not allowed for a property causes `400 Bad Request`.

The tag operators compare whole tags.
For example, `tags[has]=classic` does not match the tag `neo-classic`.

### Sorting

The `sort` query parameter is a comma-separated list of properties.
A `-` prefix sorts a property in descending order.
For example, `sort=-rating,title` sorts by rating from high to low, then by title from A to Z.

The `title` sort ignores case for the ASCII letters `A` to `Z`.
For example, "apple" comes before "Zebra".
The sort does not ignore case for other letters, and it does not use language-specific rules.
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

Error responses use RFC 9457 Problem Details with the content type `application/problem+json`.
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
    { "property": "rating", "message": "must be between 1 and 5" }
  ]
}
```

| Status | Use |
| --- | --- |
| `400 Bad Request` | malformed request, invalid query parameters, unknown or read-only properties in the body |
| `401 Unauthorized` | missing bearer token on a create, update, or delete request, or an invalid bearer token on any request |
| `403 Forbidden` | the user is not the creator of the resource |
| `404 Not Found` | the resource or the subject in the path does not exist, or the review belongs to a different subject |
| `409 Conflict` | duplicate review for a subject, delete of a subject that has reviews, or change of `title` or `kind` on a subject that has reviews |
| `415 Unsupported Media Type` | wrong request content type |
| `422 Unprocessable Content` | the request body has invalid property values |

## PocketBase Mapping

The spec uses the default PocketBase `id`, `created`, and `updated` fields without changes.
The `creator` and `subject` properties are PocketBase relation fields with the same names.
The API expands the `creator` relation and returns only the user `id` and `name`.

The PocketBase collections need this additional setup:
- the API rules of `subjects` and `reviews` are empty. Only superusers can use these collections through the PocketBase API.
  - all user access goes through the Public API
  - without this, users can skip the Public API rules, such as creator-only changes and the `409 Conflict` checks
- `tags` is a `json` field. PocketBase has no field type for a list of free-form strings.
  - the API validates the [Tags](#tags) rules
  - tag filters use SQLite `json_each` to compare whole items. The PocketBase `~` operator matches substrings of the JSON text, so tag filters do not use it.
- the list operations order by `title COLLATE NOCASE` for the `title` sort
  - the sort direction and the `id` tie-breaker come from the list query
  - SQLite `NOCASE` ignores case only for ASCII letters
  - each collection has an index on `(title COLLATE NOCASE, id)`. Without it, the `title` sort reads the full table.
- the `subject` relation on reviews is required and does not cascade on delete
  - the API checks for reviews before it deletes a subject and returns `409 Conflict`
- the `creator` relation on reviews is required and cascades on delete
- the `creator` relation on subjects is optional and does not cascade on delete
  - an `OnRecordDelete` hook on the `users` collection sets `creator` to `null` on the subjects of the user before the delete
  - the hook makes this behavior explicit. It does not depend on how PocketBase handles references to a deleted record.
- the reviews collection has a unique index on the `subject` and `creator` fields
  - two requests at the same time can both pass the duplicate check of the API
  - the API changes the unique index error to `409 Conflict`, not `500 Internal Server Error`
- the API uses custom auth middleware for the [Authentication](#authentication) rules
  - by default, PocketBase treats a request with an invalid token as anonymous
  - the custom middleware returns `401 Unauthorized` for an invalid token on all requests

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
