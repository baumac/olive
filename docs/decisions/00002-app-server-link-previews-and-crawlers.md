---
description: One PocketBase app serves the Public API, the PocketBase API, the PocketBase dashboard, and the embedded UI build. It runs the request checks, the transactions, the list queries, and the search, sends a strict Content Security Policy, writes the link preview tags of each page, and keeps all crawlers out with robots.txt.
status: implemented
---

# Motivation
[motivation]: #motivation

The app runs on one home server. See decision 00000.
Two servers would need two deploys, CORS settings, and a second origin for the session.
One PocketBase app then serves all parts:
- the Public API of decision 00001
- the PocketBase API and the dashboard for the admins
- the build of the Public UI

PocketBase gives the database, the auth records, the migrations, the dashboard, and the file storage.
The app adds the Public API on top of it. Some choices of this layer are not in the spec. This decision records them.

The UI shows text from users.
A script that an attacker injects into the page could act as the user.
Vue escapes text by default. A bug in a dependency can still let a script in.
A strict Content Security Policy stops the browser from running such a script.

Decision 00000 has this example: "I send the link of my review to a friend. The friend opens the link and reads my review."
The UI is a client-side Vue app with one `index.html` for all routes.
Chat apps do not run JavaScript when they make a link preview.
Without help from the server, a shared review shows the preview of the site, not the review.

The pages are public for friends with a link. They do not need to be in search results.
Search engines would show the names and the reviews of friends and family. AI crawlers could collect the text of the reviews.
Search engine optimization then only has costs.

Expected outcome:
- one binary serves the Public UI, the Public API, the PocketBase API, and the PocketBase dashboard
- the UI and the Public API have one origin. The API needs no CORS settings.
- the Public API implements `reviewsapi.StrictServerInterface` of decision 00001
- the server applies the migrations when it starts
- `migrations/1794096000_init.go` makes all collections and tables. It has all changes from before the first deploy.
- a data folder from the migrations before it has the collections already. The init migration then stops the start with an error. Delete the folder.
- a change of the schema after the first deploy needs a new migration. Do not change the init migration.
- developers run the UI with the Vite dev server and hot reload
- each response of the UI route has a Content Security Policy that allows scripts and styles only from the same origin
- a shared subject link shows the name and the description of the subject
- a shared review link shows the title, the feedback, and the first photo of the review
- a shared reviewer or list link shows the name of the reviewer or the list
- a link to a record that does not exist gets HTTP `404`
- crawlers that obey robots.txt read no page and no photo of the site
- the pages stay open without an account

# Scope
[scope]: #scope

In scope:
- the layout of `app/api` and its `make` targets
- the routes of the app, and how the Public API and the UI mount on the PocketBase router
- the body limits, the order of the request checks, and the transactions
- the list queries, the cursors, and the search tables
- the PocketBase mapping that all collections share
- the embedded UI build, its routes, and its cache headers
- the Content Security Policy of the UI route and its reports
- the page-meta block of `index.html` and the values that the server writes into it
- the pages with their own page meta, and the `404` of a missing record
- the Application URL warnings
- `/robots.txt`

Out of scope:
- the rules of the spec. Decision 00001 has them.
- the auth tokens, the cookie, and the rate limits. Decision 00003 has them.
- the collections of each topic. The topic decisions have them.
- the map settings of the UI route. Decision 00008 has them.
- the screens and the components of the UI. Decision 00009 has them.
- the deploy. Decision 00010 has it.
- server-side rendering of the page body, a CDN, and a separate static host
- more than one instance. See decision 00010.
- structured data, such as schema.org JSON-LD, and a sitemap
- a canonical link, an `X-Robots-Tag` header, and a `robots` meta tag
- an `Allow` group for the link preview bots that obey robots.txt. A later decision can add it after a check of the previews on the deployed site.
- blocks of crawlers that do not obey robots.txt. They need firewall rules.
- the average scores in a description, and a cover in the preview of a subject
- page meta for the list screens, such as `/` and `/restaurants`. They keep the tags of the site.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

```bash
make run-app    # Serve on 127.0.0.1:8090 with data in app/api/pb_data
make build-ui   # Build the UI into app/api/internal/webui/dist
make build-app  # Build the UI, then the binary ./bin/reviews-app
make dev-ui     # Run the Vite dev server. It sends /v1 to make run-app.
make test-app   # Run the Go tests of the app
```

