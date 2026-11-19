---
description: Record the trace and the video of an end-to-end test only on its retry, give each axe screen its own test, and run one spec file with `E2E_FILE`.
status: implemented
---

# Motivation
[motivation]: #motivation

Agents run the end-to-end tests after a UI change. On a 16-core machine with 8 workers, one project took 24 seconds in chromium and 40 to 57 seconds in firefox.
`make test-e2e` with all projects took about 106 seconds. It also failed locally for two reasons:

- The gallery test searched for the fixed name `Gallery Place`. All projects of one run share one app. A second project then found two restaurants with this name. The CI runs one project in each job and did not see this.
- The password reset test typed the email address before the router showed the new page. The text went into the Email field of the sign in page. It failed in 19 of 30 runs of the test alone.

These costs made the runs slow:

| Cost | Measurement |
|---|---|
| A trace of each test | chromium 21 s against 17 s without it, firefox 36 s against 30 s |
| A video of each test | about 3 more seconds in chromium |
| One axe test with 15 screens | 17 s in chromium. In firefox it went over the 30 s timeout and ran again. |

More workers did not help. With 16 workers chromium took 29 s against 24 s with 8 workers.

# Scope
[scope]: #scope

In scope:
- the trace and video settings of `tests/playwright.config.ts`
- the axe tests of `tests/e2e/accessibility.spec.ts`
- the `E2E_FILE` variable of `make test-e2e`
- the two test bugs above

Out of scope:
- the number of workers
- the retries. A local run still has one retry.
- a build of the UI that skips unchanged files
- a fake mail server instead of the log for the email tests. PocketBase writes its log after 3 seconds without a new entry or after 200 entries.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

While you work, run one project and the specs of your change:

```bash
make test-e2e PROJECT=chromium E2E_FILE=e2e/auth.spec.ts
```

The CI runs all projects. A failed test runs one time again. Playwright records the trace and the video of this retry. The CI uploads them.

A test that searches or filters gets its name or tag from `uniqueTag`.

# Implementation Details
[implementation-details]: #implementation-details

- `trace` and `video` are `on-first-retry`. A screenshot of each failure stays.
- A test that fails on its retry has the trace and the video of the retry. A test that passes on its retry is flaky. Its report also has them.
- Each axe screen has its own test in each theme. Each test makes its own data with `uniqueTag` names. The tests then run in parallel. The password screens and the account screens are separate tests.
- `E2E_FILE` goes to Playwright as a file filter. `PROJECT` and `E2E_FILE` can be used together.
- The name has a prefix. Make reads each environment variable as a make variable. A `FILE` of the shell then changes no run.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
