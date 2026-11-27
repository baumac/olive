---
description: The app runs as one instance on the home server with Docker Compose and a Cloudflare Tunnel. The Go, UI, and end-to-end tests check each change, with coverage minimums. A GitHub Actions workflow runs them through make, and a monthly Renovate workflow opens one pull request with all dependency updates.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00000 hosts the app on our home server at food.sponga47.com.
The operator needs one way to build, start, update, back up, and restore the app.
`pb_data` has the database and the photo files. Nothing else keeps a copy of them.
Some state is in the memory of the process, so the app cannot run as many instances.

AI coding tools write the app. See decision 00000.
Strong tests find their errors early, and agents run the tests after each change.
The tests must therefore be fast while a person or an agent works, and complete before a merge.

The unit and component tests cannot find all problems:
- the UI sends a request that the API refuses, because the SDK and the app change in different ways
- the embedded UI build, the UI route, or the cache headers break
- a flow that crosses pages breaks, such as a sign in with a `redirect` back to a form
- a color pair fails the contrast rules. jsdom does not render.

Each change goes through CI.
A failed check must name the part that failed. A person then reads one result before a merge.
The versions of the dependencies are exact, so nothing tells when a dependency has an update or a security fix.

Expected outcome:
- one command builds the image, and one command starts the app with TLS
- a backup of the database and the photos goes to a different computer each day
- a person can restore the app from a backup with written steps
- the settings with secrets, such as the SMTP password, are encrypted in the database
- the memory of the app has a limit below the memory of the server
- CI fails when the image does not build
- `make test-app` runs the Go tests in seconds
- one `make` command builds the app and runs the main flows in real browsers
- each end-to-end run starts with a new database
- accessibility checks, color contrast included, run on the main screens in both themes
- CI fails when the coverage falls below a minimum
- a failed check names its job: lint, security scan, test, end-to-end, or Docker
- one job, `CI result`, gives one pass or fail for each run
- each run step that does project work calls a `make` target
- `make lint` gives the same result on each computer and in CI
- one pull request each month has all dependency updates

# Scope
[scope]: #scope

In scope:
- `deploy/Dockerfile`, `deploy/compose.yaml`, `deploy/.env.example`, and `deploy/README.md`
- the rule of one instance
- the backup plan and the restore
- the settings encryption and the memory limits
- the Cloudflare Tunnel and the settings of the Cloudflare zone
- the PocketBase settings that the operator sets
- the environment variables of the app
- the Go tests of the app and their test database
- the UI tests
- the end-to-end tests, their app, their fakes, and their projects
- the accessibility checks
- the coverage commands and minimums
- `.github/workflows/ci.yml`, its jobs, its triggers, its concurrency, its caches, and its artifacts
- the setup action `.github/actions/setup`
- the tool Go version of the makefile
- `.github/workflows/renovate.yml`, `.github/renovate.json`, and the `renovate` targets of the makefile

Out of scope:
- a push of the image to a registry, and a deploy from CI
- the backup task of the host. The operator sets it up.
- more than one instance
- monitoring and log collection
- OAuth2 sign in. The backlog has it under "Won't do".
- the SDK tests. Decision 00001 has them.
- visual comparison with screenshots
- load tests and performance budgets. The backlog has them under "Won't do".
- tests of the PocketBase dashboard
- real mobile devices. A small viewport in Chromium covers the small screen layout.
- the coverage of the end-to-end tests, a coverage report on a pull request, and a coverage service
- a required status check. GitHub needs a paid plan or a public repository for branch protection of this private repository. A person reads `CI result` before a merge.
- runs on a schedule of `ci.yml`, and release tags
- a run of `make generate` by Renovate

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

`deploy/README.md` is the guide for the operator. It has the steps to start, update, back up, and restore.

| Service | Image | Use |
|---|---|---|
| `app` | `olive:local` | the app server on port 8090 with the `pb_data` volume |
| `cloudflared` | `cloudflare/cloudflared` | the Cloudflare Tunnel. It sends the requests of the public host name to `app:8090`. |

