---
description: Each write operation has a rate limit per user. Creates have a low limit. Updates, deletes, and list changes have a high limit.
status: implemented
---

# Motivation
[motivation]: #motivation

Only sign up, sign in, place search, and photo upload had a rate limit.
Sign up has no email verification. One script can make 10 accounts per minute per IP address.
Decision 00034 stops a new account until a superuser approves it.
An approved account could still create restaurants, reviews, and lists without a limit.
A stolen token or a bad script of an approved user could then fill the site.

Expected outcome:
- one user cannot create many restaurants, reviews, or lists in a short time
- one user cannot send many updates or deletes in a short time
- a person who uses the UI in the usual way never gets the limit
- the operator can change the limits without a new build

# Scope
[scope]: #scope

In scope:
- the `x-rate-limit` extension and the `429` response of each write operation without a limit
- the `create` and `edit` limits of the Public API and their environment variables
- the text of the UI for a `429` of a write

Out of scope:
- email verification. The backlog has it.
- a limit for all write operations of a user together. Each operation keeps its own count, the same as the other limits.
- a limit that stays after a restart. The counts are in the memory of the process. See decision 00022.
- the read operations.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

| Limit name | Operations | Default per user and hour |
|---|---|---|
| `create` | `createRestaurant`, `createReview`, `createList` | 30 for each operation |
| `edit` | `updateRestaurant`, `deleteRestaurant`, `updateReview`, `deleteReview`, `updateReviewPhoto`, `deleteReviewPhoto`, `updateList`, `deleteList`, `addListRestaurant`, `removeListRestaurant` | 300 for each operation |

The 31st new restaurant of one user in one hour gets this response:

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 2400
Content-Type: application/problem+json

{"type":"about:blank","title":"Too Many Requests","status":429,"detail":"Too many requests. Try again later."}
```

The same user can still create a review or a list. Each operation has its own count.
Another user has its own count.

Environment variables:
- `REVIEWS_CREATE_LIMIT` sets the `create` limit per hour.
- `REVIEWS_EDIT_LIMIT` sets the `edit` limit per hour.

The UI shows "You made many changes in a short time. Try again later." for a `429` of a write.

# Implementation Details
[implementation-details]: #implementation-details

- The spec has `x-rate-limit` with `key: user` and the name `create` or `edit` on each operation of the table.
- The `TooManyRequests` response tells that each operation has its own count.
- `loadSpec` refuses an operation with `x-rate-limit` and without a `429` response.
- The `user` key counts after the token check and the approval check. A user without approval gets `403` and uses no count.
- A request with a `4xx` or `5xx` response gives its count back. A failed write changes no data.
- The UI keeps the draft of a form after a `429`.
- The limits use the fixed windows of `rateLimiter` in `app/internal/publicapi/ratelimit.go`.
- `Options.CreateLimit`, `Options.EditLimit`, and `Options.WriteWindow` set the limits. Zero uses the default.
- The end-to-end tests set both limits to 10000. The rate limit project tests the `create` limit with its own app.

Why 30 and 300:
- 30 new restaurants in one hour is more than a person adds after a long trip.
- One review per restaurant and user already limits the reviews.
- A person can add and remove many favorites in a short time. 300 changes of one operation is more than this.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- A limit per day can be necessary if approved users misuse the hourly limit.
