# AGENTS.md

Guide for Coding Agents.

## Project Overview

This project allows users to review a variety of things. It is composed of the following surfaces:
- **Public API** for reviewing things with token auth
  - this is implemented and served by the app in the `app` directory
- **Public UI** to interact with the Public API
  - this is implemented in the `app/ui` directory and served by the app in the `app` directory
- **Pocketbase API** for admin operations with token auth
  - this is served by the app in the `app` directory
- **Pocketbase UI** to interact with the Management API
  - this is served by the app in the `app` directory

## Commands

```bash
make install-tools    # Install dev tools to ./bin/tools-<TOOLS_GO>/ with the Go of TOOLS_GO. Needs Go.
make lint             # Run all linters. Needs Docker and Node.js.
make lint-spec        # Lint the API specs in specs/api with vacuum
make generate         # Generate the SDKs in app/sdk/ from the API spec. Needs Docker.
make generate-sdk-go  # Generate only the Go SDK
make generate-sdk-ts  # Generate only the TypeScript SDK. Needs Docker.
make lint-go          # Run go vet and gofmt on the Go modules
make check-generate   # Fail if the generated SDKs do not match the API spec
make lint-ui          # Type-check the UI and the e2e tests. Needs Node.js.
make lint-eslint      # Run ESLint on the UI and the e2e tests. Needs Node.js.
make security-scan    # Check the Go modules for known vulnerabilities with govulncheck. Needs the network.
make test             # Run the tests of the SDKs, the app, and the UI. Needs Node.js.
make test-sdk         # Run the tests of both SDKs. Needs Node.js.
make test-sdk-go      # Run the tests of the Go SDK
make test-sdk-ts      # Run the tests of the TypeScript SDK. Needs Node.js.
make test-app         # Run the Go tests of the app with the race detector. Needs a C compiler.
make test-ui          # Run the UI tests. Needs Node.js.
make build-sdk-ts     # Build the TypeScript SDK into app/sdk/typescript/dist. Needs Node.js.
make build-ui         # Build the UI into app/internal/webui/dist. Needs Node.js.
make build-app        # Build the UI, then the app server to ./bin/reviews-app
make run-app          # Run the app server on 127.0.0.1:8090 with data in app/pb_data
make dev-ui           # Run the Vite dev server. It sends /v1 to make run-app.
make test-e2e         # Build the app, then run the end-to-end tests in tests/. PROJECT=webkit runs one project.
make install-browsers # Install the Playwright browsers. BROWSER=firefox installs one.
make install-browser-deps # Install the system packages of the browsers on Linux. Needs sudo.
make check-ci-matrix  # Fail if the e2e matrix of CI misses a Playwright project
make test-e2e-ui      # Open the Playwright UI mode for the end-to-end tests
```

The app server applies the migrations when it starts.
The PocketBase dashboard is at `http://127.0.0.1:8090/_/`.
To make the first superuser, run `go run ./app superuser create EMAIL PASSWORD --dir app/pb_data`.

`make run-app` serves the UI build that `make build-ui` made last.
For UI work, run `make run-app` and `make dev-ui` at the same time, and open the Vite URL.

Before the first `make test-e2e`, install the browsers with `make install-browsers`.
On Linux, the browsers also need system packages. Install them with `make install-browser-deps`. It needs sudo.
Run one browser with `make test-e2e PROJECT=chromium`.

`.github/workflows/ci.yml` runs on each push to `main` and on each pull request. See decision 00010.
The `lint` job runs `make lint`, and the `test` job runs `make test`.
The `security-scan` job runs `make security-scan`.
The `e2e` jobs run `make test-e2e` with one Playwright project each.
A new Playwright project needs an entry in the `e2e` matrix. `make lint` fails without it.
The `CI result` job passes only when all other jobs pass. Read it first.

## Production

- Behind a reverse proxy, set the trusted proxy header in the PocketBase settings, such as `X-Real-IP`. Otherwise all clients have the IP address of the proxy, and the sign up and sign in rate limit counts all of them together. See decision 00004.
- Set the superuser IP allow-list in the PocketBase settings, so only admin networks can open the dashboard at `/_/`.
- A migration turns on the PocketBase rate limits. Their default rules limit the PocketBase auth routes, which include the superuser sign in.
- Environment variables of the app:
  - `REVIEWS_SIGN_IN_LIMIT` sets the sign up and sign in limit per minute. The end-to-end tests raise it. Their parallel sign ups come from one IP address.
  - `REVIEWS_PHOTO_UPLOAD_LIMIT` sets the photo uploads of each user per hour. The default is 30. See decision 00008.
  - `REVIEWS_VALIDATE_RESPONSES=false` turns off the spec validation of responses. See decision 00007.
  - `REVIEWS_VALIDATE_REQUESTS=false` turns off the spec validation of parameters and body values. Invalid values can then be saved. Turn it off only with the response validation, or responses with these values get 500.

## Architecture

- **`makefile`** - the Makefile for building, developing, and testing the project
- **`specs`** - the directory containing the API specifications and the design system
  - **`specs/api/public-openapi.yaml`** - the OpenAPI specification for the Public API
    - the `restaurants` tag has the restaurant and review operations
    - the `users` tag has the sign up, sign in, and token refresh operations
  - **`specs/api/vacuum-ruleset.yaml`** - the vacuum lint rules for the API specifications
  - **`specs/design-system`** - the design system for the Public UI: tokens, component guidelines, logos, and icons
    - Start from `specs/design-system/README.md`
