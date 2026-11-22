import type { Page } from '@playwright/test'
import { test as base, expect } from '@playwright/test'
import type { TestUser } from './api'
import { TestApi } from './api'

// signIn sets the session cookie and stores the session before the next page
// loads. This is the localStorage key of app/ui/src/lib/session.ts. The init
// script runs on each load. A flag in sessionStorage makes it store the
// session one time. A reload after sign out then stays signed out.
export async function signIn(page: Page, user: TestUser) {
  const { hostname } = new URL(test.info().project.use.baseURL!)
  await page.context().addCookies([
    { name: 'reviews_session', value: user.token, domain: hostname, path: '/v1', httpOnly: true, sameSite: 'Strict' },
  ])
  const session = JSON.stringify({ expires: user.auth.expires, user: user.auth.user })
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

export const test = base.extend<{ api: TestApi; user: TestUser; noCspViolations: void }>({
  // noCspViolations fails a test if the browser blocks anything under the
  // Content Security Policy of decision 00002. It runs in every test.
  noCspViolations: [
    async ({ page }, use) => {
      const violations: string[] = []
      await page.exposeBinding('reportCspViolation', (_source, text: string) => {
        violations.push(text)
      })
      await page.addInitScript(() => {
        document.addEventListener('securitypolicyviolation', (e) => {
          const report = (window as unknown as { reportCspViolation: (t: string) => void }).reportCspViolation
          report(`${e.violatedDirective} blocked ${e.blockedURI || 'inline'} at ${e.sourceFile}:${e.lineNumber}`)
        })
      })
      // Browsers also log other policy messages, such as an ignored source.
      // Only a refused or blocked load is a violation.
      page.on('console', (msg) => {
        const text = msg.text()
        if (/Content[ -]Security[ -]Policy/i.test(text) && /refused|blocked|violates/i.test(text)) {
          violations.push(text)
        }
      })
      await use()
      expect(violations, 'Content Security Policy violations').toEqual([])
    },
    { auto: true },
  ],
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

// selectPlace searches in the restaurant form and selects the first place.
// The fake geocoder of photon.ts gives each search its own place.
export async function selectPlace(page: Page, search: string) {
  await page.getByLabel('Location').fill(search)
  await page.getByRole('list', { name: 'Places' }).getByRole('button').first().click()
  await expect(page.getByText('Selected location: Test Street 4, 08003 Testville, Testland')).toBeAttached()
}

export { expect }
