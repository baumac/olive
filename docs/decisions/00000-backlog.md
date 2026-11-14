--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## Product items needing triage
1. The default list sort is `-created`. A discovery app wants best rated or nearest first. The API has `near`, but the UI never asks for `navigator.geolocation`.
2. Photos are only on review cards. The restaurant card and the top of the restaurant page have none. A cover photo and a gallery from the review photos need no new upload.
3. The "Won't Do" list is one cluster: no email verification, no write rate limits, no reports, no password reset.
   Together they mean a public instance has no defence against spam and no recovery for a lost password. The cheapest slice is a report button that makes a record for the dashboard, plus per-user write limits.
4. A restaurant has only name, description, tags, location. Website, phone, opening hours, and price level are the facts people look for. Photon returns some OSM tags.
5. Reviews have no visit date, no helpful vote, and no owner or reader reply. Helpful votes also give the review list a better default sort than newest.
6. Reviewer page has only a count. Add the average score of the reviewer and the restaurants they added (00014 left this out).

## High Priority


## Low Priority
- Add IOS app
- Allow users to set preferred location and incorporate that into the maps feature
- Allow users to favorite restaurants and retrieve their list of favorites
- Add an index for the `overall` sort and filter of reviews
  - the sort and the filter compute the value for each review of the restaurant or the reviewer
  - SQLite does not use an index on an expression for the row value range of a cursor. See the migration `1794268808_add_restaurant_index.go`.
  - a table like `restaurant_overall` with triggers, or a generated column, fixes it. It is not needed while a restaurant has few reviews.
- Use a backup plan that copies each photo one time when `pb_data` grows near 5 GB. See decision 00022.

## Future Features
- Support reviewing cheeses, wines, and recipes
- Add load tests

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
- Support for oauth2 logins
