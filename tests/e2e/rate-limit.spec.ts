import type { RunningApp } from '../app'
import { startApp } from '../app'
import { photoFile, TestApi } from '../api'
import { expect, signIn, test } from '../fixtures'

// The app of the other tests has a high limit. This test starts its own app
// with the default limit of 10 per minute. The rate-limit project runs it in
// one browser. The count is per IP address, so the browsers share it.
let app: RunningApp

test.beforeAll(async () => {
  app = await startApp(8098, { REVIEWS_PHOTO_UPLOAD_LIMIT: '1' })
})

test.afterAll(async () => {
  await app?.stop()
})

test('too many sign ins show the rate limit alert', async ({ page }) => {
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
})

test('too many photo uploads show an error', async ({ page }) => {
  const api = new TestApi(app.baseURL)
  const user = await api.signUp('Ana')
  const restaurant = await api.createRestaurant(user, { name: 'Busy Place' })
  const review = await api.createReview(user, restaurant.id, { space: 7, service: 7, food: 7, price: 7, cleanliness: 7, title: 'Busy' })
  await api.addPhoto(user, review, 'menu.png')
  await signIn(page, user)

  await page.goto(`${app.baseURL}/restaurants/${restaurant.id}/reviews/${review.id}/edit`)
  await page.getByLabel('Add photo').setInputFiles({ name: 'menu.png', mimeType: 'image/png', buffer: photoFile('menu.png') })
  await page.getByRole('button', { name: 'Upload photo' }).click()
  await expect(page.getByText('You added many photos in a short time. Try again later.')).toBeVisible()
})
