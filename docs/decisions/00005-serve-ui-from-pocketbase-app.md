---
description: Build the Public UI in app/ui and serve it from the PocketBase app.
status: implemented
---

# Motivation
[motivation]: #motivation

The Public UI is a Vue app that calls the Public API.
The PocketBase app in `app` already serves the Public API, the PocketBase API, and the PocketBase dashboard.
If the UI runs on a separate server, the project has two servers to deploy and the UI calls the API on a different origin.
A different origin needs CORS settings and a second deployment.

We put the UI in `app/ui` and serve its built files from the PocketBase app.

The UI keeps the auth token in `localStorage`.
A script that an attacker injects into the page can read the token and act as the user until the token expires.
Decision 00001 records that the API cannot revoke tokens.
Vue escapes text by default, so most injections fail.
A bug in a dependency or a later use of `v-html` can still let a script in.
A strict Content Security Policy (CSP) stops the browser from running such a script.

Expected outcome:
- One binary serves the Public UI, the Public API, the PocketBase API, and the PocketBase dashboard.
- The UI and the Public API have the same origin. The API needs no CORS settings for the UI.
- Developers run the UI with the Vite dev server and hot reload.
- The Go and npm tools of the project work with the UI inside the Go module.
- The UI route sends a CSP that allows scripts and styles only from the same origin. A change that breaks the policy fails the end-to-end tests.

# Scope
[scope]: #scope

In scope:
- The location of the UI source and of its build output
- How the PocketBase app serves the UI
- The routes of the UI next to the other routes of the app
- The development flow and the `make` targets
- The changes to the docs that use the old `ui` path
- The CSP of the UI route, and how the tests find violations

Out of scope:
- The screens, components, and state of the UI. The design system in `specs/design-system` covers the look.
- Server-side rendering
- A CDN or a separate static host
- A CSP for the Public API under `/v1`. Its responses are JSON, which a browser does not run.
- A CSP for the PocketBase dashboard under `/_/`. PocketBase sends its own policy there.
- Trusted Types. Vue sets `innerHTML` for static template parts, which Trusted Types blocks.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

New concepts:
- **UI build**: the files that `vite build` makes from `app/ui`. The PocketBase app embeds them in its binary.
- **UI route**: a path that the UI router handles, such as `/restaurants/k3x9m2p7q1w8z4a`. The server returns `index.html` for it.

```
app/
  main.go
  internal/
    publicapi/            # the Public API
    webui/
      webui.go            # embeds dist/ and adds the UI routes
      dist/               # the UI build. Only .gitkeep is in git.
  migrations/
  ui/
    go.mod                # empty module. Go tools then skip node_modules.
    package.json          # the Vue app, an npm workspace
    vite.config.ts        # outDir ../internal/webui/dist
    index.html
    public/               # favicons and other static files
    src/
```

```bash
make build-ui    # Build the UI into app/internal/webui/dist
make build-app   # Build the UI, then the binary ./bin/reviews-app
make run-app     # Serve the UI build and the API on 127.0.0.1:8090
make dev-ui      # Run the Vite dev server. It sends /v1 to 127.0.0.1:8090.
make test-ui     # Run the UI tests
```

For UI work, run `make run-app` and `make dev-ui` at the same time.
Open the Vite dev server URL.
The Vite dev server sends `/v1` requests to the PocketBase app, so the UI uses the same paths as in production.

## Content Security Policy

Every response of the UI route has a `Content-Security-Policy` header.
The value is the constant `contentSecurityPolicy` in `app/internal/webui/webui.go`.
The Implementation Details explain each directive.

The UI must not add:
- inline scripts, `eval`, or `new Function`
- inline `<style>` elements or `style="..."` attributes in HTML. Vue `:style` bindings are allowed, because they set styles through the CSSOM.
- files from other origins, such as fonts or images from a CDN
- `data:` URLs. `assetsInlineLimit: 0` in `app/ui/vite.config.ts` makes Vite emit every asset as a file, also a small one.

A change that breaks one of these rules fails the end-to-end tests with the text of the violation.

# Implementation Details
[implementation-details]: #implementation-details

## Why the PocketBase app serves the UI

| Option | Result |
| --- | --- |
| Embed the UI build in the binary with `go:embed` | Selected. One file to deploy. The binary and the UI always have the same version. |
| Serve the UI build from a folder at runtime, such as `pb_public` | Not selected. The folder is a second thing to deploy. It can have a different version than the binary. |
| A separate static host or UI server | Not selected. It needs CORS, a second deployment, and a second origin for the auth token. |