For UI work, run `make run-app` and `make dev-ui` at the same time, and open the Vite URL.
`make run-app` serves the UI build that `make build-ui` made last.

## Routes

| Path | Handler |
|---|---|
| `/v1/{path...}` | the Public API |
| `/api/{path...}` | the PocketBase API |
| `/_/{path...}` | the PocketBase dashboard |
| `/csp-report` | the reports of the Content Security Policy |
| `/{path...}` | the UI |

- the UI route is the least specific route. The router selects the other routes first.
- a path that is not a file of the UI build gets `index.html`. The UI router then shows the screen or a not found screen.
- an unknown path under `/v1` gets the Problem Details `404` of the Public API, not `index.html`
- the UI uses the HTML5 history mode of Vue Router. Paths have no `#`.

## Layout

```
app/api/
  go.mod                       # module github.com/baumac/reviews-app/app/api
  main.go                      # the PocketBase app
  migrations/                  # the Go migrations of the collections
  internal/
    publicapi/                 # the Public API
    searchindex/               # the hooks of the search tables
    history/                   # the versions of decision 00004
    recordvalues/              # the API values of restaurants and recipes
    subjects/                  # the table names of each kind
    photocleanup/              # the job of decision 00007
    usercleanup/               # the job of decision 00003
    testapp/                   # the test database of decision 00010
    webui/
      webui.go                 # embeds dist/ and adds the UI route
      dist/                    # the UI build. Only .gitkeep is in git.
app/ui/                        # the Vue app, an npm workspace
```

`publicapi.Register(app, publicapi.DefaultOptions)` adds the Public API to a PocketBase app.
`webui.Register` adds the UI route.

## Content Security Policy

Each response of the UI route has a `Content-Security-Policy` header.
`contentSecurityPolicy` in `app/api/internal/webui/webui.go` makes it.

The UI must not add:
- inline scripts, `eval`, or `new Function`
- inline `<style>` elements or `style="..."` attributes. Vue `:style` bindings are allowed, because they set styles through the CSSOM.
- files from other origins, such as fonts or images from a CDN. The map tiles of decision 00008 are the one exception.
- `data:` URLs. `assetsInlineLimit: 0` in `app/ui/vite.config.ts` makes Vite emit each asset as a file.

A change that breaks one of these rules fails the end-to-end tests with the text of the violation.

## Link previews and crawlers

### The page-meta block

The page-meta block is the part of `app/ui/index.html` between `<!-- page-meta -->` and `<!-- /page-meta -->`.
It has the title, the description, and the link preview tags of the site.

`GET /restaurants/k3x9m2p7q1w8z4a` for the restaurant "Casa Pepe" sends:

```html
<title>Casa Pepe · Sponga47 Food</title>
<meta name="description" content="Tapas and wine." />
<meta property="og:type" content="website" />
<meta property="og:title" content="Casa Pepe" />
<meta property="og:description" content="Tapas and wine." />
<meta property="og:url" content="https://food.sponga47.com/restaurants/k3x9m2p7q1w8z4a" />
<meta property="og:image" content="https://food.sponga47.com/og-image.png" />
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

### The Application URL

- set it to the public URL of the site, such as `https://food.sponga47.com`
- the default `http://localhost:8090` is a host that the public cannot reach. The pages then have no `og:url` and the image of the site.
- the app logs a warning when it starts with such a URL, and when a settings save changes the URL to one

### robots.txt

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

## Mounting the Public API

- an `OnServe` hook adds the route `/v1/{path...}`. The route calls the handler that `reviewsapi.HandlerWithOptions` makes on its own `http.ServeMux`.
- the route puts the client IP of `RequestEvent.RealIP` in the request context. `RealIP` uses the trusted proxy settings of PocketBase. See decision 00010.
- the default PocketBase middlewares, such as the panic recovery, also apply to `/v1`
- a known path with a different method gets `405` with an `Allow` header

## Body limits

- `limitBody` of the `/v1` route replaces the body limit of PocketBase. It has the same ID, `pbBodyLimit`.
- the PocketBase limit is 32 MB, and its `413` is not Problem Details
- `limitBody` limits each body under `/v1` to 10 MiB, the largest photo upload of decision 00007. A larger `Content-Length` gets `413` before the handler.
- `readBody` limits each JSON body to 256 KB. A body without `Content-Length` fails at the first read over 256 KB with the same `413`.
- the largest valid JSON body is about 100 KB. `TestBodyLimit` sends bodies of 33 MB with and without `Content-Length`.

