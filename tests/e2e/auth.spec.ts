import { expect, test } from '../fixtures'

test('sign up signs the person in', async ({ page }) => {
  await page.goto('/sign-up')
  await page.getByLabel('Name').fill('Dana')
  await page.getByLabel('Email').fill(`dana-${Date.now()}@example.test`)
  await page.getByLabel('Password', { exact: true }).fill('correct-horse-battery')
  await page.getByLabel('Confirm password').fill('correct-horse-battery')
  await page.getByRole('button', { name: 'Create account' }).click()

  await expect(page).toHaveURL(/\/restaurants$/)
  await expect(page.getByText('Welcome to Sponga47 Reviews')).toBeVisible()
  await expect(page.getByRole('button', { name: 'DA, account menu for Dana' })).toBeVisible()
})

test('a wrong password shows an error and keeps the email', async ({ page, api }) => {
  const user = await api.signUp()
  await page.goto('/sign-in')
  await page.getByLabel('Email').fill(user.email)
  await page.getByLabel('Password', { exact: true }).fill('wrong-password')
  await page.getByRole('button', { name: 'Sign in' }).click()

  await expect(page.getByRole('alert')).toContainText('The email or password is not correct')
  await expect(page.getByLabel('Email')).toHaveValue(user.email)
  await expect(page.getByLabel('Password', { exact: true })).toHaveValue('')
  await expect(page.getByLabel('Password', { exact: true })).toBeFocused()
})

test('a signed out person signs in and comes back to the form', async ({ page, api }) => {
  const user = await api.signUp()
  await page.goto('/restaurants/new')
  await expect(page).toHaveURL(/\/sign-in\?redirect=%2Frestaurants%2Fnew|\/sign-in\?redirect=\/restaurants\/new/)

  await page.getByLabel('Email').fill(user.email)
  await page.getByLabel('Password', { exact: true }).fill(user.password)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('heading', { name: 'Add a restaurant' })).toBeVisible()
})

test('a refused token ends the session and the draft comes back', async ({ page, user }) => {
  await page.goto('/restaurants/new')
  await page.getByLabel('Name').fill('Draft Diner')

  // The API refuses the token of the first create request.
  let refused = false
  await page.route('**/v1/restaurants', async (route) => {
    if (!refused && route.request().method() === 'POST' && route.request().headers().authorization) {
      refused = true
      await route.fulfill({
        status: 401,
        contentType: 'application/problem+json',
        body: JSON.stringify({ type: 'about:blank', title: 'Unauthorized', status: 401 }),
      })
      return
    }
    await route.continue()
  })
  await page.getByRole('button', { name: 'Add restaurant' }).click()

  await expect(page).toHaveURL(/\/sign-in\?/)
  await expect(page.getByText('Your session has ended')).toBeVisible()
  await page.getByLabel('Email').fill(user.email)
  await page.getByLabel('Password', { exact: true }).fill(user.password)
  await page.getByRole('button', { name: 'Sign in' }).click()

  await expect(page.getByRole('heading', { name: 'Add a restaurant' })).toBeVisible()
  await expect(page.getByLabel('Name')).toHaveValue('Draft Diner')
})

test('sign out deletes the session', async ({ page, user }) => {
  await page.goto('/restaurants')
  await page.getByRole('button', { name: `AN, account menu for ${user.name}` }).click()
  await page.getByRole('menuitem', { name: 'Sign out' }).click()

  await expect(page.getByText('Signed out')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Sign in' })).toBeVisible()
  expect(await page.evaluate(() => localStorage.getItem('reviews-session'))).toBeNull()
})

test('sign out makes the token invalid on the server', async ({ page, user, api }) => {
  await page.goto('/restaurants')
  await page.getByRole('button', { name: `AN, account menu for ${user.name}` }).click()
  const revoked = page.waitForResponse((r) => r.url().endsWith('/v1/users/tokens') && r.request().method() === 'DELETE')
  await page.getByRole('menuitem', { name: 'Sign out' }).click()

  expect((await revoked).status()).toBe(204)
  expect(await api.refreshStatus(user.auth.token)).toBe(401)
})

test('sign out in one tab signs out the other tabs', async ({ page, user, context }) => {
  await page.goto('/my-reviews')
  await expect(page.getByRole('heading', { name: 'My reviews' })).toBeVisible()
  const other = await context.newPage()
  await other.goto('/restaurants')
  await other.getByRole('button', { name: `AN, account menu for ${user.name}` }).click()
  await other.getByRole('menuitem', { name: 'Sign out' }).click()
  await expect(other.getByText('Signed out')).toBeVisible()

  await expect(page).toHaveURL(/\/sign-in\?/)
  await expect(page.getByRole('link', { name: 'Sign in' })).toBeVisible()
})

test('sign in in one tab signs in the other tabs', async ({ page, api, context }) => {
  const user = await api.signUp()
  await page.goto('/sign-in?redirect=/my-reviews')
  const other = await context.newPage()
  await other.goto('/sign-in')
  await other.getByLabel('Email').fill(user.email)
  await other.getByLabel('Password', { exact: true }).fill(user.password)
  await other.getByRole('button', { name: 'Sign in' }).click()
  await expect(other).toHaveURL(/\/restaurants$/)

  await expect(page).toHaveURL(/\/my-reviews$/)
  await expect(page.getByRole('heading', { name: 'My reviews' })).toBeVisible()
})
