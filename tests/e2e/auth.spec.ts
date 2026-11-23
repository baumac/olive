import type { Page } from '@playwright/test'
import type { TestApi } from '../api'
import { uniqueTag } from '../api'
import { expect, selectPlace, signIn, test } from '../fixtures'

// signUp fills the sign up form and opens the link of the confirmation
// email.
async function signUp(page: Page, api: TestApi, name: string, email: string) {
  await page.goto('/sign-up')
  await page.getByLabel('Name').fill(name)
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password', { exact: true }).fill('correct-horse-battery')
  await page.getByLabel('Confirm password').fill('correct-horse-battery')
  await page.getByRole('button', { name: 'Create account' }).click()
  await expect(page.getByRole('heading', { name: 'Check your email' })).toBeVisible()
  await expect(page.getByText(`We sent an email to ${email}.`)).toBeVisible()
  await page.goto(await api.emailLink(email, '/confirm-email'))
  await page.getByLabel('Password', { exact: true }).fill('wrong-password')
  await page.getByRole('button', { name: 'Confirm email address' }).click()
  await expect(page.getByText('The password is not correct.')).toBeVisible()
  await page.getByLabel('Password', { exact: true }).fill('correct-horse-battery')
  await page.getByRole('button', { name: 'Confirm email address' }).click()
  await expect(page).toHaveURL(/:\d+\/$/)
}

test('sign up sends a link that signs the person in', async ({ page, api }) => {
  const email = `dana-${Date.now()}@example.test`
  await signUp(page, api, 'Dana', email)
  await expect(page.getByText('Welcome to Sponga47 Reviews')).toBeVisible()
  await expect(page.getByRole('button', { name: 'DA, account menu for Dana' })).toBeVisible()
})

test('a person cannot sign in before the confirmation', async ({ page }) => {
  const email = `finn-${Date.now()}@example.test`
  await page.goto('/sign-up')
  await page.getByLabel('Name').fill('Finn')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password', { exact: true }).fill('correct-horse-battery')
  await page.getByLabel('Confirm password').fill('correct-horse-battery')
  await page.getByRole('button', { name: 'Create account' }).click()
  await page.getByRole('link', { name: 'Go to sign in' }).click()

  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password', { exact: true }).fill('correct-horse-battery')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('alert')).toContainText('Confirm your email address first')

  await page.goto('/confirm-email#token=not-a-token')
  await page.getByLabel('Password', { exact: true }).fill('correct-horse-battery')
  await page.getByRole('button', { name: 'Confirm email address' }).click()
  await expect(page.getByRole('alert')).toContainText('This link does not work')
  await expect(page).toHaveURL(/\/confirm-email$/)
})

test('a person sets a new password with the link of an email', async ({ page, api }) => {
  const user = await api.signUp('Gus')
  await page.goto('/sign-in')
  await page.getByRole('link', { name: 'Forgot your password?' }).click()
  // The sign in page also has an Email field until the new page shows.
  await expect(page.getByRole('heading', { name: 'Forgot your password?' })).toBeVisible()
  await page.getByLabel('Email').fill(user.email)
  await page.getByRole('button', { name: 'Send the link' }).click()
  await expect(page.getByRole('heading', { name: 'Check your email' })).toBeVisible()

  await page.goto(await api.emailLink(user.email, '/reset-password'))
  await page.getByLabel('New password', { exact: true }).fill('better-horse-battery')
  await page.getByLabel('Confirm new password').fill('other-horse-battery')
  await page.getByRole('button', { name: 'Set the new password' }).click()
  await expect(page.getByText('The passwords do not match.')).toBeVisible()
  await page.getByLabel('Confirm new password').fill('better-horse-battery')
  await page.getByRole('button', { name: 'Set the new password' }).click()

  await expect(page).toHaveURL(/\/sign-in$/)
  await expect(page.getByText('Your password is changed')).toBeVisible()
  expect(await api.tokenStatus(user.token)).toBe(401)
  await page.getByLabel('Email').fill(user.email)
  await page.getByLabel('Password', { exact: true }).fill('better-horse-battery')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('button', { name: 'GU, account menu for Gus' })).toBeVisible()
})

