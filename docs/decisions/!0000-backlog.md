--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## For claude

### 1. One review for each user and subject, with no visit date
00001 allows one review for each user and restaurant. 00047 has the same rule for recipes. A second review gets 409 with review_exists. A review also has no visit date. The backlog puts "no visit date" under the social features, but a visit date is not a social feature.

00000 wants to "remember and share all of our food experiences". A second dinner at a restaurant or a second cook of a recipe is a new experience. The user must now change the old review. The old scores, text, and date are then lost. The app cannot show how a restaurant changed, or how often we cooked a recipe.

Decide if a review is "my opinion of this subject" or "one visit or one cook". For M1, the second is the better fit. A middle way keeps one review and adds dated visits to it.

### 2. Restaurants and recipes are locked against other approved users
00016 lets only the creator change a restaurant. Reviews of other users lock the name and the delete. 00017 locks the location, and 00048 locks the source of a recipe. The backlog accepts that the website, phone, hours, and tags stay old when the creator is inactive.

00016 gives the reason: "Changes by all users need an edit history and a report flow to undo vandalism." That reason fits an open public site. 00034 came later. Since then, a superuser approves each user, and the users are friends and family.

Example: my partner adds a restaurant with a typo in its name. I review it. Now nobody can fix the name in the UI, and only a superuser can fix it in the dashboard.

Decide if all approved users can change the facts of a restaurant or a recipe. The name, the location, and the source can stay locked. The website, the phone, the hours, the price level, and the tags are facts that anyone in the group can fix.

### 3. The Bayesian ranking fits a large site
00029 gives each restaurant 5 imaginary reviews of 5.5. Its examples have 200 reviews. In a group of friends, a restaurant has 1 to 5 reviews. The ranking then mostly counts how many of us went there:

Reviews	rating.overall	Ranking
2 reviews of 10	10	6.79
3 reviews of 9	9	6.81
00035 makes "Highest rated" the default sort. The UI does not tell the user that the order also uses the number of reviews. A user then sees a 9.0 above a 10.0 and does not know why.

Decide if a smaller prior, such as 1 review, or the plain average with a minimum review count, fits the group better. 


### 4. All pages are open to search engines
00049 removes the sitemap because "the app does not need search engine optimization". robots.txt still lets crawlers read each restaurant, recipe, review, reviewer, and list page. 00013 also gives each page a canonical URL. Search engines can then show the names and reviews of friends and family. 00033 makes all lists public, and the reviewer pages show the names of users.

00000 wants to share with "close friends, family, and fellow foodies". Its example has a friend who opens a shared link, so the pages must stay readable without an account. They do not need to be in search results.

Decide if robots.txt gets Disallow: /, or the pages get a noindex tag. The link previews of 00013 still work. Some link preview bots read robots.txt, so test them first.


### 7. Reviews have no tags
00011 removes the tags of reviews. The tags of a restaurant belong to its creator. See conflict 3. A user then cannot add a personal tag, such as date-night, to an experience. Lists of 00033 do part of this job.

This is a small conflict with M1. If conflict 3 opens the tags to all approved users, it goes away.

## For Me

- review all decisions against decision 00000-create-reviews-app.md
  - Useful prompt: `"00000-create-reviews-app.md" contains the original motivations for the creating the app. Can you review all decisions in doc/decisions against "00000-create-reviews-app.md" and create a list of any decisions that are not aligned with the original motivations?`
  - generate a list of all potential decisions that go against it
  - decide if we want to revert those decisions
- consolidate and review all decisions as there are several duplicates and related decision docs that can be merged
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
