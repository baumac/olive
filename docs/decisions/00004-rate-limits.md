---
description: Each write operation and each auth operation of the Public API has a rate limit. The spec names the limit of each operation. The limits count in the memory of the process, and environment variables change them.
status: implemented
---

# Motivation
[motivation]: #motivation

The app is on the public internet. See decision 00000.
The approval of decision 00003 stops a new account until a superuser approves it.
An approved account can still have a stolen token or a bad script.
Without limits, one client could:
- guess the passwords of a user from many IP addresses
- check many email addresses at sign up
- fill the site with restaurants, recipes, reviews, photos, or lists
- send many searches to the public geocoder, which then blocks the app
- send many emails to one address

Expected outcome:
- each write operation and each auth operation has a limit
- the spec tells the limit of each operation, so a new operation cannot skip it
- a person who uses the UI in the usual way never gets a limit
- the operator changes the limits without a new build
- the limits do not show which email addresses have an account

# Scope
[scope]: #scope

In scope:
- the `x-rate-limit` extension and the `429` response
- the limits of the Public API and their environment variables
- the limit of wrong passwords and the limit of emails
- the limiter, its windows, and its caps
- the PocketBase rate limit rules
- the text of the UI for a `429`

Out of scope:
- the read operations. They have no limit.
- `signOut`, `refreshToken`, and `getCurrentUser`. A user can then always sign out, and the UI checks the session on each page load.
- a limit that stays after a restart, and a limit that more than one server shares. The app runs as one instance. See decision 00013.
- a limit for all write operations of a user together. Each operation has its own count.
- a limit on the total photo storage of a user. The upload limit makes its growth slow.
- large bulk imports. A superuser imports old data in the dashboard or with the PocketBase API.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Limits of the Public API

| Name | Key | Operations | Default |
|---|---|---|---|
| `sign-in` | IP address | `signUp`, `confirmEmail`, `signIn`, `requestPasswordReset`, `resetPassword` | 10 per minute |
| `create` | user | `createRestaurant`, `createRecipe`, `createRestaurantReview`, `createRecipeReview`, `createList` | 30 per hour |
| `edit` | user | the updates and the deletes of restaurants, recipes, reviews, photos, lists, and the user, and the list changes | 300 per hour |
| `photo-upload` | user | `addRestaurantReviewPhoto`, `addRecipeReviewPhoto` | 30 per hour |
| `place-search` | user | `searchPlaces` | 30 per minute |

- each operation has its own count. The 31st new restaurant of a user in one hour gets `429`, but the user can still create a review.
- each user has its own count
- `create` and `edit` give back the count of a request with a `4xx` or `5xx` response. A failed write changes no data.

Other limits:

| Limit | Key | Default |
|---|---|---|
| wrong passwords of sign in, `changePassword`, and `deleteUser` | email address | 10 in 15 minutes |
| wrong passwords of `confirmEmail` | user | 10 in 15 minutes |
| emails of each kind | email address | 5 per hour |
| requests of all users to the geocoder | the app | 60 per minute. See decision 00010. |

A request over a limit gets this response:

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 2400
Content-Type: application/problem+json

