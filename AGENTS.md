# AGENTS.md

Guide for Coding Agents.

## Project Overview

This project allows users to review a variety of things. It is composed of the following surfaces:
- **Public API** for reviewing things with token auth
  - this is implemented and served by the app in the `app/api` directory
- **Public UI** to interact with the Public API
  - this is implemented in the `app/ui` directory and served by the app in the `app/api` directory
- **Pocketbase API** for admin operations with token auth
  - this is served by the app in the `app/api` directory
- **Pocketbase UI** to interact with the Management API
  - this is served by the app in the `app/api` directory

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
make test-app         # Run the Go tests of the app without the race detector
make test-race        # Run the Go tests of the app with the race detector. Needs a C compiler.
make test-ui          # Run the UI tests. Needs Node.js.
make test-coverage    # Run the Go and UI tests with coverage. Fail below the minimum. Needs Node.js and a C compiler.
make build-sdk-ts     # Build the TypeScript SDK into app/sdk/typescript/dist. Needs Node.js.
make build-ui         # Build the UI into app/api/internal/webui/dist. Needs Node.js.
make build-app        # Build the UI, then the app server to ./bin/reviews-app
make build-image      # Build the Docker image reviews-app:local from deploy/Dockerfile. Needs Docker.
make run-app          # Run the app server on 127.0.0.1:8090 with data in app/api/pb_data
make dev-ui           # Run the Vite dev server. It sends /v1 to make run-app.
make test-e2e         # Build the app, then run the end-to-end tests in tests/. PROJECT=webkit runs one project. E2E_FILE=e2e/auth.spec.ts runs one file.
make install-browsers # Install the Playwright browsers. BROWSER=firefox installs one.
make install-browser-deps # Install the system packages of the browsers on Linux. Needs sudo.
make check-ci-matrix  # Fail if the e2e matrix of CI misses a Playwright project
make test-e2e-ui      # Open the Playwright UI mode for the end-to-end tests
make renovate-local   # Find the updates with Renovate. It changes no file. The report is in bin/renovate-report.json. Needs Docker and the network.
make renovate         # Open or update the Renovate pull request on GitHub. Needs Docker, RENOVATE_TOKEN, and RENOVATE_REPOSITORIES.
```

The app server applies the migrations when it starts.
The PocketBase dashboard is at `http://127.0.0.1:8090/_/`.
To make the first superuser, run `go run ./app/api superuser create EMAIL PASSWORD --dir app/api/pb_data`.

`make run-app` serves the UI build that `make build-ui` made last.
For UI work, run `make run-app` and `make dev-ui` at the same time, and open the Vite URL.

Before the first `make test-e2e`, install the browsers with `make install-browsers`.
On Linux, the browsers also need system packages. Install them with `make install-browser-deps`. It needs sudo.
While you work, run one browser with `make test-e2e PROJECT=chromium`. Add `E2E_FILE=e2e/NAME.spec.ts` for the specs of your change. The CI runs all projects. See decision 00046.
A test that searches or filters must use a name or tag from `uniqueTag`. All projects of one run share one app.
Playwright records a trace and a video only on the retry of a failed test.

While you work, run `make test-app`. It takes seconds.
Run `make test-race` or `make test-coverage` one time before the end. Each has the race detector. Do not run both.
For one test, run `go test ./internal/publicapi -run TestName` in `app/api`. See decision 00045.

`.github/workflows/ci.yml` runs on each push to `main` and on each pull request. See decision 00010.
The `lint` job runs `make lint`. The `test` job runs `make test-sdk test-coverage`. These are the tests of `make test` with the race detector and a coverage minimum.
The `security-scan` job runs `make security-scan`.
The makefile and `app/ui/vite.config.ts` have the minimum coverage values.
The `docker` job runs `make build-image` and checks `deploy/compose.yaml`.
`.github/workflows/renovate.yml` runs `make renovate` on the first day of each month. It opens one pull request with all dependency updates. See decision 00026.
`.github/renovate.json` has its settings. The workflow uses its `GITHUB_TOKEN`. The repository setting "Allow GitHub Actions to create and approve pull requests" must be on.
A person must approve the CI run of the Renovate pull request. Renovate does not update the files in `.github/workflows`. Update them by hand.
Renovate also updates the tool versions of the makefile. A new Go tool needs a `# renovate: depName=MODULE` comment above its version.
`make renovate-local` checks a change of `.github/renovate.json`. Set `GITHUB_COM_TOKEN` to also get the updates of the GitHub Actions.
When Renovate changes `TOOLS_GO`, `OAPI_CODEGEN_VERSION`, or `OPENAPI_GENERATOR_IMAGE`, run `make generate` on its branch.
The `e2e` jobs run `make test-e2e` with one Playwright project each.
A new Playwright project needs an entry in the `e2e` matrix. `make lint` fails without it.
The `CI result` job passes only when all other jobs pass. Read it first.

