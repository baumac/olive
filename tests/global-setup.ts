import { appURL, port } from './playwright.config'
import { startApp } from './app'

// globalSetup starts the app for all tests. Playwright runs the returned
// function after the last test.
export default async function globalSetup() {
  const app = await startApp(port, { REVIEWS_SIGN_IN_LIMIT: '10000', REVIEWS_PHOTO_UPLOAD_LIMIT: '10000' }, appURL)
  return app.stop
}
