# Holistic review of 2026-11-13

This file is not a decision. It is a list of suggestions from a review of `main` at dfa152c.
A suggestion that is accepted needs a decision or a backlog item.

Method: read the spec, the decisions, the backlog, the Go packages, and the UI source.
Ran `./bin/reviews-app` on a scratch database with 3 users, 4 restaurants, 6 reviews.
Looked at the list, map, restaurant, review form, reviewer, and sign up pages on desktop and on mobile in dark mode.
Did not run the test suites. Did not test photo upload or the place search in the browser.

## Product, highest value first

1. The overall ranking has no confidence weight. `overallOf` in `app/migrations/1794268808_add_restaurant_index.go` is a plain average.
   One review of 10 beats 200 reviews of 9.4 in the `-overall` sort. Use a Bayesian average for the sort key, and keep the plain average for display.
2. The default list sort is `-created`. A discovery app wants best rated or nearest first. The API has `near`, but the UI never asks for `navigator.geolocation`.
3. Photos are only on review cards. The restaurant card and the top of the restaurant page have none. A cover photo and a gallery from the review photos need no new upload.
4. The "Won't Do" list is one cluster: no email verification, no write rate limits, no reports, no password reset.
   Together they mean a public instance has no defence against spam and no recovery for a lost password. The cheapest slice is a report button that makes a record for the dashboard, plus per-user write limits.
5. A restaurant has only name, description, tags, location. Website, phone, opening hours, and price level are the facts people look for. Photon returns some OSM tags.
6. Reviews have no visit date, no helpful vote, and no owner or reader reply. Helpful votes also give the review list a better default sort than newest.
7. Reviewer page has only a count. Add the average score of the reviewer and the restaurants they added (00014 left this out).
8. "Price" where 10 is good needs a footnote on each page. Rename the label to "Value".

## UI/UX

- Review cards show six dot rows each. Three reviews fill two screens. Show the overall and collapse the five scores, or show them as one compact row.
- The review form has 50 score buttons, and the photos section is below the Save button. Move photos above Save. Show the computed overall live.
- Mobile restaurant page: the map sits above the rating summary. The rating and "Write a review" are below the fold.
- Map view has no marker clustering. Two pins overlap at Barceloneta in the test data. It loads at most 100 restaurants in the box with no cue except `mapHasMore`.
- "Open in OpenStreetMap" only. Add a `geo:` or directions link for phones.
- `/` redirects to the list. There is no landing content for a first visitor who is signed out.
- All text is English with `en-GB` dates fixed in `app/ui/src/lib/format.ts`. No i18n layer.

## SEO and sharing

- `pagemeta.go` writes title and Open Graph tags but no schema.org JSON-LD (`Restaurant` with `AggregateRating`, `Review`). This is what gives star snippets in search results.
- No `sitemap.xml` and no `robots.txt` route. The content is only reachable through client-side lists.

## Code and libraries

- `github.com/disintegration/imaging v1.6.2` has had no release since 2019, as far as the reviewer recalls. Check upstream. It decodes untrusted uploads in `photoimage.go`. Replace with `golang.org/x/image/draw` plus a small EXIF orientation reader, or a maintained fork.
- The auth token is in `localStorage` (`app/ui/src/lib/session.ts`). The CSP is strict, so the risk is low. An HttpOnly cookie for the UI would remove it.
- Timestamps are the PocketBase format, not RFC 3339. Each client needs `parseTimestamp`. Change it before an iOS client exists.
- Review `overall` sort and filter scan. Fine now. A generated column with an index fixes it when lists grow.
- Rate limiters and the place cache are in memory. Correct for one instance, as the backlog says. State it in the deploy decision.
- No Dockerfile, deploy folder, or backup yet. This is the high priority backlog item and blocks everything public.
- CI has no coverage number and no dependency update bot. Both are on the backlog as low priority. Renovate matters more now that versions are pinned exactly.

## What is strong

Spec-first with generated SDKs and request and response validation, decision records, a11y tests, strict CSP, query plan tests, e2e on a matrix. The engineering base is ahead of the product surface.
