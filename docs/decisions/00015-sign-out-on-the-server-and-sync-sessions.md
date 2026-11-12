---
description: Make sign out end all sessions of the user on the server, keep the session equal in all browser tabs, and limit the wrong passwords of each user.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00001 accepted three risks of sessions. This decision removes them.

- Sign out deleted the token only in the browser. The token stayed valid on the server.
  A token that leaked could be refreshed without end. Only a password change could stop it, and the API has none.
- A sign out in one browser tab left the other tabs signed in until a reload.
- The sign in rate limit counted only per IP address.
  A client with many IP addresses could guess the password of one user without a limit.

Expected outcome:
- sign out makes all auth tokens of the user invalid
- all tabs of one browser show the same session without a reload
- sign in stops after 10 wrong passwords for one user in 15 minutes, from any IP address

# Scope
[scope]: #scope

In scope:
- the `signOut` operation, `DELETE /users/tokens`
- the sign out of the UI and the session sync between tabs
- the limit of wrong passwords per user and its `REVIEWS_SIGN_IN_FAILURE_LIMIT` variable
- the tests

Out of scope:
- a sign out of only one device. See the guide-level explanation.
- a list of the sessions of a user
- a stored count of wrong passwords. See the implementation details.
- a password change, a password reset, MFA, OTP, and the auth alert emails of PocketBase. They need email.
- a rate limit that more than one server shares. Decision 00004 has this limit also.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Sign out

```
DELETE /v1/users/tokens
Authorization: Bearer eyJhbGciOi...
```

The response is `204 No Content`.
All auth tokens of the user are then invalid. This includes the tokens of other devices.
A request with one of these tokens gets `401 Unauthorized`.
A sign in after it gives a valid token.

Sign out ends all sessions, not one session:
- PocketBase can only make all tokens of a user invalid. It keeps no list of tokens.
- a sign out of one session needs a table of sessions or of revoked tokens. Each request then needs a lookup.
- a user who thinks that a token leaked wants all sessions to end

The cost is that a sign out on one device also signs out the other devices of the user.

## Sessions in browser tabs

All tabs of one browser use the same session.
- a sign out in one tab signs out the other tabs. A tab on a page that needs a session goes to the sign in page.
- a sign in in one tab signs in the other tabs. A tab on the sign in page or the sign up page goes to its redirect path.
- a token refresh in one tab gives the new token to the other tabs

## Wrong passwords

Each user can get 10 wrong passwords in 15 minutes.
After that, sign in for this user gets `429 Too Many Requests` with a `Retry-After` header.
A correct password also gets `429` until the window ends. Otherwise the limit does not stop a guess.
The limit counts all IP addresses together.
A sign in with the correct password does not count.

`REVIEWS_SIGN_IN_FAILURE_LIMIT` sets the number of wrong passwords.

# Implementation Details
[implementation-details]: #implementation-details

## Sign out

- PocketBase signs each auth token with the secret of the collection and the `tokenKey` of the record
- `SignOut` calls `RefreshTokenKey` on the user record and saves it. No migration is necessary.
- the save also changes `updated` of the user
- a token without the `refreshable` claim can also sign out. A superuser makes such a token and can already change the record.
- the UI deletes its session first and then sends the request. A slow server does not delay the sign out in the browser.
  - the request has the token in its own `Authorization` header, because the session is already empty
  - `keepalive` lets the request finish when the tab closes
  - a `401` means that the token is already invalid. The UI treats it as success.
  - on another failure, a toast says that only this browser is signed out
- another device learns of the sign out with its next request. The `401` handling of the UI then ends its session.

## Sessions in browser tabs

- the session is in `localStorage`. The browser sends a `storage` event to the other tabs when it changes.
- `watchOtherTabs` in `app/ui/src/lib/session.ts` loads the session again on this event
- when the user changes, `main.ts` navigates to the current path again with `force`. The route guards then run again.
- when the user changes to a different user, the tab reloads. Its pages hold data of the first user.
- drafts are in `sessionStorage`, so each tab has its own. A tab deletes them when another tab signs out.

## Wrong passwords

- a second `rateLimiter` counts with the user ID as the key. It is the limiter of decision 00004.
- `SignIn` calls `allow` before the password check and `giveBack` after a correct password
- the check comes before the bcrypt compare. A request over the limit does not use CPU time for the hash.
- only an email address of a user makes a window. Requests with unknown email addresses cannot fill the 10,000 windows and remove the window of a user.
- the limit is not in the `x-rate-limit` extension. Its key comes from the body, and it counts only failures.
- the PocketBase rate limit rules cannot do this. They count only per IP address.

Accepted risks:
- a person who knows the email address of a user can block the sign in of this user. Each 10 requests block it for 15 minutes.
  A session that the user already has continues to work. The IP limit makes this slow from one address.
- the `429` shows that the email address has a user. Sign up already shows this. See decision 00001.
- the counts are in memory. A restart clears them. An attacker then gets at most one more window of 10 guesses.
  The app runs as a single server, so a stored count gives little.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- A sign out of one device. It needs a table of sessions.
- An email to the user when the limit of wrong passwords is reached. It needs email.