test('a user changes the name and the password on the account page', async ({ page, api }) => {
  const user = await api.signUp('Hana')
  await signIn(page, user)
  await page.goto('/restaurants')
  await page.getByRole('button', { name: 'HA, account menu for Hana' }).click()
  await page.getByRole('menuitem', { name: 'Account' }).click()
  await expect(page.getByRole('heading', { name: 'Account', level: 1 })).toBeVisible()

  await page.getByLabel('Name').fill('Ivy')
  await page.getByRole('button', { name: 'Save name' }).click()
  await expect(page.getByText('Your name is changed')).toBeVisible()
  await expect(page.getByRole('button', { name: 'IV, account menu for Ivy' })).toBeVisible()

  await page.getByLabel('Current password').fill('wrong-password')
  await page.getByLabel('New password', { exact: true }).fill('better-horse-battery')
  await page.getByLabel('Confirm new password').fill('better-horse-battery')
  await page.getByRole('button', { name: 'Change password' }).click()
  await expect(page.getByText('The password is not correct.')).toBeVisible()
  await page.getByLabel('Current password').fill(user.password)
  await page.getByRole('button', { name: 'Change password' }).click()
  await expect(page.getByText('Your password is changed')).toBeVisible()
  expect(await api.tokenStatus(user.token)).toBe(401)

  // The page has the new token in its cookie.
  await page.reload()
  await expect(page.getByRole('button', { name: 'IV, account menu for Ivy' })).toBeVisible()
})

test('a user deletes the account', async ({ page, api }) => {
  const user = await api.signUp('Jon')
  const restaurant = await api.createRestaurant(user, { name: 'Leftover Kitchen' })
  await api.createReview(user, restaurant.id, { space: 7, service: 7, food: 7, price: 7, title: 'Gone soon' })
  await signIn(page, user)
  await page.goto('/account')

  await page.getByRole('button', { name: 'Delete account' }).click()
  const dialog = page.getByRole('alertdialog')
  await dialog.getByLabel('Password', { exact: true }).fill('wrong-password')
  await dialog.getByRole('button', { name: 'Delete account' }).click()
  await expect(dialog.getByText('The password is not correct.')).toBeVisible()
  await dialog.getByLabel('Password', { exact: true }).fill(user.password)
  await dialog.getByRole('button', { name: 'Delete account' }).click()

  await expect(page.getByText('Your account is deleted')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Sign in' })).toBeVisible()
  expect(await api.tokenStatus(user.token)).toBe(401)
  await page.goto(`/restaurants/${restaurant.id}`)
  await expect(page.getByRole('heading', { name: 'Leftover Kitchen' })).toBeVisible()
  await expect(page.getByText('Gone soon')).toHaveCount(0)
})

test('a new user waits for the approval of a superuser', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Approval Kitchen' })
  const email = `eve-${Date.now()}@example.test`

  await signUp(page, api, 'Eve', email)
  const banner = page.getByRole('status').filter({ hasText: 'Your account waits for approval' })
  await expect(banner).toBeVisible()

  // A link to a form stays on the page. A direct load goes to the table.
  await page.goto('/restaurants')
  await page.getByRole('link', { name: 'Add restaurant' }).first().click()
  await expect(page).toHaveURL(/\/restaurants$/)
  await page.goto('/restaurants/new')
  await expect(page).toHaveURL(/:\d+\/$/)

  await page.goto(`/restaurants/${restaurant.id}`)
  const heart = page.getByRole('button', { name: 'Favorite Approval Kitchen' })
  await heart.click()
  await expect(heart).toHaveAttribute('aria-pressed', 'false')
  await page.getByRole('button', { name: 'Save to list' }).click()
  await expect(page.getByLabel('Name of a new list')).toHaveCount(0)

  await banner.getByRole('button', { name: 'Check again' }).click()
  await expect(page.getByText('Your account is not approved yet')).toBeVisible()

  // The stored user is from before the approval. A direct load of a form
  // gets the user again.
  await api.approveByEmail(email)
  await page.goto('/restaurants/new')
  await expect(page).toHaveURL(/\/restaurants\/new$/)
  await expect(banner).toHaveCount(0)

  await page.goto(`/restaurants/${restaurant.id}`)
  await heart.click()
  await expect(heart).toHaveAttribute('aria-pressed', 'true')
  await page.getByRole('link', { name: 'Add a visit' }).first().click()
  await expect(page).toHaveURL(new RegExp(`/restaurants/${restaurant.id}/reviews/new$`))
})

