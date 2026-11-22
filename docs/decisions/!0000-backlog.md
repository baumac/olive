--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## For claude

## For Me

- review all decisions against decision 00000-create-reviews-app.md
  - Useful prompt: `"00000-create-reviews-app.md" contains the original motivations for the creating the app. Can you review all decisions in doc/decisions against "00000-create-reviews-app.md" and create a list of any decisions that are not aligned with the original motivations?`
  - generate a list of all potential decisions that go against it
  - decide if we want to revert those decisions
  - Findings of the review on 2026-11-22:
    - Not aligned, still in force:
      - 00047: wines and cheeses are out of scope. 00000 names them. 00047 says they are under "Won't do", but the backlog has them under "For the Future".
      - 00017 and 00018: the place search sends the search text to `photon.komoot.io`. The browsers load the map tiles from `tile.openstreetmap.org`, which gets the IP address of each user. 00000 wants a self-hosted app. "For the Future" has only the Photon item. It has no item for the map tiles.
    - Partly aligned:
      - 00053: robots.txt keeps all crawlers out. Link preview bots that obey it show no preview of a shared link. The share example of 00000 then loses the preview of 00013.
      - 00036 and 00047: the app has no bulk import and no recipe import from a URL. The default create limit is 30 per hour. A user must type each past experience by hand.
    - Not aligned at first, fixed by a later decision:
      - 00001: only restaurants, no text search, and one review per user and restaurant. Fixed by 00047, the search index, and 00050.
      - 00013: canonical links for search engines. Fixed by 00053.
      - 00016: only the creator can edit, for "an open public site". Fixed by 00051 and 00055.
      - 00021: structured data and a sitemap for search engines. Fixed by 00049 and 00053.
      - 00029: a ranking for about 200 reviews per restaurant. Fixed by 00054.
      - 00038: structured data for search engines in the expected outcome. Fixed by 00049.
    - Judgement calls:
      - All reviews, reviewer names, and lists are public without an account. See 00033 and 00034. A private or unlisted option is not in the backlog.
      - Some decisions are heavier than a small group of trusted users needs: the rate limits of each operation, the version history, the end-to-end tests in many browsers, Renovate, and the coverage checks.
    - Stale text for the consolidation:
      - 00023 says "An iOS app is in the backlog". It is under "Won't do".
      - 00047 says that wines and cheeses are under "Won't do".
- consolidate and review all decisions as there are several duplicates and related decision docs that can be merged
- verify app implementation against consolidated decisions
- deploy to prod

## For the Future

- Support reviewing individual foods: for initial release reviewing individual foods won't be completed
  - reviewing cheeses
  - reviewing wines
- Replace Photon maps server that is hosted by Komoot with a self-hosted option

## Won't do
- Allow directly uploading photos to restaurants or recipes: all photos must be uploaded via a review to force photos being associated to a review
- Expose operations that can be worked around by having an admin use the PocketBase dashboard
  - No email change. UserPatch has only name; a new address needs the dashboard.
- Add IOS app: won't do for now as this requires a mac laptop and side-loading the app from that device, or a 99$ per year subscription to host the app in the app store
- Add load tests: not needed as the number of users are quite low
- Support oauth2 logins: not needed since email/password based authentication is sufficient
- Support inviting users via email
  - Note: A person can still make the sign up of an owner fail for one hour. The person signs up with the address until its email limit. The owner must then reset the password. An invite removes this.
- Social features: not needed since this is intended to be used by a low number of trusted users that communicate with each other via other means
  - Reviews have no owner or reader reply
  - Comments or replies on review
  - Helpful votes or likes on reviews
  - Following users, or an activity feed
  - Messages or notifications between users
  - Profile pictures or bios (the avatar field was removed)
  - Private or collaborative lists
  - A share button in the UI (sharing works by copying the page link)
  - Content moderation and a way for users to report content
    - a superuser must find it and delete it in the dashboard. Decision 00008 accepts this for photos.
