--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## For claude

## For Me
- Can you make a plan to consolidate decision docs into far fewer decision docs? For context, decisions are meant to be living documents that evolve as requirements change. Today, there are many decisions that were initially made and then either superseded, fixed, or replaced by a future decision. This results in duplicate information and makes the duplicates hard to follow.
  - Some examples are:
    - 00001: only restaurants, no text search, and one review per user and restaurant. Fixed by 00047, the search index, and 00050.
    - 00013: canonical links for search engines. Fixed by 00053.
    - 00016: only the creator can edit, for "an open public site". Fixed by 00051 and 00055.
    - 00021: structured data and a sitemap for search engines. Fixed by 00049 and 00053.
    - 00029: a ranking for about 200 reviews per restaurant. Fixed by 00054.
    - 00038: structured data for search engines in the expected outcome. Fixed by 00049.
- add note in agents.md decisions are meant to be living documents that evolve as requirements change
- verify app implementation against consolidated decisions
- deploy to prod

## For the Future

- Support reviewing individual foods: for initial release reviewing individual foods won't be completed
  - reviewing cheeses
  - reviewing wines
- Replace the public Photon server of Komoot and the tile server of OpenStreetMap with self-hosted options
  - `REVIEWS_PLACES_URL` and `REVIEWS_MAP_TILE_URL` already change the servers. The move needs no code change, only the servers on the home server.
  - The tile server of OpenStreetMap has no guarantee and can block heavy use. See decision 00018.

## Won't do
- Allow directly uploading photos to restaurants or recipes: all photos must be uploaded via a review to force photos being associated to a review
- Expose operations that can be worked around by having an admin use the PocketBase dashboard
  - No email change. UserPatch has only name; a new address needs the dashboard.
  - No bulk import and no recipe import from a URL. A superuser adds old restaurants, recipes, and reviews in the dashboard or with the PocketBase API. The create limit of decision 00036 is only on the Public API.
    - Add photos only through the Public API. A photo from the dashboard keeps its metadata, such as the GPS position.
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
