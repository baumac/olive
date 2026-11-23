--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## For claude
- Update the design system and the UI for the rename to "Sponga47 Food Reviews" at food.sponga47.com
  - The text and the page titles already have the new name. The drawn assets and their alt texts do not.
  - The logo lockups still draw "Sponga47 Reviews" as outlines. Draw "Food Reviews" in TeX Gyre Bonum Regular and make the `viewBox` wider.
    - Then change the `<title>` and `aria-label` of the lockups to "Sponga47 Food Reviews". They match the drawn name now.
    - The lockups are in `specs/design-system/assets/Logos` and in `app/ui/src/assets/logos`. Change both copies.
    - Check the header logo at phone width. The wider lockup can push the header buttons.
    - `specs/design-system/assets/Logos/README.md` has the rules of "Food Reviews". Remove its note about the old files.
  - `og-image.png` still shows the old name and the old domain `reviews.sponga47.com`. Make a new image and replace both copies.
    - Then change `og:image:alt` in `app/ui/index.html` and `specs/design-system/assets/Social/README.md` to the new name.
    - The copies are `specs/design-system/assets/Social/og-image.png` and `app/ui/public/og-image.png`.
  - Decide on the logo file names `sponga47-reviews-*.svg` and the `short_name` "Sponga47" of `site.webmanifest`.

## For Me
- deploy to prod

## For the Future

- Support reviewing individual foods: for initial release reviewing individual foods won't be completed
  - reviewing cheeses
  - reviewing wines
- Replace the public Photon server of Komoot and the tile server of OpenStreetMap with self-hosted options
  - `REVIEWS_PLACES_URL` and `REVIEWS_MAP_TILE_URL` already change the servers. The move needs no code change, only the servers on the home server.
  - The tile server of OpenStreetMap has no guarantee and can block heavy use. See decision 00008.

## Won't do
- Allow directly uploading photos to restaurants or recipes: all photos must be uploaded via a review to force photos being associated to a review
- Expose operations that can be worked around by having an admin use the PocketBase dashboard
  - No email change. UserPatch has only name; a new address needs the dashboard.
  - No bulk import and no recipe import from a URL. A superuser adds old restaurants, recipes, and reviews in the dashboard or with the PocketBase API. The create limit of decision 00003 is only on the Public API.
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
    - a superuser must find it and delete it in the dashboard. Decision 00007 accepts this for photos.
