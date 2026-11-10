--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## High Priority

- Add deploy folder and docker compose setup
  - need to run the built app 
  - the app will run as a single instance and deployment
  - add the `make build-image` target and the `docker` CI job of decision 00010
  - add the `docker` job to the `needs` of the `CI result` job

## Low Priority

- Rename cleanliness to hygine
- Add a backup plan for `pb_data`
  - the database and the photo files in `pb_data/storage` need backups
  - the photos of decision 00008 make `pb_data/storage` much larger than the database
- Add maps feature and add location to to each restaurant
- Add IOS app
- Add load tests

## Won't Do

- Adding email verification to sign up
  - sign up gives the same response for new and used email addresses. This removes the account enumeration risk in decision 00001.
  - the API sends a confirmation link for a new email address, or a sign in link for a used one
  - password reset needs the same email setup
