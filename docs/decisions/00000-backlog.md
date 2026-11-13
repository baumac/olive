--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## Items for Claude

## High Priority

- Add deploy folder and docker compose setup
  - Deployment details
    - the app will run as a single instance and deployment
    - add the `make build-image` target and the `docker` CI job of decision 00010
    - add the `docker` job to the `needs` of the `CI result` job\
  - Include a backup plan for `pb_data`
    - the database and the photo files in `pb_data/storage` need backups
    - the photos of decision 00008 make `pb_data/storage` much larger than the database
  - Document support for oauth2 logins

## Low Priority
- Add maps to the UI
  - decision 00017 gave each restaurant a location. Its unresolved questions have the plan.
  - Leaflet with the tiles of `tile.openstreetmap.org`, loaded by the browser. The CSP `img-src` needs the tile host.
  - the tile policy needs a visible "© OpenStreetMap contributors", a Referer, and no prefetch
  - the map opens on Barcelona. The list filter needs a bounding box.
  - a pin on a map can replace the latitude and longitude fields of the restaurant form
- Add IOS app
- Add load tests
- Add test coverage make command, and ci check
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