- `deploy/.env` has `CLOUDFLARE_TUNNEL_TOKEN`, `PB_ENCRYPTION_KEY`, and the `REVIEWS_` variables of the app
- `compose.yaml` names each `REVIEWS_` variable. A new variable of the app needs a line there.
- `make build-image` builds the image `olive:local` from `deploy/Dockerfile`
- the app applies the migrations when it starts
- to make the first superuser, run `olive superuser create EMAIL PASSWORD` with `--encryptionEnv=PB_ENCRYPTION_KEY`

## One instance

- SQLite has one writer process
- the rate limits of decision 00003 and the place cache of decision 00008 are in the memory of the process. A restart sets the limits to zero. This is accepted.
- a second instance needs a shared store for these values. That needs a new decision.

## Settings of the operator

In the PocketBase dashboard at `/_/`:
- **Backups**: turn on the auto backups. Without them, the app has no backup.
- **Application URL**: the public URL of the site. The link previews of decision 00002 start with it.
- **Trusted proxy**: the header `CF-Connecting-IP`. Otherwise all clients have the IP address of the tunnel, and the sign in limit counts them together.
- **Superuser IP allow-list**: only the admin networks can open the dashboard
- **Mail server**: set it before the first sign up. Without SMTP, the app writes each email to the log.
- **OTP and MFA** for the `_superusers` collection, after the mail server
- **Approval**: approve new users in the `users` collection. See decision 00003.

Cloudflare must not set a `Referrer-Policy` of `no-referrer`, and must not turn off `geolocation` in a `Permissions-Policy`. See decision 00008.

## Backups

- PocketBase makes each backup as one ZIP file of all of `pb_data`, with the databases and the photo files
- a cron expression sets the time, and a number sets how many backups stay
- the backups go to `pb_data/backups` on the local file system
- the backup task of the host copies this folder to a different computer. It starts after the auto backups.
- a restore on a server that runs is one action in the dashboard. A new server gets the content of the ZIP file in its volume.
- the backups do not have `PB_ENCRYPTION_KEY`. The operator keeps a copy, and a restore needs it.

## Environment variables

| Variable | Decision |
|---|---|
| `REVIEWS_SIGN_IN_LIMIT`, `REVIEWS_SIGN_IN_FAILURE_LIMIT`, `REVIEWS_CREATE_LIMIT`, `REVIEWS_EDIT_LIMIT`, `REVIEWS_PHOTO_UPLOAD_LIMIT`, `REVIEWS_PLACE_SEARCH_LIMIT`, `REVIEWS_EMAIL_LIMIT` | 00003 |
| `REVIEWS_PLACES_URL`, `REVIEWS_MAP_TILE_URL`, `REVIEWS_MAP_ATTRIBUTION` | 00008 |
| `REVIEWS_VALIDATE_RESPONSES`, `REVIEWS_VALIDATE_REQUESTS` | 00001 |
| `PB_ENCRYPTION_KEY` | this decision |
| `APP_MEMORY_LIMIT`, `APP_GOMEMLIMIT` | this decision |

## Tests

```bash
make test-app         # Go tests without the race detector. Seconds.
make test-race        # Go tests with the race detector. About 50 seconds.
make test-ui          # UI tests
make test-coverage    # Go and UI tests with coverage and the race detector. Fails below the minimum.
make test-e2e         # Build the app, then run the end-to-end tests in all projects
make test-e2e PROJECT=chromium E2E_FILE=e2e/auth.spec.ts
make test-e2e-ui      # Open the Playwright UI mode
make install-browsers # Install the Playwright browsers. BROWSER=firefox installs one.
make install-browser-deps # Install the system packages of the browsers on Linux. Needs sudo.
```

While you work:
- run `make test-app`. Run `make test-race` or `make test-coverage` one time before the end. Each has the race detector. Do not run both.
- for one Go test, run `go test ./internal/publicapi -run TestName` in `app/api`
- run one browser with `make test-e2e PROJECT=chromium`, and add `E2E_FILE` for the specs of your change. CI runs all projects.