## Request checks

The `prepare` middleware runs before the generated strict handler. The spec gives the checks of each operation. Decision 00001 has the spec validation.

| Order | Check | Status |
|---|---|---|
| 1 | a rate limit with the key `ip` | `429` |
| 2 | the token. `security: - {}` ignores it. `{}` with a scheme makes it optional. A scheme alone makes it required. | `401` |
| 3 | the approval of the user, without `x-allow-unapproved` | `403` |
| 4 | a rate limit with the key `user` | `429` |
| 5 | the names of the query parameters | `400` |
| 6 | the path and query parameters against the spec | `400` |
| 7 | the content type of the body | `415` |
| 8 | the body is one JSON object without unknown, read-only, or repeated properties | `400` |
| 9 | the body properties against the spec | `422` |
| 10 | the sort, the search, and the cursor rules that the spec cannot describe | `400` |
| 11 | the records of the path exist | `404` |
| 12 | the rules of the topic, such as the creator check of decision 00004 | `403` |
| 13 | conflicts with stored data | `409` |

- checks 1 to 9 are in `prepare`. Checks 10 to 13 are in the operation handlers.
- an operation without a spec operation gets `500`. A test checks that each generated operation has one.
- the parameter check gives kin-openapi a copy of the request without the body. The security check of kin-openapi reads the full body, also with `ExcludeRequestBody`.
- the decoders after `prepare` read the first copy of a repeated property. `prepare` then refuses repeated properties.
- the spec gives the maximum number of values of each array parameter. Each value is one SQL parameter.

## Auth

- the `loadAuthToken` middleware of PocketBase checks the token. The record must be in the `users` collection. A superuser token is then not valid for the Public API.
- PocketBase treats a request with an invalid token as anonymous. The Public API gives `401` instead. See decision 00001.
- `closeUserAccountRoutes` of `accountroutes.go` gives `404` for the PocketBase auth and account routes of `users`, such as `auth-with-password` and `request-password-reset`.
  These routes would skip the rate limits and the rules of the Public API. The check runs before PocketBase looks up a record, so the response is the same for all email addresses.
- decision 00003 has the tokens, the cookie, and the approval

## Transactions

- each write of a restaurant, a recipe, a review, a photo, or a list runs in one transaction. The transaction reads the records again, checks the rules, and saves.
- PocketBase has one SQLite writer. A parallel request then cannot break a `409` check.
- a unique index also catches a duplicate. The API changes its error to `409`, not `500`.

## Lists

- pages use a keyset. The cursor is base64url JSON with the sort values of the last item and a fingerprint.
- the fingerprint is a hash of the filters, the sort, and the parent ID. `limit` is not in it, so a client can change `limit` between pages.
- the list query selects the IDs and the sort values of a page. A second query gets the records by ID. A write between the two queries cannot skip or repeat items.
- the `id` key has the direction of the key before it. When all keys have one direction, the cursor condition is a row value comparison, such as `(created, id) < (x, y)`. SQLite can then use an index.
- mixed directions use an `OR` condition
- a cursor value must be a string or a number. Other values get `400`.
- the `name` and `title` sorts use `COLLATE NOCASE`. Their indexes are on `(name COLLATE NOCASE, id)` and `(title COLLATE NOCASE, id)`.
- `creator` and the parent name of a review come from one query for all records of a page
- a cursor keeps the form of its first fingerprint. Pages that were open before a deploy can still load more, unless the deploy changed the sort keys.

## Search

- each kind has an FTS5 table: `restaurants_fts`, `reviews_fts`, `recipes_fts`, and `recipe_reviews_fts`. Each row has the record ID in an `UNINDEXED` column.
- the `unicode61` tokenizer with `remove_diacritics 2` ignores case and accents. The `prefix = '2 3'` indexes make short prefix terms fast.
- `internal/searchindex` keeps the tables equal to the records. Each record hook runs the save or the delete and the index change in one transaction.
  - the hook starts a transaction when the caller has none. A failed index change then also removes the record change.
  - an update writes the index row only when an indexed field changes
  - the hooks also run for saves in the dashboard and for cascade deletes, because these go through `app.Save` and `app.Delete`
  - a review row has the name of its parent. A rename of the parent updates the review rows.
