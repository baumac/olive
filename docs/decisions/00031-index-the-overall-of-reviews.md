---
description: Add a generated `overall` column with indexes to the `reviews` table. The `overall` sort and filter of the review lists then read an index.
status: implemented
---

# Motivation
[motivation]: #motivation

The `overall` sort and filter of the review lists used an expression on the five score columns.
SQLite computed the value for each review of the restaurant or the reviewer. It then sorted all of them for one page.
SQLite does not use an index on an expression for the row value range of a cursor. See decision 00012.

Expected outcome:
- the `overall` sort of a review list reads one page of an index
- the `overall` filter of a review list is a range of an index
- the responses and the cursors do not change

# Scope
[scope]: #scope

In scope:
- the `overall` column of the `reviews` table and its indexes
- the `overall` sort and filter of `listReviews`, `listReviewerReviews`, and `listAllReviews`

Out of scope:
- the API spec. The values and the order do not change.
- an index for the gallery order of decision 00030. The gallery reads the photos of one restaurant. It uses the column, but it still sorts the rows.
- an `overall` sort with other filters, such as `creator[in]` or a score range. SQLite then selects the index from its statistics.
- the `overall` filter with a different sort. It does not use the new indexes. See the filter section.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

The `reviews` table has an `overall` column. SQLite computes it from the five scores of the row.
The list queries use the column where they used the expression before.

| Request | Index |
|---|---|
| `GET /v1/restaurants/{id}/reviews?sort=-overall` | `idx_reviews_restaurant_overall` |
| `GET /v1/reviewers/{id}/reviews?sort=-overall` | `idx_reviews_creator_overall` |
| `GET /v1/reviews?sort=-overall` | `idx_reviews_overall` |
| the same requests with `overall[gte]=8` | the same index, as a range |
| `GET /v1/restaurants/{id}/reviews?overall[gte]=8` | `idx_reviews_restaurant_created`, as before |

A request with a cursor starts at the cursor position in the index.

# Implementation Details
[implementation-details]: #implementation-details

## Column

The migration `1794268811_add_review_overall.go` adds the column and the indexes.

```sql
ALTER TABLE reviews ADD COLUMN overall REAL AS ((space + service + food + price + cleanliness) / 5.0);
CREATE INDEX idx_reviews_restaurant_overall ON reviews (restaurant, overall, id);
CREATE INDEX idx_reviews_creator_overall ON reviews (creator, overall, id);
CREATE INDEX idx_reviews_overall ON reviews (overall, id);
DROP INDEX idx_reviews_creator;
```

The column is virtual. Only the indexes store its value.
The expression is the expression that the queries used before. It divides in a double, the same as `toReview`.
The filter then compares the value that the response shows.

Each index ends with `id`, the last sort key of `parseSort`.
The row value comparison of the cursor is then a range of the index.

The migration removes `idx_reviews_creator` of decision 00014 from the collection.
`idx_reviews_creator_overall` starts with `creator`. It serves the count of `GetReviewer` and the other queries of the old index.

The expression is also in `toReview`. The migrations cannot use the Go code of the API.
`TestOverallColumnEqualsResponse` fails when the column and the response differ.

## Filter with a different sort

The `overall` filter uses the column as `+reviews.overall` when the sort has no `overall`.
SQLite does not use an index for a term with a unary plus.
Without it, SQLite reads the range of an `overall` index and then sorts all matches.
A request with `overall[gte]=2` and the default `-created` sort then reads almost all reviews of the restaurant for one page.
With it, SQLite reads the index of the sort and stops after one page, as before this decision.

## PocketBase

The column is not a field of the `reviews` collection.
PocketBase writes only the fields of the collection. It does not write the column.
The dashboard and the PocketBase API do not show it.
PocketBase changes the table with `ALTER TABLE` when the fields change. The column and its indexes stay.
The indexes are not in the index list of the collection. PocketBase does not drop them.

SQLite refuses to drop a column that a generated column uses.
A migration that drops a score column must first drop the indexes and the `overall` column.
A migration that adds a score to the overall must make the column again.

A superuser can change the fields of the collection in the dashboard.
`guardReviewOverall` is a validation hook of the `reviews` collection. It rejects a field with the name `overall` and the removal of a score field.
The dashboard then shows a message with the cause. Without the hook, it shows the SQLite error.
The hook does nothing when the table has no generated `overall` column. The down migrations then still work.

## Alternatives

- **A table like `restaurant_overall` with triggers.** It needs a copy of `restaurant` and `creator` for the indexes, three triggers, and a join in each query.
  The value of one review needs only the columns of its row. A generated column has no state that can differ from the row.
- **A PocketBase number field that a record hook writes.** A write from the dashboard or from SQL can then leave a wrong value.
- **Indexes on the expression.** SQLite uses them for the sort and the filter. It does not use them for the row value range of a cursor.

## Cursor

The sort keys have the same values as before. The cursor fingerprint does not change.
A cursor from before this change is still valid.

## Corner cases

- A write from the dashboard or from SQL changes the scores of a row. The column then has the new value.
- The table has two more indexes than before. Each review write is then slower. A review write is rare in comparison with a list request.
- The query plan tests compare only the index name and a short part of the range. The full plan text can change with the SQLite version.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- None.
