---
description: A new user cannot add or change data before a superuser approves the account in the PocketBase dashboard. The user can sign in and read.
status: implemented
---

# Motivation
[motivation]: #motivation

Each person with an email address could sign up and write at once.
Sign up has no email verification, and most write operations have no rate limit.
One script could make accounts and fill the site with restaurants, reviews, photos, and lists.

Expected outcome:
- a new account can do nothing that changes data until a superuser approves it
- a superuser approves an account in the dashboard, the only admin surface
- a superuser can take the approval back
- the person knows why the account cannot write yet

# Scope
[scope]: #scope

In scope:
- the `approved` field of the `users` collection and its migration
- the approval check of the Public API
- the `approved` property of `User` and the `403` response of the write operations
- the banner and the guards of the UI
- the end-to-end tests. They approve their users as a superuser.

Out of scope:
- a notice to the superusers about a new account. A superuser must look at the dashboard.
- a notice to the user about the approval. The UI finds it at the next token refresh.
- email verification. It is a different check. The backlog has it.
- an approval page in the Public UI. The Public API has no admin operations.
- a setting that turns the approval off.
- the read operations. A visitor without an account can also read.
- a limit of the number of users that wait. The sign up limit of decision 00004 and the cleanup job limit them.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Rule

A user is approved or not approved. A new user is not approved.

| A user who is not approved | Result |
|---|---|
| signs up, signs in, signs out, refreshes the token | works |
| reads restaurants, reviews, photos, reviewers, and lists | works |
| creates, changes, or deletes a restaurant, a review, a photo, or a list | `403 Forbidden` |
| adds a favorite | `403 Forbidden` |
| searches a place for a new restaurant | `403 Forbidden` |

The rule in the API: each operation that needs a token also needs an approved user. `signOut` and `refreshToken` are the exceptions.

A user who is not approved has no public profile. `getReviewer`, `listReviewerReviews`, `listReviewerLists`, `getList`, and `listListRestaurants` give `404` to other people.
The name of a new account is then not public before the approval. The user gets the own profile and lists with the own token.

```http
POST /v1/lists
Authorization: Bearer <token of a new user>

HTTP/1.1 403 Forbidden
{ "type": "about:blank", "title": "Forbidden", "status": 403,
  "detail": "The account is not approved. A superuser must approve it.",
  "code": "not_approved" }
```

`code` lets a client tell this `403` from the `403` of a resource of another user.

The `user` of each auth token response has `approved`:

```json
{ "id": "u5n1c8r3t6y0b2e", "email": "ana@example.com", "name": "Ana", "approved": false }
```

## Superuser

A superuser opens the `users` collection in the dashboard at `/_/`.
The search `approved = false` lists the users that wait.
The superuser opens a user, turns on `approved`, and saves. The token of the user then works for all operations. The user needs no new sign in.
The superuser can turn `approved` off again. The next write of the user gets `403`.
The reviews and restaurants of this user stay public. The reviewer page and the lists give `404` to other people.

A daily job deletes a user that is not approved after 30 days. A user with a review, a restaurant, or a restaurant in a list stays.

The users from before this decision are approved. The migration sets the value.

## UI

- A user who is not approved sees a banner below the header on each page: "Your account waits for approval".
- "Check again" in the banner gets the user again. After the approval, the banner goes away.
- The app also gets the user again at each start. A reload after the approval is sufficient.
- A link to a form that adds or changes data stays on the page and shows a toast with the reason.
- The heart and "Save to list" show the same toast and send no request.

# Implementation Details
[implementation-details]: #implementation-details

## Data

The migration `1794268814_add_user_approval.go` adds the bool field `approved` to `users`.
It sets `approved` for all users that exist. A deploy then does not lock them out.
`idx_users_approved` on `(approved, created)` serves the dashboard search for the users that wait.

The `users` collection has no API rules. See decision 00004. A user cannot change the own record through the PocketBase API.
Only a superuser and SQL can set `approved`.

## Server

- `loadSpec` sets `approval` on each operation with a required token. The spec extension `x-allow-unapproved: true` is the exception. `signOut` and `refreshToken` have it.
  A new operation of the `users` tag, such as a name change, then needs approval without a change of the code.
  It fails at start if such an operation has no `403` response. A new write operation then cannot skip the check or break the response validation.
- `serveOperation` checks `approved` of the user after the token check and before the user rate limit.
  A request without a token gets `401` first. A refused request does not count for the limit.
- The check reads the user record of the request. PocketBase loads it for the token check. An approval or its removal then works at the next request.
- `signUp` does not set the field. The default of a bool field is `false`.
- The favorites list hook of decision 00033 still runs for a new user. The list is empty until the approval.
- `createRestaurant`, `createReview`, `createList`, and `searchPlaces` get a `403` response in the spec. The other write operations had it.

- `findReviewer` returns no user for a user without approval, except for the user of the token. The reviewer and list read operations use it.
  `listReviewerReviews` now looks up the user for each request. Before, a page with a review was the proof of the user.
- The `webui` package gives `404` and no link preview for the reviewer page and the list pages of a user without approval. The sitemap has no list of such a user.
- `app/internal/usercleanup` is the daily job `user-cleanup`. It deletes the users that are not approved, are older than 30 days, and have no data.
  A user with data lost the approval later. The job does not delete data that people see.

`searchPlaces` needs approval. It only serves the restaurant form, and each search can cause a request to the geocoder.

## UI

- `pendingApproval` in `session.ts` is true only for a stored user with `approved` `false`.
  A session from before this decision has no value. Its user was approved by the migration.
- `ApprovalBanner` is in `App.vue` below the header. "Check again" calls `refreshSession`.
  `refreshSession` tells how the refresh ended. A failed request shows "The check did not work". An ended session shows no approval text.
- The form routes have `requiresApproval`. The router guard cancels the navigation for a user who waits.
  On a first load, the guard gets the user again before it decides. A user who was approved a moment ago then opens the form. A user who still waits opens `/restaurants`.
- The API client stores `approved` as `false` after a `403` with the code `not_approved`. The banner then tells the reason of the refused write.
- `tellPendingApproval` shows the toast. `FavoriteButton` and `SaveToListMenu` call it before a request.
- The UI checks are for the person. The API check is the security measure.

## Tests

- The Go API tests approve their users with SQL in `signUp`. `signUpPending` makes a user that waits.
- Each end-to-end app has a superuser. `startApp` signs it in and gives the token.
  `api.signUp` approves the new user through the PocketBase API, the same way as the dashboard.
  `global-setup.ts` puts the token into the environment of the test workers.

## Alternatives

- **The `verified` field of PocketBase.** It means that the user confirmed the email address. Email verification on the backlog needs it for that.
- **Block the sign in of a user who is not approved.** The person then cannot see the state of the account, and the UI cannot tell the reason.
- **An environment variable that turns the approval off.** A wrong value would remove the security measure. The tests approve their users as a superuser does.

## Corner cases

- A user is approved while a form is open in another tab. The stored user changes at the next refresh or "Check again".
- A superuser takes the approval back while the user has a form open. The save gets `403` with the code `not_approved`. The form shows its text for a missing permission, and the banner shows at once.
- A review card links to the reviewer page of a user who lost the approval. The page shows "We cannot find this page".
- A person waits for more than 30 days. The job deletes the account. The person can sign up again with the same email address.
- A superuser makes a user in the dashboard. The user is approved only when the superuser turns on `approved`.
- The data of a user stays when the approval is taken back. The reviews and lists are still public.

# Unresolved questions
[unresolved-questions]: #unresolved-questions