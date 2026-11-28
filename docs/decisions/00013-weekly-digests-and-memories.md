---
description: The home of the UI shows a weekly digest. A digest gathers the entries that the group wrote in one week. The week runs from Monday to Sunday in the zone of the reader. It has a lead, the top scores, the new subjects, and the counts. Memories show the entries of the same week in earlier years. The UI builds the digest from the review lists. The Public API only gets a `date` filter.
status: proposed
---

# Motivation
[motivation]: #motivation

The table of decision 00009 lists all entries by the time they were written, the newest first.
A small group writes few entries. A quiet week then looks empty. An endless feed also reads like social media.
The backlog has the item "redesign or rename table concept".

A weekly digest gathers the entries of one week on one page, like a small newspaper.
A quiet week then looks planned, not empty. A finished digest rarely changes. A person can then share its link in the chat of the group.
Readers in the time zone of the group see the same digest.
A food journal also remembers. The entries of the same week in earlier years come back as memories.

Expected outcome:
- the home shows the entries of this week so far, the digest of the last finished week, and the memories
- each week has a page at `/weeks/{date}` with its digest and every entry
- the archive at `/weeks` lists the weeks with entries
- the link of a digest shows a preview with the counts
- the weeks start on Monday at 00:00 in the time zone of the browser
- the Public API keeps its operations. The review lists only get a `date` filter.

# Scope
[scope]: #scope

In scope:
- the week and its time zone
- the `date` filter of the review lists
- the rules of the lead, the top scores, the new subjects, and the memories in the UI
- the home, the digest page, and the archive of the UI
- the link previews of the digest pages
- the words and the components of the design system for the digest

