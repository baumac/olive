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
make install-tools   # vacuum and oapi-codegen into ./bin/tools-<TOOLS_GO of the makefile>
make build-app       # build the UI and the app to ./bin/reviews-app
make run-app         # serve on http://127.0.0.1:8090 with data in app/api/pb_data
```

Make a superuser for the dashboard:

```bash
go run ./app/api superuser create EMAIL PASSWORD --dir app/api/pb_data
```

For UI work, run `make run-app` and `make dev-ui` at the same time, and open the Vite URL.

## Check your work

```bash
make lint       # spec lint, go vet, gofmt, type checks, the generated SDK check, and the CI matrix check
make test       # Go, SDK, and UI unit tests
make test-coverage   # the Go and UI tests. It fails below the minimum coverage.
make test-e2e   # the Playwright projects chromium, firefox, webkit, chromium-small, and rate-limit
```

Before the first `make test-e2e`, run `make install-browsers`.
On Linux, the browsers also need system packages. Install them with `make install-browser-deps`. It needs sudo.
CI runs the same commands. See `.github/workflows/ci.yml`.

## Layout

| Folder | What |
| --- | --- |
| `specs/api` | the OpenAPI spec of the Public API |
| `specs/design-system` | tokens, component rules, logos, and icons of the UI |
| `app` | the PocketBase app, the Public API, and the migrations |
| `app/sdk` | the TypeScript and Go SDKs, generated from the spec |
| `app/ui` | the Public UI |
| `tests` | the end-to-end tests |
| `docs/decisions` | the design decisions. Start with `!0000-backlog.md`. |

`AGENTS.md` has all commands and the rules for code comments.

## Deploy

`deploy/README.md` has the Docker Compose setup with the reverse proxy, and the backup settings.
