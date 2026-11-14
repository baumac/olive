---
description: Run Renovate in a GitHub Actions workflow each month, and let it update the tool versions of the makefile.
status: implemented
---

# Motivation
[motivation]: #motivation

- decision 00024 added `.github/renovate.json` for the hosted Renovate app. The app needs an install on the repository, and a third party then runs the updates.
- Renovate did not read the tool versions of the makefile. A person had to find their updates.

Expected outcome:
- a workflow of the repository runs Renovate on the first day of each month
- one pull request has all updates, with the tool versions of the makefile

# Scope
[scope]: #scope

In scope:
- `.github/workflows/renovate.yml`
- the custom managers of `.github/renovate.json`, and the `renovate` comments of the makefile
- the `make renovate` and `make renovate-local` targets

Out of scope:
- the repository setting "Allow GitHub Actions to create and approve pull requests". The owner of the repository turns it on one time.
- the updates of the files in `.github/workflows`. The `GITHUB_TOKEN` cannot change them. A person updates the actions and the runners of the workflows.
- a run of `make generate` by Renovate with `postUpgradeTasks`. A person runs the command on the branch.
  The Renovate container then needs Go, the Docker socket of the runner, and the `/tmp` folder of the runner.
  The global config must also allow the command with `allowedCommands`.
- the minimum coverage values, and the Node.js version of `.github/actions/setup/action.yml`

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

The `Renovate` workflow runs at 04:17 UTC on the first day of each month. It opens or updates one pull request with the branch `renovate/all`.
Start it by hand from the Actions tab at other times, for example to rebase the pull request. The `log-level` input `debug` shows each dependency that Renovate reads.

The pull request has the updates of decision 00024 and these versions of the makefile:

| Variable | Renovate datasource |
| --- | --- |
| `VACUUM_VERSION`, `OAPI_CODEGEN_VERSION`, `GOVULNCHECK_VERSION` | `go`, with the module of the `renovate` comment |
| `TOOLS_GO` | `golang-version` |
| `OPENAPI_GENERATOR_IMAGE`, `RENOVATE_IMAGE` | `docker` |

A new Go tool in the makefile needs a comment with its module above the version:

```make
# renovate: depName=github.com/daveshanley/vacuum
VACUUM_VERSION := v0.33.2
```

A new image needs a variable with a name that ends in `_IMAGE`. It needs no comment.

Renovate does not do a major update of a Go tool with a major version in its module, such as `oapi-codegen/v2`.
The install command of the makefile has this module path, and Renovate changes only the version. A person changes the two for a major update.

`make renovate-local` runs Renovate in Docker on the files of the work folder. It changes no file and opens no pull request.
It writes the dependencies and their updates to `bin/renovate-report.json`. Use it to check a change of `.github/renovate.json` or a new `renovate` comment.
Without the environment variable `GITHUB_COM_TOKEN`, it skips the updates of the GitHub Actions. `LOG_LEVEL=debug` shows more.

The workflow runs `make renovate` with its `GITHUB_TOKEN`. The repository needs no secret and no personal token.
Turn on "Allow GitHub Actions to create and approve pull requests" in the Actions settings of the repository. Without it, Renovate cannot open the pull request.
GitHub does not start CI for a pull request of the `GITHUB_TOKEN`. A person with write access approves the run on the pull request.
The pull request has no updates of `.github/workflows/ci.yml` and `.github/workflows/renovate.yml`. It has the updates of `.github/actions/setup/action.yml`.

An update of `TOOLS_GO`, `OAPI_CODEGEN_VERSION`, or `OPENAPI_GENERATOR_IMAGE` can change the generated SDKs. The `lint` job of the pull request then fails.
Run `make generate` on the branch and push the result.

# Implementation Details
[implementation-details]: #implementation-details

- `make renovate` runs the Renovate image in Docker, as `renovatebot/github-action` does. The workflow does not use the action.
  The makefile then has the one Renovate version of the workflow and of `make renovate-local`. A person can also run the same command.
- `make renovate` fails without `RENOVATE_TOKEN` or `RENOVATE_REPOSITORIES`. Docker gets the token from the environment, so the token is not in the command.
- the workflow gives its `GITHUB_TOKEN` to Renovate as `RENOVATE_TOKEN`. The job gives the token write access to contents, pull requests, and issues.
  The token stops at the end of the job. A personal token has more access and no end, and a person must renew it.
- GitHub refuses a push of the `GITHUB_TOKEN` that changes a file in `.github/workflows`. With `group:all`, one such change stops the push of all updates.
  A package rule of `.github/renovate.json` turns off the updates of these files.
- `RENOVATE_REPOSITORIES` is the repository of the workflow. The makefile sets `RENOVATE_ONBOARDING` to `false`, because the repository has its config.
- `.github/renovate.json` no longer has the `schedule:monthly` preset. The cron of the workflow is the schedule.
  With the preset, a late run or a run by hand is out of the schedule, and Renovate then does nothing.
- `group:all` puts all updates in one pull request. `prHourlyLimit` is `0`, so no limit stops this pull request.
- three custom managers of the type `regex` read the makefile. Each one has one form of version.
- a package rule turns off the major updates of the Go tools of the makefile that have a `/vN` module path.
- `.github/renovate.json` no longer has `timezone`. Only the schedule of the preset used it.
- `RENOVATE_IMAGE` in the makefile has the Renovate version. Renovate updates it with the other images.
- the pull request and the Dependency Dashboard issue change only when the workflow runs. A click in the dashboard has its effect in the next run.
- `make renovate-local` uses `--platform=local` and `--dry-run=lookup`.

Checked on 2026-11-14 with `make renovate-local` and Renovate 44.239.1. It read the six versions of the makefile and put their updates in `renovate/all`.
`make renovate` and the workflow did not run on GitHub before the merge. A scheduled workflow runs only from the default branch.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
