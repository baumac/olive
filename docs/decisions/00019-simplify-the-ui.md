---
description: Make the Public UI simpler. Show a score as one number, remove repeated controls and text, and shorten the forms.
status: implemented
---

# Motivation
[motivation]: #motivation

The UI had a calm base, but repetition made it feel busy.

- A restaurant page with three reviews showed 24 rows of ten dots. Each row gave one value three times.
- The restaurants list had three rows of controls before the first restaurant.
- The "Price" score needed a note on three pages, because a high price score is good.
- The review form had 50 buttons and ten lines of help text. The photos came after the save button.

Expected outcome:
- a person sees the overall score of a restaurant or a review at a glance
- each screen starts with its content
- the forms ask only for what the API needs

# Scope
[scope]: #scope

In scope:
- the score display, the cards, the header, the list toolbar, the restaurant page, the review form, and the sign up form of the UI
- the texts and the previews of `specs/design-system` for these parts
- the tests

Out of scope:
- the Public API. The score property stays `price`. The removed sorts stay in the API.
- a scale other than 1 to 10
- marker clusters on the map, a landing page, and other items of `docs/reviews/2026-11-13-holistic-review.md`

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Scores

A score shows as a number without dots and without "/ 10".

- A card shows only the overall score, such as "7.2", with the review count or the title next to it.
- The page of a restaurant or a review shows the overall score in large type. "out of 10" follows it one time.
  One line below has the five scores with their labels, such as "Space 6.7" and "Value 5.7".
- The label of the `price` score is "Value". The note about the price score is gone.

## Restaurants list

- One toolbar row has the search, the filters, the sort, the count, the List and Map buttons, and "Add restaurant".
- The heading "Restaurants" is visually hidden. Each screen still has one `h1`.
- The sort has "Newest", "Highest rated", and "Name".
- A restaurant card has no footer with the creator. The full card is the link.

## Restaurant page

- On a wide page, the map is in a column next to the summary. On a narrow page, a "Show map" button opens it.
- The address is the link to OpenStreetMap.
- The text "Other people have reviewed this restaurant. You cannot delete it." is on the edit page.
- The reviews are a flat list with dividers.
- A list with fewer than 10 reviews has no search, no filters, and no sort. A filter or a sort in the URL shows them.
- The review sort has "Newest", "Highest rated", and "Lowest rated".

## Header and theme

- The header has the logo and the account menu. The logo opens the restaurants list.
- A signed in person selects the theme in the account menu. A signed out person selects it in the page footer.

## Forms

- Each score of the review form has one line of help. The meanings of 1 and 10 are below the ends of the button row.
- The photos are above the save button.
- Sign up has no "Confirm password" field. The show password button lets the person check the password.
  The API has no password reset, so a typing error can lose the account. The help text tells the person to check the password.

## Avatars

Only the account menu and the top of the reviewer page show an avatar. A list of one reviewer does not repeat the name on each review.

# Implementation Details
[implementation-details]: #implementation-details

- `RatingDisplay` shows the number. It keeps `role="img"` and the name "Food: 8 out of 10". `showLabel` and `size` select the form.
- `ScoreSummary` in `components/app` shows the large overall score and the line of five scores. The restaurant page and the review page use it.
- `scoreFields` in `lib/reviews.ts` has the label "Value" for `price` and short `lowest` and `highest` words. `scoresOf` and `scoreLegend` are removed.
- `RatingInput` links two elements to the group: the description and the ends. Hidden words make "1 means poor value. 10 means great value." for a screen reader.
- `ReviewCard` is an `article`, not a `Card`. `showCreator` replaces `linkCreator`.
- `ReviewList` shows `FilterBar` when the list has 10 reviews or more, or when the URL has a filter or a sort.
  A restaurant gives the count. A list of one reviewer uses the loaded page. Without a filter, this page has all reviews or it has a next page.
- A URL with a removed sort, such as `?sort=-food`, shows the default sort. `useListQuery` already ignores unknown values.
- `FilterBar` has the `end` slot. The label of the sort is `sr-only`, so the name "Sort by" stays.
- `RestaurantCard` makes the link cover the card with an `after` element. The card has one link, so no other control is below it.
- `RestaurantPage` uses `useMediaQuery` at 1024px. It makes only one `MapView`. Leaflet then never starts in a hidden element.
- The save button of the review form is outside the `form` element. Its `form` attribute joins it to the form.
- `useTheme` has the theme state. `ThemeToggle` and `UserMenu` use it. `AppFooter` shows `ThemeToggle` only without a user.
- `App.vue` is a column with the minimum height of the window. The footer then stays at the bottom of a short page.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
