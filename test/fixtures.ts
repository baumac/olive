import type { Page } from '@playwright/test'
import { test as base, expect } from '@playwright/test'
import type { TestUser } from './api'
import { TestApi } from './api'

// signIn stores the session before the next page loads. This is the
// localStorage key of app/ui/src/lib/session.ts. The init script runs on
// each load. A flag in sessionStorage makes it store the session one time.
// A reload after sign out then stays signed out.
export async function signIn(page: Page, user: TestUser) {
  const session = JSON.stringify({ token: user.auth.token, expires: user.auth.expires, user: user.auth.user })
  await page.addInitScript(
    ({ value, flag }) => {
      if (!sessionStorage.getItem(flag)) {
        sessionStorage.setItem(flag, '1')
        localStorage.setItem('reviews-session', value)
      }
    },
    { value: session, flag: `e2e-signed-in:${user.auth.user.id}` },
  )
}

export const test = base.extend<{ api: TestApi; user: TestUser }>({
  api: async ({ baseURL }, use) => {
    await use(new TestApi(baseURL!))
  },
  // user is a signed in user. The page starts with the session.
  user: async ({ api, page }, use) => {
    const user = await api.signUp()
    await signIn(page, user)
    await use(user)
  },
})

export { expect }
