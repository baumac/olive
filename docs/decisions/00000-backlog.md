--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

- Add a README.md file to the project for newcomers to understand and get started with the project.
- Implement the Public API OAS in `specs/api/public-openapi.yaml` in the `api` directory
- Generating client libraries and documentation for the Public API
- Implement the UI based on the Public API Spec
- Define end-to-end tests in the `tests` directory
- Adding rating aggregate field to restaurants
- Adding search
- Adding email verification to sign up
  - sign up gives the same response for new and used email addresses. This removes the account enumeration risk in decision 00001.
  - the API sends a confirmation link for a new email address, or a sign in link for a used one
  - password reset needs the same email setup