test('the auth token is only in an HttpOnly cookie', async ({ page, api, context }) => {
  const user = await api.signUp()
  await page.goto('/sign-in')
  await page.getByLabel('Email').fill(user.email)
  await page.getByLabel('Password', { exact: true }).fill(user.password)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page).toHaveURL(/:\d+\/$/)

  const cookie = (await context.cookies()).find((c) => c.name === 'reviews_session')
  expect(cookie).toMatchObject({ httpOnly: true, sameSite: 'Strict', path: '/v1' })
  expect(await api.tokenStatus(cookie!.value)).toBe(200)
  const stored = await page.evaluate(() => ({ cookies: document.cookie, session: localStorage.getItem('reviews-session') }))
  expect(stored.cookies).not.toContain(cookie!.value)
  expect(stored.session).not.toContain(cookie!.value)
  expect(JSON.parse(stored.session!)).not.toHaveProperty('token')
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
  await selectPlace(page, uniqueTag('draft'))
  await page.getByLabel('Name').fill('Draft Diner')

  // The API refuses the first create request. A route handler of WebKit
  // gets no Cookie header, so the handler cannot check the cookie.
  let refused = false
  await page.route('**/v1/restaurants', async (route) => {
    if (!refused && route.request().method() === 'POST') {
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
  await expect(page.getByText('Selected location: Test Street 4, 08003 Testville, Testland')).toBeAttached()
})

test('sign out deletes the session', async ({ page, user }) => {
  await page.goto('/restaurants')
  await page.getByRole('button', { name: `AN, account menu for ${user.name}` }).click()
  await page.getByRole('menuitem', { name: 'Sign out' }).click()

  await expect(page.getByText('Signed out')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Sign in' })).toBeVisible()
  expect(await page.evaluate(() => localStorage.getItem('reviews-session'))).toBeNull()
  await expect.poll(async () => (await page.context().cookies()).map((c) => c.name)).not.toContain('reviews_session')
})

test('sign out makes the token invalid on the server', async ({ page, user, api }) => {
  await page.goto('/restaurants')
  await page.getByRole('button', { name: `AN, account menu for ${user.name}` }).click()
  const revoked = page.waitForResponse((r) => r.url().endsWith('/v1/users/tokens') && r.request().method() === 'DELETE')
  await page.getByRole('menuitem', { name: 'Sign out' }).click()

  expect((await revoked).status()).toBe(204)
  expect(await api.tokenStatus(user.token)).toBe(401)
})

test('sign out in one tab signs out the other tabs', async ({ page, user, context }) => {
  await page.goto('/account')
  await expect(page.getByRole('heading', { name: 'Account', level: 1 })).toBeVisible()
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
  await expect(other.getByRole('heading', { name: 'On the table', level: 1 })).toBeVisible()

  await expect(page).toHaveURL(new RegExp(`/reviewers/${user.auth.user.id}$`))
  await expect(page.getByRole('heading', { name: user.name, level: 1 })).toBeVisible()
})
