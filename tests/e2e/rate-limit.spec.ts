import type { RunningApp } from '../app'
import { startApp } from '../app'
import { photoFile, TestApi } from '../api'
import { expect, signIn, test } from '../fixtures'

// The app of the other tests has high limits. Each test here starts its own
// app with low limits on its own port. Workers then share no app, and a
// failed test does not skip the other one.
async function withApp(port: number, env: Record<string, string>, run: (app: RunningApp) => Promise<void>) {
  const app = await startApp(port, env)
  try {
    await run(app)
  } finally {
    await app.stop()
  }
}

// The default sign in limit is 10 per minute. The count is per IP address.
test('too many sign ins show the rate limit alert', async ({ page }) => withApp(8098, {}, async (app) => {
  const body = JSON.stringify({ email: 'nobody@example.test', password: 'wrong-password' })
  for (let i = 0; i < 10; i++) {
    await fetch(`${app.baseURL}/v1/users/tokens`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
    })
  }
  await page.goto(`${app.baseURL}/sign-in`)
  await page.getByLabel('Email').fill('nobody@example.test')
  await page.getByLabel('Password', { exact: true }).fill('wrong-password')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('alert')).toContainText('Too many tries')
}))

test('too many photo uploads show an error', async ({ page }) => withApp(8099, { REVIEWS_PHOTO_UPLOAD_LIMIT: '1' }, async (app) => {
  const api = new TestApi(app.baseURL, app.superuserToken)
  const user = await api.signUp('Ana')
  const restaurant = await api.createRestaurant(user, { name: 'Busy Place' })
  const review = await api.createReview(user, restaurant.id, { space: 7, service: 7, food: 7, price: 7, cleanliness: 7, title: 'Busy' })
  await api.addPhoto(user, review, 'menu.png')
  await signIn(page, user)

  await page.goto(`${app.baseURL}/restaurants/${restaurant.id}/reviews/${review.id}/edit`)
  await page.getByLabel('Add photo').setInputFiles({ name: 'menu.png', mimeType: 'image/png', buffer: photoFile('menu.png') })
  await page.getByRole('button', { name: 'Upload photo' }).click()
  await expect(page.getByText('You added many photos in a short time. Try again later.')).toBeVisible()
}))
