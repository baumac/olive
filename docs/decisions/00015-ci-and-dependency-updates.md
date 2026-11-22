---
description: A GitHub Actions workflow runs the lint, the security scan, the tests with coverage, the end-to-end tests of each Playwright project, and the Docker build. Each step calls make, and one CI result job gives the result. A monthly Renovate workflow opens one pull request with all dependency updates.
status: implemented
---

# Motivation
[motivation]: #motivation

AI coding tools write the app, and each change goes through CI. See decision 00000.
A failed check must name the part that failed. A person then reads one result before a merge.
The versions of the dependencies are exact, so nothing tells when a dependency has an update or a security fix.

Expected outcome:
- a failed check names its job: lint, security scan, test, end-to-end, or Docker
- one job, `CI result`, gives one pass or fail for each run
- each run step that does project work calls a `make` target
- `make lint` gives the same result on each computer and in CI
- one pull request each month has all dependency updates

# Scope
[scope]: #scope

In scope:
- `.github/workflows/ci.yml`, its jobs, its triggers, its concurrency, its caches, and its artifacts
- the setup action `.github/actions/setup`
- the tool Go version of the makefile
- `.github/workflows/renovate.yml`, `.github/renovate.json`, and the `renovate` targets of the makefile

Out of scope:
- the tests themselves. Decision 00014 has them.
- a push of the image to a registry, and a deploy from CI
- a required status check. GitHub needs a paid plan or a public repository for branch protection of this private repository. A person reads `CI result` before a merge.
- runs on a schedule of `ci.yml`, and release tags
- a run of `make generate` by Renovate

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## CI

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

## Renovate

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

## The workflow

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

## The setup action

- `.github/actions/setup/action.yml` runs `actions/setup-go` with `go-version-file: app/api/go.mod` and `actions/setup-node` with Node.js 22
- one cache has the Go modules, the Go build cache, and the npm cache. Its key is a hash of the `go.sum` files and `package-lock.json`.
- only the `test` job saves the cache. The other jobs only restore it. The built-in caches of the setup actions are off, because the jobs would race for one key.

## The tool Go version

- the Go SDK embeds the spec as gzip data. The Go version that builds `oapi-codegen` changes these bytes. See decision 00001.
- the makefile sets `TOOLS_GO`. `install-tools` runs `GOTOOLCHAIN=$(TOOLS_GO) go install`, so Go downloads that version when the computer has another one.
- the tools go to `bin/tools-$(TOOLS_GO)/`. A new `TOOLS_GO` gets a new folder. The old tools stay until a person deletes them.
- `oapi-codegen` and `vacuum` then have the same bytes on each computer and in CI, and `check-generate` compares the same output
- the tests and the app build keep the Go version of `app/api/go.mod`

## The make targets

| Target | Command |
|---|---|
| `install-browsers` | `cd tests && npx playwright install $(BROWSER)`. An empty `BROWSER` installs all browsers. |
| `install-browser-deps` | `cd tests && npx playwright install-deps $(BROWSER)`. It needs sudo on Linux. |
| `check-ci-matrix` | `node tests/check-ci-matrix.mjs` |
| `build-image` | `docker build -f deploy/Dockerfile -t reviews-app:local .` |
| `security-scan` | `govulncheck` on the Go modules |

`test-e2e` takes `PROJECT` and `E2E_FILE`. See decision 00014.

## Renovate

- `make renovate` runs the Renovate image in Docker. The workflow does not use `renovatebot/github-action`, so the makefile has the one Renovate version.
- `make renovate` fails without `RENOVATE_TOKEN` or `RENOVATE_REPOSITORIES`. Docker gets the token from the environment, so the token is not in the command.
- the workflow gives its `GITHUB_TOKEN` to Renovate. The job gives the token write access to contents, pull requests, and issues. The token stops at the end of the job.
- GitHub refuses a push of the `GITHUB_TOKEN` that changes a file in `.github/workflows`. With `group:all`, one such change would stop all updates, so a package rule turns them off.
- `.github/renovate.json` has `config:recommended` and `group:all`. `rangeStrategy` is `pin`, so the versions stay exact. `prHourlyLimit` is `0`.
- the cron of the workflow is the schedule. The config has no `schedule` preset, because a late run or a run by hand would then do nothing.
- `gomodTidy` runs `go mod tidy` after a Go update
- three custom managers of the type `regex` read the makefile. A package rule turns off the major updates of the Go tools with a `/vN` module.
- the Dependency Dashboard issue changes only when the workflow runs. A click in the dashboard has its effect in the next run.
- `make renovate-local` uses `--platform=local` and `--dry-run=lookup`

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
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
