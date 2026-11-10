---
description: Split the GitHub Actions workflow into lint, test, end-to-end, and Docker jobs that call make, with one job that gives the result.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00006 added `.github/workflows/ci.yml` with two jobs:
- `lint-and-test` runs `make lint` and `make test`
- `e2e` installs the browsers with `npx playwright install --with-deps` and runs `make test-e2e`

The backlog asks for more:
- a job for lint, a job for test, and a job for the Docker build
- each task of a workflow runs through the makefile
- one result that combines the jobs into pass or fail

The workflow also has a problem today.
Each push to `main` since e9bcfa4 fails `make lint` at `check-generate`.
The end-to-end job passes, also in WebKit.
The cause is the Go version that builds `oapi-codegen`:
- the Go SDK embeds the spec as gzip data. The Go version of the `oapi-codegen` binary changes these bytes.
- the committed `app/sdk/go/reviewsapi.gen.go` comes from a binary that Go 1.26.8 built
- CI builds `oapi-codegen` with Go 1.27.3, which `actions/setup-go` selects for `go 1.27` in `app/go.mod`
- the same spec and `oapi-codegen` v2.8.2 built with Go 1.27.3 give an embedded spec that differs in 279 lines

Expected outcome:
- a failed check names its job: lint, test, end-to-end, or Docker
- one job, `CI result`, gives one pass or fail for each run
- each run step that does project work calls a `make` target
- `make lint` gives the same result on each computer and in CI

# Scope
[scope]: #scope

In scope:
- the jobs, their `make` targets, and their order in `.github/workflows/ci.yml`
- the `CI result` job
- the new `make` target `install-browsers`, and the `build-image` target that comes with the deploy item
- one Go version for the tools in `bin/`
- the triggers, the concurrency, the timeouts, the caches, and the artifacts of the workflow

Out of scope:
- the Dockerfile and the deploy folder. The backlog item "Add deploy folder and docker compose setup" has them.
- a push of the image to a registry, and deploys
- a required status check. `CI result` is not a required check for now. GitHub needs a paid plan or a public repository for branch protection of this private repository. The API answers `403`. A person reads `CI result` before a merge.
- runs on a schedule, and release tags
- the system packages of WebKit on developer computers

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

New concepts:
- **Job**: one part of the workflow. Jobs run at the same time on their own runners.
- **CI result**: the last job. It passes only when each other job passes.
- **Tool Go version**: the one Go version that builds the tools in `bin/`, such as `oapi-codegen`.

| Job | Steps after the setup | What it needs |
| --- | --- | --- |
| `lint` | `make install-tools`, `make lint` | Go, Node.js, Docker for OpenAPI Generator |
| `test` | `make test` | Go, Node.js |
| `e2e` | `make install-browsers BROWSER=...`, `make install-browser-deps BROWSER=...`, `make test-e2e PROJECT=...` | Go, Node.js, one Playwright browser. A matrix runs one job for each Playwright project. |
| `docker` | `make build-image` | Docker. It comes with the deploy item. |
| `CI result` | one shell step that reads the results of the other jobs | nothing |

The setup steps are actions, not `make` targets: `actions/checkout`, `actions/setup-go`, and `actions/setup-node`.
A local action `.github/actions/setup` has them, so each job has one setup step.

A pull request shows each job.
A person reads `CI result` first.
A red `CI result` points to the job that failed.

```yaml
jobs:
  lint:
    runs-on: ubuntu-24.04
    timeout-minutes: 20
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/setup
      - run: make install-tools
      - run: make lint
  # test has the same form. The End-to-end matrix section shows e2e.
  result:
    name: CI result
    if: ${{ !cancelled() }}
    needs: [lint, test, e2e]
    runs-on: ubuntu-24.04
    steps:
      - if: contains(needs.*.result, 'failure') || contains(needs.*.result, 'cancelled') || contains(needs.*.result, 'skipped')
        run: exit 1
```

# Implementation Details
[implementation-details]: #implementation-details

## Workflow