Out of scope:
- new operations or response schemas for the digest. See [Alternatives](#alternatives).
- a weekly email. The group talks in its own chats. See "Won't do" in the backlog.
- memories of one month or six months ago. Memories are whole years.
- the memories of one reviewer on the journal
- a digest that a person edits or curates
- a week that starts on Sunday
- a setting for the time zone of the weeks. The browser gives it.
- comments, likes, or reactions on a digest. Decision 00000 puts social features out of scope.
- an average score of a week

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

The design canvas "Weekly digest mockup" has the five screens of this decision.
They are the home on a Saturday and on a Monday, the digest page, the archive, and a link preview.

## The concepts

- **Week**: Monday 00:00 to the next Monday 00:00 in the time zone of the reader. The date of its Monday identifies it, such as `2026-11-16`.
- **Digest**: the page of one week. It shows the entries that were written in the week. A week that ended has a finished digest. The current week has a digest in progress.
- **Lead**: the first story of a digest. It is a subject that two or more people reviewed in the week, or the best entry.
- **Top scores**: the best entry of each kind after the lead.
- **New on the table**: the restaurants, recipes, and pantry items that users added in the week.
- **Memory**: an entry with a `date` in the same days of an earlier year. "On this day" marks a memory with the month and the day of today.
- **Archive**: the list of the weeks with entries at `/weeks`.
- **Time zone**: the zone of the weeks. It is the zone of the browser of the reader.

Examples:
- On Monday 23 November 2026, Ana opens Olive. "This week so far" says "Nothing yet this week". Below it is the digest of last week. Its lead is "Priya and Marco both went to Luigi's Trattoria" with the photo of Priya.
- Priya copies the link `/weeks/2026-11-16` into the chat of the family. The preview shows "Week of 16 November 2026" and "14 entries from 6 people".
- Marco writes a review on Monday 23 November at 00:30 CET about the dinner of Sunday. For a reader in CET, it is in the week of 23 November. The week of an entry comes from the time it was written. Its card still shows "Visited on 22 Nov 2026".
- On Saturday 28 November, Jules opens the home. The memories show the pho that Ana ate on 28 November 2025 with "On this day". They also show the banana bread that Marco cooked on 24 November 2023.
- Jules opens `/weeks/2026-08-03`, a week without entries. The page says "A quiet week" and links to the weeks before and after it.

## The home

```
SATURDAY 28 NOVEMBER
On the table
What everyone ate, cooked, and tasted lately.

THIS WEEK SO FAR               4 entries from 4 people
  Kin Khao Noodle Bar · Broth worth the queue      8.8
  Shakshuka · Better with feta                     7.5
  Comté 24 months · Nutty, with a long finish      9.0
  Osteria del Ponte · Lovely room, slow kitchen    6.5
  Open this week

══════════════════════════════════════════════════════
Last week
16–22 November · 14 entries from 6 people · 3 new on the table
  [photo of the lead]                            (8.4)
  LUIGI'S TRATTORIA · 2 VISITS
  Priya and Marco both went to Luigi's Trattoria
  Priya  The best carbonara in town ........... 8.6
  Marco  Loud but worth it .................... 8.2
Top scores
  ATE OUT  Casa Pepe .......................... 8.8
           Jules · Croquetas to remember
  COOKED   Dal makhani ........................ 8.4
           Ana · Worth the slow simmer
  TASTED   Rioja Reserva ...................... 8.1
           Sam · Soft tannins, ripe cherry
New on the table
  (Luigi's Trattoria) (Miso cod) (Rioja Reserva)
  Read the digest    All weeks

Memories
A YEAR AGO THIS WEEK
  [On this day] Pho 24 · Rare beef, clear broth    8.0
3 YEARS AGO THIS WEEK
  Banana bread · Too sweet, great crust            7.0
```

- "This week so far" shows the five newest entries of the current week as short rows. Its link "Open this week" opens the page of the current week.
- without entries in the current week, it says "Nothing yet this week". Below it is "The first visit, cook, or tasting shows here." A signed in user then sees "Add review".
- the digest is the last finished week with entries. Its heading is "Last week" for the week before the current week. An older week has "Week of 2 November".
- a double rule in `foreground` starts the digest, like the masthead of a newspaper
- "Read the digest" opens the page of the week. "All weeks" opens the archive.
- the memories are the memories of the current week. Without memories, the section is not there.
- the heading of the memories is "Memories". The groups are "A year ago this week" and "3 years ago this week".
- a signed out visitor sees the note of decision 00009 above "This week so far"
- the home has no `KindSwitch`. The digest shows all kinds on purpose.
- without any entries, the home shows the empty state "Nothing on the table yet" of decision 00009
- without a finished week with entries, the home has no digest section
- `BottomNav` and `AppHeader` keep "Table". It opens the home.

## The parts of a digest

The home and the digest page show the parts of a digest in the same way.

- the lead is a photo card, like the photo form of `EntryCard`. The score stamp sits on its photo.
- the kicker of a shared lead is the subject and the count, such as "Luigi's Trattoria · 2 visits". The words are "visits", "cooks", and "tastings".
- the heading of a shared lead is its headline. Below it, each entry of the lead has one line.
- such a line has the reviewer, the title, a dotted leader, and the overall
- the stamp of a shared lead shows the average overall of its entries. 8.6 and 8.2 give 8.4.
- the UI gets the average from the exact values and shows one decimal, like each overall. See decision 00005.
- a lead with one entry is the photo card of that entry with its title
- a lead with one entry has a headline such as "Ana tasted Comté 24 months". It names the reviewer, the verb, and the subject.
- the archive rows use the headline of each lead
- the top scores are one card, like a menu. Each kind has a line with the kind, the subject, a dotted leader, and the overall.
- the reviewer and the title of a top score are below its line
- "New on the table" is a row of chips with the icon of the kind and the `accent` fill. More than six subjects add the text "and 2 more".
- "On this day" is a tag with the `accent` fill above the subject of a memory. `saffron` stays only for the score stamp. See decision 00009.

## The digest page

- the page of a week is at `/weeks/{date}`. A date that is not a Monday opens the week of the date. The URL then changes to its Monday.
- the heading is "Week of 16 November 2026". The current week has "In progress" below it.
- the page has the counts, the lead, the top scores, "New on the table", the memories, and "Every entry"
- the counts have the kinds below them, such as "6 visits · 5 cooks · 3 tastings"
- the memories of a digest page have the dates of their year as group headings, such as "16–22 November 2025"
- "Every entry" lists all entries of the week as short rows, the newest written first
- the `KindSwitch` of "Every entry" has "All", "Ate out", "Cooked", and "Tasted". `?kind=` keeps the selection.
- the links at the end are "Week of 9 November", "Week of 23 November", and "All weeks"
- they open the weeks before and after with entries, and the archive
- a week without entries says "A quiet week". It still has the links.
- a week after the current week shows the not found screen
- a date that does not exist, such as `2026-02-30` or `abc`, also shows it. So does a date before `1900-01-01`.

## The archive

- `/weeks` lists the weeks with entries, the newest first
- the heading is "All weeks", with "The weeks with entries, the newest first." below it
- each row has "Week of 16 November 2026", the counts, and the headline and the photo of the lead
- a lead without a photo shows the icon of its kind. A row has no score.
- the row of the current week has the tag "In progress". Its counts end with "so far".
- "Load more weeks" loads the next weeks
- the digest page and the home link to it

# Implementation Details
[implementation-details]: #implementation-details

## The time zone

- the weeks of the UI use the zone of the browser from `Intl.DateTimeFormat().resolvedOptions().timeZone`
- the UI reads the zone and the current week when a route loads. It reads them again when the tab becomes visible.
- a new current week then reloads the data of the page. A tab open across Monday 00:00 moves to the new week.
- a device that moves to another zone gets the new weeks at the same moments
- `REVIEWS_TIME_ZONE` sets the zone of the link previews only. It is an IANA name. The default is `CET`.
- an unknown name stops the app at start with an error, like the other environment variables
- the app embeds the zones of `time/tzdata`. It then starts on a host without zone files. Go reads the zone files of the host first.
- the Docker image also has the `tzdata` package
- `CET` has the summer time CEST from the last Sunday of March to the last Sunday of October
- the week of 23 to 29 March 2026 has 167 hours in `CET`. The week of 19 to 25 October 2026 has 169 hours.
- only the weeks and today use the zone. The `date` check of decision 00005 keeps UTC+14.

## The `date` filter

The Public API gets one change. The nine review lists with the `created` filter also get a `date` filter.

- `DateFilter` is a shared parameter, like `CreatedFilter`. Its schema `DateRange` has the keys `lt`, `lte`, `eq`, `gte`, and `gt`, as in `TimestampRange`.
- each value is a `CalendarDate` in the form `YYYY-MM-DD`. Its format `calendar-date` needs a date that exists. All comparisons must match.
- `ReviewDate` does not fit. Its format refuses a date after today, and a filter can end in the future.
- `listAllRestaurantReviews` with `date[gte]=2025-11-23` and `date[lte]=2025-11-29` gives the restaurant reviews of that week in 2025
- the filter compares the text of `date`. The form `YYYY-MM-DD` sorts like the dates.
- each review table has the index `date, created, id`. The filter uses it.
- the review lists keep their other filters and sorts. The API gets no new operation. `DateRange` and `CalendarDate` are its only new schemas.

## Requests

The UI calls the Public API only through `@reviews/sdk`, as before.
`loadAll` gets each page of a list with `limit=100`.
`listAll*Reviews` names `listAllRestaurantReviews`, `listAllRecipeReviews`, and `listAllPantryItemReviews`.

| Data | Requests | Lists and filters |
|---|---|---|
| the entries of a week | 3 | `listAll*Reviews` with the `created` range of the week, all pages |
| the new subjects of a week | 3 | `listRestaurants`, `listRecipes`, and `listPantryItems` with the same `created` range, all pages |
| the nearest earlier week with entries | 3 | `listAll*Reviews` with `created[lt]` of the start, `sort=-created`, and `limit=1` |
| the nearest later week with entries | 3 | `listAll*Reviews` with `created[gte]` of the end, `sort=created`, and `limit=1` |
| the oldest entry | 3 | `listAll*Reviews` with `sort=date` and `limit=1` |
| the memories of one earlier year | 3 | `listAll*Reviews` with the `date` range of the week in that year, all pages |

- the requests of one step run at the same time
- the home loads the entries from the Monday before the current week. It loads one list of each kind, with all pages. It splits them into the two weeks.
- the home looks for an older week only when last week has no entries
- the UI keeps the oldest entry until the next load of the app. A save of a review with an older `date` replaces it.
- the memories look back from the year before to the year of the oldest entry. They look back at most five years.
- a home in the first year makes about 9 requests. A home with three earlier years makes about 18.
- these numbers count one page of each list. A list with more than 100 entries needs one more request for each page.
- the digest page loads the entries, the new subjects, the weeks before and after, and the memories of its week
- the page of the current week does not ask for a later week. Its link to a later week is not there.
- "Every entry" shows the loaded entries of the week. `KindSwitch` filters them in the UI. They need no more requests.
- "Load more entries" shows the next 20 loaded entries

## The weeks in the UI

- `lib/weeks.ts` changes between instants and the dates of the zone with `Intl.DateTimeFormat` and its `timeZone`
- the UI has no date library
- the start of a week is the first instant on or after Monday 00:00 in the zone. The UI finds it from the offset of the zone at that time.
- in a few zones the clocks jump at midnight. Monday 00:00 then does not exist, and the week starts at the first instant after the jump.
- in a few zones the clocks go back over midnight. Monday 00:00 then happens two times, and the week starts at the first one. Asia/Jerusalem had this on 24 September 2001.
- `webui` uses the same rule for `REVIEWS_TIME_ZONE`
- the `created` filters get the start and the end as RFC 3339 timestamps
- today is the date of now in the zone

## The rules

All rules are in `lib/weeks.ts`. They use the entries that were written in the week. `created` gives the week of an entry, as on the table of decision 00009.

The lead:
1. A shared subject has entries of two or more reviewers in the week. The shared subject with the most reviewers is the lead. Its entries of the week are the entries of the lead, the newest first.
2. Without a shared subject, the lead is the entry with photos and the highest overall.
3. Without photos in the week, the lead is the entry with the highest overall.

- in rule 1, a tie goes to the higher average overall of the entries of the week. The next tie goes to the newest entry.
- in rules 2 and 3, a tie goes to the newest `created`, then to the higher ID
- one reviewer with two entries of a subject does not make it shared
- the headline of a shared lead names the reviewers, such as "Priya and Marco both went to Luigi's Trattoria"
- three or more reviewers get "all", such as "Priya, Ana, and Marco all cooked Shakshuka". The verbs are "went to", "cooked", and "tasted".
- the photo of the lead is the first photo of the newest entry with photos

The top scores:
- each kind has the entry with the highest overall that is not in the lead
- the order is "Ate out", "Cooked", and "Tasted". A kind without such an entry is not there.
- a tie goes to the newest `created`, then to the higher ID
- the rows show no rank numbers

New on the table:
- the restaurants, recipes, and pantry items with a `created` in the week, the newest first
- a subject without reviews is also there

The counts:
- the number of entries is the sum of the three kinds
- the number of people counts each user with an entry one time
- the UI writes "14 entries from 6 people · 3 new on the table". It leaves out a count of zero.

The memories:
- a memory has a `date` from the Monday to the Sunday of the week in an earlier year
- 29 February shifts to 28 February in a year without it
- the memories come from at most three years, the nearest first. Each year has at most two memories.
- in the current week, a memory with the month and the day of today goes first in its year
- then a memory with photos goes first, then the higher overall, then the newest `created`
- the UI ranks all entries of the week in that year. The requests load each page, so no memory falls out before the ranking.
- the UI shows "On this day" on these memories of the current week. On a finished week, no memory has it.
- the memories use `date`, not `created`. A memory is about the meal.
- a user writes a review today of a dinner of 2023. It is a memory of the week of that dinner.
- the memories have no part for one person. "On this day" uses the date of today in the zone of the reader.

## The archive in the UI

- `/weeks` loads the reviews of all kinds with `sort=-created`, 100 at a time
- `takeMerged` of `lib/entries.ts` merges the three kinds. See decision 00009.
- the UI groups the merged entries by week. A week is complete when the merge has passed its start.
- each row gets its counts and its lead from the entries of its week
- "Load more weeks" loads the next pages until it has 10 more complete weeks
- the archive reads all reviews when a person scrolls to the end. This is fast for a few thousand reviews.
- the backlog expects a low number of users

## Link previews

- `webui` writes the page meta of `/weeks/{date}`. See decision 00002.
- it uses `REVIEWS_TIME_ZONE` for the start and the end of the week. A crawler has no browser zone.
- a reader in another zone can see counts that differ from the preview. Only entries near Monday 00:00 move.
- one query counts the reviews of each kind and the people. It has no copy of the rules.
- it joins the creator IDs of the three kinds with `UNION` and counts each ID one time. A person with entries of two kinds is one person, as in the UI.
- the title is "Week of 16 November 2026". `og:type` is `article`.
- the description has the counts, such as "14 entries from 6 people: 6 visits, 5 cooks, and 3 tastings."
- the description of the current week starts with "So far". A week without entries has "A quiet week on Olive."
- `og:image` is the image of the site
- `og:url` has the Monday of the week
- a week after the current week gets `404` with the default `index.html`
- a date that does not exist or is before `1900-01-01` gets the same `404`
- `/weeks` keeps the tags of the site, like the other list screens
- the `<title>` of the page meta has the format of `usePageTitle`, as for the other pages

## The UI

- `TablePage.vue` at `/` shows the current week and the last digest
- `WeekPage.vue` is at `/weeks/:date`. `WeeksPage.vue` is at `/weeks`.
- `composables/useDigest.ts` makes the requests of one week. `composables/useWeeks.ts` makes the requests of the archive.
- `lib/weeks.ts` has the weeks, the rules, the two headlines of the lead, and the average of a shared lead
- it also has the texts of the counts and the group headings of the memories
- the short rows use `EntryCard` with `variant="row"`
- `groupByWritten` of `lib/entries.ts` goes away. The journal keeps `groupByMonth` and `useEntries`.
- `specs/design-system` gets the lead, the top scores, "New on the table", the memories, and the week rows
- the words of the design system change. "Past week" and the month groups go away.
- the home gets "This week so far", "Last week", and "A year ago this week"
- the digest page gets "Week of 16 November" and "Every entry"

## Corner cases

- the first week of an install has no finished digest. The home then shows only "This week so far".
- an edit of a score after the end of a week can change its lead. A delete can remove the lead. The next load shows the new result.
- an edit does not move a review to another week. `created` does not change.
- for a reader in CET, an entry written on Monday at 00:30 CET is in the new week. In UTC, it is 23:30 on Sunday.
- a person in New York sees the weeks of New York. A review written at 00:30 CET on Monday is then in the week before.
- the same link can then show other entries to that person near the start or the end of the week
- a device with a wrong clock shows the wrong current week
- the week of 28 December 2026 to 3 January 2027 has memories of two months. They are from December and January.
- the first year of an install has no memories. Old reviews that a superuser adds in the dashboard can fill it. See the backlog.
- a week with more than 100 entries of a kind needs more pages. `loadAll` gets them.
- a review saved while a page loads can be in one request and not in another. The next load shows it.

## Changes to other decisions

When this decision is implemented, these decisions change with it:

| Decision | Change |
|---|---|
| 00002 | the page meta of `/weeks/{date}` and its row in the table of the pages |
| 00005 | the `date` filter of the review lists |
| 00009 | the concept "Table", the home, the two sections "The table", and `?kind=` on the home. The table sections point here. |

`AGENTS.md` gets `REVIEWS_TIME_ZONE`, the `date` filter, and the pages.
The design system README gets the new words and components.
The backlog item "redesign or rename table concept" is done.

## Work order

1. the `date` filter, the generated SDKs, `REVIEWS_TIME_ZONE`, and the page meta. One pull request.
2. the home, the digest page, the archive, and the design system. A second pull request.

Deploy the two pull requests together.
The UI has no screen at `/weeks/{date}` before the second pull request. A shared link then shows a preview and opens the not found screen.

The second pull request also moves the end-to-end tests of the table.

- `tests/e2e/pantry.spec.ts` opens `/?kind=pantry` today. Its check moves to "Every entry" of the week of its review with `?kind=pantry`.
- each test that looks for its entry on `/` looks for it in "Every entry" of the week of its `created`. The home shows only five entries.
- the test gets that week from the `created` of the saved review. A run across Monday 00:00 then still finds the entry.
- the checks of the note for visitors and of the heading "On the table" stay on `/`
- `tests/e2e/accessibility.spec.ts` waits until the status of `/` is not "Loading entries". The home keeps this status text.

Each pull request has its Go, unit, and end-to-end tests.

- the Go tests check each operator of the `date` filter on a review list of each kind
- they check the page meta of a finished week, of the current week, and of a quiet week
- they check the `404` of a week after the current week, of `2026-02-30`, and of a date before 1900
- they check that a person with entries of two kinds counts one time
- they check an unknown name in `REVIEWS_TIME_ZONE`
- the unit tests of `lib/weeks.ts` check the rules with fixed data and a fixed time of now
- they check the ties of the lead and the top scores
- they check the memories of 29 February and of the turn of the year
- they check that a memory with a low overall and the date of today still goes first
- they check the weeks of the summer time changes in `Europe/Berlin`
- they check the zones with a jump at Monday 00:00, such as `Asia/Tehran` on 22 March 2021. `America/Santiago` jumps only on a Sunday. It checks the start of a day.
- `specs/api/week-starts.json` has these weeks and days. The Go tests and the UI tests both read it. Go and TypeScript then give the same start.
- they run in the time zone `America/Los_Angeles` and give the zone `Europe/Berlin` to the week code. They then show that the code uses the given zone.
- all projects of one end-to-end run share one app and one week. The end-to-end tests check only their own entries.
- an end-to-end test finds its entry in "Every entry" by a name from `uniqueTag`

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| A `weeks` tag with `getCurrentWeek`, `getWeek`, and `listWeeks` | Not selected. It adds three operations and the first `discriminator` of the spec. The review lists already have the data. |
| An inline schema for the `date` filter | Not selected. The generator makes one Go type for each operation. |
| `ReviewDate` for the values of the `date` filter | Not selected. Its format refuses a date after today. |
| A filter of the month and the day of `date` | Not selected. It is a new kind of filter. The `date` range works like the `created` range. |
| Memories from more than five years back | Not selected. Each year costs three requests. |
| The 20 best memories of each kind and year | Not selected. An entry of today with a low overall could fall out before the ranking. |
| The photo of the lead in the link preview | Not selected. `webui` would need a copy of the lead rule. |
| One time zone for the site in `/digest-config.json` | Not selected. The group lives in one zone. Its browsers already agree. The file was one more request that can fail. |
| A setting for the time zone in the UI | Not selected. The group lives in one zone. The weeks of a traveller differ only near Monday 00:00. |
| A link preview in the zone of the reader | Not selected. A crawler has no zone. The preview uses `REVIEWS_TIME_ZONE`. |
| Weeks in UTC | Not selected. An entry of Monday at 00:30 CET would be in the week before. |
| The live digest of the current week as the home | Not selected. Its lead would change all week. The home shows the last finished digest. |
| Saved digests from a weekly job | Not selected. Each edit and each delete of a review would need to change them. |
| A weekly email | Not selected. The group talks in its own chats. A person shares the link of a digest there. |
| The ISO week in the URL, such as `2026-W47` | Not selected. A date is easier to read. ISO week years also have corner cases. |
| The top score first as the lead rule | Not selected. A shared subject tells more about the group. |
| The overall of one entry on the stamp of a shared lead | Not selected. The stamp would show one entry as the score of the subject. |
| A headline only for a shared lead | Not selected. Each row of the archive needs a headline. |
| `saffron` for the tag "On this day" | Not selected. The design system keeps `saffron` for the score stamp. |
| Rank numbers on the top scores | Not selected. They make the digest a leaderboard. People then avoid honest low scores. |
| An average score of the week | Not selected. A low average would read like a bad week. |
| `KindSwitch` for the whole digest | Not selected. The digest shows all kinds on purpose. "Every entry" has it. |
| Memories of one exact day | Not selected. A small group has few entries on one day. A week has about seven times more. |
| Memories of one month or six months ago | Not selected. Memories are whole years. |
| Memories by `created` | Not selected. A memory is about the meal. It uses `date`. |
| The own memories of the reader first | Not selected. The digest has no part for one person. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
