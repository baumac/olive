--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## For Claude
3. Design gaps
- Only the creator can edit a restaurant. If the creator is inactive, the website, phone, hours and tags are frozen. In a trusted group, let any approved user edit the non-identity fields.
- No GET /users/me. The UI calls refreshToken on every load just to learn who is signed in, minting a new token each time.
- No email change. UserPatch has only name; a new address needs the dashboard.
- listListRestaurants has no search, sort or filters, so favorites cannot be shown on a map or sorted by distance. A list[eq] filter on listRestaurants would give all of that for free.
- No price_level filter, though the field exists.
- Increase test coverage
- Improve test performance

## For Me
- write 00000-create-reviews-app decision doc that covers the decision to create the app, its intended users, and what it does/does not need to do
  - rename and/or move the backlog doc to the /docs directory
- consolidate and review all decisions as there are several duplicates and related decision docs that can be merged
- review features such as SEO, and social features that don't align with the 00000-create-reviews-app
- deploy to prod

## Future
- Support reviewing cheeses, wines, and recipes

## Won't do
- Add IOS app: won't do for now as this requires a mac laptop and side-loading the app from that device, or a 99$ per year subscription to host the app in the app store
- Add load tests: not needed as the number of users are quite low
- Support oauth2 logins: not needed since email/password based authentication is sufficient
- Social features: not needed since this is intended to be used by a low number of users that communicate with each other via other means
  - Reviews have no visit date and no owner or reader reply
  - Comments or replies on review
  - Helpful votes or likes on reviews
  - Following users, or an activity feed
  - Messages or notifications between users
  - Profile pictures or bios (the avatar field was removed)
  - Private or collaborative lists
  - A share button in the UI (sharing works by copying the page link)
  - A way for users to report content
    - a superuser must find it and delete it in the dashboard. Decision 00008 accepts this for photos.