- the API changes a search into an FTS5 query. Each word with a letter or a digit becomes a quoted term, such as `"carb"*`. A quote in a word is doubled. FTS5 syntax in a search is then plain text.
- a word of one character is a whole-word term without `*`. The prefix indexes do not cover one character.
- the list query has an `INNER JOIN` on the matches with their score. The score adds the weight of each column that matches, with an FTS5 column filter such as `name : ("carb"*)`.
- the cursor of a `relevance` sort has the score of the last item
- the tokenizer keeps a word of a script without spaces as one word. `ラーメン屋` is one word.

## PocketBase mapping

- the API rules of all collections are empty. Only superusers use the collections through the PocketBase API. Without this, users could skip the rules of the Public API.
- the spec uses the PocketBase `id`, `created`, and `updated` fields. PocketBase stores the timestamps with a space, such as `2024-11-04 01:26:55.002Z`. Decision 00001 changes them to RFC 3339.
- PocketBase stores an empty string for a text field without a value. An empty `description` or `feedback` is not in the responses.
- an update saves only when a value changes. A merge patch without changes keeps `updated`.
- `creator` is a relation field. The API expands it and returns only the `id` and the `name` of the user.
- `tags` is a `json` field. PocketBase has no field type for a list of free-form strings. Decision 00004 has the tag tables.
- PocketBase v0.40.6 needs Go 1.27. The app module and `go.work` use it.

## The UI build

- `app/api/internal/webui/webui.go` has `//go:embed all:dist`. The binary and the UI always have the same version.
- `go:embed` fails without `dist`. `dist/.gitkeep` is in git, so `go build`, `go vet`, and the tests work without a UI build.
- without a UI build, `/` returns a short page that says to run `make build-ui`
- `vite.config.ts` sets `outDir` to `../api/internal/webui/dist` and `emptyOutDir: true`. `make build-ui` makes `dist/.gitkeep` again after `vite build`.
- the UI route returns `405` for methods other than `GET` and `HEAD`. A route `GET /{path...}` would conflict with `/v1/{path...}` in `http.ServeMux`.
- the UI route hides the files whose names start with a dot, such as `.gitkeep`
- the root `package.json` is an npm workspace. The UI depends on `@reviews/sdk` with `"*"`, and npm links the local SDK. `make build-ui` builds the SDK first.

## Cache headers

- Vite puts a content hash in the names of the files in `assets/`. These files get `Cache-Control: public, max-age=31536000, immutable`.
- `index.html` gets `Cache-Control: no-cache`. A new deploy then loads the new asset names.
- a missing file in `assets/` gets `index.html` with `no-cache`, so a wrong file is never cached
- the theme script is the file `public/theme.js`, not an inline script. The policy then allows it with `'self'`.

## Content Security Policy

| Directive | Value | Reason |
|---|---|---|
| `default-src` | `'self'` | the base for all other types |
| `script-src` | `'self'` | the Vite chunks and `theme.js`. No inline scripts. |
| `style-src` | `'self'` and one hash | the Tailwind CSS file and the inline style of Reka UI `SelectViewport` |
| `img-src` | `'self' blob:` and the origin of the map tiles | the logos, the icons, and the photos of `/v1`. `blob:` shows the preview of a selected photo. Decision 00008 has the tile origin. |
| `font-src` | `'self'` | the UI uses system fonts |
| `connect-src` | `'self'` | the Public API is on the same origin |
| `manifest-src` | `'self'` | `site.webmanifest` |
| `object-src` | `'none'` | no plugins |
| `base-uri` | `'none'` | an injected `<base>` cannot move the relative URLs |
| `form-action` | `'self'` | forms send only to this origin |
| `frame-ancestors` | `'none'` | other sites cannot put the UI in a frame |
| `report-uri`, `report-to` | `/csp-report`, `csp` | browsers send violations to the app |

- `/v1` has no policy. Its responses are JSON, which a browser does not run. PocketBase sends its own policy for `/_/`.
- the Vite dev server does not send the header. Hot reload needs inline scripts and a WebSocket.
- Reka UI `SelectViewport` renders a `<style>` element with a fixed text. Its SHA-256 hash allows it without `'unsafe-inline'`.
- the UI route also sends `Reporting-Endpoints: csp="/csp-report"`. Browsers that support `report-to` use it. Other browsers use `report-uri`.
- `POST /csp-report` logs the report as a warning with the user agent, and returns `204`. It keeps at most 16 KB of each body. A PocketBase rate limit rule allows 20 reports per minute per IP address.
- the log shows a violation that the tests did not find, such as after a Reka UI update in a screen without a test