### Go tests

- a test makes its app with `tests.NewTestApp(testapp.Dir(t))`. This database has the production schema.
- a test with many sign ups uses `testapp.LowCostDir(t)`. The `publicapi` tests use it.
- each test package needs a `TestMain` that calls `testapp.Main`
- each test starts with `t.Parallel()`. A test that changes a package variable or calls `t.Setenv` has no `t.Parallel()`.
- the tests send requests through `apis.NewRouter`, so they test the Public API with the PocketBase middlewares
- query plan tests check that the list queries use their indexes. They compare only the index name and a short part of the range.

### End-to-end tests

```ts
test('a signed out person signs in to write a review', async ({ page, api }) => {
  const restaurant = await api.createRestaurant({ name: uniqueTag('Luigi') })
  const user = await api.signUp()
  await page.goto(`/restaurants/${restaurant.id}`)
  await page.getByRole('link', { name: 'Write a review' }).click()
  await expect(page).toHaveURL(/\/sign-in\?redirect=/)
  await page.getByLabel('Email').fill(user.email)
  await page.getByLabel('Password').fill(user.password)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('heading', { name: 'Write a review' })).toBeVisible()
})
```

- tests find elements by role and accessible name, as a screen reader does. They use no CSS classes or test IDs.
- a test that cannot find an element by its role and name shows an accessibility problem
- tests make their data through `@reviews/sdk`, not through the UI
- each test makes its own users with unique email addresses
- a test that searches or filters gets its name or tag from `uniqueTag`. All projects of one run share one app.
- `api.signUp` confirms and approves the user with the superuser of the test app. `api.signUpPending` only confirms.
- `api.emailLink` reads the link of an email from the log of the test app. The test app has no mail server.
- `tests/photon.ts` is the fake geocoder and the fake map tiles. The tests need no network.

| Project | Browser | Viewport |
|---|---|---|
| `chromium`, `firefox`, `webkit` | each browser | 1280×800 |
| `chromium-small` | Chromium | 375×812 |
| `rate-limit` | Chromium | only `rate-limit.spec.ts`, with its own app and the default limits |

A new project needs an entry in the `e2e` matrix of CI. `make check-ci-matrix` fails without it.

### Coverage

| Part | Minimum | Where |
|---|---|---|
| Go statements of `app/api` | 90% | `COVERAGE_MIN_APP` of the makefile |
| UI statements and lines | 55% | `app/ui/vite.config.ts` |

Each minimum is about three points below the value. A small change does not fail CI, and a large fall does.
Raise a minimum when the value grows.

## CI and dependency updates

### CI

`.github/workflows/ci.yml` runs on each push to `main` and on each pull request.

| Job | Command | Needs |
|---|---|---|
| `lint` | `make install-tools`, `make lint` | Go, Node.js, and Docker for OpenAPI Generator |
| `security-scan` | `make security-scan` | Go and the network |
| `test` | `make test-sdk test-coverage` | Go, Node.js, and a C compiler for the race detector |
| `docker` | `make build-image`, then `docker compose config` of `deploy/compose.yaml` | Docker |
| `e2e` | `make install-browsers`, `make install-browser-deps`, and `make test-e2e` for one project | Go, Node.js, and one browser. A matrix has one job for each Playwright project. |
| `CI result` | one shell step that reads the results of the other jobs | nothing |

- `test` runs the tests of `make test` with the race detector and the coverage minimums
- a pull request shows each job. A person reads `CI result` first. A red `CI result` points to the job that failed.
- a new Playwright project needs an entry in the `e2e` matrix. `make check-ci-matrix` fails without it, and `make lint` runs it.

### Renovate

