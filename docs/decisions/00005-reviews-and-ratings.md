---
description: A review is one visit to a restaurant or one cook of a recipe. It has a date, four scores, an overall that the API computes, a title, feedback, and photos. The rating of a subject counts each reviewer one time, and the overall sort uses this rating. The UI adds a review from each screen and shows the reviews of a subject as a timeline of visits or cooks.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00000 wants to remember and share all food experiences.
A second dinner at a restaurant or a second cook of a recipe is a new experience.
Each one then gets its own review with its own date, so the old scores and text stay.

A reader compares reviews faster with one value than with four scores.
A value that the reviewer types can disagree with the scores, so the API computes it.

The group is small. A restaurant has 1 to 5 reviews.
The order of "Highest rated" must be the order of the values on the cards.
A person who visits a restaurant often must not move its rating more than a person who visits one time.

Expected outcome:
- a user writes many reviews of one restaurant or recipe, one for each visit or cook
- each review has the date of the visit or the cook, and the lists sort by it
- each review has an `overall` that the API computes from its scores
- `rating` of a subject counts each reviewer one time. `review_count` counts each review.
- the `overall` sort is the order of `rating.overall` on the cards
- the sorts and the filters of the scores read one page of an index

# Scope
[scope]: #scope

In scope:
- the scores of each kind and the `overall` of a review
- the `date` of a review and its rules
- the review operations, their sorts, their filters, and their search
- `rating`, `review_count`, and `own_review_count` of a subject
- the `overall` sort and filter of `listRestaurants` and `listRecipes`
- the `overall` column and the overall tables with their triggers and indexes
- the review form, the review card, and the review page of the UI

Out of scope:
- the photos of a review. Decision 00007 has them.
- the reviewer pages and the reviewer averages. Decision 00006 has them.
- an overall that the reviewer types
- tags on reviews
- a `date` filter, and a count of the visits of one user, such as "You visited 3 times"
- a `reviewer_count` property in the responses
- a minimum number of reviews for the `overall` sort
- replies, comments, and likes on reviews. Decision 00000 puts social features out of scope.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The scores

Each score is an integer from 1 to 10.

| Kind | Score | Question |
|---|---|---|
| restaurant | `space` | decoration, comfort, atmosphere, cleanliness, and kitchen setup |
| restaurant | `service` | professionalism, speed, friendliness, and attentiveness of the staff |
| restaurant | `food` | presentation, flavor, ingredient quality, and culinary execution |
| restaurant | `price` | value for money. A high score is good. |
| recipe | `taste` | how good is the result? |
| recipe | `ease` | how easy is it to make? A high score is easy. |
| recipe | `instructions` | are the steps clear, and do the times and amounts work? |
| recipe | `value` | is the result worth the cost of the ingredients and the time? |

## A review

```json
{
  "id": "r1b8c5e2g9i6k3m",
  "restaurant": "k3x9m2p7q1w8z4a",
  "restaurant_name": "Luigi's Trattoria",
  "date": "2026-11-14",
  "overall": 7.5,
  "space": 7, "service": 9, "food": 8, "price": 6,
  "title": "Great food, a bit expensive",
  "feedback": "Friendly staff and great pasta",
  "creator": { "id": "u5n1c8r3t6y0b2e", "name": "Ana" },
  "photos": [],
  "created": "2026-11-14T21:02:11.120Z",
  "updated": "2026-11-14T21:02:11.120Z"
}
```

- `title` is a `Title` of 1 to 255 characters. `feedback` is an optional `LongText`.
- `overall` is the average of the four scores. It is exact with two decimals, such as 7.75. The UI shows one decimal.
- a body with `overall` gets `400`. It is read only.
- `restaurant_name` or `recipe_name` is the current name of the subject. A list of reviews from many subjects then shows the subject without more requests.
- a recipe review has `recipe`, `recipe_name`, and the recipe scores
- `created` is the time that the user wrote the review. `date` is the day of the visit or the cook.

## The date

