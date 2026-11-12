---
description: The app server writes the title, description, and link preview tags of restaurant and review pages into index.html.
status: implemented
---

# Motivation
[motivation]: #motivation

The Public UI is a client-side Vue app. Decision 00005 serves the same `index.html` for all UI routes.
`index.html` has one title, one description, and one set of Open Graph tags.

Chat apps and social sites do not run JavaScript when they make a link preview.
A shared restaurant or review then shows the preview of the site, not the restaurant or the review.
Some search engines also do not run JavaScript. They see only the title and the description of the site.

Expected outcome:
- A shared restaurant link shows the name and the description of the restaurant.
- A shared review link shows the title and the feedback of the review, and its first photo.
- Each restaurant and review page has a canonical URL.
- A link to a restaurant or review that does not exist gets HTTP `404`.
- The UI, the Content Security Policy, and the Vite dev server do not change.

# Scope
[scope]: #scope

In scope:
- The `<title>`, `description`, canonical link, Open Graph tags, and Twitter tags of `/restaurants/{id}` and `/restaurants/{id}/reviews/{id}`
- The HTTP status of these pages when the record does not exist
- The escaping of the values

Out of scope:
- Server-side rendering of the page body. See Implementation Details.
- Structured data, such as a schema.org `Restaurant` in JSON-LD. A later decision can add it in the same block.
- The average scores in the description of a restaurant. They need the rating query of the Public API in `app/internal/webui`.
- Tags for the other screens, such as `/restaurants` and `/my-reviews`. They keep the tags of the site.
- A sitemap
- Showing average score in a restaurant's description

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

New concepts:
- **page-meta block**: the part of `app/ui/index.html` between `<!-- page-meta -->` and `<!-- /page-meta -->`. It has the title, the description, and the link preview tags of the site.
- **page meta**: the values that the app server writes into the page-meta block for one restaurant or review.

`app/ui/index.html`:

```html
<meta property="og:site_name" content="Sponga47 Reviews" />
<!-- page-meta -->
<title>Sponga47 Reviews</title>
<meta name="description" content="Find a restaurant. Score it. Say why." />
<meta property="og:title" content="Sponga47 Reviews" />
...
<!-- /page-meta -->
```

`GET /restaurants/k3x9m2p7q1w8z4a` for the restaurant "Casa Pepe" sends:

```html
<title>Casa Pepe · Sponga47 Reviews</title>
<meta name="description" content="Tapas and wine." />
<link rel="canonical" href="https://reviews.sponga47.com/restaurants/k3x9m2p7q1w8z4a" />
<meta property="og:type" content="website" />
<meta property="og:title" content="Casa Pepe" />
<meta property="og:description" content="Tapas and wine." />
<meta property="og:url" content="https://reviews.sponga47.com/restaurants/k3x9m2p7q1w8z4a" />
<meta property="og:image" content="https://reviews.sponga47.com/og-image.png" />
...
<meta name="twitter:card" content="summary_large_image" />
<meta name="twitter:title" content="Casa Pepe" />
<meta name="twitter:description" content="Tapas and wine." />
```

A review page has `og:type` `article`, the title of the review, and its feedback.
`og:title` adds the name of the restaurant, for example `Great grill · Preview Grill`.
If the review has photos, `og:image` is the `full` file of the first photo, with its size and alt text.

The canonical URL, `og:url`, and the photo URL start with the Application URL of the PocketBase settings.
In production, set it to the public URL of the site, such as `https://reviews.sponga47.com`.
An Application URL on a host that the public cannot reach has no use for link previews. The default `http://localhost:8090` is such a URL.
With it, the pages have no canonical URL, no `og:url`, and the image of the site.
The app logs a warning when it starts to serve with such a URL.
After the start, it logs a warning when a settings save changes the URL to such a URL.

The default `og:image` and its size and alt text come from the page-meta block of `index.html`.
The site name in the titles comes from its `<title>`.

# Implementation Details
[implementation-details]: #implementation-details

## Where the tags come from

