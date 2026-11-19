---
description: Sort restaurants by a Bayesian average of their reviews. A restaurant with few reviews no longer beats a restaurant with many good reviews. The responses still show the plain average.
status: implemented
---

# Motivation
[motivation]: #motivation

The `overall` sort of `GET /v1/restaurants` used the plain average of the reviews.
One review of 10 put a restaurant above a restaurant with 200 reviews and an average of 9.4.
The "Highest rated" order of the UI then started with restaurants that have one review.

Expected outcome:
- the `overall` sort gives less weight to an average of few reviews
- `rating.overall` and the `overall` filter do not change
- the sort still reads one page of an index

# Scope
[scope]: #scope

In scope:
- the `ranking` column of `restaurant_overall` and the triggers that write it
- the `overall` sort of `listRestaurants`
- the description of the sort in the API spec

Out of scope:
- the `overall` filter. It compares `rating.overall`, the value that the response shows.
- the `overall` sort of the review lists. One review has one value.
- a prior from the average of all reviews. See the alternatives.
- a UI change. The UI does not show the ranking value. See the UI section.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

The ranking of a restaurant is a Bayesian average.
Each restaurant starts with 5 reviews that have an overall of 5.5.
The reviews of users then move the value away from 5.5.

```
ranking = (5 * 5.5 + sum of the overall values of the reviews) / (5 + number of reviews)
```

| Reviews | `rating.overall` | Ranking |
|---|---|---|
| 1 review of 10 | 10 | 6.25 |
| 5 reviews of 10 | 10 | 7.75 |
| 200 reviews of 9.4 | 9.4 | 9.30 |
| 1 review of 1 | 1 | 4.75 |
| 200 reviews of 2 | 2 | 2.09 |

`sort=-overall` puts the restaurant with 200 reviews of 9.4 first.
`sort=overall` puts the restaurant with 200 reviews of 2 first.
A restaurant without reviews is last in both directions, as before.

The order can differ from the order of the `rating.overall` values on the page.
The spec tells this in the description of the `sort` parameter.
The ranking of a restaurant with few reviews is near 5.5. It is not near the middle of the list.

## UI

The "Highest rated" sort of the restaurant list uses `sort=-overall`.
The UI does not tell users that this order also uses the number of reviews.
It has no help text, no tooltip, and no other label for this sort.

Each restaurant card shows the average score and the number of reviews.
A user can see that a restaurant with 1 review of 10 is below a restaurant with 200 reviews of 9.4.
Many review sites use a "Highest rated" label for an order of this type.

The recipe list has the same sort and the same cards. See decision 00047.

# Implementation Details
[implementation-details]: #implementation-details

## Table

The migration `1794268810_add_overall_ranking.go` changes `restaurant_overall` of decision 00012.

```sql
CREATE TABLE restaurant_overall (
  restaurant TEXT PRIMARY KEY NOT NULL,
  overall REAL,
  unrated INTEGER AS (overall IS NULL),
  rated INTEGER AS (overall IS NOT NULL),
  ranking REAL,
  ranking_or_0 REAL AS (COALESCE(ranking, 0))
) WITHOUT ROWID;
CREATE INDEX idx_restaurant_overall_asc ON restaurant_overall (unrated, ranking_or_0, restaurant);
CREATE INDEX idx_restaurant_overall_desc ON restaurant_overall (rated, ranking_or_0, restaurant);
CREATE INDEX idx_restaurant_overall_value ON restaurant_overall (overall);
```

`ranking` is NULL for a restaurant without reviews, the same as `overall`.
`ranking_or_0` replaces `overall_or_0` in the sort keys and in the two sort indexes.
The index names do not change.
`idx_restaurant_overall_value` and the `overall` filter do not change.

The triggers of decision 00012 write `overall` and `ranking` in one statement.
They compute the two values from the reviews of one restaurant.
The migration computes `ranking` for the rows that exist.

The migration runs one time, when the app starts.
It writes the table again for each column change. It then reads the reviews of each restaurant through `idx_reviews_restaurant_creator`.
The app serves no request before the migration ends. Its time grows with the number of restaurants and reviews.

`ranking` has no rounding.
Two restaurants with different reviews then seldom have equal values.
Equal values sort by the restaurant id, as before.

## Prior

The prior is fixed: 5 reviews of 5.5.
5.5 is the middle of the score range of 1 to 10.
The two values are constants in the SQL of the migration.
A change of them needs a new migration that computes all rows again.

A weight of 5 is small. A restaurant with 20 reviews has a ranking near its average.
A site with many reviews for each restaurant can need a larger weight.

## Alternatives

- **A help text or a tooltip on the "Highest rated" sort.** A tooltip does not work on touch screens.
  A help text adds a line to the filter bar of each list. The review count on each card gives the same fact.
- **A different label, such as "Best and most reviewed".** It is longer than the other sort labels.
  The review sorts also have "Highest rated" for the plain `overall` value of one review.

- **The average of all reviews as the prior.** This is the usual form of the Bayesian average.
  Each new review changes the average of all reviews. The ranking of each restaurant then changes.
  A trigger must then write all rows of `restaurant_overall` on each review write, or a job must compute them later.
  The fixed prior keeps the cost of a review write at one row.
- **The lower bound of a confidence interval.** It ranks a restaurant with few reviews below a bad restaurant with many reviews.
  The ascending sort needs a second column with the upper bound.
- **A minimum number of reviews.** It hides new restaurants from the sort.

## Cursor

The cursor fingerprint of a request with the `overall` sort has the version 3.
The API rejects a cursor from before this change with 400.
The second key of an old cursor is a rounded average, not a ranking.

## Corner cases

- A prior of 5.5 is below the average of most review sites. A new restaurant with one review of 8 has a ranking of 5.92.
  It is below a restaurant with many reviews of 6. More reviews move it up.
- Sort fields after `overall`, such as `name` in `overall,name`, do not change the order. This is the same as in decision 00012.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
