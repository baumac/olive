---
description: Users sign up with an email confirmation, sign in with a password, and wait for the approval of a superuser before they can write. The UI keeps the auth token in an HttpOnly cookie, and sign out ends all sessions. Each write operation and each auth operation has a rate limit that the spec names.
status: implemented
---

# Motivation
[motivation]: #motivation

The users are friends, family, and fellow foodies. See decision 00000.
Each of them makes an account to write reviews. Visitors without an account can read.

The app is on the public internet. Each person with an email address can sign up.
These risks shape the accounts:
- a script could make accounts and fill the site with data. A superuser therefore approves each new user before the user can write.
- a person could sign up with the address of another person. The owner of the address therefore confirms it with a link.
- sign up could show which addresses have an account. Sign up therefore gives the same response for all addresses.
- a script on the page could take the auth token. The UI therefore keeps it in an HttpOnly cookie.
- a token could leak. Sign out therefore ends all tokens of the user on the server.

The approval stops a new account until a superuser approves it.
An approved account can still have a stolen token or a bad script.
Without limits, one client could:
- guess the passwords of a user from many IP addresses
- check many email addresses at sign up
- fill the site with restaurants, recipes, reviews, photos, or lists
- send many searches to the public geocoder, which then blocks the app
- send many emails to one address

Expected outcome:
- a person signs up, confirms the address with a link, and signs in
- a new user can read, but cannot write until a superuser approves the account
- the person knows why the account cannot write yet
- no script on the page can read the auth token of the UI
- sign out makes all tokens of the user invalid. All tabs of one browser show the same session.
- a person who forgot the password resets it by email
- a user changes the name and the password, and deletes the account
- each write operation and each auth operation has a limit
- the spec tells the limit of each operation, so a new operation cannot skip it
- a person who uses the UI in the usual way never gets a limit
- the operator changes the limits without a new build
- the limits do not show which email addresses have an account

# Scope
[scope]: #scope

In scope:
- the operations of the `users` tag
- the auth tokens, the `reviews_session` cookie, and the check against cross-origin requests
- the approval of new users, its emails, and the cleanup job of users who wait
- the emails of the sign up and the password reset
- the session of the UI, its sync between tabs, and the sign up and account pages
- the `x-rate-limit` extension and the `429` response
- the limits of the Public API and their environment variables
- the limit of wrong passwords and the limit of emails
- the limiter, its windows, and its caps
- the PocketBase rate limit rules
- the text of the UI for a `429`

Out of scope:
- a change of the email address. A superuser changes it in the dashboard.
- a sign in link without a password, OAuth2, MFA, and OTP for users
- a sign out of one device, and a list of the sessions of a user
- a sign up with an invite. The backlog has it under "Won't do".
- an approval page in the Public UI. The dashboard is the only admin surface.
- a setting that turns the approval off
- templates of the emails in the dashboard. The texts are in `app/api/internal/publicapi/mail.go`.
- a rate limit on the read operations
- a rate limit on `signOut`, `refreshToken`, and `getCurrentUser`. A user can then always sign out, and the UI checks the session on each page load.
- a limit that stays after a restart, and a limit that more than one server shares. The app runs as one instance. See decision 00010.
- a limit for all write operations of a user together. Each operation has its own count.
- a limit on the total photo storage of a user. The upload limit makes its growth slow.
- large bulk imports. A superuser imports old data in the dashboard or with the PocketBase API.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Operations

| Operation | Request | Result |
|---|---|---|
| `signUp` | `POST /users` | `202` without a body. The API sends an email. |
| `confirmEmail` | `POST /users/confirmations` | an auth token for the token of the link and the password |
| `signIn` | `POST /users/tokens` | an auth token |
| `signOut` | `DELETE /users/tokens` | `204`. All tokens of the user stop. |
| `refreshToken` | `POST /users/tokens/refresh` | a new auth token |
| `requestPasswordReset` | `POST /users/password-resets` | `202` without a body |
| `resetPassword` | `PUT /users/password` | `204`. All tokens of the user stop. |
| `getCurrentUser` | `GET /users/me` | the user of the token, without a new token |
| `updateUser` | `PATCH /users/me` | the user with the new name |
| `changePassword` | `PUT /users/me/password` | a new auth token. All other tokens stop. |
| `deleteUser` | `POST /users/me/deletion` | `204` |