## Production

- `deploy/README.md` has the steps to start, update, back up, and restore the app with Docker Compose.
- Turn on the backups in the PocketBase settings. They go to `pb_data/backups` on the local file system. The backup task of the host copies them off the server. The app has no backup without them. See decision 00022.
- Run only one instance. The rate limits and the place cache are in the memory of the process. See decision 00022.
- Behind a reverse proxy, set the trusted proxy header in the PocketBase settings, such as `X-Real-IP`. Otherwise all clients have the IP address of the proxy, and the sign up and sign in rate limit counts all of them together. See decision 00004.
- Set the Application URL in the PocketBase settings to the public URL of the site. The canonical links and link previews of restaurant and review pages start with it. See decision 00013.
  The session cookie gets `Secure` from the `X-Forwarded-Proto` header of the reverse proxy. Without this header, it gets `Secure` when the Application URL is `https` and has the host of the request. See decision 00027.
  `/sitemap.xml` also needs it. Without a public URL, the sitemap gets 404. See decision 00021.
- The place search sends the search text of users to the public Photon server. This server has no guarantee and blocks heavy use. The app sends the Application URL in its `User-Agent`. See decision 00017.
- The browsers of users load the map tiles from `tile.openstreetmap.org`. This server gets their IP address and the origin of the site. It has no guarantee and blocks heavy use. Do not set a `Referrer-Policy` of `no-referrer` on the UI. The tile server needs the `Referer` header. See decision 00018.
- The "Use my current location" button needs HTTPS. Do not turn off `geolocation` in a `Permissions-Policy` header of the reverse proxy. See decision 00035.
- Set the superuser IP allow-list in the PocketBase settings, so only admin networks can open the dashboard at `/_/`.
- Set the mail server in the PocketBase settings, then turn on OTP and MFA for the `_superusers` collection. `deploy/README.md` has the steps.
- `PB_ENCRYPTION_KEY` of `deploy/.env` encrypts the PocketBase settings in the database. The backups do not have it. A restore needs the same key. See decision 00022.
- `deploy/compose.yaml` limits the memory and the open files of the app. `deploy/Caddyfile` limits the request body to 11 MB. `deploy/README.md` has the values.
- Set the mail server in the PocketBase settings before the first sign up. A new user confirms the email address with a link from an email. Without SMTP, the app writes each email to the log. See decision 00037.
- Approve new users in the dashboard. A user without the `approved` value of the `users` collection can only sign in and read. `deploy/README.md` has the steps. See decision 00034.
  A daily job deletes a user that waits for more than 30 days and has no data.
- A migration turns on the PocketBase rate limits. Their default rules limit the PocketBase auth routes, which include the superuser sign in.
- Environment variables of the app:
  - `REVIEWS_SIGN_IN_LIMIT` sets the sign up and sign in limit per minute. The end-to-end tests raise it. Their parallel sign ups come from one IP address.
  - `REVIEWS_SIGN_IN_FAILURE_LIMIT` sets the wrong passwords of each email address per 15 minutes. The default is 10. See decisions 00015 and 00039.
  - `REVIEWS_PHOTO_UPLOAD_LIMIT` sets the photo uploads of each user per hour. The default is 30. See decision 00008.
  - `REVIEWS_PLACES_URL` sets the Photon geocoder of the place search. The default is the public server `https://photon.komoot.io`. `off` turns the search off. See decision 00017.
  - `REVIEWS_PLACE_SEARCH_LIMIT` sets the place searches of each user per minute. The default is 30.
  - `REVIEWS_CREATE_LIMIT` sets the restaurants, reviews, or lists that each user can create per hour. Each operation has its own count. The default is 30. See decision 00036.
  - `REVIEWS_EMAIL_LIMIT` sets the emails of each kind that one address can get per hour. The default is 5. See decision 00037.
  - `REVIEWS_EDIT_LIMIT` sets the updates, deletes, and list changes of each operation per user and hour. The default is 300. See decision 00036.
  - `REVIEWS_MAP_TILE_URL` sets the tile server of the maps. It is a URL template with `{z}`, `{x}`, and `{y}`. The default is the tile server of OpenStreetMap. `off` turns the maps off. See decision 00018.
  - `REVIEWS_MAP_ATTRIBUTION` sets the credit of a tile server that is not from OpenStreetMap.
  - `REVIEWS_VALIDATE_RESPONSES=false` turns off the spec validation of responses. See decision 00007.
  - `REVIEWS_VALIDATE_REQUESTS=false` turns off the spec validation of parameters and body values. Invalid values can then be saved. Turn it off only with the response validation, or responses with these values get 500.

