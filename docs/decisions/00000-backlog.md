--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## Items for Claude
- Add tests to `app/internal/searchindex`
  - the package has no test files. The tests of `publicapi` use it through the record hooks.
- Sync the session between browser tabs
  - `app/ui/src/lib/session.ts` has no `storage` listener
  - a sign out in one tab leaves the other tabs signed in until a reload
- Remove the full scans of the overall sort and filter
  - `joinOverall` in `app/internal/publicapi/ratings.go` groups all rows of `reviews` on each request
  - the tag filters use `json_each` without an index
- Add the missing CI checks
  - no `govulncheck`
  - no ESLint for the UI and the tests
  - `make test-app` does not use `-race`. `go test -race ./...` passes today.
- Fix the doc drift
  - `AGENTS.md` names `docs/decisions/backlog/`. The backlog is this file.
  - `README.md` says `app/pb_ddata`. The folder is `app/pb_data`.
  - `README.md` says `cd tests && npx playwright install`. `AGENTS.md` says `make install-browsers`.
- Limit sign in per account
  - `serveOperation` in `app/internal/publicapi/server.go` counts sign in only per client network
  - many IP addresses can guess the password of one account. No lockout stops them.
  - the limiter is in memory. A restart resets all counts.
- Limit the photo uploads that wait for a decode
  - `readPhoto` in `app/internal/publicapi/photos.go` reads up to 10 MB before it waits for a decode slot
  - there are 2 decode slots. Each upload that waits holds its bytes in memory.
- Fix the dead ends of restaurant ownership
  - only the creator can change a restaurant
  - one review of any user blocks the delete and the rename
  - nobody can change a restaurant through the API after its creator is deleted
  - the creator can still replace the description and the tags after reviews exist
- Explain the score scales in the review form
  - the form shows only the label and the numbers 1 to 10
  - the spec defines Price as value for money. A person can read 10 as expensive.
  - the spec has a description for each of the five scores
- Add an overall score to a review, and a page for each reviewer
  - a review card shows five scores and no single value
  - a person can list only their own reviews
- Give shared links their own preview
  - `app/ui/index.html` has one set of Open Graph tags for all pages
  - a shared restaurant or review shows the preview of the site
  - search engines get little content from the client-side app
- Add server-side sign out
  - sign out deletes the token only in the browser
  - a leaked token can be refreshed without end. Decision 00001 accepts this risk.
  - without a password change, the user cannot stop it

## High Priority

- Add deploy folder and docker compose setup
  - Key components
    - the app will run as a single instance and deployment
    - add the `make build-image` target and the `docker` CI job of decision 00010
    - add the `docker` job to the `needs` of the `CI result` job\
  - Include a backup plan for `pb_data`
    - the database and the photo files in `pb_data/storage` need backups
    - the photos of decision 00008 make `pb_data/storage` much larger than the database

## Low Priority
- Add maps feature, add location to to each restaurant, and prevent duplicates
  - a restaurant has only a name, a description, and tags
  - two records of one place split its reviews
  - the location of the maps item above can help
- Add IOS app
- Add load tests
- Add renovate on a once per month scheudle that groups all updates into a single PR

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
