---
description: The wrong password limit counts each email address, with or without a user. `confirmEmail` needs the password of the user and the token of the link.
status: implemented
---

# Motivation
[motivation]: #motivation

A review of the Public API found two gaps in the account operations.

- The wrong password limit of decision 00015 counted only users. After 10 wrong passwords, sign in gave `429` for an email address with a user and `401` for an address without one. A person could find out which addresses have an account. Decisions 00001 and 00037 keep this fact secret.
- A second sign up replaces the password of a user without confirmation. The link of the email then confirmed this password. A person who does not own the address could sign up with it at the right time. The owner then opened a real link and confirmed a user with the password of the other person. A superuser approved this user, and the other person could sign in.
  - The time is right after the sign up of the owner, before the owner opens the link.
  - The time is also before the sign up of the owner, when the address is at its email limit. The sign up of the owner then changes nothing, and the owner opens an older link.

# Scope
[scope]: #scope

In scope:
- the key of the wrong password limit
- the `password` property of `confirmEmail` and the "Password" field of `ConfirmEmailPage`
- version 4.0.0 of the Public API

Out of scope:
- a sign up with an invite. It removes the open sign up and its emails. It is a larger change.
- the lock of a user by a person who sends wrong passwords for the address. A password reset removes the count.
- the other findings of the review

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Wrong passwords

- Each email address can get 10 wrong passwords in 15 minutes. `REVIEWS_SIGN_IN_FAILURE_LIMIT` changes the number.
- The count is the same for an address with a user and for an address without one. Sign in gives `401` for the first 10 wrong passwords and then `429` with the `code` `wrong_password_limit`.
- The count ignores the case of the address.
- `changePassword` and `deleteUser` count for the address of their user.

## Confirmation

- `confirmEmail` needs `token` and `password`. The `password` is the password of the last sign up with the address.
- A wrong `password` gets `422` with the property `password`. The token stays valid.
- The confirmation has its own count of wrong passwords for each user, with the same limit. Only a request with a valid token adds to it. A person without the link then cannot block the confirmation with wrong passwords at sign in.
- The owner of an address cannot confirm a user with a password that another person set. The owner signs up again or resets the password. A password reset sets a new password and confirms the address.
- `ConfirmEmailPage` has a "Password" field and the link "Forgot your password?".

Example: Ana signs up with `ana@example.com`. Eve then signs up with the same address and her own password. Ana gets a second email and opens its link. The page asks for the password. The password of Ana is wrong, so the user stays without confirmation. Eve has no link, so she cannot confirm the user.

# Implementation Details
[implementation-details]: #implementation-details

- `passwordLimitKey` changes an email address to lower case. It is the key of the `signInFailures` limiter.
- `SignIn` counts the request before it looks for the user. An address at the limit gets `429` without a lookup and without a password check.
- the limiter now has a cap of 100,000 windows. Clients without an account make its keys, so a limiter without a cap can use all memory. A client with many IPv6 networks can send a new address in each request.
  - `Credentials.email` has a maximum of 255 characters. The limiter then uses less than 40 MB.
  - at the cap, a new key removes the oldest window. This is the same for an address with a user and for one without. The limit then still does not show which addresses have a user.
  - decision 00015 had no cap, so that other keys cannot remove the window of a user. A client now needs 100,000 requests with new addresses to get 10 more tries for one user.
- `ConfirmEmail` first checks the token, then the password with `checkPassword`. Its key is `confirm:` and the ID of the user. No email address is equal to this key.
- the change of `confirmEmail` breaks clients, so the version is 4.0.0. The UI is the only client. The path stays `/v1`.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- A person can still make the sign up of an owner fail for one hour. The person signs up with the address until its email limit. The owner must then reset the password. An invite removes this.
