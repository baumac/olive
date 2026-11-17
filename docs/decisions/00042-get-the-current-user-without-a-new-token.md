---
description: GET /users/me returns the signed in user. The UI uses it at start and refreshes the token only after half of its time.
status: implemented
---

# Motivation
[motivation]: #motivation

The API had no operation that only returns the signed in user.
The UI called `refreshToken` at each start to check the stored session and to get the current user.
Each page load then made a new token and a new `reviews_session` cookie.
The "Check again" button of the approval banner and the route guard of decision 00034 also made a new token.

A new token on each load has these problems:
- each tab of the UI gets a new token on each load. Many valid tokens of one user then exist at the same time.
- a client cannot check a token without a change of its token
- the end-to-end tests check a token with a refresh

Expected outcome:
- a client gets the user of a token without a new token
- the UI makes a new token only after half of the token time or when the token must move into the cookie
- a person who visits the UI regularly stays signed in, as before

# Scope
[scope]: #scope

In scope:
- the `getCurrentUser` operation and version 5.1.0 of the spec
- the session check of the UI at start, in the approval banner, and in the route guard
- the tests

Out of scope:
- a change of `refreshToken`. Other clients keep it.
- a change of the token duration
- a sign out of one device. Decision 00015 has the reason.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Get the signed in user

```
GET /v1/users/me
Authorization: Bearer eyJhbGciOi...
```

The response is `200 OK` with the `User`:

```json
{
  "id": "u5n1c8r3t6y0b2e",
  "email": "ana@example.com",
  "name": "Ana",
  "approved": false,
  "created": "2024-11-01T08:12:40.517Z",
  "updated": "2024-11-01T08:12:40.517Z"
}
```

- the `reviews_session` cookie works in the place of the bearer token
- the response has no token and no `Set-Cookie` header
- a user who is not approved also gets the user. The operation has `x-allow-unapproved`. See decision 00034.
- a missing or invalid token gets `401`. An invalid cookie gets `401` and the response deletes the cookie. See decision 00027.
- the operation has no rate limit. It changes no data.

## The UI

`checkSession` replaces `refreshSession`. It sends one of two requests:

| Stored session | Request |
|---|---|
| a `token` from an older version | `refreshToken` with the token and `cookie=true` |
| no `issued` value | `refreshToken` with `cookie=true` |
| half of the time from `issued` to `expires` is over | `refreshToken` with `cookie=true` |
| all other sessions | `getCurrentUser` |

`issued` is the browser time when the UI got the token. `saveSession` stores it with `expires`.
A session from before this decision has no `issued` value. Its first check refreshes the token.

After `getCurrentUser`, the UI keeps the stored `expires` and `issued`.
It writes the user to `localStorage` only when the user changed.
Each write sends a storage event to the other tabs.

Each hour, the UI refreshes the token when half of its time is over.
The UI before this decision refreshed the token at each load and when less than one day was left.

## The session length

The UI before this decision made a new token at each load.
A person who visited the UI more often than the token duration stayed signed in.

With a token duration of 7 days, the UI now refreshes the token on the first visit after 3.5 days.
A person who visits at least one time in each 3.5 days stays signed in.
A person who visits less often can find an expired session. This was also true before.

# Implementation Details
[implementation-details]: #implementation-details

- `GetCurrentUser` in `app/api/internal/publicapi/users.go` returns the user of the request state
- `authenticate` and `prepare` check the token as for each other operation. A wrong token gets the 401 of decision 00027.
- the cross-origin check of decision 00027 lets each `GET` pass. The CORS settings do not allow credentials. A cross-origin page cannot read the email address.
- `checkSession` in `app/ui/src/composables/useAuth.ts` shares one request between calls at the same time, as `refreshSession` did
- the `tokenStatus` helper of the end-to-end tests uses `getCurrentUser`. It replaces `refreshStatus`.

A tab can show an old `expires` after another tab refreshed the token.
The storage event of decision 00015 gives the new `expires` to all tabs.
A tab that missed the event refreshes the token one time too early. This does no harm.

`issued` uses the clock of the browser. `expires` uses the clock of the server.
A browser clock that is ahead makes the refresh earlier. A clock that is ahead by days can refresh at each load.
A browser clock that is behind makes the refresh later.
A clock that is behind by more than half of the token duration lets the token expire. The person then signs in again.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
