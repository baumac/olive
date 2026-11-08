---
description: Add end-to-end tests in the test directory that run the UI in real browsers against the built app.
status: implemented
---

# Motivation
[motivation]: #motivation

The project has unit and component tests:
- the Go tests send requests to the Public API through the PocketBase router
- the UI tests run Vue components in jsdom with a fake `fetch`

No test runs the UI and the app together.
These problems can then reach a release without a failed test:
- the UI sends a request that the API refuses, because the SDK and the app change in different ways
- the embedded UI build, the UI route, or the cache headers of decision 00005 break
- a flow that crosses pages breaks, such as sign in with a `redirect` back to the review form
- a color pair fails the contrast rules. jsdom does not render, so the component tests cannot check contrast.

Expected outcome:
- one `make` command builds the app and runs the main flows in real browsers
- each test starts with a new database
- accessibility checks, color contrast included, run on the main screens in the light and the dark theme

# Scope
[scope]: #scope

In scope:
- the test tool and the layout of the `test` directory
- how the tests start the app and make test data
- the flows that the first set of tests covers
- accessibility checks in the browser
- the `make` target

Out of scope:
- running the tests in CI. A later decision chooses the CI system.
- visual comparison with screenshots
- load tests and performance budgets
- tests of the PocketBase dashboard
- mobile devices. The tests use a small viewport in a desktop browser.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

New concepts:
- **End-to-end test**: a test that uses the UI in a browser, as a person does. The UI calls the real Public API in the built app.
- **Fixture**: a Playwright function that gives a test what it needs, such as a signed in user or a restaurant.

```bash
make test-e2e      # Build the app, then run the end-to-end tests in Chromium, Firefox, and WebKit
make test-e2e-ui   # Open the Playwright UI mode to run and debug single tests
```

The first run downloads the browsers with `npx playwright install`.

A test looks like this:

```ts
import { expect, test } from './fixtures'

test('a signed out person signs in to write a review', async ({ page, api }) => {
  const restaurant = await api.createRestaurant({ name: "Luigi's Trattoria" })
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

Tests find elements by role and accessible name, as a screen reader does.
They do not use CSS classes or test IDs.
A test that cannot find an element by its role and name shows an accessibility problem.

# Implementation Details
[implementation-details]: #implementation-details

## Why Playwright

| Option | Result |
| --- | --- |
| Playwright | Selected. It runs Chromium, Firefox, and WebKit with one API. It waits for elements before it acts, so tests do not need sleeps. `@axe-core/playwright` runs axe in the real browser. |
| Cypress | Not selected. Its WebKit support is experimental. A test runs inside the browser, so calls to the API for test data are harder. |
| Vitest browser mode | Not selected. It tests components, not the built app with its routes. |

## Layout

```
test/
  package.json          # npm workspace @reviews/tests
  playwright.config.ts  # browsers, base URL, and the app server
  global-setup.ts       # starts the app for the run and stops it after
  app.ts                # starts ./bin/reviews-app with a new data folder
  fixtures.ts           # the api fixture and the signed in page fixture
  api.ts                # test data through @reviews/sdk
  e2e/
    restaurants.spec.ts
    reviews.spec.ts
    auth.spec.ts
    rate-limit.spec.ts
    accessibility.spec.ts
```

- the root `package.json` adds `test` to its workspaces
- the versions are pinned: `@playwright/test` 1.64.1 and `@axe-core/playwright` 4.14.0
- `test/package.json` also pins `playwright-core` 1.64.1. `@axe-core/playwright` has it as a peer dependency. Without the pin, npm installs a newer `playwright-core`, and the `Page` types do not match.
- `test` uses TypeScript 6.0.3, like the SDK and the UI

## The app under test

- `make test-e2e` runs `make build-app` first. The tests use `./bin/reviews-app` with the embedded UI build, as a release does.
- `global-setup.ts` calls `startApp` of `app.ts`. It starts `./bin/reviews-app serve --dir <temp folder> --http 127.0.0.1:8097` and waits for `/api/health`. The function that it returns stops the app after the run.
- `startApp` makes a new folder with the `reviews-e2e-` prefix in the temp folder of the system. `stop` deletes only this folder. No environment variable can point the delete to another folder.
- `app.ts` finds the binary relative to its own file. Playwright can then run from any folder.
- commands that do not run tests, such as `--list`, do not run the global setup. They make no folder.
- the app applies the migrations when it starts
- the base URL is `http://127.0.0.1:8097`. Port 8097 is not the port of `make run-app`, so both can run at the same time.