The user object goes only to the user of the token:

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

- `email` has at most 255 characters
- `password` has 8 to 71 characters, and its UTF-8 encoding has at most 72 bytes
- `name` is a `Name` of decision 00001. Other people see it as the `name` of a `creator`.
- passwords are never in a response
- an auth token response has `token`, `expires`, and `user`. Clients do not read `token`. With the cookie, the response has no `token`.

## Sign up and confirmation

- a new address gets a link to `/confirm-email#token=...`. The link works one time and for one day.
- the address of a confirmed user gets an email with links to the sign in page and the password reset. The user does not change.
- the address of a user without confirmation gets a new link. The user gets the new name and password.
- `confirmEmail` needs the token of the link and the password of the last sign up with the address. A wrong password gets `422` with the property `password`, and the token stays valid.
- the page of the link sends the token only after a click on "Confirm email address". A mail scanner that opens the link does not use the token.
- a sign in with the correct password before the confirmation gets `403` with `email_not_confirmed`. The API sends a new link.
- a password reset sets a new password and also confirms the address

Example: Ana signs up with `ana@example.com`. Eve then signs up with the same address and her own password.
Ana gets a second email and opens its link. The page asks for the password. The password of Ana is wrong, so the user stays without confirmation.
Eve has no link, so she cannot confirm the user. Ana signs up again or resets the password.

The sign up form has "Name", "Email", "Password", and "Confirm password".
When the two passwords are not the same, the form shows "The passwords do not match." and sends no request.

## Sign in and wrong passwords

