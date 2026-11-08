--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## High Priority


## Low Priority

- Add review lists that are not per restaurant to the Public API
  - the `AppHeader` design has a "Reviews" link, and the `UserMenu` design has "My reviews"
  - the Public API lists reviews only for one restaurant. The UI does not show these items until the API can list them.
- Adding rating aggregate field to restaurants
- Adding search
- Adding email verification to sign up
  - sign up gives the same response for new and used email addresses. This removes the account enumeration risk in decision 00001.
  - the API sends a confirmation link for a new email address, or a sign in link for a used one
  - password reset needs the same email setup