- the `Renovate` workflow runs at 04:17 UTC on the first day of each month. It opens or updates one pull request with the branch `renovate/all`.
- start it by hand from the Actions tab at other times. The `log-level` input `debug` shows each dependency.
- the pull request has the updates of the Go modules, the npm packages, the GitHub Actions of `.github/actions/setup`, the image tags of `deploy/`, and the tool versions of the makefile
- major updates are in the same pull request. A person can remove one update when CI fails.
- the repository setting "Allow GitHub Actions to create and approve pull requests" must be on
- GitHub does not start CI for a pull request of the `GITHUB_TOKEN`. A person with write access approves the run.
- Renovate does not update the files in `.github/workflows`. A person updates them.
- when Renovate changes `TOOLS_GO`, `OAPI_CODEGEN_VERSION`, or `OPENAPI_GENERATOR_IMAGE`, the `lint` job fails. Run `make generate` on the branch and push the result.

The makefile versions:

| Variable | Datasource |
|---|---|
| `VACUUM_VERSION`, `OAPI_CODEGEN_VERSION`, `GOVULNCHECK_VERSION` | `go`, with the module of the `renovate` comment |
| `TOOLS_GO` | `golang-version` |
| `OPENAPI_GENERATOR_IMAGE`, `RENOVATE_IMAGE` | `docker` |

A new Go tool needs a comment with its module above the version:

```make
# renovate: depName=github.com/daveshanley/vacuum
VACUUM_VERSION := v0.33.2
```

A new image needs a variable with a name that ends in `_IMAGE`.
Renovate does not do a major update of a Go tool with a `/vN` module, such as `oapi-codegen/v2`. A person changes the module path and the version.

```bash
make renovate-local   # Find the updates. It changes no file. The report is in bin/renovate-report.json.
make renovate         # Open or update the pull request. Needs RENOVATE_TOKEN and RENOVATE_REPOSITORIES.
```

`make renovate-local` checks a change of `.github/renovate.json`. Set `GITHUB_COM_TOKEN` to also get the updates of the GitHub Actions.

# Implementation Details
[implementation-details]: #implementation-details

## Image