- **`app/sdk`** - the SDKs generated from `specs/api/public-openapi.yaml`. Do not edit generated files. See decision 00003.
  - **`app/sdk/typescript`** - the TypeScript client `@reviews/sdk` that the UI uses
  - **`app/sdk/go`** - the Go module with the server interface and the types that the API uses
- **`app`** - the directory containing the app that serves the Public UI, Public API, Pocketbase API, and Pocketbase UI
  - Implements the OpenAPI specification defined in `specs/api/public-openapi.yaml`
  - Written in the Go programming language
  - Built using the Pocketbase framework. See decision 00004.
  - **`app/main.go`** - the PocketBase app
  - **`app/migrations`** - the Go migrations of the `users`, `restaurants`, `reviews`, and `review_photos` collections
    - They also make the indexes, the FTS5 search tables, and the rate limit rules
  - **`app/internal/publicapi`** - the Public API. It implements the strict server interface of `app/sdk/go`.
    - It validates each request and response against the spec that `app/sdk/go` embeds. See decision 00007.
    - It checks, changes, and serves the photos of reviews. See decision 00008.
  - **`app/internal/searchindex`** - the record hooks that keep the FTS5 search tables equal to the records. See decision 00004.
  - **`app/internal/photocleanup`** - the daily job that deletes the photo files without a record. See decision 00009.
  - **`app/internal/webui`** - embeds the UI build and serves it. See decision 00005.
- **`app/ui`** - the Public UI, the npm workspace `@reviews/ui`
  - Calls the Public API only through `@reviews/sdk`
  - Written in TypeScript with Vue, Vue Router, and shadcn-vue on Tailwind CSS. See decision 00002.
  - Follows `specs/design-system`. Components live in `app/ui/src/components/ui`, route-aware components in `app/ui/src/components/app`, and screens in `app/ui/src/pages`.
  - `app/ui/go.mod` is an empty Go module. The Go tools of `app` then skip `app/ui/node_modules`.
- **`tests`** - the end-to-end tests, the npm workspace `@reviews/tests`. See decision 00006.
  - Playwright runs the specs in `tests/e2e` against `./bin/reviews-app` with a new database
  - Tests make their data through `@reviews/sdk` and find elements by role and accessible name
  - **`tests/photos`** - the small test images of the photo tests

## Code Comment Style (Mandatory)

This rule applies to **every** comment you write or touch: Go doc comments, inline
comments, test comments, commit messages, and code blocks in Markdown. It applies
to new comments and to comments you edit for any other reason.

Write comments in **ASD-STE100 Simplified Technical English**.

### Comment Only When Needed

The default is **no comment**. Write a comment only if it gives the reader a fact
that the code cannot show. Valid reasons:

- The code does something unexpected. The comment tells why.
- A constraint, invariant, or side effect is not visible in the code.
- The code is a workaround for an external bug. Include the issue link.
- Go requires a doc comment on an exported identifier.

Do not write a comment that:

- Tells what the code does. The code already shows this.
- Describes your change, for example `added X`, `now uses Y`, `fixed bug`.
  Put this in the commit message.
- Labels an obvious block, for example `// Loop over servers`.
- Tells the steps of the function one by one.

If a better name or a smaller function removes the need for a comment, change the
code. Do not write the comment.

When a comment is necessary, keep only the essential fact. One sentence is the
target. Use more only when the reader needs each sentence.

Do not delete existing comments that you do not touch. This section applies to
comments you write or edit. Commit messages are always necessary.

### Hard Rules

1. **One idea per sentence.** Split any sentence that has two clauses joined by
   `and`, `but`, `so`, `because`, `which`, `where`, or a colon.
2. **Maximum 20 words per sentence.** Count them. If over, split.
3. **One sentence is the target. Maximum 3 sentences per comment block.** If you
   need more, the code needs a better name or a smaller function, not a longer
   comment.
4. **Present tense, active voice.** Write `X changes Y`, not `X is used to
   transform Y` or `Y gets transformed by X`.
5. **One meaning per word.** Use the simplest verb that is correct:
- `change` (not transform, translate, convert, marshal into, coerce)
- `map` only when the code literally maps keys to values
- `get` (not retrieve, fetch, obtain, acquire)
- `send` (not dispatch, propagate, emit, forward on)
- `make` (not construct, instantiate, materialize)
- `use` (not leverage, utilize, employ)
- `start` / `stop` (not initiate, terminate, tear down)
6. **No parenthetical asides and no em dashes.** If the aside matters, make it its
   own sentence. If it does not matter, delete it.
7. **No filler.** Ban list: `simply`, `basically`, `essentially`, `note that`,
   `it is worth noting`, `of course`, `obviously`, `in order to`, `please be aware`.
8. **No hedging narrative.** Do not write `so that an update that leaves it out
   does not silently clear it`. Write the fact: `the stored value stays`.
9. **Do not restate the code.** If a sentence repeats the code or the function
   signature in prose, delete it. If no sentence stays, delete the comment.
10. **Go doc comments start with the identifier name.** `// FooBar does X.`
11. **Wrap at 80 columns.** Never break a line mid-sentence in a way that leaves a
    word glued to the next comment marker.
12. **No emoji, no exclamation marks.**

## Design Decisions

- Substantive design decisions are recorded as decisions under `docs/decisions/`
  Start from `docs/decisions/00000-template.md`.
- Backlog items for a future date are in `docs/decisions/00000-backlog.md`
