---
description: Store the overall average and the tags of each restaurant in indexed tables. The overall sort, the overall filter, and the tag filters then do not read all reviews or all restaurants.
status: implemented
---

# Motivation
[motivation]: #motivation

`GET /v1/restaurants` with the `overall` sort or an `overall` filter grouped all rows of `reviews` on each request.
The tag filters read the `tags` JSON of each restaurant with `json_each`.
SQLite cannot index the items of a JSON array.
The cost of these requests grew with all reviews and all restaurants, not with the page size.

Expected outcome:
- the `overall` sort reads one page of an index. A cursor starts the read at its position.
- the `overall` filter reads a range of an index
- a tag filter reads only the restaurants with the tag
- the responses do not change

# Scope
[scope]: #scope

In scope:
- the `restaurant_overall` and `restaurant_tags` tables and a migration that makes and fills them
- the SQLite triggers on `restaurants` and `reviews` that keep the tables equal to the records
- the overall sort, the overall filter, and the tag filters of `listRestaurants`
- a test that checks the query plans with `EXPLAIN QUERY PLAN`

Out of scope:
- the other averages of `rating`. No sort or filter uses them. They read only the reviews of the restaurants on the page with an index.
- the full-text search. Decision 00011 and the `searchindex` package cover it.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

Each restaurant has one row in `restaurant_overall`.
Its `overall` is the rounded overall average of the reviews of the restaurant.
It is NULL for a restaurant without reviews.

Each string tag of a restaurant has one row in `restaurant_tags`.
`tags[has]=italian` then reads the `italian` rows of the primary key.

A save or a delete of a restaurant or a review changes the rows in the same statement.
All writes change them: the Public API, the PocketBase dashboard, cascade deletes, and plain SQL.
`rating.overall` of the responses is the stored value.

# Implementation Details
[implementation-details]: #implementation-details

## Tables

This is the schema of this decision.
Decision 00029 replaces `overall_or_0` with `ranking` and `ranking_or_0` in the table and in the two sort indexes. It has the current schema.

```sql
CREATE TABLE restaurant_overall (
  restaurant TEXT PRIMARY KEY NOT NULL,
  overall REAL,
  unrated INTEGER AS (overall IS NULL),
  rated INTEGER AS (overall IS NOT NULL),
  overall_or_0 REAL AS (COALESCE(overall, 0))
) WITHOUT ROWID;
CREATE INDEX idx_restaurant_overall_asc ON restaurant_overall (unrated, overall_or_0, restaurant);
CREATE INDEX idx_restaurant_overall_desc ON restaurant_overall (rated, overall_or_0, restaurant);
CREATE INDEX idx_restaurant_overall_value ON restaurant_overall (overall);

CREATE TABLE restaurant_tags (
  tag TEXT NOT NULL,
  restaurant TEXT NOT NULL,
  PRIMARY KEY (tag, restaurant)
) WITHOUT ROWID;
CREATE INDEX idx_restaurant_tags_restaurant ON restaurant_tags (restaurant);
```

The tables are not PocketBase collections.
The PocketBase API and the dashboard do not show them.
A save of a restaurant record then cannot write an old average over a new one.

## Sort

Decision 00029 changes the second sort key from `overall_or_0` to `ranking_or_0`, a Bayesian average.

The `overall` sort puts restaurants without reviews last in both directions.
`overall` sorts by `unrated, overall_or_0, restaurant`.
`-overall` sorts by `rated DESC, overall_or_0 DESC, restaurant DESC`.
All keys of a sort have one direction.
The keyset condition of the cursor is then one row value comparison.

SQLite uses an index on columns for a row value range.
It does not use an index on expressions for it.
The sort keys are then generated columns.

The `restaurant` key is unique.
`parseSort` adds no key after a unique key, and it adds no `id` key.
All keys of `overall` and `-overall` are then columns of one index, and SQLite sorts no rows.
Sort fields after `overall`, such as `name` in `overall,name`, cannot change the order.

The query joins `restaurant_overall` with an inner join.
SQLite can then read `restaurant_overall` first, in the order of the index.
The triggers give each restaurant a row. The inner join then keeps all restaurants.

The cursor fingerprint of a request with the `overall` sort has a version.
The API then rejects a cursor from before this change with 400.
Without it, a `-overall` cursor with the old keys gives a wrong next page.

## Filters

`overall[gte]=4` reads `idx_restaurant_overall_value`.
A restaurant without reviews has NULL and matches no filter.

`tags[has_any]=a&tags[has_any]=b` adds
`restaurants.id IN (SELECT restaurant FROM restaurant_tags WHERE tag IN ('a', 'b'))`.
`tags[has]` and each `tags[has_all]` value add one such condition.

## Triggers

The migration makes the triggers. The table shows what each one does.

| Record change | Index change |
|---|---|
| a restaurant is made | insert an `overall` row, insert its tags |
| the tags of a restaurant change | replace its tags |
| a restaurant is deleted | delete its `overall` row and its tags |
| a review is made or deleted | compute the `overall` row of its restaurant again |
| the scores or the restaurant of a review change | compute the rows of the new and the old restaurant again |

A change of only the title or the feedback of a review starts no trigger.
A change of only the name or the description of a restaurant starts no trigger.

The computation reads only the reviews of one restaurant through `idx_reviews_restaurant_creator`.
It does not add to or subtract from the stored value.
The stored value then cannot drift from the reviews.

The triggers use the old and new rows of SQLite.
They do not depend on `Record.Original` or on a Go hook registration.

The tags keep only the strings of a JSON array.
The API accepts only arrays of strings. The dashboard accepts all JSON.
Before this change, a JSON string such as `"italian"` matched `tags[has]=italian`. It does not match now.

### Triggers and later migrations

PocketBase changes the table of a collection with `ALTER TABLE`.
SQLite changes the triggers when a column gets a new name.
SQLite refuses to drop a column that a trigger uses.
A migration that drops a score column, `reviews.restaurant`, or `restaurants.tags` must first drop the triggers.
It then makes the triggers again without the column.
PocketBase also drops and adds the column of a relation field when its maximum changes from one to more.
This also applies to `reviews.restaurant`.

## Corner cases

- A restaurant without reviews has an `overall` row with NULL. The inner join then keeps it.
- A restaurant that plain SQL inserts also gets a row.
- A deleted restaurant has no row. A later review delete in the same cascade changes no row.
- A tag that occurs two times in a restaurant has one row.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
