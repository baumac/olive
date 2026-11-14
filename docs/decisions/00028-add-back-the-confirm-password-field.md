---
description: Add the "Confirm password" field back to the sign up form of the Public UI.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00019 removed the "Confirm password" field of sign up. The show password button was the only check of the password.

- The API has no password reset. A typing error in the password loses the account.
- A person does not always use the show password button. A second field catches the typing error before the account exists.

Expected outcome:
- a person cannot make an account with a password that has a typing error in only one of the two fields

# Scope
[scope]: #scope

In scope:
- the sign up form of the UI
- the `SignUpForm` text and preview of `specs/design-system`
- the tests

Out of scope:
- the Public API. The `signUp` operation does not get the second password.
- password reset. It stays in `docs/decisions/00000-backlog.md`.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

- The sign up form has the fields "Name", "Email", "Password", and "Confirm password", in this order.
- "Confirm password" has the show password button, the same as "Password".
- The help text of "Password" is "8 to 71 characters." again.
- When the two passwords are not the same, "Confirm password" shows "The passwords do not match.". The form does not send the request.

The other changes of decision 00019 stay.

# Implementation Details
[implementation-details]: #implementation-details

- `SignUpForm` has the field `confirm` with the id `sign-up-confirm` and the name `confirm-password`.
- The UI checks `confirm` when the person leaves the field and on submit, like the other fields.
- The `submit` event has only `name`, `email`, and `password`.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
