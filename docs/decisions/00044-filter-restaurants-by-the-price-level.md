---
description: Add the `price_level` filter to `listRestaurants`. The restaurants page selects a range of price levels.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00038 added the `price_level` of restaurants. No operation could filter by it.
A person who wants an inexpensive dinner had to open each restaurant.

# Scope
[scope]: #scope

In scope:
- the `price_level` query parameter of `listRestaurants`
- the price level filter of the restaurants page

Out of scope:
- a sort by the price level
- a filter for the restaurants without a price level
- the `price_level` filter on `listListRestaurants`. That operation has no filters. See the backlog.
- an index of the `price_level` column

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

`price_level` is a `deepObject` parameter, like `overall` and `created`.
Its keys are `lt`, `lte`, `eq`, `gte`, and `gt`. Each value is a `PriceLevel` from 1 to 4. All comparisons must match.

| Query | Restaurants |
|---|---|
| `price_level[lte]=2` | inexpensive and moderate |
| `price_level[eq]=4` | very expensive |
| `price_level[gte]=2&price_level[lte]=3` | moderate and expensive |
| `price_level[lte]=0`, `price_level[in]=1`, `price_level=1` | `400` |

A restaurant without a `price_level` matches no `price_level` filter.
The `overall` filter has the same rule for a restaurant without reviews.

The "Filters" popover of the restaurants page has a "Price level" select:

| Option | Link value | Parameters |
|---|---|---|
| Inexpensive | `1` | `price_level[eq]=1` |
| Moderate or less | `1-2` | `price_level[gte]=1&price_level[lte]=2` |
| Expensive or less | `1-3` | `price_level[gte]=1&price_level[lte]=3` |
| Moderate or more | `2-4` | `price_level[gte]=2&price_level[lte]=4` |
| Expensive or more | `3-4` | `price_level[gte]=3&price_level[lte]=4` |
| Very expensive | `4` | `price_level[eq]=4` |

The select has no option for all four levels. It would only leave out the restaurants without a price level.
The chip is "Price level: Moderate or less".

Version 5.3.0 of the spec adds the parameter. A client that does not send it gets the same results as before.

# Implementation Details
[implementation-details]: #implementation-details

- the `PriceLevelRange` schema has the five comparisons. `priceLevelComparisons` gives them to `compare` of `list.go`, like the helpers of the other ranges.
- the `price_level` column stores 0 for a restaurant without a price level. PocketBase makes each number column `NUMERIC DEFAULT 0 NOT NULL`, so the column cannot store NULL. See decision 00038.
  `priceLevelColumn` of `restaurants.go` is `NULLIF([[restaurants.price_level]], 0)`. It is NULL for these restaurants, like `overallColumn` for a restaurant without reviews.
  A comparison with NULL matches no row. Without the expression, `price_level[lt]=2` would also match these restaurants.
  A later sort or filter of the price level uses the same expression.
- the cursor fingerprint has all the parameters, so a cursor works only with the same `price_level` filter
- the column has no index. It has only four values, so an index selects a large part of the table. The query reads the rows in the order of the sort index and checks each row.
- `priceLevelRanges` of `restaurantFilters.ts` has the options and their parameters. `useListQuery` ignores a link value that is not an option.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
