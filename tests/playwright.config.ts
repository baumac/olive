import { defineConfig, devices } from '@playwright/test'

export const port = 8097
export const baseURL = `http://127.0.0.1:${port}`
// appURL is the Application URL of the test app. The link previews use it.
// The app does not serve it.
export const appURL = 'https://reviews.example.com'

const desktop = { viewport: { width: 1280, height: 800 } }
// The rate limit test has its own app and project. See rate-limit.spec.ts.
const rateLimit = /rate-limit\.spec\.ts/

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  retries: 1,
  reporter: [['list'], ['html', { open: 'never' }]],
  globalSetup: './global-setup.ts',
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    { name: 'chromium', testIgnore: rateLimit, use: { ...devices['Desktop Chrome'], ...desktop } },
    { name: 'firefox', testIgnore: rateLimit, use: { ...devices['Desktop Firefox'], ...desktop } },
    { name: 'webkit', testIgnore: rateLimit, use: { ...devices['Desktop Safari'], ...desktop } },
    {
      name: 'chromium-small',
      testIgnore: rateLimit,
      use: { ...devices['Desktop Chrome'], viewport: { width: 375, height: 812 } },
    },
    { name: 'rate-limit', testMatch: rateLimit, use: { ...devices['Desktop Chrome'], ...desktop } },
  ],
})
