# Holistic review of 2026-11-13

This file is not a decision. It is a list of suggestions from a review of `main` at dfa152c.
A suggestion that is accepted needs a decision or a backlog item.

Method: read the spec, the decisions, the backlog, the Go packages, and the UI source.
Ran `./bin/reviews-app` on a scratch database with 3 users, 4 restaurants, 6 reviews.
Looked at the list, map, restaurant, review form, reviewer, and sign up pages on desktop and on mobile in dark mode.
Did not run the test suites. Did not test photo upload or the place search in the browser.

Updated on 2026-11-14 for decision 00019, which made the UI simpler. The items that it completed are removed.
It completed the compact score display, the "Value" label, the photos above the save button, and the map position on a narrow page.
Updated again on 2026-11-14 for decision 00020, which fixed most items of the UI/UX section.
Updated again on 2026-11-14 for decision 00021, which fixed the items of the SEO and sharing section.
Updated again on 2026-11-14 for decisions 00022 to 00025, which fixed most items of the Code and libraries section.

## Product, highest value first

1. The overall ranking has no confidence weight. `overallOf` in `app/migrations/1794268808_add_restaurant_index.go` is a plain average.
   One review of 10 beats 200 reviews of 9.4 in the `-overall` sort. Use a Bayesian average for the sort key, and keep the plain average for display.
2. The default list sort is `-created`. A discovery app wants best rated or nearest first. The API has `near`, but the UI never asks for `navigator.geolocation`.
3. Photos are only on review cards. The restaurant card and the top of the restaurant page have none. A cover photo and a gallery from the review photos need no new upload.
4. The "Won't Do" list is one cluster: no email verification, no write rate limits, no reports, no password reset.
   Together they mean a public instance has no defence against spam and no recovery for a lost password. The cheapest slice is a report button that makes a record for the dashboard, plus per-user write limits.
   Decision 00019 removed the "Confirm password" field of sign up. A typing error in the password now loses the account, so the password reset matters more.
5. A restaurant has only name, description, tags, location. Website, phone, opening hours, and price level are the facts people look for. Photon returns some OSM tags.
6. Reviews have no visit date, no helpful vote, and no owner or reader reply. Helpful votes also give the review list a better default sort than newest.
7. Reviewer page has only a count. Add the average score of the reviewer and the restaurants they added (00014 left this out).

## UI/UX

Decision 00020 completed the other items of this section on 2026-11-14.
It added the overall score to the review form, the scores to a review card, map clusters, the directions link, the old sort links, and the intro for a signed out visitor.

- The review form still has 50 score buttons. Decision 00020 keeps them, because one click selects a score.
- The map view loads at most 100 restaurants in the box. A line below the map now tells this, but the person must zoom in to see the others.
- The restaurants list has no visible heading and the header has no navigation links. A second top-level screen needs both again.
- `/` redirects to the list. A signed out visitor sees only two sentences of intro. There is no landing page.
- The directions link selects the map app from the user agent. This is not tested on a real phone.
- All text is English with `en-GB` dates fixed in `app/ui/src/lib/format.ts`. No i18n layer.

## SEO and sharing

Decision 00021 completed the two items of this section on 2026-11-14.
It added schema.org JSON-LD to the restaurant and review pages, and the `/robots.txt` and `/sitemap.xml` routes.

- The sitemap is one file with at most 50,000 URLs. A larger site needs a sitemap index.
- The JSON-LD is not checked with the Rich Results Test of Google. It needs a public URL.
- The restaurants list and the reviewer pages have no structured data.

## Code and libraries

Decisions 00022 to 00025 completed most items of this section on 2026-11-14.
They replaced the `imaging` library in the photo upload, changed the timestamps to RFC 3339, added the deploy folder with backups, and added the coverage check and Renovate.
Decision 00022 states that the rate limiters and the place cache need one instance.

- The auth token is in `localStorage` (`app/ui/src/lib/session.ts`). The CSP is strict, so the risk is low. An HttpOnly cookie for the UI would remove it. It needs a spec change. The backlog has the item.
- Review `overall` sort and filter scan. Fine now. The backlog has the item.
- `github.com/disintegration/imaging` is still in `app/go.sum`, because PocketBase needs it. The app does not call it.
- The backups stay on the host. The operator must copy them to a different computer. The restore steps and the `caddy` service are not tested on a real server.
- Renovate needs its GitHub app on the repository. It does not update the tool versions of the makefile and the image tags of `deploy/`.

## What is strong

Spec-first with generated SDKs and request and response validation, decision records, a11y tests, strict CSP, query plan tests, e2e on a matrix. The engineering base is ahead of the product surface.
