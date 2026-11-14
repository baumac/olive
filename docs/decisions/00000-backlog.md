--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## Items for Claude

## High Priority

- Document support for oauth2 logins

## Low Priority
- Add IOS app
- Add load tests
- Allow users to set preferred city and incorporate that into the maps feature
- Allow users to favorite restaurants and retrieve their list of favorites
- Add an index for the `overall` sort and filter of reviews
  - the sort and the filter compute the value for each review of the restaurant or the reviewer
  - SQLite does not use an index on an expression for the row value range of a cursor. See the migration `1794268808_add_restaurant_index.go`.
  - a table like `restaurant_overall` with triggers, or a generated column, fixes it. It is not needed while a restaurant has few reviews.
- Keep the auth token of the UI in an HttpOnly cookie
  - the UI keeps the token in `localStorage`. A script that runs on the page can read it. The Content Security Policy makes this risk low.
  - the sign in, token refresh, and sign out operations must then set and clear the cookie
  - a response to a request with the cookie must not have the token in its body. This is a spec change.
  - requests with the cookie need a defence against cross-site requests
- Copy the backups of `deploy/` to a different computer, and test the restore steps
- Let Renovate update the tool versions of the makefile and the image tags of `deploy/`

## Won't Do
- Add rate limits to the write operations
  - only sign up, sign in, and photo upload have `x-rate-limit` in `specs/api/public-openapi.yaml`
  - sign up has no email verification. One script can make 10 accounts per minute per IP address.
  - each account can then create restaurants and reviews without a limit
  - this is a spec change. It needs new SDKs.
- Adding email verification to sign up
  - sign up gives the same response for new and used email addresses. This removes the account enumeration risk in decision 00001.
  - the API sends a confirmation link for a new email address, or a sign in link for a used one
  - password reset needs the same email setup
- Add password reset, password change, name change, and account deletion
  - a person who forgets the password loses the account
  - decision 00001 leaves these out. Password reset needs the email setup of the Won't Do item.
- Add a way for users to report content
  - nobody can report a review, a photo, or a restaurant
  - a superuser must find it and delete it in the dashboard. Decision 00008 accepts this for photos.
