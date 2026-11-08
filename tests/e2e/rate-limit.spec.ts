import type { RunningApp } from '../app'
import { startApp } from '../app'
import { expect, test } from '../fixtures'

// The app of the other tests has a high limit. This test starts its own app
// with the default limit of 10 per minute. The rate-limit project runs it in
// one browser. The count is per IP address, so the browsers share it.
let app: RunningApp

test.beforeAll(async () => {
  app = await startApp(8098)
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
