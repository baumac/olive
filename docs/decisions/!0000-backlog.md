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
