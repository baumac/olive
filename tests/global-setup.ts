import { appURL, photonPort, port } from './playwright.config'
import { superuserTokenEnv } from './api'
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
        REVIEWS_CREATE_LIMIT: '10000',
        REVIEWS_EDIT_LIMIT: '10000',
        REVIEWS_PLACES_URL: photon.url,
        REVIEWS_MAP_TILE_URL: `${photon.url}/tiles/{z}/{x}/{y}.png`,
      },
      appURL,
    )
    // The test workers get the environment of this process.
    process.env[superuserTokenEnv] = app.superuserToken
    return async () => {
      await app.stop()
      await photon.stop()
    }
  } catch (err) {
    await photon.stop()
    throw err
  }
}
