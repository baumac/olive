---
description: Build the Public UI in app/ui and serve it from the PocketBase app.
status: proposed
---

# Motivation
[motivation]: #motivation

The Public UI is a Vue app that calls the Public API.
The PocketBase app in `app` already serves the Public API, the PocketBase API, and the PocketBase dashboard.
If the UI runs on a separate server, the project has two servers to deploy and the UI calls the API on a different origin.
A different origin needs CORS settings and a second deployment.

We put the UI in `app/ui` and serve its built files from the PocketBase app.

Expected outcome:
- One binary serves the Public UI, the Public API, the PocketBase API, and the PocketBase dashboard.
- The UI and the Public API have the same origin. The API needs no CORS settings for the UI.
- Developers run the UI with the Vite dev server and hot reload.
- The Go and npm tools of the project work with the UI inside the Go module.

# Scope
[scope]: #scope

In scope:
- The location of the UI source and of its build output
- How the PocketBase app serves the UI
- The routes of the UI next to the other routes of the app
- The development flow and the `make` targets
- The changes to the docs that use the old `ui` path

Out of scope:
- The screens, components, and state of the UI. The design system in `specs/design-system` covers the look.
- Server-side rendering
- A CDN or a separate static host
- A Content Security Policy. The backlog has it.

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

- the root `package.json` lists `app/ui` and `sdk/typescript` as workspaces
- the UI depends on `@reviews/sdk` with `"@reviews/sdk": "*"`. npm links the local SDK.
- `make build-ui` builds the SDK before the UI

## Caching

- Vite puts a content hash in the names of the files in `assets/`. These files get `Cache-Control: public, max-age=31536000, immutable`.
- `index.html` gets `Cache-Control: no-cache`. A new deploy then loads the new asset names.
- a middleware on the UI route sets these headers

## Auth token

- the UI keeps the auth token in `localStorage` and sends it as a bearer token
- the UI and the API have the same origin, so the API needs no CORS settings for the UI
- the API does not use cookies. It does not need CSRF protection.

## Docs to change

- `AGENTS.md`: the `ui` paths become `app/ui`, and the UI commands replace the TODO
- `specs/design-system/README.md` and the component READMEs: `ui/src/...` becomes `app/ui/src/...`, and `ui/public/` becomes `app/ui/public/`
- decision 00003: the `ui` app becomes the UI in `app/ui`

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
