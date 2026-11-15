--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## High Priority
1. Improve restaurant location UX
  - Allow users to set preferred location and incorporate that into the maps feature
  - The default list sort is `-created`, update it to be `best rated` or `near`. 
  - The API has `near`, but the UI never asks for `navigator.geolocation`.

## Production Readiness

Document/implement the following in deploy.md:
- Document SMTP and superuser MFA in deploy/README.md. Both are dashboard settings. The dashboard is the only admin surface, so its sign in matters most.
- Enable settings encryption
  - Turn on the settings encryption. Add --encryptionEnv to the start command of the image and a key variable to deploy/.env.example. The key must stay out of the backups. A lost key makes the settings of a restored backup unreadable, so the operator keeps a copy.
- Set GOMEMLIMIT and a memory limit for the app service in deploy/compose.yaml
- Add ulimits.nofile to the app service.
- Add request_body { max_size 10MB } to deploy/Caddyfile.
- Configuring oauth2 logins using Gmail/Google


### Additional items
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
  - decision 00001 leaves these out. Password reset needs the email setup of the Ice Box item.
- Use a backup plan that copies each photo one time when `pb_data` grows near 5 GB. See decision 00022.

## Future
- Add IOS app
- Support reviewing cheeses, wines, and recipes
- Add more information to restaurant
  - Potential additional fields are Website, phone, opening hours, and price level are the facts people look for. Photon returns some OSM tags.
- Improve social aspects of reviews
  - Reviews have no visit date, no helpful vote, and no owner or reader reply. Helpful votes also give the review list a better default sort than newest.
- Add a way for users to report content
  - nobody can report a review, a photo, or a restaurant
  - a superuser must find it and delete it in the dashboard. Decision 00008 accepts this for photos.


## Ice Box
- Add load tests
