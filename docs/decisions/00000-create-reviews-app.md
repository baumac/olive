---
description: Create a self-hosted app to save, review, and share food experiences.
status: implemented
---

# Motivation
[motivation]: #motivation

My partner and I are foodies.
We love to cook, to eat, and to try all kinds of food.
We want to remember and share all of our food experiences.
These include recipes, restaurants, wines, and cheeses.
We want one app to save each experience and to review it.

We also want to share our food experiences with others including close friends, family, and fellow foodies. Through this app we want to be able to share our experiences with others.

Expected outcome:
- Users can save and review food experiences
- Users can recall past food experience
- Users can search for new food experiences 
- Users can make an account, write their own reviews, and read others
- Users can share lists of their favorite things
- The app is hosted on our home server

# Scope
[scope]: #scope

In scope:
- Defining the goal of the app
- Defining users of the app
- Defining potential things that can be reviewed
- Defining the parts of the app: an API, a UI, and an admin dashboard
- Defining how and where the app will be hosted

Out of scope:
- The choice of technology. Later decisions cover it. See [Implementation Details](#implementation-details).
- Specifying the exact things that can be reviewed
- Wines and cheeses in the first release. The first release has restaurants and recipes. A later decision can add them. The backlog has them under "For the Future".
- Self-hosted maps and place search in the first release. The first release uses the free public Photon server of Komoot and the tile server of OpenStreetMap. These servers get the search text and the IP address of users. A later decision can move them to our home server. The backlog has this under "For the Future".
- Private or unlisted reviews and lists. All data is public, so a shared link works for each person. A user who wants less exposure can use a nickname as the name.
- Social features, such as comments, likes, follows, and messages. The users are few and talk to each other in other ways.
- Ads or other income

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

We make the app with AI coding tools.
The agents follow `AGENTS.md`.
Each design choice is recorded as a decision in `docs/decisions`.

The users are few, but the app has the checks of a larger app.
Each write operation has a rate limit.
Each change of a restaurant or a recipe has a version.
The tests run in many browsers and have a coverage minimum.
Renovate updates the dependencies each month.
These checks keep the app robust on the public internet.
They also find the errors of the AI coding tools early.

The app has an API and a UI that users use.
It also has an admin dashboard for admins.
The app runs on our home server.
It is reachable via the public internet at reviews.sponga47.com.

Each page of the app is public.
A person without an account can read each review, reviewer page, and list.
A shared link then works for each friend.
Only approved users can write.
Search engines and AI crawlers that obey robots.txt do not get the pages. See decision 00012.

New concepts:
- **Subjects**: a thing that users review. The first release has restaurants and recipes. Decision 00005 gives the pattern for more kinds, such as wines and cheeses.
- **Review**: the scores, the text, and the photos of one user for one subject.
- **Reviewer**: a user who writes reviews. Each reviewer has a public profile page.
- **List**: a named group of subjects. Each user has a favorites list.

Examples:
- After dinner at a new restaurant, I add the restaurant to the app.
  I write a review with scores and a photo.
  I add the restaurant to my favorites list.
  I send the link of my review to a friend.
  The friend opens the link and reads my review without an account.
- My wife cooks a new recipe.
  She adds the recipe with its ingredients and steps, then reviews it.
  Later, I search for the recipe by name and cook it again.
- A friend wants to join.
  The friend signs up and confirms the email address.
  We approve the friend in the admin dashboard.
  Then the friend can write reviews.

# Implementation Details
[implementation-details]: #implementation-details

The other decisions cover the implementation. Each one covers one topic:

- [00001](00001-public-api-spec-and-sdks.md): the spec of the Public API, the SDKs, and the validation
- [00002](00002-app-server-on-pocketbase.md): the PocketBase app, the UI route, and the Content Security Policy
- [00003](00003-accounts-and-access.md): the sign up, the email confirmation, the approval, and the session
- [00004](00004-rate-limits.md): the rate limits
- [00005](00005-restaurants-and-recipes.md): the restaurants, the recipes, and their tags
- [00006](00006-editing-and-history.md): the edit and delete rules and the history of changes
- [00007](00007-reviews-and-ratings.md): the reviews, the overall, and the ratings
- [00008](00008-reviewers-and-lists.md): the reviewer pages and the lists
- [00009](00009-photos.md): the photos, the cover, and the gallery
- [00010](00010-places-and-maps.md): the place search, the maps, and the preferred location
- [00011](00011-public-ui-and-design-system.md): the Public UI and its design system
- [00012](00012-link-previews-and-crawlers.md): the link previews and robots.txt
- [00013](00013-deployment-and-operations.md): the deploy with Docker Compose and the backups
- [00014](00014-tests.md): the Go, UI, and end-to-end tests
- [00015](00015-ci-and-dependency-updates.md): the CI workflow and Renovate

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