## Architecture

- **`makefile`** - the Makefile for building, developing, and testing the project
- **`deploy`** - the Dockerfile and the Docker Compose file with Caddy. Start from `deploy/README.md`. See decision 00022.
- **`specs`** - the directory containing the API specifications and the design system
  - **`specs/api/public-openapi.yaml`** - the OpenAPI specification for the Public API
    - the `restaurants` tag has the restaurant and review operations
    - the `recipes` tag has the recipe and recipe review operations. Recipes have ingredients, steps, and four scores. See decision 00047.
    - the `users` tag has the sign up, email confirmation, sign in, sign out, token refresh, and password reset operations
    - the operations of `/users/me` change the name and the password and delete the user. See decision 00037.
    - `getCurrentUser` returns the user of the token without a new token. The UI uses it at start. See decision 00042.
    - `confirmEmail` needs the token of the link and the password. The wrong password limit counts each email address, with or without a user. Its limiter has a cap of 100,000 addresses. See decision 00039.
    - each write operation has an `x-rate-limit`. An operation with it must have a `429` response. Sign out and the token refresh are the exceptions. See decision 00036.
    - each `403` and `409` problem has a `code`. A new reason needs a value in the `code` enum of `Problem`. See decision 00040.
    - a schema with `x-trim: true` loses the ASCII spaces around its value. Names, titles, addresses, and alt texts have it. See decision 00040.
    - a `Tag` has the lowercase letters and digits of all scripts. A schema with `x-nfc: true` gets Unicode normalization form C before the validation, in the body and in the query. See decision 00041.
    - each operation that needs a token also needs an approved user. It must have a `403` response. `x-allow-unapproved: true` is the exception for sign out, token refresh, and the operations of `/users/me`. See decision 00034.
    - the UI keeps its auth token in the HttpOnly `reviews_session` cookie. Other clients send a bearer token. See decision 00027.
    - the `reviewers` tag has the public profile of a user and the list of the reviews of a user
    - the `places` tag has the place search for the location of a restaurant
    - the `lists` tag has the public lists of restaurants of users. Each user has a favorites list. See decision 00033.
    - `listRestaurants` with `list[eq]` searches, sorts, and filters the restaurants of a list. See decision 00043.
  - **`specs/api/vacuum-ruleset.yaml`** - the vacuum lint rules for the API specifications
  - **`specs/design-system`** - the design system for the Public UI: tokens, component guidelines, logos, and icons
    - Start from `specs/design-system/README.md`
- **`app/sdk`** - the SDKs generated from `specs/api/public-openapi.yaml`. Do not edit generated files. See decision 00003.
  - **`app/sdk/typescript`** - the TypeScript client `@reviews/sdk` that the UI uses
  - **`app/sdk/go`** - the Go module with the server interface and the types that the API uses
