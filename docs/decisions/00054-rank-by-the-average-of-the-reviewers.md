---
description: Each reviewer counts one time in the rating of a restaurant or a recipe. The overall sort uses the rating that the card shows, then the number of reviewers. The Bayesian ranking of decision 00029 is gone. This is version 8.1.0 of the spec.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00029 sorts restaurants and recipes by a Bayesian average with a prior of 5 reviews of 5.5.
Its examples have 200 reviews for each restaurant. The users of this app are a group of friends. See decision 00000.
A restaurant of the group has 1 to 5 reviews. The ranking then mostly counts the reviews:

| Reviews | `rating.overall` | Ranking of decision 00029 |
|---|---|---|
| 2 reviews of 10 | 10 | 6.79 |
| 3 reviews of 9 | 9 | 6.81 |

Decision 00035 makes "Highest rated" the default sort of the restaurant list.
The UI does not tell the user that this order also uses the number of reviews.
A user then sees a 9.0 above a 10.0 and does not know why.
A smaller prior does not remove this. With 1 review of 5.5, 3 reviews of 9 get 8.13 and 1 review of 10 gets 7.75.

Decision 00050 allows a review for each visit or cook.
`rating` and the ranking count each review. A user who visits a restaurant often then moves its value more.
Four visits of Ana with 10 and one visit of Ben with 6 give 9.2. Ana and Ben do not agree on 9.2.

Expected outcome:
- the order of "Highest rated" is the order of the values on the cards
- each reviewer counts one time in `rating`, in the `overall` filter, and in the `overall` sort
- `review_count` still counts each visit or cook
- the sort still reads one page of an index

# Scope
[scope]: #scope

In scope:
- `rating` of restaurants and recipes
- the `overall` sort of `listRestaurants` and `listRecipes`, and its cursor
- the `restaurant_overall` and `recipe_overall` tables, their triggers, and their sort indexes
- the descriptions of the spec, version 8.1.0
- the tests

Out of scope:
- `review_count` and `own_review_count`. Each review counts, as in decision 00050.
- the reviewer averages of `Reviewer`. They have the reviews of one user only.
- the `overall` value of one review and the `overall` sort of the review lists
- the order of the photos of a gallery. The photos of the review with the highest overall come first, as before.
- a `reviewer_count` property in the responses. See the corner cases.
- a minimum number of reviews. See the alternatives.
- a UI change. The cards already show `rating.overall` and the number of reviews.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Each reviewer counts one time

The rating first gets the average of each reviewer. It then gets the average of these values.

Ana visits Luigi's four times. Each of her reviews has the overall 10. Ben visits one time and gives 6.

```
Ana:    (10 + 10 + 10 + 10) / 4 = 10
Ben:    6 / 1                   = 6
rating.overall = (10 + 6) / 2   = 8.0
review_count   = 5
```

Decision 00050 gave 46 / 5 = 9.2.
Each score of `rating` uses the same rule. `rating.overall` is still the average of the four score averages.
The API rounds each value to one decimal, as before.

## The overall sort

`sort=-overall` sorts by `rating.overall`, the value that the card shows.
Two equal values sort by the number of reviewers. More reviewers come first.
Equal values with equal reviewers sort by the id, as before.

| Restaurant | Reviews | `rating.overall` | Reviewers | Place in `-overall` |
|---|---|---|---|---|
| A | Ana 10, Ben 10 | 10.0 | 2 | 1 |
| B | Ana 10 | 10.0 | 1 | 2 |
| C | Ana 9, Ben 9, Cleo 9 | 9.0 | 3 | 3 |
| D | Ana 10 four times, Ben 6 | 8.0 | 2 | 4 |
| E | none | | 0 | last |

`sort=overall` is the reverse order. Restaurants and recipes without reviews are last in both directions, as before.
The `overall` filter compares `rating.overall`. A filter and the sort then always agree.

One review of 10 is above many reviews of 9.
The group accepts this. The card shows the number of reviews next to the value.

# Implementation Details
[implementation-details]: #implementation-details

## Tables

The migration `1794268823_rank_by_the_average_of_the_reviewers.go` changes `restaurant_overall` and `recipe_overall`:

```sql
CREATE TABLE restaurant_overall (
  restaurant TEXT PRIMARY KEY NOT NULL,
  overall REAL,
  unrated INTEGER AS (overall IS NULL),
  rated INTEGER AS (overall IS NOT NULL),
  reviewers INTEGER NOT NULL DEFAULT 0,
  review_count INTEGER NOT NULL DEFAULT 0,
  space REAL,
  service REAL,
  food REAL,
  price REAL,
  overall_or_0 REAL AS (COALESCE(overall, 0))
) WITHOUT ROWID;
CREATE INDEX idx_restaurant_overall_asc ON restaurant_overall (unrated, overall_or_0, reviewers, restaurant);
CREATE INDEX idx_restaurant_overall_desc ON restaurant_overall (rated, overall_or_0, reviewers, restaurant);
CREATE INDEX idx_restaurant_overall_value ON restaurant_overall (overall);
```