- `date` has the form `YYYY-MM-DD`. It has no time and no time zone.
- a create needs it. A patch can change it.
- the date must exist. `2026-02-30` gets `422`.
- the date must be `1900-01-01` or later. It must not be after the current date in UTC+14, the latest time zone. A user anywhere can then use the local date of today.
- the UI shows "Visited on 14 Nov 2026" or "Cooked on 14 Nov 2026"

## Operations

| Path | Operations |
|---|---|
| `/restaurants/{restaurantId}/reviews` | `createRestaurantReview`, `listRestaurantReviews` |
| `/restaurants/{restaurantId}/reviews/{reviewId}` | `getRestaurantReview`, `updateRestaurantReview`, `deleteRestaurantReview` |
| `/restaurant-reviews` | `listAllRestaurantReviews` |
| `/recipes/{recipeId}/reviews` | `createRecipeReview`, `listRecipeReviews` |
| `/recipes/{recipeId}/reviews/{reviewId}` | `getRecipeReview`, `updateRecipeReview`, `deleteRecipeReview` |
| `/recipe-reviews` | `listAllRecipeReviews` |

- a user can write many reviews of one subject. A create never gets `409`.
- only the creator of a review changes or deletes it. Other users get `403` with `not_creator`.
- the scores, `title`, `feedback`, and `date` can change
- a review of a different subject than the path gets `404`. An unknown subject in the path gets `404`, also for the list.
- the lists have the sorts `overall`, the scores, `relevance`, `title`, `date`, `created`, and `updated`. The default is `-date`. Reviews of one day sort by `created`.
- the lists have the filters `overall`, the scores, `created`, `updated`, and `creator`. The lists of all reviews also have `restaurant` or `recipe` with `eq` and `in`.
- the search looks in `title` 10, the name of the subject 2, and `feedback` 1
- decision 00006 has the reviews of one reviewer

## The rating of a subject

The rating first gets the average of each reviewer. It then gets the average of these values.

Ana visits Luigi's four times, and each review has the overall 10. Ben visits one time and gives 6.

```
Ana:    (10 + 10 + 10 + 10) / 4 = 10
Ben:    6 / 1                   = 6
rating.overall = (10 + 6) / 2   = 8.0
review_count   = 5
```

- each score of `rating` uses the same rule. `rating.overall` is the average of the reviewer overalls.
- the API rounds each value to one decimal
- `rating` is `null` for a subject without reviews
- `review_count` counts each review. `own_review_count` counts the reviews of the user of the token. It is 0 without a token.
- a new, changed, or deleted review changes the rating at once

## The overall sort

`sort=-overall` sorts by `rating.overall`, the value that the card shows.
Two equal values sort by the number of reviewers. More reviewers come first.

| Restaurant | Reviews | `rating.overall` | Reviewers | Place in `-overall` |
|---|---|---|---|---|
| A | Ana 10, Ben 10 | 10.0 | 2 | 1 |
| B | Ana 10 | 10.0 | 1 | 2 |
| C | Ana 9, Ben 9, Cleo 9 | 9.0 | 3 | 3 |
| D | Ana 10 four times, Ben 6 | 8.0 | 2 | 4 |
| E | none | | 0 | last |

- `sort=overall` is the reverse order. Subjects without reviews are last in both directions.
- the `overall` filter compares the rounded `rating.overall`. A subject without reviews matches no `overall` filter.
- sort properties after `overall`, such as `name` in `overall,name`, do not change the order
- one review of 10 is above many reviews of 9. The group accepts this. The card shows the number of reviews next to the value.

## The UI

