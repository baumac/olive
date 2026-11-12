---
description: Give each review an overall score that the API computes from its five scores, and give each reviewer a public page with their reviews.
status: implemented
---

# Motivation
[motivation]: #motivation

A review card shows five scores and no single value.
A reader must add the scores in their head to compare two reviews.
A restaurant already has `rating.overall`, so a review without one is not consistent.

A person can list only their own reviews on the my reviews page.
A reader who trusts a reviewer cannot see the other reviews of that person.

Expected outcome:
- each review has an `overall` value. The review card and the review page show it first.
- the review lists can sort and filter by `overall`
- each reviewer has a page at `/reviewers/{id}`. The name on a review card links to it.
- the page shows only public data: the name, the number of reviews, and the reviews

# Scope
[scope]: #scope

In scope:
- the `overall` property of `Review`, the `overall` review sort, and the `overall[...]` review filter
- the `getReviewer` and `listReviewerReviews` operations and the `Reviewer` schema
- the reviewer page, the reviewer link, and the overall row and filter of the UI
- the link preview of the reviewer page. See decision 00013.
- the tests

Out of scope:
- an overall score that a reviewer enters. See the guide-level explanation.
- a stored `overall` column or an index on it. The review lists filter on the scores with a scan today.
- a profile that the user edits, such as a bio or an avatar
- a list of all reviewers
- the restaurants that the user created. A reviewer page shows only the reviews of the user.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Overall score

The API computes `overall`. It is the average of the five scores:

```json
{
  "id": "r1b8c5e2g9i6k3m",
  "overall": 7.6,
  "space": 7, "service": 9, "food": 8, "price": 6, "cleanliness": 8,
  "title": "Great food, a bit expensive"
}
```

`(7 + 9 + 8 + 6 + 8) / 5` is `7.6`. Each value has at most one decimal, so no rounding occurs.

The value is derived, not entered:
- `rating.overall` of a restaurant is the stored value of decision 00012.
  It is the rounded average of the `overall` values of its reviews.
  A derived value keeps the review and the restaurant consistent.
- an entered value can disagree with the five scores. The reader then does not know which to trust.
- a derived value needs no migration and no new form field

A request body with `overall` gets `400 Bad Request`. The property is not in `ReviewCreate` or `ReviewPatch`.

`GET /v1/reviews?sort=-overall&overall[gte]=7.5` lists the best reviews first.
`listReviews`, `listAllReviews`, and `listReviewerReviews` have the same sort and filter.

## Reviewer page

`GET /v1/reviewers/{reviewerId}` gives the public profile of a user:

```json
{ "id": "u5n1c8r3t6y0b2e", "name": "Ana", "review_count": 12 }
```

It has the fields of `Creator` and the review count. It has no email address and no timestamps.

`GET /v1/reviewers/{reviewerId}/reviews` gives the reviews of the user.
It has the filters, sorts, and pages of `listAllReviews` without `creator[eq]` and `creator[in]`.
An unknown user gets `404`. `GET /v1/reviews?creator[eq]=` gives an empty list for the same user.

A deleted user has no page. Both operations give `404` for the user.
The reviews of the user are deleted with the user, so no review card links to the page.

The UI page `/reviewers/{id}` shows the name, the review count, and the review list with its filter bar.
The name on each review card and on the review page links to it.
A list of the reviews of one reviewer shows the name without a link.
A malformed ID gets `400` from the API. The UI then shows the not found page.
A shared link of the page has the name of the reviewer in its preview.
The my reviews page uses `listReviewerReviews` for the signed in user.

# Implementation Details
[implementation-details]: #implementation-details

Spec:
- `Review` has the required `overall`, an `AverageScore`
- `ReviewSort` has `overall`. `ReviewOverallFilter` compares it with an `AverageRange`.
- the new tag `reviewers` has `getReviewer` and `listReviewerReviews`
- the SDKs are generated again. See decision 00003. The TypeScript SDK has a new `ReviewersApi`.

Server:
- `toReview` divides the integer sum of the scores by 5 as a `float64`
- the SQL expression `reviewOverall` divides the same sum by `5.0`. SQLite also uses a double.
  Both give the double nearest to the decimal value, so `overall[eq]=7.6` matches the response value `7.6`.
- `ListReviewerReviews` sets `creator[eq]` to the path user and calls `reviewPage`
- it looks up the user only for an empty page. A page with a review proves that the user exists.
  The lookup runs after the parameter checks, so `400` comes before `404`.
- `overallDetail` rejects a NaN overall filter for the restaurant and the review lists
- its cursor fingerprint has the `reviewer-reviews` prefix and the user ID. A cursor of one list does not work in another.
- `GetReviewer` counts the reviews with `idx_reviews_creator`
- the `webui` package gives `/reviewers/{id}` its own page-meta block with `og:type` `profile`.
  An unknown user gets `404`. The block has no email address.

Cursors:
- the new `overall` parameter has `omitempty`. The fingerprints of earlier requests do not change.
- a cursor of an `overall` sort has the overall value as its first key

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