- one workflow file, `.github/workflows/ci.yml`. `needs` works only inside one workflow.
- the triggers stay: each push to `main` and each pull request
- `permissions: contents: read` stays. No job writes to the repository.
- `concurrency` cancels an older run of the same pull request. Its group is `ci-` and the ref for a pull request.
- each push to `main` gets the group `ci-` and its run ID. A group keeps at most one running and one waiting run, and GitHub cancels an older waiting run. A shared group for `main` could then cancel the run of a push.
- `cancel-in-progress` is true only for pull requests
- each job has `timeout-minutes: 20`. A hung job then fails, and `CI result` fails with it.
- `CI result` runs with `if: ${{ !cancelled() }}`. Without a condition, a failed job skips `CI result`, and GitHub shows a skipped job as not failed.
- a cancelled run skips `CI result`. `always()` would run it, and each cancelled run of a pull request would then show a red `CI result`.
- `CI result` fails for `failure`, `cancelled`, and `skipped` results. A skipped job did not prove that the code is good.
- each `e2e` matrix job keeps the upload of its Playwright report and its traces for 7 days after a failure

## End-to-end matrix

The `e2e` job has a matrix with one entry for each project of `tests/playwright.config.ts`:

| `project` | `browser` |
| --- | --- |
| `chromium` | `chromium` |
| `firefox` | `firefox` |
| `webkit` | `webkit` |
| `chromium-small` | `chromium` |
| `rate-limit` | `chromium` |

```yaml
  e2e:
    name: e2e (${{ matrix.project }})
    strategy:
      fail-fast: false
      matrix:
        include:
          - { project: chromium, browser: chromium }
          - { project: firefox, browser: firefox }
          - { project: webkit, browser: webkit }
          - { project: chromium-small, browser: chromium }
          - { project: rate-limit, browser: chromium }
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/setup
      - run: make install-browsers BROWSER=${{ matrix.browser }}
      - run: make install-browser-deps BROWSER=${{ matrix.browser }}
      - run: make test-e2e PROJECT=${{ matrix.project }}
```

- each job installs only its browser and its system packages. Playwright also installs ffmpeg for each browser that `install` names. The video of a failed test then works in each job.
- the workflow does not cache the Playwright browsers. The download of all browsers takes about a minute, and one browser takes less. A cache also does not keep the system packages of `install-browser-deps`.
- `fail-fast: false` lets the other browsers finish after one fails. The run then shows each failed browser.
- `needs: [e2e]` in `CI result` waits for all matrix jobs. The result of `e2e` is `failure` if one matrix job fails.
- each failed job uploads its report as `playwright-report-<project>`. The names then do not collide.
- each job builds the app with `make test-e2e`. Five builds use about 5 to 10 more runner minutes than one. A build job that shares `bin/reviews-app` and the SDK build can replace them later. It needs a `make` target that runs the tests without the build.
- a new project in `tests/playwright.config.ts` needs a new matrix entry. Without it, CI does not run the project.
- `make check-ci-matrix` compares the matrix with the projects of `npx playwright test --list`. It fails for a missing or an extra project. `make lint` runs it, so the `lint` job fails for a missing project.
- each test of the rate limit project starts its own app on its own port, 8098 or 8099. The workers then share no app. A failed test does not skip the other test.

## Setup action

`.github/actions/setup/action.yml` is a composite action:
- `actions/setup-go@v5` with `go-version-file: app/go.mod`
- `actions/setup-node@v4` with Node.js 22
- one cache for the Go modules, the Go build cache, and the npm cache. Its key is a hash of `app/go.sum`, `app/sdk/go/go.sum`, and `package-lock.json`.
- only the `test` job saves the cache, with the input `save-cache: 'true'`. The other jobs only restore it with `actions/cache/restore`.
- the built-in caches of `setup-go` and `setup-node` are off. They save in each job, and seven jobs then race for one key.
- the `docker` job does not use it. It needs only Docker.

## New make targets