- a wrong email address and a wrong password get the same `401`. The response does not say which one is wrong.
- after too many wrong passwords for one email address, sign in gets `429` with `wrong_password_limit`, also with the correct password. [Rate limits](#rate-limits) has the numbers and their environment variables.
- the count is the same for an address with a user and for an address without one. The limit then does not show which addresses have a user.
- `changePassword` and `deleteUser` need the current password. A wrong password gets `422` and counts for the same limit.
- a password reset removes the count

## Approval

A new user is not approved.

| A user who is not approved | Result |
|---|---|
| signs in, signs out, refreshes the token, uses `/users/me` | works |
| reads restaurants, recipes, reviews, photos, reviewers, and lists | works |
| creates, changes, or deletes data, adds a favorite, or searches a place | `403` with `not_approved` |

- each operation that needs a token also needs an approved user. It has a `403` response in the spec.
- `x-allow-unapproved: true` is the exception. `signOut`, `refreshToken`, `getCurrentUser`, `updateUser`, `changePassword`, and `deleteUser` have it.
- a user who is not approved has no public profile. The reviewer and list operations give `404` to other people. The name of a new account is then not public before the approval.

A superuser opens the `users` collection in the dashboard at `/_/`.
The search `approved = false` lists the users that wait.
The superuser turns on `approved` and saves. The token of the user then works for all operations at the next request.
The superuser can turn `approved` off again. The data of the user stays public. The reviewer page and the lists then give `404` to other people.

Emails:
- the superusers get an email when a user without approval confirms the address
- a confirmed user gets an email when a superuser turns on `approved`

A daily job deletes a user that waits for more than 30 days and has no data. The person can sign up again with the same address.

In the UI:
- a user who waits sees a banner below the header on each page: "Your account waits for approval". "Check again" gets the user again.
- a link to a form that adds or changes data stays on the page and shows a toast with the reason
- the heart and "Save to list" show the same toast and send no request

## The session cookie

The UI sends `cookie=true` with `signIn`, `confirmEmail`, `refreshToken`, and `changePassword`. The response then has this header and no `token`:

```
Set-Cookie: reviews_session=eyJhbGciOi...; Path=/v1; Max-Age=432000; HttpOnly; Secure; SameSite=Strict
```

- the browser sends the cookie with each request to `/v1`. The API uses its token as the bearer token.
- an `Authorization` header wins over the cookie
- a `refreshToken` with the cookie sets a new cookie
- a `signOut` or a `deleteUser` with the cookie deletes the cookie
- a `401` for a cookie that is not valid deletes the cookie
- other clients send a bearer token, as before

The API ignores the cookie and the `cookie` parameter on a cross-origin write from a browser.
A cross-origin write then gets `401`. A cross-origin read uses the cookie, but CORS does not let the other page read the response.

## Sign out

Sign out ends all sessions of the user. A sign out on one device also signs out the other devices.
- PocketBase can only make all tokens of a user invalid. It keeps no list of tokens.
- a sign out of one session needs a table of sessions or of revoked tokens, with a lookup on each request
- a user who thinks that a token leaked wants all sessions to end

## Sessions in the UI

- all tabs of one browser have the same session. A sign out in one tab signs out the other tabs. A sign in in one tab signs in the other tabs.
- a token refresh in one tab gives the new `expires` to the other tabs
- at start, the UI calls `getCurrentUser`. It refreshes the token only after half of the token time.
- the token duration is 5 days, the PocketBase default of the `users` collection. A person who visits at least one time in 2.5 days stays signed in.

## Delete of an account

- `deleteUser` deletes the reviews, the photos, and the lists of the user
- the restaurants and recipes of the user stay. Their `creator` becomes `null`. The reviews of other users can be about them.
- this is the same as a delete in the dashboard

## Rate limits

### Limits of the Public API

| Name | Key | Operations | Default |
|---|---|---|---|
| `sign-in` | IP address | `signUp`, `confirmEmail`, `signIn`, `requestPasswordReset`, `resetPassword` | 10 per minute |
| `create` | user | `createRestaurant`, `createRecipe`, `createPantryItem`, `createRestaurantReview`, `createRecipeReview`, `createPantryItemReview`, `createList` | 30 per hour |
| `edit` | user | the updates and the deletes of restaurants, recipes, pantry items, reviews, photos, lists, and the user, and the list changes | 300 per hour |
| `photo-upload` | user | `addRestaurantReviewPhoto`, `addRecipeReviewPhoto`, `addPantryItemReviewPhoto` | 30 per hour |
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
| requests of all users to the geocoder | the app | 60 per minute. See decision 00008. |

A request over a limit gets this response:

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 2400
Content-Type: application/problem+json

{"type":"about:blank","title":"Too Many Requests","status":429,"detail":"Too many requests. Try again later."}
```

The wrong password limit adds the `code` `wrong_password_limit`.
At the email limit, the request still gets its usual response, but the app sends no email.

Why these values:
- 30 new restaurants in one hour is more than a person adds after a long trip
- a person can add and remove many favorites in a short time. 300 changes of one operation is more than this.
- 5 photos for each review and 30 uploads per hour are the values of the first release. Real use can change them.

### Environment variables

| Variable | Limit |
|---|---|
| `REVIEWS_SIGN_IN_LIMIT` | `sign-in` per minute. The end-to-end tests raise it, because their parallel sign ups come from one IP address. |
| `REVIEWS_SIGN_IN_FAILURE_LIMIT` | wrong passwords per 15 minutes |
| `REVIEWS_CREATE_LIMIT` | `create` per hour |
| `REVIEWS_EDIT_LIMIT` | `edit` per hour |
| `REVIEWS_PHOTO_UPLOAD_LIMIT` | `photo-upload` per hour |
| `REVIEWS_PLACE_SEARCH_LIMIT` | `place-search` per minute |
| `REVIEWS_EMAIL_LIMIT` | emails of each kind per address and hour |

### The UI

- a `429` of a write shows "You made many changes in a short time. Try again later."
- the form keeps its draft after a `429`

# Implementation Details
[implementation-details]: #implementation-details

## Tokens

- the tokens are PocketBase auth tokens of the `users` collection. PocketBase signs each one with the secret of the collection and the `tokenKey` of the record.
- `SignOut` calls `RefreshTokenKey` on the user and saves it. The save skips the field validation, so a record that breaks a newer field rule can still sign out.
- `refreshToken` makes a new token only for a token with the `refreshable` claim. A superuser can make a token without it to act as a user.
- `expires` comes from the `exp` claim
- the verification and password reset tokens are the PocketBase tokens. A new token key makes them invalid after one use.
- the token of a link is after the `#`. Browsers do not send this part to a server, so it is not in access logs.
- `password` auth is on, and `email` is the only identity field. The unique index on `email` uses `COLLATE NOCASE`.
- the `AuthRule` of `users` is `null`. The PocketBase auth routes of `users` are closed. See decision 00002.

## Sign up and sign in

- sign up runs the PocketBase validation also for a used address. Both cases get the same `422`.
- the app sends each email after the response. The response time is then the same with and without an email.
- at the email limit, a sign up does not change a user without confirmation. The last link then stays valid.
- the emails to users do not have the name of the user. A person who does not own the address can set this name.
- with SMTP off, the app writes each email with its link to the log. See decision 00010.
- sign in compares the password with a dummy bcrypt hash when no user has the address. Sign in then takes the same time for an unknown address and a wrong password.
- `ConfirmEmail` checks the token first, then the password. Its count of wrong passwords has the key `confirm:` and the user ID. A person without the link then cannot block the confirmation.
- sign up and sign in ignore the `Authorization` header. A stored token that is not valid does not block them.

## Approval

- `approved` is a bool field of `users`. Its default is `false`. `idx_users_approved` on `(approved, created)` serves the dashboard search.
- only a superuser and SQL can set `approved`. The `users` collection has no API rules.
- `loadSpec` sets the approval check on each operation with a required token, unless it has `x-allow-unapproved`. It fails at start when such an operation has no `403` response.
- `prepare` checks `approved` after the token check and before the rate limit of the user. A refused request does not count for the limit.
- the check reads the user record that PocketBase loads for the token. A change of the approval then works at the next request.
- `findReviewer` returns no user without approval, except for the user of the token. The reviewer and list read operations use it. The `webui` package gives `404` and no link preview for these pages.
- `app/api/internal/usercleanup` is the daily job `user-cleanup`. It deletes the users that are not approved, are older than 30 days, and have no review, restaurant, recipe, pantry item, or list item.
  A user with data who lost the approval stays. The job does not delete data that people see.
- `mail.go` sends the approval email from an `OnRecordAfterUpdateSuccess` hook of `users`

## Cookie

- `Path=/v1` sends the cookie only to the API. The UI build, the photos in `<img>`, and the PocketBase routes do not get it.
- `HttpOnly` hides the cookie from scripts. `SameSite=Strict` stops the browser from sending it on a request from another site.
- `Max-Age` is the time until the `exp` claim of the token
- `Secure` comes from TLS, or from the `X-Forwarded-Proto` header of the reverse proxy. Without the header, it comes from an `https` Application URL with the host of the request. See decision 00010.
- a client can send a wrong `X-Forwarded-Proto`. This changes only its own cookie.
- the name has no `__Host-` prefix. This prefix needs `Path=/`.
- `authenticate` in `server.go` reads the cookie when the request has no `Authorization` header
- the `cookieAuth` scheme is in each operation that has `bearerAuth`. A test checks that the cookie names of the spec and the code are equal.

Cross-origin check:
- `SameSite=Strict` does not stop a request from another origin of the same site, such as another subdomain
- each request with the cookie or `cookie=true` gets `http.CrossOriginProtection` of the Go standard library
- a `Sec-Fetch-Site` of `same-origin` or `none` passes. Without it, an `Origin` with the host of the request passes. A request without `Origin` does not come from a browser and passes.
- `GET`, `HEAD`, and `OPTIONS` always pass. The CORS settings of PocketBase do not allow credentials.
- the check does not refuse a request. It removes the cookie and the `cookie` parameter.
- the check of `signIn` stops a login CSRF. A cross-origin page cannot sign the browser in as another user.
- a request with a bearer token has no check. A page cannot add an `Authorization` header to a request of another origin without CORS.

## The UI

- `localStorage` keeps `expires`, `issued`, and `user` of the session. They are not secret. The token is only in the cookie.
- `issued` is the browser time when the UI got the token
- `checkSession` in `app/ui/src/composables/useAuth.ts` calls `refreshToken` when half of the time from `issued` to `expires` is over. Otherwise it calls `getCurrentUser`.
- `checkSession` shares one request between calls at the same time. The UI also checks the token each hour.
- a middleware sends requests with `credentials: 'omit'` while the UI has no session. A cookie that a failed sign out left then has no effect.
- a `401` on a request with credentials ends the session. The middleware sends the request again without credentials. A read then works. A page that needs an account opens the sign in page with `reason=session-ended`.
- the browser sends a `storage` event to the other tabs when the session changes. `watchOtherTabs` of `session.ts` loads it again.
  On a change of the user, `main.ts` navigates again only on a page that needs a session after a sign out, or on a guest page after a sign in.
  A change to a different user reloads the tab.
- drafts of forms are in `sessionStorage`, so each tab has its own. A tab deletes them when another tab signs out.
- the sign out deletes the session first, then sends the request with `keepalive`.
  A `401` with `error="invalid_token"` means that a sign out on another device already changed the token key. The UI treats it as success.
  Another failure shows a toast that only this browser is signed out.
- `pendingApproval` of `session.ts` is true for a stored user with `approved` `false`. `ApprovalBanner` is in `App.vue`.
- the form routes have `requiresApproval`. The guard cancels the navigation for a user who waits. On a first load, it gets the user again before it decides.
- after a `403` with `not_approved`, the API client stores `approved` as `false`. The banner then shows the reason of the refused write.
- the UI checks are for the person. The API check is the security measure.

## Corner cases

- `issued` uses the clock of the browser. `expires` uses the clock of the server. A browser clock that is ahead makes the refresh earlier. A clock that is behind by more than half of the token time lets the token expire.
- a superuser takes the approval back while a form is open. The save gets `403` with `not_approved`, and the banner shows at once.
- a superuser makes a user in the dashboard. The user is approved only when the superuser turns on `approved`.

Accepted risks:
- a person who knows an address can block its sign in with wrong passwords. A session that the user has continues to work.
- a person can make the sign up of an owner fail for one hour with sign ups up to the email limit. The owner then resets the password.
- a failed sign out leaves the cookie in the browser until it expires. The token stays valid on the server. The UI does not send this cookie while it has no session.
- a script on the page can send requests as the user while the page is open. The Content Security Policy of decision 00002 is the defence.
- the counts of the rate limits are in memory. A restart clears them. An attacker then gets at most one more window.

## Tests

- the Go tests approve their users with SQL in `signUp`. `signUpPending` makes a user that waits.
- each end-to-end app has a superuser. `api.signUp` confirms and approves the user through the PocketBase API. `api.signUpPending` only confirms.
- `api.emailLink` reads the link of an email from the log of the test app. The test app has no mail server.
- not tested: an email through a real SMTP server

## Rate limits

### The spec

- `x-rate-limit` has a `key` and a `name`. The key is `ip` or `user`. The name selects the settings: `sign-in`, `create`, `edit`, `photo-upload`, or `place-search`.
- `loadSpec` refuses an operation with `x-rate-limit` and without a `429` response
- each write operation must have `x-rate-limit`. `signOut` and `refreshToken` are the exceptions.
- the `TooManyRequests` response tells that each operation has its own count
- the wrong password limit and the email limit are not in the extension. Their key comes from the body, and they count only some requests.

### Order of the checks

- the `ip` key counts before the token check. Sign up and sign in have no user.
- the `user` key counts after the token check and the approval check. A request without a token gets `401`, and a user without approval gets `403`. Neither uses a count.
- sign in counts the wrong password limit before it looks for the user. An address at the limit gets `429` without a lookup and without a bcrypt compare.
- a sign in with the correct password does not count. A correct password over the limit also gets `429`. Otherwise the limit would not stop a guess.

### The limiter

- `rateLimiter` in `app/api/internal/publicapi/ratelimit.go` counts in fixed windows in the memory of the process
- each request removes the expired windows first
- a limiter keeps at most 10,000 windows. At the cap, a new key removes the oldest window.
- `Retry-After` is the number of seconds until the window ends, at least 1
- an IPv6 address counts as its /64 network. One host often gets a full /64 network.
- the IP address comes from `RequestEvent.RealIP`. Behind a reverse proxy, the trusted proxy header must be set. See decision 00010.
- `Options` of `publicapi` has the limits and the windows. Zero uses the default.

### The wrong password limit

- `signInFailures` counts with the key of `passwordLimitKey`, the email address in lower case
- the count is the same for an address with a user and for an address without one. The limit then does not show which addresses have a user.
- its cap is 100,000 windows. Clients without an account make its keys, so a limiter without a cap could use all memory.
  `Credentials.email` has at most 255 characters, so the limiter uses less than 40 MB.
- at the cap, a new key removes the oldest window, the same for all addresses. A client needs 100,000 requests with new addresses to get 10 more tries for one user.
- `confirmEmail` counts with the key `confirm:` and the user ID. No email address is equal to this key.
- the PocketBase rate limit rules cannot do this. They count only per IP address.

### PocketBase rules

- a migration turns on the rate limits of PocketBase. Their default rules limit the auth routes under `/api`, which include the superuser sign in.
- a migration adds the rule `POST /csp-report` with 20 requests per minute. See decision 00002.
- these rules do not apply to `/v1`

### Tests

- the end-to-end tests set the `create` and `edit` limits to 10000
- the rate limit project of the end-to-end tests checks the `create` limit with its own app. See decision 00010.
- the Go tests use high limits. Some tests use a low limit.

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| The token in `localStorage` with a bearer header in the UI | Removed. A script on the page could read the token and use it from any device. |
| Sign out only in the browser | Removed. A leaked token stayed valid and could be refreshed without end. |
| `409` for a sign up with a used address | Removed. It showed which addresses have an account. |
| A sign up that signs the user in at once | Removed. A person who does not own the address could then use it. |
| `confirmEmail` with only the token of the link | Removed. A second sign up could set the password that the link confirmed. |
| A wrong password limit only for addresses with a user | Removed. The different status showed which addresses have a user. |
| Skip the confirmation with SMTP off | Not selected. The API would have two behaviors. |
| Fail the sign up with SMTP off | Not selected. Local development and the tests would need a mail server. |
| The `verified` field of PocketBase for the approval | Not selected. It means that the user confirmed the address. |
| Block the sign in of a user who waits | Not selected. The person could not see the state of the account. |
| A refresh of the token at each load of the UI | Removed. Each load made a new token, so many valid tokens of one user existed. |
| No "Confirm password" field | Removed. A typing error in the password could lose the account. |
| A limit only on sign up, sign in, uploads, and the place search | Removed. An approved account with a stolen token could fill the site. |
| A wrong password limit with the user ID as the key | Removed. An address without a user got `401` where an address with a user got `429`. |
| A wrong password limiter without a cap | Removed. Clients without an account make its keys, so it could use all memory. |
| A count in the database | Not selected. The app runs as one instance, so a stored count gives little. |
| A table of operations and limits in Go | Not selected. The spec already names each operation. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