- the build context is the root of the repository. `.dockerignore` keeps `node_modules`, `bin`, `pb_data`, the docs, and the tests out of it.
- each base image has an exact version tag. Renovate updates the tags. See [Renovate](#renovate).
- stage `ui` uses the Node.js 22 image. It runs `npm ci` for the SDK and UI workspaces and builds both.
- stage `build` uses the Go image with the toolchain version of `app/api/go.mod`. It copies the UI build into `app/api/internal/webui/dist` and builds the app with `CGO_ENABLED=0`.
- the last stage uses Alpine with `tzdata` and `ca-certificates`. The app runs as the user `reviews` with the ID 10001.
- the image has a health check on `/api/health`

## Settings encryption

- the start command has `--encryptionEnv=PB_ENCRYPTION_KEY`. `compose.yaml` stops without the variable.
- the app does not start with a key that does not have 32 characters. The example value of `.env.example` is such a key.
- a `olive` command with `docker compose exec` does not use the start command. It needs the flag, or it cannot read the settings.

## Limits

| Limit | Value | Reason |
|---|---|---|
| `mem_limit` of `app` | 768 MB | The server has 1 GB. `cloudflared` and the system get the rest. |
| `GOMEMLIMIT` | 600 MiB | The Go runtime frees memory before Docker stops the container. The photo uploads of decision 00007 peak at about 264 MB. |
| `ulimits.nofile` | 65536 | Each connection and each SQLite file uses a file descriptor. |
| request body of Cloudflare | 100 MB | The plan of the account sets it. The limit of the app is 10 MiB. A body between the two limits gets the `413` of the app with Problem Details. |

## Cloudflare Tunnel

- `cloudflared` opens an outbound connection to Cloudflare. The server needs no open port and no certificate.
- the tunnel is remotely managed. Its public hostname and its target `app:8090` are in the Cloudflare dashboard. `deploy/.env` has only its token.
- Cloudflare has the certificate of the site and compresses the responses
- Cloudflare sets `CF-Connecting-IP` to the address of the client. Only `cloudflared` can reach the app, so a client cannot set this header.
- the session cookie of decision 00003 gets `Secure` from the `X-Forwarded-Proto` header that Cloudflare sets
- Cloudflare caches only the paths with a file extension of its default list. It keeps the `Cache-Control` of the app. The photo paths have no extension, so Cloudflare does not cache them.
- Cloudflare stops a request after 100 seconds with a `524`. "New backup" in the dashboard waits for the ZIP file, so a large backup can get it. The backup continues, and the operator looks for it in the list.
- Rocket Loader, the email obfuscation, and the automatic Web Analytics add scripts. The Content Security Policy blocks them, so the zone turns them off.
- Bot Fight Mode is off. Its challenge blocks SDK clients with a bearer token and link preview bots.
- Cloudflare decrypts the traffic at its edge. It sees the passwords and the session cookies. This is accepted.

## Limits of the backup plan

- each ZIP file has all photos again. The disk needs space for `pb_data` and for `pb_data` times the number of backups.
- the app can be slower while a backup runs
- PocketBase advises the `sqlite3` and `rsync` plan for a `pb_data` of 5 GB or more. The site then needs a new decision.
- a failed backup shows only in the logs of the dashboard
- many host backups skip `/var/lib/docker`. The operator adds the folder of the volume to the task.
- a copy of the database files that the host task makes while the app runs can be broken. A restore uses a ZIP file.

## Daily jobs

- `photo-cleanup` deletes the photo files without a record. See decision 00007.
- `user-cleanup` deletes the users that wait for more than 30 days and have no data. See decision 00003.
- superusers see and start both under Settings, Crons in the dashboard

## Checks of the deploy

- the `docker` job of CI runs `make build-image` and checks `deploy/compose.yaml` with `docker compose config`
- tested by hand: the image builds and gets the state `healthy`, a backup ZIP file restores on a new server, the key rules, the memory limits, and the `413` of the app
- not tested: the `cloudflared` service with a real tunnel, the 100 MB limit of Cloudflare, `CF-Connecting-IP`, the `Secure` cookie behind the tunnel, the copy by a real host backup task, and the steps for the mail server and the MFA

## Tests

### The Go test database

- `testapp.Dir` makes its template folder one time for each test binary with a `sync.Once`. It calls `tests.NewTestApp` with an empty folder, which runs the migrations of the binary.
- `tests.NewTestApp` copies the template folder for each app. `RunAllMigrations` then finds no new migration, and the tests share no database.
- `testapp.LowCostDir` copies the folder of `Dir` one time and sets the bcrypt cost of each auth collection to 4. Production and `testapp.Dir` have the default cost 10.
- `testapp.Main` deletes the template folders after `m.Run`
- `testapp` does not import `migrations`. The tests of `migrations` can then use it without an import cycle.
- the record hooks are not in the template. Each test registers them after the migrations.
- each `publicapi.Register` makes its own rate limiters
- `make test-coverage-app` gives `-coverpkg` all packages except `internal/testapp`
- `TestDummyHashUsesThePasswordCost` checks the dummy hash of sign in with the cost 10 and the cost 4

### The UI tests

- Vitest with Vue Test Utils in jsdom, and `@vitest/coverage-v8`
- the component tests run `axe-core` directly. `vitest-axe` has no release since 2022.
- jsdom has no layout, so the UI tests do not test `MapView`

### The end-to-end app

- `make test-e2e` runs `make build-app` first. The tests use `./bin/olive` with the embedded UI build, as a release does.
- `global-setup.ts` calls `startApp` of `app.ts`. It starts the binary on `127.0.0.1:8097` with a new folder and waits for `/api/health`. The fake geocoder runs on port 8096.
- `startApp` makes a folder with the `reviews-e2e-` prefix in the temp folder. `stop` deletes only this folder.
- each app has a superuser. `startApp` signs it in, and `global-setup.ts` gives the token to the test workers. `tests/app.ts` sets the Application URL `https://reviews.example.com`.
- `global-setup.ts` raises the sign up, create, edit, photo upload, and place search limits of the app. Parallel sign ups come from one IP address.
- the `noCspViolations` fixture of `tests/fixtures.ts` runs in each test. See decision 00002.
- `@playwright/test`, `playwright-core`, and `@axe-core/playwright` have pinned versions. `@axe-core/playwright` has `playwright-core` as a peer dependency, and a newer one would not match the types.

### Accessibility

- `accessibility.spec.ts` runs `@axe-core/playwright` on the main screens in the light and the dark theme
- each screen has its own test in each theme, with its own data. The tests then run in parallel.
- the color contrast rule runs, because the browser renders the page
- a test fails on each violation of WCAG 2.2 level A or AA
- keyboard tests check the skip link, the focus order of the sign in form, and that Escape closes a dialog

### Failures

- a failed test runs again one time. A test that passes only on the retry is flaky in the report.
- `trace` and `video` are `on-first-retry`. A trace and a video of each test make a run about one third slower. A screenshot of each failure stays.
- Playwright keeps the results in `tests/test-results`. `make test-e2e` prints the path of the HTML report. `.gitignore` ignores both folders.
- `E2E_FILE` goes to Playwright as a file filter. The name has a prefix, because make reads each environment variable as a make variable.
- WebKit on Linux needs system packages. `make install-browser-deps` installs them.

## CI and dependency updates

### The workflow

- one workflow file, because `needs` works only inside one workflow
- `permissions: contents: read`. No job writes to the repository.
- `concurrency` cancels an older run of the same pull request. A push to `main` gets its run ID in the group, so GitHub never cancels a waiting run of `main`.
- each job has `timeout-minutes: 20`. A hung job then fails, and `CI result` fails with it.
- `CI result` runs with `if: ${{ !cancelled() }}`. Without a condition, a failed job skips it, and GitHub shows a skipped job as not failed. `always()` would show a red result for each cancelled run.
- `CI result` fails for `failure`, `cancelled`, and `skipped`. A skipped job did not prove that the code is good.
- the `e2e` matrix has `fail-fast: false`, so the other projects finish after one fails
- each `e2e` job installs only its browser and its system packages. Playwright also installs ffmpeg, so the video of a retry works.
- the workflow does not cache the browsers. One browser downloads in less than a minute, and a cache does not keep the system packages.
- a failed `e2e` job uploads `playwright-report-<project>` with the traces for 7 days
- each `e2e` job builds the app. A shared build job can replace this later.
- `tests/check-ci-matrix.mjs` compares the matrix with the projects of `npx playwright test --list`

### The setup action

- `.github/actions/setup/action.yml` runs `actions/setup-go` with `go-version-file: app/api/go.mod` and `actions/setup-node` with Node.js 22
- one cache has the Go modules, the Go build cache, and the npm cache. Its key is a hash of the `go.sum` files and `package-lock.json`.
- only the `test` job saves the cache. The other jobs only restore it. The built-in caches of the setup actions are off, because the jobs would race for one key.

### The tool Go version

- the Go SDK embeds the spec as gzip data. The Go version that builds `oapi-codegen` changes these bytes. See decision 00001.
- the makefile sets `TOOLS_GO`. `install-tools` runs `GOTOOLCHAIN=$(TOOLS_GO) go install`, so Go downloads that version when the computer has another one.
- the tools go to `bin/tools-$(TOOLS_GO)/`. A new `TOOLS_GO` gets a new folder. The old tools stay until a person deletes them.
- `oapi-codegen` and `vacuum` then have the same bytes on each computer and in CI, and `check-generate` compares the same output
- the tests and the app build keep the Go version of `app/api/go.mod`

### The make targets

| Target | Command |
|---|---|
| `install-browsers` | `cd tests && npx playwright install $(BROWSER)`. An empty `BROWSER` installs all browsers. |
| `install-browser-deps` | `cd tests && npx playwright install-deps $(BROWSER)`. It needs sudo on Linux. |
| `check-ci-matrix` | `node tests/check-ci-matrix.mjs` |
| `build-image` | `docker build -f deploy/Dockerfile -t olive:local .` |
| `security-scan` | `govulncheck` on the Go modules |

`test-e2e` takes `PROJECT` and `E2E_FILE`. See [Tests](#tests).

### Renovate

- `make renovate` runs the Renovate image in Docker. The workflow does not use `renovatebot/github-action`, so the makefile has the one Renovate version.
- `make renovate` fails without `RENOVATE_TOKEN` or `RENOVATE_REPOSITORIES`. Docker gets the token from the environment, so the token is not in the command.
- the workflow gives its `GITHUB_TOKEN` to Renovate. The job gives the token write access to contents, pull requests, and issues. The token stops at the end of the job.
- GitHub refuses a push of the `GITHUB_TOKEN` that changes a file in `.github/workflows`. With `group:all`, one such change would stop all updates, so a package rule turns them off.
- `.github/renovate.json` has `config:recommended` and `group:all`. `rangeStrategy` is `pin`, so the versions stay exact. `prHourlyLimit` is `0`.
- each action has a commit SHA and a comment with its full version. A moved tag then changes no run.
- `helpers:pinGitHubActionDigestsToSemver` keeps this rule for the updates of `.github/actions/setup`
- a person pins an action of `.github/workflows` to the SHA of a release. `gh api repos/OWNER/REPO/commits/TAG --jq .sha` shows it.
- the cron of the workflow is the schedule. The config has no `schedule` preset, because a late run or a run by hand would then do nothing.
- `gomodTidy` runs `go mod tidy` after a Go update
- three custom managers of the type `regex` read the makefile. A package rule turns off the major updates of the Go tools with a `/vN` module.
- the Dependency Dashboard issue changes only when the workflow runs. A click in the dashboard has its effect in the next run.
- `make renovate-local` uses `--platform=local` and `--dry-run=lookup`

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| A `backup` service with `sqlite3 .backup` and `rsync` | Removed. It needs a script, tools in the image, and a second step to get the files off the server. |
| The backups of PocketBase on S3 storage | Removed. The host backup already copies the files off the server. A bucket and an access key are one more thing to keep. |
| `DefaultEncryptionEnv` in `main.go` | Not selected. A local run with the variable in its environment would encrypt the local `pb_data`. |
| nginx with certbot | Not selected. It needs a second container and a renew job. |
| PocketBase serves TLS without a proxy | Not selected. The compression, HTTP/3, and a later second site need a proxy. |
| Caddy on the ports 80 and 443 | Removed. The home server then needs open ports and a public IP address. The tunnel needs neither. |
| More than one instance | Not selected. The limits and the cache are in memory, and SQLite has one writer. |
| Cypress | Not selected. Its WebKit support is experimental, and calls to the API for test data are harder. |
| Vitest browser mode | Not selected. It tests components, not the built app with its routes. |
| A new migrated database for each Go test | Removed. With the race detector, a new app took 2.5 seconds, and `make test-app` took 7 minutes. |
| The race detector in `make test-app` | Removed. It made the run of each change slow. `make test-race` and `make test-coverage` have it. |
| The bcrypt cost 10 in all tests | Removed for `publicapi`. One hash took 0.55 seconds with the race detector. |
| A trace and a video of each test | Removed. They made each run slower. The retry records them. |
| One axe test for all screens | Removed. It went over the timeout in Firefox. |
| More workers | Not selected. 16 workers were slower than 8. |
| A fake mail server | Not selected. The log of the test app has the links. |
| One job that runs all make targets | Removed. A failure did not name the part that failed. |
| One workflow for each job, with a `workflow_run` workflow for the result | Not selected. `workflow_run` runs in the context of the default branch and starts later. |
| A third-party action for the result | Not selected. One shell step does the same without code from another repository. |
| `check-generate` compares the decoded spec | Not selected. A later generator change can change other output. |
| `toolchain` with a fixed patch version in `go.mod` | Not selected. Only the tools need it. |
| One `e2e` job for all browsers | Not selected. It takes about 9 minutes and hides which browsers pass. |
| The hosted Renovate app | Removed. It needs an install on the repository, and a third party runs the updates. |
| A personal token for Renovate | Not selected. It has more access and no end, and a person must renew it. |
| A run of `make generate` by Renovate | Not selected. The container would need Go, the Docker socket, and `allowedCommands`. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