- the restaurant page always has "Add a visit". The recipe page always has "Add a cook". Each opens a new review of the subject.
- "Add review" of the navigation opens `/reviews/new`, a new review without a subject. The form starts with "What did you eat?" and the choices "I ate out" and "I cooked".
- the restaurant or recipe field searches the existing subjects. With a preferred location, the nearest restaurants come first. See decision 00008.
- "Add a new restaurant" or "Add a new recipe" below the field saves a draft of the review, then opens the form of decision 00004. After the save, the UI goes back to `/reviews/new` with the new subject and restores the draft.
- when the place or the source already has a subject, the form of decision 00004 links "Review this restaurant" or "Review this recipe". "Cancel" also goes back to `/reviews/new`.
- "Change" and a switch of the kind keep the fields and the photos of the review in the draft of the kind
- the date has "Today", "Yesterday", and "Pick a date". The line below shows "Visited on 22 Nov 2026" or "Cooked on 22 Nov 2026". The form starts with the local date of today and allows no later date.
- the form shows the overall of the selected scores before the save. Each score has ten buttons, so one click selects it.
- the title and the feedback are the last fields, under "In a few words". The photos section follows the form. See decision 00007.
- an entry of the table or the journal shows the overall, the title, the reviewer, and the date. Decision 00009 has the entries.
- the review page starts with the first photo. Then it shows the subject, the title, the reviewer, the date, the score block, the feedback, the photos, the subject, and more reviews of the same reviewer.
- the review page shows the date and "Written" with the time of `created`
- the restaurant page lists its reviews under "Every visit". The recipe page lists them under "Every cook".
- each item of this timeline has the date, the title, the overall, two lines of the feedback, and the reviewer. The own reviews show "You".
- the sort control has "Newest visit" or "Newest cook", "Highest rated", and "Lowest rated". A link can also select the oldest and the written orders.
- `lib/reviewKinds.ts` has the score fields, the labels, and the API calls of each kind

# Implementation Details
[implementation-details]: #implementation-details

## Collections

| Collection | Fields |
|---|---|
| `reviews` | `restaurant`, `creator`, `date`, `space`, `service`, `food`, `price`, `title`, `feedback`, `created`, `updated` |
| `recipe_reviews` | `recipe`, `creator`, `date`, `taste`, `ease`, `instructions`, `value`, `title`, `feedback`, `created`, `updated` |

- the parent relation is required and has no cascade. A delete of a subject deletes the reviews itself. See decision 00004.
- `creator` is required and cascades. A deleted user takes the reviews.
- the score fields are required numbers from 1 to 10
- `date` is a required text field with the pattern `^\d{4}-\d{2}-\d{2}$`
- `idx_reviews_restaurant_creator` and `idx_recipe_reviews_recipe_creator` are not unique. They serve the own reviews and the rating triggers.
- the `date` sort has indexes on `(parent, date, created, id)`, `(creator, date, created, id)`, and `(date, created, id)`
- each create, update, and delete runs in one transaction with a read of the parent. See decision 00002.

## The date check

- `ReviewDate` has the format `review-date`, not `format: date`. The TypeScript generator would change `format: date` to a `Date` in UTC.
- `checkReviewDate` of `reviewdate.go` checks the format for both kinds. The response validation does not check it.
- `reviewDateGuard` checks `date` on each save through the PocketBase API. A save in the dashboard gets the same check.
- `parseDay` of `format.ts` reads the date as a local date. `new Date('2026-11-14')` is midnight UTC, which is the day before west of UTC.
- the form gets the local date again on each focus of the date input and on each submit

## The overall column

```sql
ALTER TABLE reviews ADD COLUMN overall REAL AS ((space + service + food + price) / 4.0);
CREATE INDEX idx_reviews_restaurant_overall ON reviews (restaurant, overall, id);
CREATE INDEX idx_reviews_creator_overall ON reviews (creator, overall, id);
CREATE INDEX idx_reviews_overall ON reviews (overall, id);
```

- `recipe_reviews` has the same column from its four scores and the same three indexes
- the column is virtual. Only the indexes store its value.
- each index ends with `id`, the last sort key. The row value comparison of a cursor is then a range of the index.
- the column is not a field of the collection. PocketBase does not write it, and the dashboard does not show it.
  PocketBase keeps the column and its indexes when it changes the table with `ALTER TABLE`.
- `toReview` computes the same value in Go. `TestOverallColumnEqualsResponse` fails when the column and the response differ.
- the `overall` filter uses the column as `+reviews.overall` when the sort has no `overall`. SQLite then reads the index of the sort and stops after one page.
- `guardReviewOverall` of `reviewschema.go` refuses a dashboard change that adds a field named `overall` or removes a score field. The dashboard then shows the cause, not the SQLite error.

