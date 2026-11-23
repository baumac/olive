# Sponga47 Food

A shared food journal for a small group of friends and family at food.sponga47.com.
Each visit to a restaurant and each cook of a recipe is a dated entry with photos, four scores, and a few words.

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
| `specs/api/sdk` | the SDKs, generated from the spec: the TypeScript client in `client/typescript` and the Go server in `server/go` |
| `specs/design-system` | tokens, component rules, logos, and icons of the UI |
| `app` | the PocketBase app, the Public API, and the migrations |
| `app/ui` | the Public UI |
| `tests` | the end-to-end tests |
| `docs/decisions` | the design decisions. Start with `!0000-backlog.md`. |

`AGENTS.md` has all commands and the rules for code comments.

## Deploy

`deploy/README.md` has the Docker Compose setup with the Cloudflare Tunnel, and the backup settings.
