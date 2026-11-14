---
description: Fix the UI items of the holistic review of 2026-11-13. Show the overall score in the review form, the scores on a review card, map clusters, a directions link, old sort links, and an intro for visitors.
status: implemented
---

# Motivation
[motivation]: #motivation

The holistic review of 2026-11-13 lists UI problems that decision 00019 did not fix.

- The review form does not show the overall score that the five scores give.
- A review card shows only the overall score.
- Pins that are near each other overlap on the map of the restaurants. The limit of 100 restaurants shows only in the count.
- The address opens OpenStreetMap, but no link starts a route on a phone.
- A link with a sort that decision 00019 removed shows the default sort.
- A signed out visitor gets no text about the site.

Expected outcome:
- a person sees the result of the scores before the save
- a reader sees the five scores of a review without a new page
- each restaurant on the map has a pin that a person can select
- old links keep their order

# Scope
[scope]: #scope

In scope:
- the review form, the review card, the map of the restaurants, the restaurant page, and the restaurants list of the UI
- the tests

Out of scope:
- the Public API
- the number of score buttons in the review form. Ten buttons per score stay, because one click selects a score.
- visible navigation links and a visible list heading. The UI has one top-level screen.
- an i18n layer. It needs a decision about the languages.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Review form

A line below the five scores shows "Overall" and the average, such as "7.6". Before the fifth score, it shows a short help text.

## Review card

Each card has a "Show scores" button after the date. It shows one line with the five scores. "Hide scores" removes the line.

## Map of the restaurants

- Pins that are less than 44 pixels apart join into one round pin with a count.
- A click on this pin zooms the map until its pins are apart.
- At the last zoom level, pins at one point show next to each other.
- The selected pin stays out of a cluster.
- A line below the map tells when the area has more than 100 restaurants.

## Restaurant page

A "Directions" link is next to the address.

- An Android device gets a `geo:` link. It opens the map app.
- An Apple device gets a link to Apple Maps.
- Other devices get the directions page of OpenStreetMap.

## Sorts

A link with an old sort, such as `?sort=-food`, keeps its order. The sort control then shows this sort as one more option. The option goes away when the person selects a different sort.

## Restaurants list

A signed out visitor sees two sentences above the toolbar. They tell what the scores are and link to the sign up page. A filter or the map view hides them.

# Implementation Details
[implementation-details]: #implementation-details

- `overallOf` in `lib/reviews.ts` makes the average like the API. The line in the form has `role="status"`.
- `ReviewCard` keeps the open state of its scores. The button has `aria-expanded` and the title of the review in its name.
- `MapView` has the `cluster` and `clusterLabel` props. Only the map of the restaurants sets them.
  It makes the groups from the pixel distance at the zoom level and draws again after each zoom. No library is added.
- `directionsLink` in `lib/places.ts` selects the link from the user agent. An iPad can send the user agent of a Mac, so a Mac also gets Apple Maps.
- `useListQuery` has the `oldSortOptions` parameter. `oldReviewSortOptions` and `oldRestaurantSortOptions` have the sorts from before decision 00019.
- `RestaurantsPage` shows the intro when no user is signed in, no filter is active, and the list view is open.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
