---
description: The app server writes schema.org JSON-LD into restaurant and review pages, and serves robots.txt and sitemap.xml.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00013 gives each restaurant and review page its title and link preview tags. It left structured data and a sitemap out of scope.

- A search engine cannot show the score of a restaurant in its results. It needs schema.org data for that.
- The restaurant and review pages are reachable only through lists that JavaScript loads. A crawler that does not run JavaScript finds no page.
- The site has no `robots.txt`. Crawlers then also get the dashboard and the sign in page.

Expected outcome:
- a restaurant page tells its name, address, average score, and review count as data
- a review page tells its title, text, score, author, and restaurant as data
- a crawler gets the URL of each restaurant and each review from one file

# Scope
[scope]: #scope

In scope:
- JSON-LD in the page-meta block of `/restaurants/{id}` and `/restaurants/{id}/reviews/{id}`
- the `/robots.txt` and `/sitemap.xml` routes
- the tests

Out of scope:
- the Public API and the UI
- structured data for the restaurants list and the reviewer pages
- a sitemap index. One sitemap file has at most 50,000 URLs.
- the reviewer pages in the sitemap. They have only a name and a list.
- the reviews of a restaurant inside its JSON-LD. Each review has its own page with its data.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Structured data

`GET /restaurants/k3x9m2p7q1w8z4a` has this element in its head. The example has line breaks for the reader.

```html
<script type="application/ld+json">
{"@context":"https://schema.org","@type":"Restaurant",
 "name":"Casa Pepe","description":"Tapas and wine.",
 "url":"https://reviews.sponga47.com/restaurants/k3x9m2p7q1w8z4a",
 "address":"Carrer Nou 1, Barcelona",
 "geo":{"@type":"GeoCoordinates","latitude":41.39,"longitude":2.16},
 "aggregateRating":{"@type":"AggregateRating","ratingValue":7.4,"reviewCount":12,"worstRating":1,"bestRating":10}}
</script>
```

- A restaurant without reviews has no `aggregateRating`.
- A restaurant without a location has no `address` and no `geo`.
- `ratingValue` is the `rating.overall` of the API.

A review page has a `Review` object.

- `name` is the title, and `reviewBody` is the full feedback.
- `reviewRating` has the `overall` value of the review with `worstRating` 1 and `bestRating` 10.
- `author` is a `Person` with the name and the URL of the reviewer page.
- `itemReviewed` is the `Restaurant` with its name, URL, and address.
- `datePublished` is the creation time.

Without a public Application URL, the objects have no `url`.

## robots.txt

```
User-agent: *
Disallow: /_/
Disallow: /api/
Disallow: /sign-in
Disallow: /sign-up
Disallow: /my-reviews

Sitemap: https://reviews.sponga47.com/sitemap.xml
```

The paths under `/v1` stay open. Link preview bots that obey `robots.txt` need the photo files there.
Without a public Application URL, the file has no `Sitemap` line.

## sitemap.xml

The sitemap lists `/restaurants`, each restaurant page, and each review page. Each record has its `updated` time as `lastmod`.
The records with the newest change come first. The file stops at 50,000 URLs.
Without a public Application URL, `/sitemap.xml` gets `404`. A sitemap needs absolute URLs.

# Implementation Details
[implementation-details]: #implementation-details

- `structureddata.go` in `app/internal/webui` makes the objects. `sitemap.go` has the two routes.
- `encoding/json` writes the JSON-LD. It escapes `<`, `>`, and `&`, so a value cannot end the script element.
- the browser does not run a script of the type `application/ld+json`. The Content Security Policy does not change. The end-to-end tests check this.
- the restaurant object needs one more query. It reads the review count and the row of `restaurant_overall`. The review object reads the user record of the author.
- a database error of the rating query gives the default page, like the other errors of decision 00013.
- `/robots.txt` and `/sitemap.xml` have `Cache-Control: public, max-age=3600`. The sitemap query reads all restaurants and reviews, so a proxy cache can keep it.
- the UI route checks the two paths before the static files. A `robots.txt` in the UI build is not used.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
