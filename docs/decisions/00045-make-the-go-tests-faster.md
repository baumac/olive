---
description: Copy one migrated database into each Go test app, use bcrypt cost 4 in the publicapi tests, and run the race detector only in `make test-race` and `make test-coverage`.
status: implemented
---

# Motivation
[motivation]: #motivation

Agents and developers run the Go tests after each change.
`make test-app` used the race detector and took about 7 minutes on a 16-core machine. The `publicapi` package took 431 seconds of this time.
Two costs made the run slow:

| Cost | With race | Without race |
|---|---|---|
| A new test app with an empty database and all migrations | 2.5 s | 0.15 s |
| One bcrypt hash or compare at the default cost 10 | 0.55 s | 0.045 s |

Almost each of the 122 tests of `publicapi` makes a new app. The tests sign up users 93 times. Each sign up hashes and compares a password.

# Scope
[scope]: #scope

In scope:
- one migrated database for each Go test binary
- the bcrypt cost of the test apps
- the race detector of `make test-app`

Out of scope:
- the UI, SDK, lint, and end-to-end targets. Each takes less than 30 seconds.
- the speed of the `migrations` tests. With race, they take about 46 seconds. They use the production schema with the bcrypt cost 10.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

| Command | Race detector | Time before | Time after |
|---|---|---|---|
| `make test-app` | no | 7 min | 6 s |
| `make test-race` | yes | new | about 50 s |
| `make test-coverage-app` | yes | 7 min | 51 s |

While you work, run `make test-app`.
Run `make test-race` or `make test-coverage` one time before the end. CI runs `make test-coverage` with the race detector.

A Go test makes its app with `tests.NewTestApp(testapp.Dir(t))`. This database has the production schema.
A test with many sign ups uses `testapp.LowCostDir(t)`. The `publicapi` tests use it.
Each test package has a `main_test.go` with a `TestMain` that calls `testapp.Main`.

# Implementation Details
[implementation-details]: #implementation-details

- `testapp.Dir` makes its template folder one time for each test binary. It uses a `sync.Once`.
  - It calls `tests.NewTestApp` with an empty folder. This runs the migrations that the test binary registers.
  - It closes the databases with `ResetBootstrapState`. The folder stays.
- `testapp.LowCostDir` copies the folder of `Dir` one time. It sets the `Cost` of the password field of each auth collection to `bcrypt.MinCost`.
- `tests.NewTestApp` copies the template folder for each app. `RunAllMigrations` then finds no new migration.
- `testapp.Main` deletes the two template folders after `m.Run`.
- `testapp` does not import `migrations`. The tests of `migrations` can then use it without an import cycle.
- The test binaries of `publicapi` and `photocleanup` have internal and external test files. One `TestMain` is in the external package.
- The migrations tests and the tests of the other packages use `Dir`. They check the schema that production has.
- The sign in of an unknown email checks a dummy hash with the cost of the `users` password field. `TestDummyHashUsesThePasswordCost` checks the default cost 10 and the cost 4.
- `make test-coverage-app` gives `-coverpkg` all packages except `internal/testapp`. The coverage then counts only the code of the app.
- The tests run with `t.Parallel()`. With race, the slowest package takes about 30 seconds instead of 59 seconds.
  - No test calls `t.Setenv` or `os.Setenv`. Only `main.go` reads environment variables.
  - Each `publicapi.Register` makes its own rate limiters. `newEnv` calls it for each test.
  - `tests.NewTestApp` copies the template folder for each app. The tests share no database.
  - `TestWarningOfBuildWithoutMetaBlock` changes the package variable `warnNoMetaBlock`. It has no `t.Parallel()`. Go runs it before the parallel tests continue.


The record hooks of `searchindex` and `publicapi` are not in the template. The tests registered these hooks after the migrations before this decision too.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.