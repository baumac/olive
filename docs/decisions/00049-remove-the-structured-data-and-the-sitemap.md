---
description: Remove the schema.org JSON-LD of the pages and the sitemap of decision 00021. The app does not need search engine optimization. robots.txt stays without its Sitemap line.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00021 added schema.org JSON-LD to the restaurant, recipe, review, reviewer, and list pages. It also added `/robots.txt` and `/sitemap.xml`.

The app does not need search engine optimization. The structured data and the sitemap then only have costs:

- each restaurant, recipe, and reviewer page needs one more database query.
- each review page reads the user record of the author.
- the sitemap query reads all restaurants, recipes, reviews, and lists.
- each new field of a page needs a decision about its schema.org property. Decisions 00032, 00033, 00038, and 00047 had to do this.

Expected outcome:
- the pages have no `application/ld+json` script
- `/sitemap.xml` is not a route of the app
- `/robots.txt` stays without its `Sitemap` line
- the link preview tags of decision 00013 do not change

# Scope
[scope]: #scope

In scope:
- the removal of `structureddata.go` and `sitemap.go` in `app/api/internal/webui`
- the removal of the `Sitemap` line of `/robots.txt`
- the tests

Out of scope:
- the link preview tags, the canonical link, and the page titles. Decision 00013 has them. Chat apps and social sites use them.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

The page-meta block has only the tags of decision 00013.

## robots.txt

`/robots.txt` stays. It is not search engine optimization.

- It keeps crawlers out of the dashboard, the PocketBase API, and the account screens.
- The photo files under `/v1` stay open. Link preview bots that obey `robots.txt` need them.
- It has no `Sitemap` line. It does not need a public Application URL.

```
User-agent: *
Disallow: /_/
Disallow: /api/
Disallow: /sign-in
Disallow: /sign-up
Disallow: /my-reviews
Disallow: /account
Disallow: /confirm-email
Disallow: /forgot-password
Disallow: /reset-password
```

## sitemap.xml

`/sitemap.xml` gets the same response as other unknown paths of the UI.

# Implementation Details
[implementation-details]: #implementation-details

- `robots.go` in `app/api/internal/webui` has the robots.txt route. It keeps `Cache-Control: public, max-age=3600`.
- `reviewedKind` has no schema.org type and no data function.
- The reviewer page needs no review count. The review page does not read the author. The list page still counts its items for the description.
- The end-to-end tests do not check the JSON-LD and the sitemap.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