| Option | Result |
| --- | --- |
| The Go UI route replaces the page-meta block of the embedded `index.html` | Selected. The UI, the build, and the policy do not change. One request reads at most three records. |
| Server-side rendering of the Vue app | Not selected. It needs a JavaScript runtime in the server or a second server. Decision 00005 puts it out of scope. |
| A prerender service for bots | Not selected. It is a second deployment, and user agent detection is not reliable. |
| The UI sets the tags at runtime | Not selected. Link preview bots do not run JavaScript. The UI already sets `document.title`. |

## Route

- `webui.Register` reads `index.html` one time and cuts it around the page-meta block. It also reads the site name and the default `og:image` tags of the block.
- the reader of the default tags accepts any attribute order and quote style, and tag names in upper case
- a host that the public cannot reach is `localhost`, a name that ends in `.localhost`, or a loopback, private, link-local, or unspecified IP address
- the URLs use only the scheme, the host, and the path of the Application URL. A user name, a password, a query, or a fragment is not used.
- a hook on `OnSettingsReload` checks the Application URL after each settings save. It warns only if the URL changed. CLI commands, such as `superuser upsert`, do not serve, so they do not warn.
- the UI route sends its own page only for `restaurants/{id}` and `restaurants/{id}/reviews/{id}`, with an optional trailing slash. `restaurants` and `reviews` ignore case, like Vue Router. Each `{id}` must match the `RecordId` pattern of the spec, `^[a-z0-9]{15}$`.
  - `/restaurants/new` does not match the pattern. It gets the default `index.html`.
  - the edit and new review pages need an account. They keep the default tags.
- `html/template` writes the values. It escapes them for the attribute or text context. A restaurant name such as `<script>` stays text.
- the description puts all whitespace on one line. A text of more than 200 characters is cut at the end of a word and ends with `…`.
- without a description or feedback, the description is `Reviews of NAME on Sponga47 Reviews.` or `A review of NAME on Sponga47 Reviews.`
- the `<title>` has the format of `usePageTitle` in the UI. The title does not change when the UI starts. The end-to-end tests compare the two titles.
- the headers of decision 00005 do not change: the CSP, `Reporting-Endpoints`, and `Cache-Control: no-cache`

## Missing records

- a restaurant that does not exist, a review that does not exist, and a review of a different restaurant get `404`
- the body is the default `index.html`. The UI then shows its not found screen.
- search engines then do not index the URL as a page
- on another database error, the app logs the error and sends the default `index.html` with `200`

## Without a page-meta block

- the placeholder page of decision 00005 has no page-meta block. A block without `<title>` or `og:image` is not used.
- if the UI build has no usable block, the app logs a warning when it starts. Pages then have the tags of the site.
- `make test-app` checks the block of the UI build when the build exists The UI route then sends it for all paths.
- a Go test checks that `app/ui/index.html` has the block and its tags. A Vite change that removes the comments then fails `make test-app`. The end-to-end tests check the built page.

## Server-rendered content for crawlers

The UI route does not add content to the page body.
- Vue removes the content of `#app` when it mounts. Users see the content for a short time before the UI starts.
- Google runs JavaScript, so it sees the rendered page. Link preview bots read only the head.
- the end-to-end tests find elements by role and name. Two versions of the same heading can make them unstable.

A later decision can add JSON-LD in the page-meta block. JSON-LD is a data block, so the CSP does not block it.

## Photos

- `og:image` is `/v1/restaurants/{id}/reviews/{id}/photos/{id}/full` of the first photo by `created` and `id`. This is the order of the API.
- `publicapi.PhotoFilePath` makes the path. The API responses use the same function.
- the photo file route needs no account, so link preview bots can get it. Decision 00008 has its headers.

## Tests

- `app/internal/webui/pagemeta_test.go` checks the tags, the escaping, the photo, the `404` cases, and the paths that keep the default tags
- `tests/e2e/link-previews.spec.ts` checks the tags of the built app in a browser, the photo file, the titles, and the not found screen after a `404`
- the end-to-end app has the Application URL `https://reviews.example.com`. `tests/app.ts` makes a superuser before the start and sets the URL through `PATCH /api/settings`. A failure of these steps stops the start with its own error.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
