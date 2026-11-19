---
description: Sign up sends a confirmation link and gives the same response for all email addresses. The API also gets a password reset, a name change, a password change, an account delete, and email notices for the approval.
status: implemented
---

# Motivation
[motivation]: #motivation

- sign up told with `409` that an email address has an account. Decision 00001 accepted this risk.
- a person who forgot the password lost the account
- a user could not change the name or the password, and could not delete the account
- a superuser had to search for new users in the dashboard. A user did not know about the approval.

# Scope
[scope]: #scope

In scope:
- the confirmation of the email address at sign up
- the password reset by email
- the operations of `/users/me`: name change, password change, and account delete
- the email to the superusers about a new user, and the email to the user about the approval
- the pages of the UI for these operations

Out of scope:
- a change of the email address
- a sign in link without a password
- templates of the emails in the dashboard. The texts are in `app/internal/publicapi/mail.go`.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

Version 3.0.0 of the spec has these operations. The path stays `/v1`.

| Operation | Request | Result |
| --- | --- | --- |
| `signUp` | `POST /users` | `202` without a body. The API sends an email. |
| `confirmEmail` | `POST /users/confirmations` | an auth token for the token of the link |
| `requestPasswordReset` | `POST /users/password-resets` | `202` without a body |
| `resetPassword` | `PUT /users/password` | `204`. All auth tokens of the user stop. |
| `updateUser` | `PATCH /users/me` | the user with the new name |
| `changePassword` | `PUT /users/me/password` | a new auth token. All other tokens stop. |
| `deleteUser` | `POST /users/me/deletion` | `204` |

Sign up:
- a new email address gets a link to `/confirm-email#token=...`. The link works one time and for one day.
- the email address of a confirmed user gets an email with links to the sign in page and the password reset. The user does not change.
- the email address of a user without confirmation gets a new link. The user gets the new name and password. A person who does not own the address then cannot set its password.
- a sign in with the correct password before the confirmation gets `403` with the `code` `email_not_confirmed`. The API sends a new link.

Account operations:
- `changePassword` and `deleteUser` need the current password. A wrong password gets `422` and counts for the wrong password limit of sign in.
- `deleteUser` deletes the reviews, the photos, and the lists of the user. The restaurants stay without a creator. This is the same as a delete in the dashboard.
- a user without approval can use the operations of `/users/me`
- a password reset also confirms the email address

Emails:
- the superusers get an email when a user without approval confirms the email address
- a confirmed user gets an email when a superuser turns on `approved`
- with SMTP off, the app writes each email with its link to the log. Local development then needs no mail server.
- one address gets at most 5 emails of each kind in one hour. `REVIEWS_EMAIL_LIMIT` changes the number.
- at this limit, a sign up does not change a user without confirmation. The last link then stays valid.
- the emails to users do not have the name of the user. A person who does not own the address can set this name.
- the page of the link sends the token only after a click on "Confirm email address". A mail scanner that opens the link does not use the token.
- a password reset removes the wrong password count of the user

# Implementation Details
[implementation-details]: #implementation-details

- the tokens are the verification and password reset tokens of PocketBase. A new token key of the user makes them invalid after one use.
- the token is after the `#` of the link. Browsers do not send this part to a server, so it is not in access logs.
- sign up runs the PocketBase validation also for a used email address. Both cases then get the same `422`.
- the app sends each email after the response. The response time is then the same with and without an email.
- a migration sets `verified` for the users from before this change
- the PocketBase account routes of `users` stay closed
- `robots.txt` disallows the new pages

| Option for SMTP off | Result |
| --- | --- |
| write the email to the log | Selected. The API has one behavior. |
| skip the confirmation | Not selected. The API then has two behaviors, and the risk of decision 00001 stays. |
| fail the sign up | Not selected. Local development and the tests then need a mail server. |

## Tests

- `account_test.go` has the Go tests of the operations and the emails
- `app/ui/test/account.test.ts` has the unit tests of the pages
- `tests/e2e/auth.spec.ts` reads the links from the log of the test app. `api.signUp` confirms a user as a superuser.

Not tested: an email through a real SMTP server.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.