| Target | Command | Why |
| --- | --- | --- |
| `install-browsers` | `cd tests && npx playwright install $(BROWSER)` | the browsers come from `make`, like the other tasks. An empty `BROWSER` installs all browsers. It needs no sudo. |
| `install-browser-deps` | `cd tests && npx playwright install-deps $(BROWSER)` | the system packages of the browsers. It needs sudo on Linux. A developer who has them can skip it. |
| `check-ci-matrix` | `node tests/check-ci-matrix.mjs` | the `e2e` matrix and the Playwright projects stay equal |
| `build-image` | `docker build -f deploy/Dockerfile -t reviews-app:local .` | the `docker` job and developers build the image the same way |

- `test-e2e` gets the variable `PROJECT`. A value adds `--project=$(PROJECT)` to the Playwright command. An empty `PROJECT` runs all projects, as before.
- `install-browsers` depends on the npm install target. `npx` then finds the Playwright version of `tests/package.json`.
- `build-image` comes with the deploy folder. The deploy decision owns the Dockerfile.
- AGENTS.md lists `install-browsers`, `install-browser-deps`, `check-ci-matrix`, and the `PROJECT` variable of `test-e2e`

## One tool Go version

- the makefile sets `TOOLS_GO := go1.27.3`
- `install-tools` runs `GOTOOLCHAIN=$(TOOLS_GO) go install ...`. Go then downloads that version if the computer has a different one.
- the tools go to `bin/tools-go1.27.3/`. The tool targets are files, so `make` does not build a tool that exists. A new `TOOLS_GO` then gets a new folder, and `make` builds the tools in it.
- the old tools in `bin/` stay until a person deletes them. The makefile does not use them.
- `oapi-codegen` and `vacuum` then have the same bytes on each computer and in CI. `check-generate` compares the same output.
- the change runs `make install-tools` and `make generate` one time. The commit has the new `reviewsapi.gen.go`. It is equal to the output of `oapi-codegen` that Go 1.27.3 built in a test before the change.
- an update of `TOOLS_GO` is a change of its own. It runs `make generate` again.
- the tests and the app build keep the Go version of `app/go.mod`. Only the tools need the fixed version.

## Order with the deploy item

- the `docker` job needs `make build-image` and `deploy/Dockerfile`
- the change of this decision lands before the deploy item. The workflow has `lint`, `test`, and `e2e`, and `CI result` needs these three.
- the deploy change adds `make build-image`, the `docker` job, and the `docker` job in the `needs` of `CI result`. The backlog item of the deploy folder has these steps.

## Docs changed

- AGENTS.md: the tool folder, the new targets, the `PROJECT` variable, and the CI jobs
- decision 00006: the CI section names the new jobs
- the backlog: the CI item is gone. The deploy item has the `docker` job.

## Tests

- a pull request with a failing unit test shows a red `test` and a red `CI result`. `lint` and `e2e` stay green.
- a pull request that changes the spec without `make generate` shows a red `lint`
- `make lint` passes on a computer with Go 1.27.0 and in CI with Go 1.27.3. The tool Go version is the same for both.
- a cancelled run of a pull request does not leave a green `CI result`
- a failed WebKit test shows a red `e2e (webkit)`. The other `e2e` jobs stay green, and `CI result` is red.

## Alternatives

| Alternative | Result |
| --- | --- |
| One job that runs all make targets | Not selected. A failure then does not name the part that failed. The jobs also cannot run at the same time. |
| One workflow file for each job, with a `workflow_run` workflow for the result | Not selected. `workflow_run` runs in the context of the default branch and starts after the other workflows. `needs` in one workflow is simpler. |
| A third-party action for the result, such as `re-actors/alls-green` | Not selected. One shell step does the same. The workflow then runs no code from another repository. |
| `check-generate` compares the decoded spec, not the bytes | Not selected. A later generator change can change other output. One tool Go version gives the same bytes everywhere. |
| `toolchain go1.27.3` in `app/go.mod` | Not selected. The app and the tests do not need a fixed patch version. Only the tools in `bin/` need it. |
| One `e2e` job for all browsers | Not selected. It takes about 9 minutes. One failed browser also hides which other browsers pass. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
