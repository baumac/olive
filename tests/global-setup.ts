import { appURL, photonPort, port } from './playwright.config'
import { startApp } from './app'
import { startPhoton } from './photon'

// globalSetup starts the app for all tests. Playwright runs the returned
// function after the last test.
export default async function globalSetup() {
  const photon = await startPhoton(photonPort)
  try {
    const app = await startApp(
      port,
      {
        REVIEWS_SIGN_IN_LIMIT: '10000',
        REVIEWS_PHOTO_UPLOAD_LIMIT: '10000',
        REVIEWS_PLACE_SEARCH_LIMIT: '10000',
        REVIEWS_PLACES_URL: photon.url,
      },
      appURL,
    )
    return async () => {
      await app.stop()
      await photon.stop()
    }
  } catch (err) {
    await photon.stop()
    throw err
  }
}
