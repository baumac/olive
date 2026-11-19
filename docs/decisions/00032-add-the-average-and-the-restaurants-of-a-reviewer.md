---
description: Add the average score of a reviewer and the number of restaurants they added to the reviewer profile. The reviewer page shows the average and lists the restaurants.
status: implemented
---

# Motivation
[motivation]: #motivation

The reviewer page of decision 00014 shows the name, the number of reviews, and the reviews.
A reader cannot see if the reviewer gives high or low scores without reading each review.
A reader also cannot see the restaurants that the reviewer added. Decision 00014 left them out.

Expected outcome:
- the profile of a reviewer has the average of the `overall` values of their reviews
- the profile has the number of restaurants that the user created
- the reviewer page shows the average and lists these restaurants

# Scope
[scope]: #scope

In scope:
- the `average_overall` and `restaurant_count` properties of `Reviewer`
- the average and the "Restaurants added" section of the reviewer page
- the tests

Out of scope:
- a new list operation. `listRestaurants` already has `creator[eq]`.
- the average of each of the five scores of a reviewer
- a sort or a filter of reviewers. The API has no list of reviewers.
- the my reviews page. It shows only the review list.
- the link preview and the structured data of the reviewer page. See decisions 00013 and 00021. Decision 00049 removes the structured data.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

`GET /v1/reviewers/{reviewerId}` gives two more values:

```json
{ "id": "u5n1c8r3t6y0b2e", "name": "Ana", "review_count": 12, "average_overall": 7.4, "restaurant_count": 3 }
```

`average_overall` is the average of the `overall` values of the reviews of the user. The API rounds it to one decimal.
It is `null` for a user without reviews, the same as `rating` of a restaurant without reviews.

`restaurant_count` is the number of restaurants with the user as their creator.
`GET /v1/restaurants?creator[eq]={reviewerId}` lists them.

The reviewer page shows the average below the name:

```
Ana
6.2  average of 2 reviews · 1 restaurant added
```

A reviewer without reviews has "No reviews yet" and no average.
A reviewer with restaurants has a "Restaurants added" section above the reviews.
The section has the restaurant cards of the restaurants page, the newest first. It shows 3 cards and a "Load more restaurants" button.
On a wide screen, the 3 cards are one row.
The review list has pages of 20 reviews. Below it, the section of a reviewer with many reviews is far down the page.
Above it, a "Load more" of the reviews does not move the section.
A reviewer without restaurants has no section.

# Implementation Details
[implementation-details]: #implementation-details

Spec:
- `Reviewer` has the required `average_overall`, an `AverageScore` or `null`
- `Reviewer` has the required `restaurant_count`, an integer
- the SDKs are generated again. See decision 00003.

Server:
- `reviewerStats` reads the review count, the average, and the restaurant count in one statement.
  The count and the average then come from the same reviews.
- the average is `ROUND(AVG(overall), 1)` of the generated `overall` column of decision 00031
- the statement finds the reviews through `idx_reviews_creator_overall` and counts the restaurants in `idx_restaurants_creator_created`.
  SQLite computes a virtual column from the row. It then reads the row of each review of the user. A stored value like `restaurant_overall` is not necessary for the number of reviews of one user.
- the migration `1794268812_index_restaurants_creator_created.go` adds the index on `(creator, created, id)`.
  `listRestaurants` with `creator[eq]` and the sort `-created` reads one page of it.
  The index replaces `idx_restaurants_creator`, because it starts with `creator`.

UI:
- `ReviewerPage` shows the average with `RatingDisplay` and the label "Average score"
- `ReviewerRestaurants` loads `listRestaurants` with `creator[eq]`, the sort `-created`, and a limit of 3
- the page order is the header, the lists of decision 00033, the `KindSwitch` of decision 00047, "Restaurants added", and the reviews
- decision 00047 changes `ReviewerRestaurants` to `ReviewerAdded`. The switch selects "Restaurants added" or "Recipes added", and the reviews of the same kind.
  The section of the other kind then does not push the reviews down. The header has the counts of both kinds.
- the cards are in 3 columns from the `lg` width
- a change of the query keeps the scroll position. A change of a filter, the sort, or the kind then does not move the page to the top.
- the page shows the section only for a `restaurant_count` above 0. A reviewer without restaurants causes no list request.
- `RestaurantCard` has a `heading` property. The cards of the section use `h3` below the `h2` of the section.
- the restaurant list has no search, filter, or sort. The filters of the review list are in the URL of the page.
- `PagedList` has an `unfiltered` property. A failed load of the section then has a "Try again" button and no text about filters.

Corner cases:
- A user deletes a restaurant while the page is open. The count in the header is then one more than the list until a reload.
- The list can be empty with a count above 0, after a delete between the two requests. The section then tells that the restaurants are not available.
- The scores of the API are from 1 to 10. A score outside this range from SQL gives an average that the response validation rejects, the same as for `rating` of a restaurant.
- A restaurant keeps its creator when the creator has no review of it. It is in the list.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
