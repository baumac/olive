--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

## For Me
- write 00000-create-reviews-app decision doc that covers the decision to create the app, its intended users, and what it does/does not need to do
  - rename and/or move the backlog doc to the /docs directory
- consolidate and review all decisions as there are several duplicates and related decision docs that can be merged
- review features such as SEO, and social features that don't align with the 00000-create-reviews-app
- deploy to prod

## For Claude
- Add IOS app

## Future
- Support reviewing cheeses, wines, and recipes

## Won't do
- Add load tests
  - Load testing is not needed as the number of users are quite low
- Social features
  - Supporting oauth2 logins
  - Improve social aspects of reviews
    - reviews have no visit date, no helpful vote, and no owner or reader reply. Helpful votes also give the review list a better default sort than newest.
  - Add a way for users to report content
    - nobody can report a review, a photo, or a restaurant
    - a superuser must find it and delete it in the dashboard. Decision 00008 accepts this for photos.
