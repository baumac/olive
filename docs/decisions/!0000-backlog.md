--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## For claude
- generate list of unresolved questions and resolve them with the user
- Undo decision 00021-add-structured-data-and-a-sitemap.md to add structured data and a sitemap for search engines
- Update decision 00018 with improved map UX
  - encode the position of the map in the url. Currently it is not in the URL, so a reload opens Barcelona again
  - allow users to set a prefered location that will be the default location the map opens for them
- Update decision 00029 to address the following
  - The UI does not tell users that "Highest rated" uses the number of reviews. The restaurant cards show the number of reviews.
- Update decision 00032 to address the following
  - The section is below a review list of up to 20 reviews. A reviewer with many reviews has the restaurants far down the page.


## For Me
- review all decisions against decision 00000-create-reviews-app.md
  - generate a list of all potential decisions that go against it
  - decide if we want to revert those decisions
- consolidate and review all decisions as there are several duplicates and related decision docs that can be merged
- deploy to prod

## Won't do
- Allow directly uploading photos to restaurants or recipes: all photos must be uploaded via a review to force photos being associated to a review
- Support reviewing individual foods: for now reviewing individual foods is out of scope for this project
  - support reviewing cheeses
  - support reviewing wines
- Expose operations that can be worked around by having an admin use the PocketBase dashboard
  - No email change. UserPatch has only name; a new address needs the dashboard.
  - Only the creator can edit a restaurant. If the creator is inactive, the website, phone, hours and tags are frozen.
- Add IOS app: won't do for now as this requires a mac laptop and side-loading the app from that device, or a 99$ per year subscription to host the app in the app store
- Add load tests: not needed as the number of users are quite low
- Support oauth2 logins: not needed since email/password based authentication is sufficient
- Support inviting users via email
  - Note: A person can still make the sign up of an owner fail for one hour. The person signs up with the address until its email limit. The owner must then reset the password. An invite removes this.
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
