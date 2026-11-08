# Sponga47 Reviews

Find a restaurant. Score it. Say why.

One Go binary serves the whole app:

| Path | What |
| --- | --- |
| `/` | the Public UI, a Vue app in `app/ui` |
| `/v1` | the Public API, defined in `specs/api/public-openapi.yaml` |
| `/api` | the PocketBase API, for superusers |
| `/_/` | the PocketBase dashboard, for superusers |

## Get started

You need Go, Node.js 22, and Docker.

```bash
make install-tools   # vacuum and oapi-codegen into ./bin
make build-app       # build the UI and the app to ./bin/reviews-app
make run-app         # serve on http://127.0.0.1:8090 with data in app/pb_data
```

Make a superuser for the dashboard:

```bash
go run ./app superuser create EMAIL PASSWORD --dir app/pb_data
```

For UI work, run `make run-app` and `make dev-ui` at the same time, and open the Vite URL.

## Check your work

```bash
make lint       # spec lint, go vet, gofmt, type checks, and the generated SDK check
make test       # Go, SDK, and UI unit tests
make test-e2e   # Playwright tests in Chromium, Firefox, and WebKit
```

Before the first `make test-e2e`, run `cd test && npx playwright install`.
CI runs the same commands. See `.github/workflows/ci.yml`.

## Layout

| Folder | What |
| --- | --- |
| `specs/api` | the OpenAPI spec of the Public API |
| `specs/design-system` | tokens, component rules, logos, and icons of the UI |
| `app` | the PocketBase app, the Public API, and the migrations |
| `app/sdk` | the TypeScript and Go SDKs, generated from the spec |
| `app/ui` | the Public UI |
| `test` | the end-to-end tests |
| `docs/decisions` | the design decisions. Start with `00000-backlog.md`. |

`AGENTS.md` has all commands and the rules for code comments.