- **`app`** - the directory with the three parts of the app: `app/api`, `app/ui`, and `app/sdk`
- **`app/api`** - the Go module of the app that serves the Public UI, Public API, Pocketbase API, and Pocketbase UI
  - Implements the OpenAPI specification defined in `specs/api/public-openapi.yaml`
  - Written in the Go programming language
  - Built using the Pocketbase framework. See decision 00004.
  - **`app/api/main.go`** - the PocketBase app
  - **`app/api/migrations`** - the Go migrations of the `users`, `restaurants`, `reviews`, `review_photos`, `lists`, `list_items`, `recipes`, `recipe_reviews`, and `recipe_review_photos` collections
    - SQLite triggers keep the `restaurant_overall`, `restaurant_tags`, `recipe_overall`, and `recipe_tags` tables equal to the records. See decision 00012.
    - `restaurant_overall` has the plain average for the responses and a Bayesian average for the `overall` sort. See decision 00029.
    - Drop these triggers before a migration drops a column that they use. SQLite refuses the drop.
    - The `overall` column of `reviews` and `recipe_reviews` is a generated column with indexes. It is not a field of the collection. See decision 00031.
    - Drop it and its indexes before a migration drops a score column of `reviews` or `recipe_reviews`. SQLite refuses the drop.
    - A hook of `app/api/internal/publicapi` rejects this change from the dashboard with a message.
    - They also make the indexes, the FTS5 search tables, and the rate limit rules
  - **`app/api/internal/publicapi`** - the Public API. It implements the strict server interface of `app/sdk/go`.
    - It validates each request and response against the spec that `app/sdk/go` embeds. See decision 00007.
    - It checks, changes, and serves the photos of reviews. See decision 00008.
    - Its timestamps are RFC 3339. The database has the PocketBase format with a space. See decision 00023.
    - It gets the places of the place search from a Photon geocoder. See decision 00017.
    - A record hook gives each new user a favorites list. See decision 00033.
    - A record hook gives a restaurant or recipe save in the dashboard the checks of the spec. See decision 00016.
    - A record hook sets the `source_key` of each recipe save. Its unique index stops two recipes with one source. See decision 00047.
    - `mail.go` sends the emails of the sign up, the password reset, and the approval. See decision 00037.
  - **`app/api/internal/searchindex`** - the record hooks that keep the FTS5 search tables equal to the records. See decision 00004.
  - **`app/api/internal/subjects`** - the table names of each kind of thing that users review. The Public API and the search index use them. See decision 00047.
  - **`app/api/internal/photocleanup`** - the daily job that deletes the photo files without a record. See decision 00009.
  - **`app/api/internal/usercleanup`** - the daily job that deletes the users without approval and without data after 30 days. See decision 00034.
  - **`app/api/internal/testapp`** - the migrated database of the Go tests. See decision 00045.
    - A test makes its app with `tests.NewTestApp(testapp.Dir(t))`. Each test package needs a `TestMain` that calls `testapp.Main`.
    - `testapp.LowCostDir` has the bcrypt cost 4 in the password fields. The `publicapi` tests use it. Production and `testapp.Dir` have the default cost 10.
    - `make test-coverage` does not count `testapp`.
  - **`app/api/internal/webui`** - embeds the UI build and serves it. See decision 00005.
    - It writes the title and link preview tags of restaurant, review, reviewer, and list pages into `index.html`. See decision 00013.
    - It writes the schema.org JSON-LD of these pages and of reviewer pages, and serves `/robots.txt` and `/sitemap.xml`. See decision 00021.
    - It serves `/map-config.json` with the tile server of the maps and allows its origin in the Content Security Policy. See decision 00018.
- **`app/ui`** - the Public UI, the npm workspace `@reviews/ui`
  - Calls the Public API only through `@reviews/sdk`
  - Keeps the preferred location of a person in the `localStorage` of the browser. The restaurant list and the maps use it. See decision 00035.
  - Written in TypeScript with Vue, Vue Router, and shadcn-vue on Tailwind CSS. See decision 00002.
  - Follows `specs/design-system`. Components live in `app/ui/src/components/ui`, route-aware components in `app/ui/src/components/app`, and screens in `app/ui/src/pages`.
- **`tests`** - the end-to-end tests, the npm workspace `@reviews/tests`. See decision 00006.
  - Playwright runs the specs in `tests/e2e` against `./bin/reviews-app` with a new database
  - Tests make their data through `@reviews/sdk` and find elements by role and accessible name
  - `api.signUp` confirms and approves the user with the superuser of the test app. `api.signUpPending` only confirms.
  - `api.emailLink` reads the link of an email from the log of the test app. The test app has no mail server.
  - **`tests/photos`** - the small test images of the photo tests
  - **`tests/photon.ts`** - the fake geocoder of the place search and the fake map tiles. The tests need no network.

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
