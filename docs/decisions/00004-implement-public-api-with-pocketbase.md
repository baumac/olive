---
description: Implement the Public API in the app with PocketBase and the Go SDK.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00001 defines the Public API and its PocketBase mapping.
Decision 00003 generates a Go server interface from the spec.
The PocketBase app in `app` must implement the spec on PocketBase.
Some choices of the implementation are not in these decisions.
This decision records them.

Expected outcome:
- The PocketBase app in `app` serves the Public API under `/v1`, the PocketBase API under `/api`, and the PocketBase dashboard under `/_/`.
- The Public API implements `reviewsapi.StrictServerInterface` from `app/sdk/go`.
- Migrations make the `restaurants` and `reviews` collections and change `users`.
- Tests send requests through the PocketBase router to a new database.

# Scope
[scope]: #scope

In scope:
- The layout of the PocketBase app in `app`
- How the Public API mounts on the PocketBase router
- The order of the request checks
- The rate limit, the cursor format, and the PocketBase details that decision 00001 does not give

Out of scope:
- Deployment, backups, and email settings of PocketBase
- A rate limit that is shared by more than one server
- The `ui` app

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

```bash
make run-app    # Serve on 127.0.0.1:8090 with data in app/pb_data
make test-app   # Run the tests of the app
make build-app  # Build ./bin/reviews-app
```

The server applies the migrations when it starts.

```
app/
  go.mod                       # module github.com/baumac/reviews-app/app
  main.go                      # the PocketBase app
  migrations/                  # Go migrations of the collections
  internal/publicapi/
    server.go                  # Register, the route table, auth, and request checks
    body.go                    # the body schemas
    validate.go                # the value rules
    users.go                   # signUp, signIn, refreshToken
    restaurants.go             # the restaurant operations
    reviews.go                 # the review operations
    list.go                    # filters, sorts, and cursors
    ratelimit.go               # the sign up and sign in rate limit
    publicapi_test.go          # tests through the PocketBase router
```

`publicapi.Register(app, publicapi.DefaultOptions)` adds the Public API to a PocketBase app.

# Implementation Details
[implementation-details]: #implementation-details

## Mounting

- an `OnServe` hook adds the route `/v1/{path...}` to the PocketBase router
- the route calls one `http.Handler`. It has the handler that `reviewsapi.HandlerWithOptions` makes on its own `http.ServeMux`.
- the route puts the client IP from `RequestEvent.RealIP` in the request context. `RealIP` uses the trusted proxy settings of PocketBase.
- the default PocketBase middlewares, such as the body limit and the panic recovery, also apply to `/v1`
- a path under `/v1` that the spec does not define gets `404 Not Found`. A known path with a different method gets `405 Method Not Allowed` with an `Allow` header.

## Request checks

A `prepare` middleware runs before the generated strict handler. A route table gives the checks of each operation.
An operation without a route table entry gets `500`. A test checks that each generated operation has an entry.
The checks run in this order:

| Order | Check | Status |
| --- | --- | --- |
| 1 | the rate limit of sign up and sign in | `429` |
| 2 | the bearer token. Sign up and sign in ignore it. | `401` |
| 3 | the names of the query parameters, and the number of values of each parameter | `400` |
| 4 | the content type of the body | `415` |
| 5 | the body is one JSON object without unknown, read-only, or repeated properties | `400` |
| 6 | the JSON types, required properties, `null` values, and value rules of the body properties | `422` |
| 7 | the IDs in the path, and the values of the query parameters | `400` |
| 8 | the restaurant and the review exist | `404` |
| 9 | the user is the creator | `403` |
| 10 | the review kind and the properties of other kinds | `422`, `400` |
| 11 | duplicate reviews, and restaurants with reviews | `409` |

- checks 1 to 6 are in `prepare`. Checks 7 to 11 are in the operation handlers.
- check 6 gives all invalid properties in one response
- the review body schemas have the properties of all kinds. The handler then rejects the scores of other kinds with `400`.
- a JSON number for an integer property must be an integer literal of at most 9 digits. `5.0` gets `422`. The value then fits a Go `int` on all platforms.
- a body larger than 256 KB gets `413`. The largest valid body is about 100 KB.
- more than 20 tags give one `tags` error, without an error for each item. The size of the response then does not grow with the request.
- the strict handler decodes the body after `prepare`. Its decode errors therefore do not occur for client input.
- the decoders after `prepare` read the first copy of a repeated property. `prepare` therefore rejects repeated properties.
- an email must be a bare address. `mail.ParseAddress` also accepts `Ana <ana@example.com>`, so `prepare` rejects a display name.
- a parameter that is not an array has one value. `tags[has_any]` and `tags[has_all]` have at most 20 values. `creator[in]` and `kind[in]` have at most 100 values. Each value is one SQL parameter.

