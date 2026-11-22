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
- Social features, such as comments, likes, follows, and messages. The users are few and talk to each other in other ways.
- Ads or other income

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

We make the app with AI coding tools.
The agents follow `AGENTS.md`.
Each design choice is recorded as a decision in `docs/decisions`.

The app has an API and a UI that users use.
It also has an admin dashboard for admins.
The app runs on our home server.
It is reachable via the public internet at reviews.sponga47.com.

New concepts:
- **Subjects**: a thing that users review. The first release has restaurants and recipes. Decision 00047 gives the pattern for more kinds, such as wines and cheeses.
- **Review**: the scores, the text, and the photos of one user for one subject.
- **Reviewer**: a user who writes reviews. Each reviewer has a public profile page.
- **List**: a named group of subjects. Each user has a favorites list.

Examples:
- After dinner at a new restaurant, I add the restaurant to the app.
  I write a review with scores and a photo.
  I add the restaurant to my favorites list.
  I send the link of my review to a friend.
  The friend opens the link and reads my review.
- My wife cooks a new recipe.
  She adds the recipe with its ingredients and steps, then reviews it.
  Later, I search for the recipe by name and cook it again.
- A friend wants to join.
  The friend signs up and confirms the email address.
  We approve the friend in the admin dashboard.
  Then the friend can write reviews.

# Implementation Details
[implementation-details]: #implementation-details

Later decisions cover the implementation.
The decisions below make the base of the app:

- [00001](00001-define-public-api-spec.md): the OpenAPI spec of the Public API
- [00002](00002-add-design-system.md): the design system of the Public UI
- [00003](00003-generate-sdks-from-api-spec.md): the SDKs that we generate from the spec
- [00004](00004-implement-public-api-with-pocketbase.md): the Public API on PocketBase
- [00005](00005-serve-ui-from-pocketbase-app.md): the UI that the PocketBase app serves
- [00022](00022-deploy-with-docker-compose.md): the deploy with Docker Compose and the backups
- [00034](00034-require-superuser-approval-for-new-users.md): the approval of new users
- [00037](00037-add-email-confirmation-and-account-operations.md): the email confirmation of new users
- [00047](00047-support-reviewing-recipes.md): the reviews of recipes

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