## Test data

- tests make their data through the Public API with `@reviews/sdk`, not through the UI. This is faster, and a test checks only one flow.
- each test makes its own users with unique email addresses, such as `ana-<random>@example.test`. Tests then do not share state and can run in parallel.
- the `user` fixture signs up a user through the API. It writes the token to `localStorage` before the page opens, so the test starts signed in.
- the init script writes the token one time per tab. A flag in `sessionStorage` stops it on later loads, so a reload after sign out stays signed out.
- tests do not delete data. Each run starts with an empty database.

## Rate limit

- sign up and sign in allow 10 requests per IP address per minute (decision 00004). Parallel tests make more requests than that.
- the app reads `REVIEWS_SIGN_IN_LIMIT` from the environment. The default stays 10.
- `playwright.config.ts` sets it to 10000 for the app server
- `rate-limit.spec.ts` starts its own app on port 8098 with the default limit. It checks the "Too many tries" alert.
- the `rate-limit` project runs only this file, in Chromium. The other projects ignore it. The count is per IP address, so one browser is enough. `--project=rate-limit` selects it.

## First set of tests

`restaurants.spec.ts`:
- the list shows restaurants. A tag filter in the URL selects them. "Load more" adds the next page.
- a signed in user adds a restaurant, edits it, and deletes it
- the edit and delete buttons show only to the creator
- a restaurant with reviews cannot change its name or be deleted. The UI shows the error.

`reviews.spec.ts`:
- a signed in user writes a `basic` review and a `joc_de_cartes` review
- "Write a review" opens the edit form when the user already has a review
- the kind and score filters select reviews. The sort puts other kinds last.
- the creator edits and deletes a review

`auth.spec.ts`:
- sign up signs the person in and shows the welcome toast
- a wrong password shows the error and keeps the email
- a signed out person who opens a form signs in and comes back to it
- a token that the API refuses ends the session. The form opens the sign in page with "Your session has ended". After sign in, the draft is back.
- sign out deletes the session

## Accessibility

- `accessibility.spec.ts` runs `@axe-core/playwright` on these screens: the restaurant list, a restaurant page with reviews, the review form, the sign in page, and the 404 page
- each screen runs in the light and the dark theme. The test sets `colorScheme` in the browser context.
- the color contrast rule runs, because the browser renders the page. This covers the contrast values of `tokens.json`.
- a test fails on any axe violation of WCAG 2.2 level A or AA
- the first run found that the account menu button broke WCAG 2.5.3 Label in Name. It showed "AN", and its name was "Account menu for Ana". Its name now starts with the visible text: "AN, account menu for Ana". The `UserMenu` design has the same change.
- keyboard tests check the skip link, the focus order of the sign in form, and that Escape closes the delete dialog

## Browsers and viewports

- the tests run in Chromium, Firefox, and WebKit
- WebKit on Linux needs system packages. `sudo npx playwright install-deps` installs them.
- each browser runs at 1280×800. Chromium also runs at 375×812 for the small screen layout.

## Failures

- Playwright keeps a trace, a screenshot, and a video of a failed test in `test/test-results`
- `make test-e2e` prints the path of the HTML report
- `.gitignore` ignores `test/test-results` and `test/playwright-report`
- a failed test runs again one time. A test that passes only on the retry shows as flaky in the report.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- The CI system that runs `make test-e2e`. The browsers need system packages, which `npx playwright install --with-deps` installs.