## Routes

The app has these routes:

| Path | Handler |
| --- | --- |
| `/v1/{path...}` | the Public API. See decision 00004. |
| `/api/{path...}` | the PocketBase API |
| `/_/{path...}` | the PocketBase dashboard |
| `/{path...}` | the UI |

- the UI route is the least specific route. The router selects the other routes first.
- `webui.Register` adds the UI route with `apis.Static(fsys, true)` in an `OnServe` hook
- the UI route has no method. `GET /{path...}` and `/v1/{path...}` conflict in `http.ServeMux`, because neither is more specific. The UI route returns `405` for methods other than `GET` and `HEAD`.
- `true` turns on the index fallback. A path that is not a file in the UI build gets `index.html`. The UI router then shows the screen or a not found screen.
- an unknown path under `/v1` still gets the Problem Details `404` of decision 00004. It does not get `index.html`.
- the UI uses HTML5 history mode in Vue Router. Paths have no `#`.

## Embedding

- `app/internal/webui/webui.go` has `//go:embed all:dist`
- `go:embed` fails when `dist` does not exist. `dist/.gitkeep` is in git, so `go build`, `go vet`, and the tests work without a UI build.
- `.gitignore` ignores the other files in `app/internal/webui/dist`
- without a UI build, `/` returns a short page that says to run `make build-ui`
- `dist` is outside the Vite root, so the Vite config sets `emptyOutDir: true`. This also deletes `.gitkeep`.
- `make build-ui` makes `dist/.gitkeep` again after `vite build`

## Go tools and node_modules

- `go build ./...` and `go vet ./...` walk all folders of the module, also `node_modules`
- some npm packages have `.go` files. These files then break the Go build of the app.
- `app/ui/go.mod` makes `app/ui` a separate, empty Go module. The Go tools of the `app` module then skip it.
- `go.work` does not list `app/ui`

## npm workspace

- the root `package.json` lists `app/ui` and `app/sdk/typescript` as workspaces
- the UI depends on `@reviews/sdk` with `"@reviews/sdk": "*"`. npm links the local SDK.
- `make build-ui` builds the SDK before the UI

## Caching

- Vite puts a content hash in the names of the files in `assets/`. These files get `Cache-Control: public, max-age=31536000, immutable`.
- `index.html` gets `Cache-Control: no-cache`. A new deploy then loads the new asset names.
- the UI route hides files whose names start with a dot, such as `.gitkeep`
- the theme script is the file `public/theme.js`, not an inline script. A Content Security Policy can then allow it with `'self'`.
- the UI route sets these headers. A missing file in `assets/` gets `index.html` with `no-cache`, so a wrong file is never cached.

## Auth token

- the UI keeps the auth token in `localStorage` and sends it as a bearer token
- the UI and the API have the same origin, so the API needs no CORS settings for the UI
- the API does not use cookies. It does not need CSRF protection.

## UI structure

- `/my-reviews` lists the reviews of the signed in user at all restaurants. Its cards name the restaurant. "My reviews" in `UserMenu` opens it.
- the UI has no page with the reviews of all users. `AppHeader` has only the "Restaurants" link.
- the restaurant page and `/my-reviews` use one `ReviewList` component. Without a restaurant, it calls `listAllReviews` with `creator[eq]` set to the user. The filters and sorts are in `src/lib/reviewFilters.ts`.

- `src/lib/api.ts` makes one `Configuration` for `RestaurantsApi` and `UsersApi` with the base path `/v1`
- a middleware of the SDK handles a `401` on a request with a token. It deletes the session and sends the request again without the token. A read then works. A write gets `401`, and a page that needs an account opens the sign in page with `reason=session-ended`.
- forms save a draft in `sessionStorage` before each request that needs an account. They restore it after sign in.
- list filters and the sort are in the URL query. A change loads the first page again.
- the components come from the shadcn-vue CLI, then follow the design system: a solid 2px focus outline in `ring`, the dark `destructive` fill at 50%, a `success` alert, and `radius-md` badges
- the `Form` parts are small project components, not the vee-validate parts of shadcn-vue. They set `id`, `aria-describedby`, and `aria-invalid`.

## Tests

- `make test-ui` runs Vitest with Vue Test Utils in jsdom
- the component tests run `axe-core` directly. `vitest-axe` has no release since 2022. jsdom does not render, so the color contrast rule does not run. `tokens.json` has the contrast values.
- `app/internal/webui` has Go tests for the routes and the cache headers