## The overall tables

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

- `recipe_overall` has the same columns with `recipe` and the recipe scores
- the tables are not PocketBase collections. A save of a record cannot write an old average over a new one.
- each subject has one row. A subject without reviews has NULL values and the counts 0, so the inner join of a list keeps it.
- SQLite triggers write all values of a row in one statement. The inner query groups the reviews of one subject by `creator`:

```sql
SELECT ROUND(AVG(reviewer_overall), 1), COUNT(*), COALESCE(SUM(reviewer_reviews), 0),
  ROUND(AVG(space), 1), ROUND(AVG(service), 1), ROUND(AVG(food), 1), ROUND(AVG(price), 1)
FROM (
  SELECT SUM(space + service + food + price) * 1.0 / (4 * COUNT(*)) AS reviewer_overall,
    COUNT(*) AS reviewer_reviews, AVG(space) AS space, AVG(service) AS service, AVG(food) AS food, AVG(price) AS price
  FROM reviews WHERE reviews.restaurant = ? GROUP BY creator
)
```

| Record change | Table change |
|---|---|
| a subject is made | insert its row |
| a subject is deleted | delete its row |
| a review is made or deleted | compute the row of its subject again |
| the scores, the subject, or the creator of a review change | compute the rows of the new and the old subject again |

- the triggers compute from the reviews. They do not add to or subtract from a stored value, so the value cannot drift.
- the triggers run for all writes: the Public API, the dashboard, cascade deletes, and plain SQL
- `ratingsQuery` reads the stored rating of each subject of a page by the primary key. It reads no reviews.
- `subjectKind` of the migrations has the names of each kind. One code makes the tables and the triggers of both kinds.

## The overall sort

- `-overall` sorts by `rated DESC, overall_or_0 DESC, reviewers DESC, restaurant DESC`. `overall` sorts by `unrated, overall_or_0, reviewers, restaurant`.
- all keys have one direction, so the cursor condition is one row value comparison
- SQLite uses an index on columns for a row value range, not an index on expressions. The sort keys are then generated columns.
- the subject key is unique, so `parseSort` adds no `id` key. SQLite sorts no rows.
- the list joins the overall table with an inner join. SQLite then reads it first, in the order of the index.
- the sort key is the rounded value. Reviewer averages of 8.04 and 7.96 both give 8.0, and the number of reviewers decides.

## Migrations and the derived columns

- SQLite refuses to drop a column that a trigger or a generated column uses
- a migration that drops a score column must first drop the triggers, the sort indexes, the `overall` column, and its indexes. It makes them again afterwards.
- it also drops the score column of the overall table
- PocketBase drops and adds the column of a relation field when its maximum changes from one to more. This also applies to the parent relation of reviews.
- `subjectKind` of `migrations/subjects.go` makes the SQL of the overall column and the rating from the list of scores

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| One review for each user and subject | Removed. A second visit had to change the old review, and its scores, text, and date were lost. |
| One review with a list of dated visits | Not selected. Each visit has its own scores, text, and photos. |
| A `cleanliness` score | Removed. `space` covers cleanliness, so it was scored two times and counted two times in `overall`. |
| Review kinds with a `kind` and a single `rating` | Removed. One kind is simpler for clients, filters, and sorts. |
| Tags on reviews | Removed. They repeated the tags of the subject. |
| An overall that the reviewer types | Not selected. It can disagree with the scores. |
| A rating that counts each review | Removed. A person who visits often moved the value more. |
| A Bayesian average for the `overall` sort | Removed. With 1 to 5 reviews it mostly counted the reviews, so a 9.0 came above a 10.0 without a reason on the page. |
| A minimum number of reviewers | Not selected. Most subjects of the group have one or two reviewers. |
| The latest review of each reviewer | Not selected. It loses the older visits. |
| A rating computed on each read | Removed. The `overall` sort and filter then read all reviews. |
| A table with triggers for the overall of one review | Not selected. The value needs only the columns of its row, so a generated column has no state that can drift. |
| A PocketBase number field that a hook writes | Not selected. A write from the dashboard or from SQL could leave a wrong value. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
