---
description: The app server writes the title, the description, and the link preview tags of each subject, review, reviewer, and list page into index.html. robots.txt keeps all crawlers out. The app has no structured data, no sitemap, and no canonical link.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00000 has this example: "I send the link of my review to a friend. The friend opens the link and reads my review."
The UI is a client-side Vue app with one `index.html` for all routes.
Chat apps do not run JavaScript when they make a link preview.
Without help from the server, a shared review shows the preview of the site, not the review.

The pages are public for friends with a link. They do not need to be in search results.
Search engines would show the names and the reviews of friends and family. AI crawlers could collect the text of the reviews.
Search engine optimization then only has costs.

Expected outcome:
- a shared subject link shows the name and the description of the subject
- a shared review link shows the title, the feedback, and the first photo of the review
- a shared reviewer or list link shows the name of the reviewer or the list
- a link to a record that does not exist gets HTTP `404`
- crawlers that obey robots.txt read no page and no photo of the site
- the pages stay open without an account

# Scope
[scope]: #scope

In scope:
- the page-meta block of `index.html` and the values that the server writes into it
- the pages with their own page meta, and the `404` of a missing record
- the Application URL warnings
- `/robots.txt`

Out of scope:
- server-side rendering of the page body
- structured data, such as schema.org JSON-LD, and a sitemap
- a canonical link, an `X-Robots-Tag` header, and a `robots` meta tag
- an `Allow` group for the link preview bots that obey robots.txt. The backlog has a check of the previews after the deploy.
- blocks of crawlers that do not obey robots.txt. They need firewall rules.
- the average scores in a description, and a cover in the preview of a subject
- page meta for the list screens, such as `/restaurants` and `/my-reviews`. They keep the tags of the site.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The page-meta block

The page-meta block is the part of `app/ui/index.html` between `<!-- page-meta -->` and `<!-- /page-meta -->`.
It has the title, the description, and the link preview tags of the site.

`GET /restaurants/k3x9m2p7q1w8z4a` for the restaurant "Casa Pepe" sends:

```html
<title>Casa Pepe · Sponga47 Reviews</title>
<meta name="description" content="Tapas and wine." />
<meta property="og:type" content="website" />
<meta property="og:title" content="Casa Pepe" />
<meta property="og:description" content="Tapas and wine." />
<meta property="og:url" content="https://reviews.sponga47.com/restaurants/k3x9m2p7q1w8z4a" />
<meta property="og:image" content="https://reviews.sponga47.com/og-image.png" />
<meta name="twitter:card" content="summary_large_image" />
```

| Page | Values |
|---|---|
| `/restaurants/{id}`, `/recipes/{id}` | the name and the description |
| `/restaurants/{id}/reviews/{id}`, `/recipes/{id}/reviews/{id}` | `og:type` `article`, the title with the name of the subject, the feedback, and the first photo |
| `/reviewers/{id}` | `og:type` `profile` and the name of the reviewer |
| `/lists/{id}` | "NAME by CREATOR" and the number of restaurants and recipes |

- `og:image` of a review with photos is the `full` file of its first photo, with its size and alt text. Other pages have the image of the site.
- a missing record, a review of a different subject, and a reviewer without approval get `404` with the default `index.html`. The UI then shows its not found screen.
- `og:url` and the photo URL start with the Application URL of the PocketBase settings

## The Application URL

- set it to the public URL of the site, such as `https://reviews.sponga47.com`
- the default `http://localhost:8090` is a host that the public cannot reach. The pages then have no `og:url` and the image of the site.
- the app logs a warning when it starts with such a URL, and when a settings save changes the URL to one

## robots.txt

```
User-agent: *
Disallow: /
```

- the rule covers the UI, the photo files under `/v1`, the dashboard, and the PocketBase API
- it also covers the AI crawlers that obey robots.txt, such as GPTBot, ClaudeBot, and CCBot
- a link preview bot that obeys robots.txt does not get the page or the photo. The chat app then shows the plain link.

| Bot | Reads robots.txt |
|---|---|
| Twitterbot of X | yes |
| LinkedInBot | yes |
| Slackbot-LinkExpanding | no |
| facebookexternalhit, Meta-ExternalFetcher | not always. A fetch that a user asks for can skip robots.txt. |
| Discordbot, TelegramBot, WhatsApp, iMessage, Signal | not known |

The friends and family share links in chat apps. X and LinkedIn are not the usual place.
Google can still show a URL without a title when a public page links to it. Few public pages link to the app.

# Implementation Details
[implementation-details]: #implementation-details

## The route

- `pagemeta.go` of `app/api/internal/webui` has the page meta. `webui.Register` reads `index.html` one time and cuts it around the block. It also reads the site name and the default `og:image` tags of the block.
- the UI route sends its own page only for the paths above, with an optional trailing slash. The path words ignore case, like Vue Router. Each ID must match `^[a-z0-9]{15}$`.
  `/restaurants/new` does not match. The edit and new pages need an account and keep the default tags.
- `html/template` writes the values. A name such as `<script>` stays text.
- the description puts all whitespace on one line. A text of more than 200 characters is cut at the end of a word and ends with `…`.
- without a description or feedback, the description is `Reviews of NAME on Sponga47 Reviews.` or `A review of NAME on Sponga47 Reviews.`
- the `<title>` has the format of `usePageTitle` in the UI. The end-to-end tests compare the two titles.
- on a database error other than a missing record, the app logs the error and sends the default `index.html` with `200`
- `publicapi.PhotoFilePath` makes the photo path. The API responses use the same function.
- a host that the public cannot reach is `localhost`, a name that ends in `.localhost`, or a loopback, private, link-local, or unspecified IP address
- the URLs use only the scheme, the host, and the path of the Application URL
- a hook on `OnSettingsReload` checks the Application URL after each settings save. It warns only when the URL changed.
- a block without `<title>` or `og:image` is not used. The app then logs a warning, and the pages have the tags of the site.
- a Go test checks that `app/ui/index.html` has the block. A Vite change that removes the comments then fails `make test-app`.
- the route does not add content to the body. Vue removes the content of `#app` when it mounts, and two versions of a heading would make the end-to-end tests unstable.

## robots.txt

- `robotsTxt` of `robots.go` has the two lines. The route sends `Cache-Control: public, max-age=3600`.
- `TestRobotsTxt` compares the body with the two lines
- `TestPageMeta` and `tests/e2e/link-previews.spec.ts` check that no page has a canonical link

## Tests

- `pagemeta_test.go` checks the tags, the escaping, the photo, the `404` cases, and the paths that keep the default tags
- `tests/e2e/link-previews.spec.ts` checks the tags of the built app, the photo file, the titles, and the not found screen
- the end-to-end app has the Application URL `https://reviews.example.com`. `tests/app.ts` sets it through `PATCH /api/settings`.

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| Server-side rendering of the Vue app | Not selected. It needs a JavaScript runtime in the server or a second server. |
| A prerender service for bots | Not selected. It is a second deploy, and user agent detection is not reliable. |
| Tags that the UI sets at runtime | Not selected. Link preview bots do not run JavaScript. |
| schema.org JSON-LD and a sitemap | Removed. The app does not need search engine optimization. Each page needed more queries, and each new field needed a schema.org property. |
| A canonical link on each page | Removed. It helped search engines show the pages of friends and family. |
| A robots.txt that allowed the pages and the photos | Removed. Search engines could show the names and reviews of friends and family. |
| `X-Robots-Tag: noindex` | Not selected. A crawler that obeys `Disallow: /` does not get the page, so it does not see a noindex. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