## Content Security Policy

### Directives

| Directive | Value | Why |
| --- | --- | --- |
| `default-src` | `'self'` | the base for all other types |
| `script-src` | `'self'` | the Vite chunks and `theme.js`. No inline scripts. |
| `style-src` | `'self'` and one hash | the Tailwind CSS file and the inline style of Reka UI `SelectViewport` |
| `img-src` | `'self' blob:` and the origin of the map tiles | the logos, the icons, and the photos of `/v1`. `blob:` shows the preview of a selected photo. See decision 00008. Decision 00018 added the origin of the map tiles. |
| `font-src` | `'self'` | the UI uses system fonts. No web fonts. |
| `connect-src` | `'self'` | the Public API is on the same origin |
| `manifest-src` | `'self'` | `site.webmanifest` |
| `object-src` | `'none'` | no plugins |
| `base-uri` | `'none'` | an injected `<base>` cannot move the relative URLs |
| `form-action` | `'self'` | forms send only to this origin |
| `frame-ancestors` | `'none'` | other sites cannot put the UI in a frame. This also stops clickjacking. |
| `report-uri`, `report-to` | `/csp-report`, `csp` | browsers send violations to the app. See Reports. |

- `app/internal/webui/webui.go` sets the header on every response of the UI route
- the Vite dev server of `make dev-ui` does not send the header. Hot reload needs inline scripts and a WebSocket.

### Reports

- the UI route also sends `Reporting-Endpoints: csp="/csp-report"`. Browsers that support `report-to` use it. Other browsers use `report-uri`.
- `POST /csp-report` logs the report as a warning in the PocketBase log, with the user agent. It returns `204`.
- anybody can send a report. The handler keeps at most 16 KB of each body. A PocketBase rate limit rule allows 20 reports per minute per IP address.
- the log shows a violation that the tests did not find, for example after a Reka UI update in a screen without a test

### Uploads

PocketBase serves uploaded files under `/api/files` on the origin of the UI, which `'self'` trusts.
A migration removes the default `avatar` file field of `users`. The Public API has no profile pictures.
No collection then has a file field, and nobody can upload a file that the policy would trust.
A later file field needs a check of this risk, for example a separate host for files.

Decision 00008 adds the photo file fields of `review_photos`. They are protected, so `/api/files` does not send them to clients.
The Public API sends only JPEG files that it encoded itself, with `nosniff` and a `sandbox` policy.

### The style hash

Reka UI `SelectViewport` renders a `<style>` element that hides the scrollbar of select lists.
Its text is fixed, so its SHA-256 hash allows it without `'unsafe-inline'`.

Other options:

| Option | Result |
| --- | --- |
| A hash of the fixed text | Selected. The policy stays static. The tests find a new text after a Reka UI update. |
| A nonce | Not selected. The server would need to rewrite `index.html` for each response with a new nonce, and pass it to `SelectViewport`. |
| `'unsafe-inline'` for styles | Not selected. An injected `<style>` can then change the page, for example to hide or fake content. |

### Tests

- the Go test of `app/internal/webui` checks that the UI route sends the policy and `/v1` does not
- the `noCspViolations` fixture in `tests/fixtures.ts` runs in every end-to-end test
  - it listens to `securitypolicyviolation` events and to console messages about the policy
  - a test with a violation fails and shows the text of the violation
- the fixture ignores other policy messages in the console, such as an ignored source. Only a refused or blocked load fails a test.
- `tests/e2e/csp.spec.ts` opens the parts of the UI that change styles at runtime: the theme menu, Load more, the filter popover with a score select, the sort select, the rating inputs, the success and error toasts, the delete dialog, and the account menu

### Rollout

The backlog item said to send the policy first as `Content-Security-Policy-Report-Only`.
The policy is enforced at once instead:
- report-only shows a violation only in the console and the log. The tests find violations before release.
- the end-to-end tests run the UI under the enforced policy in Chromium, Firefox, WebKit, and a small viewport. They found the Reka UI style before release.

## Docs changed

- `AGENTS.md`: the `app/ui` paths and the UI commands
- `specs/design-system/README.md` and the component READMEs: `ui/src/...` became `app/ui/src/...`, and `ui/public/` became `app/ui/public/`
- decision 00003: the `ui` app became the UI in `app/ui`

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
