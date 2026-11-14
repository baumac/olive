---
description: Add a test coverage command with a minimum, a coverage job in CI, and a monthly Renovate pull request.
status: implemented
---

# Motivation
[motivation]: #motivation

- nobody saw when a change removed tests or added code without tests. CI had no coverage number.
- the versions of the dependencies are exact. Nothing told when a dependency had an update or a security fix.
  `make security-scan` finds only known vulnerabilities of the Go modules.

Expected outcome:
- CI fails when the coverage falls below a minimum
- one pull request each month has all dependency updates

# Scope
[scope]: #scope

In scope:
- the `make test-coverage` target, and its use in the `test` job of CI
- `.github/renovate.json`

Out of scope:
- the coverage of the end-to-end tests
- a coverage report on a pull request, and a coverage service
- the tool versions of the makefile. Renovate does not read them without custom rules.
- the install of the Renovate app on the repository. The owner of the repository does this one time.

Decision 00026 replaces the Renovate app with a workflow, removes the `schedule:monthly` preset, and adds the tool versions of the makefile.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

`make test-coverage` runs the Go tests of the app and the UI tests with coverage.

| Part | Minimum | Value on 2026-11-14 | Where the minimum is |
| --- | --- | --- | --- |
| Go statements of `app` | 65% | 68.8% | `COVERAGE_MIN_APP` in the makefile |
| UI statements | 55% | 58.2% | `app/ui/vite.config.ts` |
| UI lines | 55% | 59.1% | `app/ui/vite.config.ts` |

Each minimum is about three points below the value. A small change then does not fail CI, and a large fall does.
Raise a minimum when the value grows.

Renovate opens one pull request on the first day of each month. It has all updates of the Go modules, the npm packages, the GitHub Actions, and the image tags of `deploy/Dockerfile` and `deploy/compose.yaml`.
Major updates are in the same pull request. A person can remove one update from it when CI fails.

# Implementation Details
[implementation-details]: #implementation-details

- the Go command uses `-coverpkg=./...`. A test of one package then counts the code that it runs in other packages.
- the Go target uses the race detector, like `make test-app`. One run then gives the test result and the coverage.
- the UI uses `@vitest/coverage-v8` with the version of Vitest
- the `test` job of CI runs `make test-sdk test-coverage`. The tests of the app and the UI then run one time for each push.
- Renovate uses the presets `config:recommended`, `group:all`, and `schedule:monthly`. `rangeStrategy` is `pin`, so the versions stay exact.
- `gomodTidy` runs `go mod tidy` after a Go update

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