Uploads:
- PocketBase serves uploaded files under `/api/files` on the origin of the UI, which `'self'` trusts
- the migrations remove the `avatar` file field of `users`. Only the photo collections of decision 00007 have file fields.
- their files are protected, so `/api/files` does not send them to clients. The Public API sends only the JPEG files that it encoded, with `nosniff` and a `sandbox` policy.
- a new file field needs a check of this risk

Tests:
- the Go tests of `app/api/internal/webui` check the routes, the cache headers, and that `/v1` has no policy
- the `noCspViolations` fixture of `tests/fixtures.ts` runs in each end-to-end test. A refused or blocked load fails the test with the text of the violation.
- `tests/e2e/csp.spec.ts` opens the parts of the UI that change styles at runtime, such as the menus, the selects, the toasts, and the dialogs

## Link previews and crawlers

### The route

- `pagemeta.go` of `app/api/internal/webui` has the page meta. `webui.Register` reads `index.html` one time and cuts it around the block. It also reads the site name and the default `og:image` tags of the block.
- the UI route sends its own page only for the pages of the table in [The page-meta block](#the-page-meta-block), with an optional trailing slash. The path words ignore case, like Vue Router. Each ID must match `^[a-z0-9]{15}$`.
  `/restaurants/new` does not match. The edit and new pages need an account and keep the default tags.
- `html/template` writes the values. A name such as `<script>` stays text.
- the description puts all whitespace on one line. A text of more than 200 characters is cut at the end of a word and ends with `…`.
- without a description or feedback, the description is `Reviews of NAME on Sponga47 Food.` or `A review of NAME on Sponga47 Food.`
- the `<title>` has the format of `usePageTitle` in the UI. The end-to-end tests compare the two titles.
- on a database error other than a missing record, the app logs the error and sends the default `index.html` with `200`
- `publicapi.PhotoFilePath` makes the photo path. The API responses use the same function.
- a host that the public cannot reach is `localhost`, a name that ends in `.localhost`, or a loopback, private, link-local, or unspecified IP address
- the URLs use only the scheme, the host, and the path of the Application URL
- a hook on `OnSettingsReload` checks the Application URL after each settings save. It warns only when the URL changed.
- a block without `<title>` or `og:image` is not used. The app then logs a warning, and the pages have the tags of the site.
- a Go test checks that `app/ui/index.html` has the block. A Vite change that removes the comments then fails `make test-app`.
- the route does not add content to the body. Vue removes the content of `#app` when it mounts, and two versions of a heading would make the end-to-end tests unstable.

### robots.txt

- `robotsTxt` of `robots.go` has the two lines. The route sends `Cache-Control: public, max-age=3600`.
- `TestRobotsTxt` compares the body with the two lines
- `TestPageMeta` and `tests/e2e/link-previews.spec.ts` check that no page has a canonical link

### Tests

- `pagemeta_test.go` checks the tags, the escaping, the photo, the `404` cases, and the paths that keep the default tags
- `tests/e2e/link-previews.spec.ts` checks the tags of the built app, the photo file, the titles, and the not found screen
- the end-to-end app has the Application URL `https://reviews.example.com`. `tests/app.ts` sets it through `PATCH /api/settings`.

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| A UI build in a folder at runtime, such as `pb_public` | Not selected. The folder is a second thing to deploy, with its own version. |
| A separate static host or UI server | Not selected. It needs CORS, a second deploy, and a second origin for the session. |
| SQL triggers for the search tables | Not selected. A trigger blocks a later `DROP COLUMN` of a column that it uses. |
| An external-content FTS5 table | Not selected. It needs a stable `rowid`, and PocketBase uses text IDs. |
| `bm25` for the search score | Not selected. It depends on all rows, so each write moves the ranks under an open cursor. |
| A nonce for the inline style | Not selected. The server would rewrite `index.html` for each response. |
| `'unsafe-inline'` for styles | Not selected. An injected `<style>` could hide or fake content. |
| Trusted Types | Not selected. Vue sets `innerHTML` for static template parts. |
| `Content-Security-Policy-Report-Only` first | Not selected. The end-to-end tests run the UI under the enforced policy in all browsers and find violations before a release. |
| An empty Go module in `app/ui` | Removed. The UI moved out of the Go module, so the Go tools do not walk its `node_modules`. |
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