{"type":"about:blank","title":"Too Many Requests","status":429,"detail":"Too many requests. Try again later."}
```

The wrong password limit adds the `code` `wrong_password_limit`.
At the email limit, the request still gets its usual response, but the app sends no email. See decision 00003.

Why these values:
- 30 new restaurants in one hour is more than a person adds after a long trip
- a person can add and remove many favorites in a short time. 300 changes of one operation is more than this.
- 5 photos for each review and 30 uploads per hour are the values of the first release. Real use can change them.

## Environment variables

| Variable | Limit |
|---|---|
| `REVIEWS_SIGN_IN_LIMIT` | `sign-in` per minute. The end-to-end tests raise it, because their parallel sign ups come from one IP address. |
| `REVIEWS_SIGN_IN_FAILURE_LIMIT` | wrong passwords per 15 minutes |
| `REVIEWS_CREATE_LIMIT` | `create` per hour |
| `REVIEWS_EDIT_LIMIT` | `edit` per hour |
| `REVIEWS_PHOTO_UPLOAD_LIMIT` | `photo-upload` per hour |
| `REVIEWS_PLACE_SEARCH_LIMIT` | `place-search` per minute |
| `REVIEWS_EMAIL_LIMIT` | emails of each kind per address and hour |

## The UI

- a `429` of a write shows "You made many changes in a short time. Try again later."
- the form keeps its draft after a `429`

# Implementation Details
[implementation-details]: #implementation-details

## The spec

- `x-rate-limit` has a `key` and a `name`. The key is `ip` or `user`. The name selects the settings: `sign-in`, `create`, `edit`, `photo-upload`, or `place-search`.
- `loadSpec` refuses an operation with `x-rate-limit` and without a `429` response
- each write operation must have `x-rate-limit`. `signOut` and `refreshToken` are the exceptions.
- the `TooManyRequests` response tells that each operation has its own count
- the wrong password limit and the email limit are not in the extension. Their key comes from the body, and they count only some requests.

## Order of the checks

- the `ip` key counts before the token check. Sign up and sign in have no user.
- the `user` key counts after the token check and the approval check. A request without a token gets `401`, and a user without approval gets `403`. Neither uses a count.
- sign in counts the wrong password limit before it looks for the user. An address at the limit gets `429` without a lookup and without a bcrypt compare.
- a sign in with the correct password does not count. A correct password over the limit also gets `429`. Otherwise the limit would not stop a guess.

## The limiter

- `rateLimiter` in `app/api/internal/publicapi/ratelimit.go` counts in fixed windows in the memory of the process
- each request removes the expired windows first
- a limiter keeps at most 10,000 windows. At the cap, a new key removes the oldest window.
- `Retry-After` is the number of seconds until the window ends, at least 1
- an IPv6 address counts as its /64 network. One host often gets a full /64 network.
- the IP address comes from `RequestEvent.RealIP`. Behind a reverse proxy, the trusted proxy header must be set. See decision 00013.
- `Options` of `publicapi` has the limits and the windows. Zero uses the default.

## The wrong password limit

- `signInFailures` counts with the key of `passwordLimitKey`, the email address in lower case
- the count is the same for an address with a user and for an address without one. The limit then does not show which addresses have a user.
- its cap is 100,000 windows. Clients without an account make its keys, so a limiter without a cap could use all memory.
  `Credentials.email` has at most 255 characters, so the limiter uses less than 40 MB.
- at the cap, a new key removes the oldest window, the same for all addresses. A client needs 100,000 requests with new addresses to get 10 more tries for one user.
- `confirmEmail` counts with the key `confirm:` and the user ID. No email address is equal to this key.
- the PocketBase rate limit rules cannot do this. They count only per IP address.

## PocketBase rules

- a migration turns on the rate limits of PocketBase. Their default rules limit the auth routes under `/api`, which include the superuser sign in.
- a migration adds the rule `POST /csp-report` with 20 requests per minute. See decision 00002.
- these rules do not apply to `/v1`

## Tests

- the end-to-end tests set the `create` and `edit` limits to 10000
- the rate limit project of the end-to-end tests checks the `create` limit with its own app. See decision 00014.
- the Go tests use high limits. Some tests use a low limit.

Accepted risks:
- the counts are in memory. A restart clears them. An attacker then gets at most one more window.
- a person who knows an address can block its sign in. Decision 00003 has this risk.

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| A limit only on sign up, sign in, uploads, and the place search | Removed. An approved account with a stolen token could fill the site. |
| A wrong password limit with the user ID as the key | Removed. An address without a user got `401` where an address with a user got `429`. |
| A wrong password limiter without a cap | Removed. Clients without an account make its keys, so it could use all memory. |
| A count in the database | Not selected. The app runs as one instance, so a stored count gives little. |
| A table of operations and limits in Go | Not selected. The spec already names each operation. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
