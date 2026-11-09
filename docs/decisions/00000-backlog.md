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
- Add OAS validation middleware
  - should be built using https://github.com/getkin/kin-openapi
  - it should be able to validate requests
  - it should be able to validate responses using a buffered response writer wrapper that buffers the response body, headers, and status code to be modified. To write to the buffered response the Flush() function must be called, otherwise underlying writer will never be written. When an invalid response is detected a generic 500 error response should be returned.
  - it should replace what is in app/internal/publicap/validate.go
- Add CI integration that depends on GitHub actions, split the commands into the following
  - lint
  - UI tests
  - API tests
  - E2E tests

## Low Priority

- Remove reviews from to nav bar
- Rename cleanliness to hygine
- Add maps feature and add location to to each restaurant
- Add IOS app

## Won't Do

- Adding email verification to sign up
  - sign up gives the same response for new and used email addresses. This removes the account enumeration risk in decision 00001.
  - the API sends a confirmation link for a new email address, or a sign in link for a used one
  - password reset needs the same email setup
