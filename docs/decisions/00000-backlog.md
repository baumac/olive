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
- Add CI integration that depends on GitHub actions.
  - Job for lint
  - Job for test
  - Job for docker build
  - all workflows should use the makefile to execute their tasks
  - look in AGENTS.md for the full list of potential make commands
  - workflow result to capture the result of the above workflows and combine them into a single pass fail result

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