`recipe_overall` has the same columns with `recipe` and the scores `taste`, `ease`, `instructions`, and `value`.
The score columns have the rounded averages of `rating`. `review_count` counts each review.
The migration drops `ranking` and `ranking_or_0`. It drops the triggers and the two sort indexes first. SQLite refuses the drop of a column that they use.
The index names do not change.
The migration computes the values again for each row.
A later migration that drops a score column of the reviews also drops its column of the table.

## Triggers

The triggers of decision 00012 write all values of a row in one statement:

```sql
SELECT ROUND(AVG(reviewer_overall), 1), COUNT(*), COALESCE(SUM(reviewer_reviews), 0),
  ROUND(AVG(space), 1), ROUND(AVG(service), 1), ROUND(AVG(food), 1), ROUND(AVG(price), 1)
FROM (
  SELECT SUM(space + service + food + price) * 1.0 / (4 * COUNT(*)) AS reviewer_overall,
    COUNT(*) AS reviewer_reviews, AVG(space) AS space, AVG(service) AS service, AVG(food) AS food, AVG(price) AS price
  FROM reviews WHERE reviews.restaurant = ? GROUP BY creator
)
```

The inner query reads the reviews of one restaurant through `idx_reviews_restaurant_creator`. A Go test checks this plan.
The rule of the averages is in this one statement. The API reads the stored values.
A review write still writes one row of the table.
The update trigger also watches `creator`. A change of the creator in the dashboard moves a review to another reviewer.
A restaurant without reviews gets NULL values and the counts 0.
`overallKind` of the migration has the names of each kind. One code makes the triggers of both tables.
The down migration uses `makeOverallSort` of decision 00029 and `makeRecipeRankingSort` of decision 00047.

## Public API

- `overallSort` has the keys `rated` or `unrated`, `overall_or_0`, `reviewers`, and the id. All keys have one direction.
- `ratingsQuery` reads the stored rating of each parent of a page by the primary key of the table.
  It reads no reviews. `review_count` of the responses is the stored `review_count`.
- `overallSortVersion` is 4. The fingerprint of each `overall` sort of `listRestaurants` and `listRecipes` has it.
  The API rejects a cursor from before this change with 400. The recipe cursors had no version before.

## Spec

- The descriptions of `RestaurantSort` and `RecipeSort` tell the `overall` sort and its order of equal values.
- The descriptions of `RestaurantRating` and `RecipeRating` tell that each reviewer counts one time.
- The version is 8.1.0. No schema changes. The values and the order change.
- `make generate` makes the SDKs again. Only the descriptions change.

## Down migration

The down migration adds `ranking` and `ranking_or_0` again. It computes them with the prior of decision 00029.
It computes `overall` again with each review counted.

## Alternatives

- **A smaller prior, such as 1 review of 5.5.** Some 9.0 restaurants still come before 10.0 restaurants. The UI still does not tell why.
- **A minimum number of reviewers.** A restaurant below the minimum sorts after all others.
  Most restaurants of the group have one or two reviewers. The minimum then hides most of them.
- **The latest review of each reviewer.** It loses the older visits. Decision 00050 keeps each visit because the visits differ.
- **The per-reviewer average only in the sort.** The card then shows another value than the sort uses. This is the problem of decision 00029.
- **A help text on the "Highest rated" sort.** Decision 00029 has the reasons against it.

## Corner cases

- Equal values with different reviewers sort by the number of reviewers. The card shows the number of reviews, not of reviewers.
  The two numbers differ only for repeated visits. The order of equal values then can look random. A `reviewer_count` property can fix this later.
- `sort=overall` puts fewer reviewers first among equal values. It is the reverse of `-overall`. One index serves both directions.
- A deleted user takes the reviews with a cascade delete. The triggers then compute the values without this reviewer.
- Sort fields after `overall`, such as `name` in `overall,name`, do not change the order. The id is the last key.
- The sort key is the rounded `overall`. Reviewer averages of 8.04 and 7.96 both give 8.0.
  The number of reviewers then decides the order, not the hidden decimals. The card shows 8.0 for both. The order then agrees with the cards.

## Tests

- Go tests:
  - the triggers count each reviewer one time, for both kinds
  - a change of the creator moves a review to the other reviewer
  - the migration computes the old rows again, and the down migration brings the ranking back
  - the `overall` sort puts equal values with more reviewers first, and the cursor follows the four keys
  - `rating` and `review_count` of a restaurant with repeated visits
  - the cursors of versions 2 and 3, and an old recipe cursor, get 400
  - the triggers store the score averages and the review count
  - the plan of the trigger query uses the creator indexes
  - the plan of the rating query uses the primary key of the overall table
- End-to-end tests:
  - a restaurant with three visits of one user and one visit of another user has the average of the two users. The list shows it in this order.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