## Authentication

- a router middleware returns `404` for the PocketBase account routes of `users`: password reset, verification, email change, and one-time passwords. Decision 00001 does not have these flows. The middleware runs before PocketBase looks up a record, so the response is the same for all email addresses.

- the `loadAuthToken` middleware of PocketBase checks the token with `FindAuthRecordByToken`. The route passes its result in the request context. The record must be in the `users` collection.
- a superuser token is therefore not valid for the Public API
- `refreshToken` makes a new token only for a token with the `refreshable` claim. A superuser can make a token without it to act as a user.
- the `AuthRule` of `users` is `null`. This stops the PocketBase auth endpoints of `users`. All sign ins go through the Public API and its rate limit.
- sign up uses the PocketBase record API in Go. It does not send `passwordConfirm`.
- sign in compares the password with a dummy bcrypt hash when no user has the email address. The hash uses the cost of the `password` field.

## Rate limit

- sign up and sign in each allow 10 requests per IP address per minute
- the count uses fixed windows in the memory of the server
- at 10,000 windows, the limiter removes the expired windows. It does this at most one time per window.
- `Retry-After` is the number of seconds until the window ends, at least 1
- a restart clears the counts. Each server has its own counts.
- the IP address comes from `RequestEvent.RealIP`. Behind a reverse proxy, the PocketBase setting `trustedProxy.headers` must name the header that the proxy sets, such as `X-Real-IP`. Without it, `RealIP` is the address of the proxy for all clients, and the limit counts all clients together.
- a second migration turns on the PocketBase rate limits. Their default rules limit the auth routes of `/api`, which include the superuser sign in. They do not apply to `/v1`.

## Transactions

Update and delete of a restaurant, and create, update, and delete of a review run in one transaction.
The transaction reads the records again, checks the rules, and saves.
PocketBase uses one SQLite writer, so a parallel request cannot break the `409` checks.
The unique index on `restaurant` and `creator` of `reviews` also changes a duplicate to `409`.

## Lists

- pagination uses a keyset. The cursor is base64url JSON with the sort values of the last item and a fingerprint.
- the fingerprint is a hash of the filters, the sort, and the restaurant ID. `limit` is not in it, so a client can change `limit` between pages.
- a sort on a score has two keys. The first key is `0` for reviews of the kind of the score and `1` for other reviews. This puts the other kinds last in both directions.
- the `name` and `title` sorts use `COLLATE NOCASE`
- the `id` key has the direction of the key before it. When all keys have one direction, the cursor condition is a row value comparison, such as `(created, id) < (x, y)`. SQLite can then use the index. Mixed directions use an `OR` condition.
- a cursor value must be a string or a number. Other values get `400`.
- filters on a score also compare `kind`
- tag filters use `json_each`. A value in `tags` that is not valid JSON matches no tag filter.

## PocketBase mapping

- PocketBase stores an empty string for a text field that has no value. An empty `description` or `feedback` is therefore not in responses.
- the score fields have the minimum 1 and the maximum of their kind. PocketBase does not check these limits for the stored `0`.
- an update saves only when a value changes. A merge patch without changes keeps `updated`.
- the unique `email` index of `users` uses `COLLATE NOCASE`. The down migration restores the PocketBase defaults of `users`.
- PocketBase removes the user ID from `restaurants.creator` in the delete transaction of the user. The API has no hook for it.
- `restaurants` has indexes on `creator` and on `(created, id)`. `reviews` has indexes on `creator` and on `(restaurant, created, id)`.

## Go modules

- PocketBase v0.40.6 needs Go 1.27. The `app` module and `go.work` therefore use Go 1.27.
- `app/go.mod` has a `replace` line for `app/sdk/go`. `go mod tidy` does not use `go.work`, and the SDK module is not published.

## Tests

- the tests make a PocketBase test app with an empty data directory. It applies the system migrations and the migrations of this project.
- the tests send requests through `apis.NewRouter`. They test the Public API with the PocketBase middlewares.
- the tests use a high rate limit. One test uses a low limit.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- The rate limit is in memory. More than one server needs a shared store or a limit at the proxy.
