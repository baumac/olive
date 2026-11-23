import { uniqueTag } from '../api'
import type { Page } from '@playwright/test'
import { expect, test } from '../fixtures'

// The fake server of photon.ts gives the map tiles. The noCspViolations
// fixture checks that the browser loads them under the policy.
// The List and Map buttons exist below the lg breakpoint. From lg, the list
// and the map are side by side. See decision 00008.
test.use({ viewport: { width: 900, height: 800 } })

const barceloneta = { latitude: 41.3766, longitude: 2.1893, address: 'Carrer de Sant Carles 4, Barcelona' }
const eixample = { latitude: 41.39, longitude: 2.16, address: 'Carrer Nou 1, Barcelona' }

// restaurantMap gives the map of the restaurant page. A narrow page shows
// the map only after a click.
async function restaurantMap(page: Page, name: string) {
  const map = page.getByRole('region', { name: `Map of ${name}` })
  const showMap = page.getByRole('button', { name: 'Show map' })
  await expect(map.or(showMap)).toBeVisible()
  if (await showMap.isVisible()) {
    await showMap.click()
  }
  await expect(map).toBeVisible()
  return map
}

test('the restaurant page shows the location on a map', async ({ page, api }) => {
  const restaurant = await api.createRestaurant(await api.signUp('Joe'), { name: 'Can Sole', location: barceloneta })

  const tiles = page.waitForResponse((res) => res.url().includes('/tiles/') && res.ok())
  await page.goto(`/restaurants/${restaurant.id}`)
  const map = await restaurantMap(page, 'Can Sole')
  await tiles
  await expect(map.getByRole('button', { name: 'Can Sole' })).toBeVisible()
  await expect(map.getByRole('link', { name: '© OpenStreetMap contributors' })).toHaveAttribute('href', 'https://www.openstreetmap.org/copyright')
  await expect(page.getByRole('link', { name: /^Address: .+, open in OpenStreetMap$/ })).toHaveAttribute(
    'href',
    'https://www.openstreetmap.org/?mlat=41.3766&mlon=2.1893#map=18/41.3766/2.1893',
  )
  // The link depends on the device of the browser.
  await expect(page.getByRole('link', { name: 'Directions' })).toHaveAttribute(
    'href',
    /^(geo:41\.3766,2\.1893\?q=|https:\/\/maps\.apple\.com\/\?daddr=|https:\/\/www\.openstreetmap\.org\/directions\?to=)41\.3766(,|%2C)2\.1893$/,
  )
})

test('the map view joins near pins into one pin with a count', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('cluster')
  await api.createRestaurant(owner, { name: 'Can Sole', tags: [tag], location: barceloneta })
  await api.createRestaurant(owner, { name: 'Next Door', tags: [tag], location: { ...barceloneta, latitude: 41.3771 } })

  await page.goto(`/restaurants?tags=${tag}&view=map`)
  const map = page.getByRole('region', { name: 'Map of the restaurants' })
  await expect(map.getByRole('button', { name: 'Can Sole' })).toHaveCount(0)
  await map.getByRole('button', { name: '2 restaurants, zoom in to see them' }).click()
  await expect(map.getByRole('button', { name: 'Can Sole' })).toBeVisible()
  await expect(map.getByRole('button', { name: 'Next Door' })).toBeVisible()
  await expect(map.getByRole('button', { name: '2 restaurants, zoom in to see them' })).toHaveCount(0)
})

test('the map view shows the restaurants of the filters', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('map')
  await api.createRestaurant(owner, { name: 'Can Sole', tags: [tag], location: barceloneta })
  await api.createRestaurant(owner, { name: 'Bar Central', tags: [tag], location: eixample })
  await api.createRestaurant(owner, { name: 'Far Away', tags: [tag] })
  await api.createRestaurant(owner, { name: 'Other Tag', location: eixample })

  await page.goto(`/restaurants?tags=${tag}`)
  await expect(page.getByRole('heading', { level: 2 })).toHaveCount(3)
  await page.getByRole('button', { name: 'Map', exact: true }).click()
  await expect(page).toHaveURL(new RegExp(`\\?tags=${tag}&view=map$`))

  const map = page.getByRole('region', { name: 'Map of the restaurants' })
  await expect(page.getByText('2 restaurants on the map')).toBeVisible()

  // An open map does not load again without a move.
  let loads = 0
  page.on('request', (req) => {
    if (req.url().includes('/v1/restaurants?') && req.url().includes('within')) {
      loads++
    }
  })
  // eslint-disable-next-line playwright/no-wait-for-timeout -- The test checks that no load starts in this time.
  await page.waitForTimeout(1500)
  expect(loads).toBe(0)
  await expect(map.getByRole('button', { name: 'Bar Central' })).toBeVisible()
  await expect(map.getByRole('button', { name: 'Far Away' })).toHaveCount(0)
  await expect(map.getByRole('button', { name: 'Other Tag' })).toHaveCount(0)

  // Enter on a pin selects it, like a click.
  await map.getByRole('button', { name: 'Can Sole' }).press('Enter')
  await expect(page.getByRole('link', { name: 'Can Sole' })).toBeVisible()
  await expect(page.getByText('Carrer de Sant Carles 4, Barcelona')).toBeVisible()

  // A zoom replaces the tiles. The pins stay.
  await map.getByRole('button', { name: 'Zoom in' }).click()
  await map.getByRole('button', { name: 'Zoom out' }).click()
  await expect(map.getByRole('button', { name: 'Bar Central' })).toBeVisible()

  // A filter change keeps the map view.
  await page.getByRole('searchbox', { name: 'Search restaurants' }).fill('central')
  await expect(page.getByText('1 restaurant on the map')).toBeVisible()
  await expect(page).toHaveURL(/view=map/)
  await expect(map.getByRole('button', { name: 'Can Sole' })).toHaveCount(0)

  await page.getByRole('button', { name: 'List', exact: true }).click()
  await expect(page).not.toHaveURL(/view=map/)
  await expect(page.getByRole('heading', { level: 2 })).toHaveText(['Bar Central'])
})

test('a location of the device puts the nearest restaurants first', async ({ page, api, context }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('near')
  const far = await api.createRestaurant(owner, { name: 'Far Place', tags: [tag], location: { latitude: 40.4168, longitude: -3.7038, address: 'Calle Mayor 1, Madrid' } })
  await api.createRestaurant(owner, { name: 'Mid Place', tags: [tag], location: eixample })
  await api.createRestaurant(owner, { name: 'Near Place', tags: [tag], location: barceloneta })
  await api.createReview(owner, far.id, { space: 9, service: 9, food: 9, price: 9, title: 'Top' })
  await context.grantPermissions(['geolocation'])
  await context.setGeolocation({ latitude: barceloneta.latitude, longitude: barceloneta.longitude })
  const names = page.getByRole('heading', { level: 2, name: / Place$/ })
  const sort = page.getByRole('combobox', { name: 'Sort by' })

  // Only Far Place has a review. The other two have no fixed order.
  await page.goto(`/restaurants?tags=${tag}`)
  await expect(names).toHaveCount(3)
  await expect(names.first()).toHaveText('Far Place')
  await page.getByRole('button', { name: 'Set location' }).click()
  await page.getByRole('button', { name: 'Use my current location' }).click()

  // Madrid is not in the default distance of 10 km.
  const menu = page.getByRole('button', { name: 'Location: Current location' })
  await expect(menu).toBeVisible()
  await expect(sort).toHaveText('Nearest')
  await expect(names).toHaveText(['Near Place', 'Mid Place'])
  await expect(page.getByText('Less than 100 m')).toBeVisible()
  await expect(page).not.toHaveURL(/latitude|sort=/)

  await menu.click()
  await page.getByRole('combobox', { name: 'Distance' }).click()
  await page.getByRole('option', { name: 'Any distance' }).click()
  await page.keyboard.press('Escape')
  await expect(sort).toHaveText('Nearest')
  await expect(names).toHaveText(['Near Place', 'Mid Place', 'Far Place'])

  // The browser keeps the location. The map shows it.
  await page.reload()
  await expect(menu).toBeVisible()
  await page.getByRole('button', { name: 'Map', exact: true }).click()
  const map = page.getByRole('region', { name: 'Map of the restaurants' })
  await expect(map.getByRole('button', { name: 'Near Place' })).toBeVisible()
  await expect(page.getByText('The blue dot is your location.')).toBeVisible()

  await menu.click()
  await page.getByRole('button', { name: 'Clear location' }).click()
  await expect(page.getByRole('button', { name: 'Set location' })).toBeVisible()
  await expect(page.getByText('The blue dot is your location.')).toHaveCount(0)
})

const madrid = { latitude: 40.4168, longitude: -3.7038, address: 'Calle Mayor 1, Madrid' }
const madridAt = /at=40\.4168(,|%2C)-3\.7038(,|%2C)13/

test('the map view keeps its position in the URL', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('at')
  await api.createRestaurant(owner, { name: 'Can Sole', tags: [tag], location: barceloneta })
  await api.createRestaurant(owner, { name: 'Casa Mayor', tags: [tag], location: madrid })
  await page.addInitScript(() => {
    localStorage.setItem('reviews-location', JSON.stringify({ latitude: 41.3766, longitude: 2.1893, label: 'Barceloneta', radius: null }))
  })
  const map = page.getByRole('region', { name: 'Map of the restaurants' })

  // A zoom keeps the preferred location in the center. The URL then does
  // not show it.
  await page.goto(`/restaurants?tags=${tag}&view=map`)
  await expect(map.getByRole('button', { name: 'Can Sole' })).toBeVisible()
  await expect(page.getByText('1 restaurant on the map')).toBeVisible()
  await map.getByRole('button', { name: 'Zoom in' }).click()
  await expect(page.getByText('1 restaurant on the map')).toBeVisible()
  // eslint-disable-next-line playwright/no-wait-for-timeout -- The test checks that the URL does not change in this time.
  await page.waitForTimeout(1000)
  await expect(page).not.toHaveURL(/at=/)

  await map.focus()
  await page.keyboard.press('ArrowRight')
  await expect(page).toHaveURL(/at=41\.\d+(,|%2C)2\.\d+(,|%2C)14$/)
  const at = new URL(page.url()).searchParams.get('at')
  await page.reload()
  await expect(map.getByRole('button', { name: 'Can Sole' })).toBeVisible()
  expect(new URL(page.url()).searchParams.get('at')).toBe(at)

  // The map view opens again at the last view after the list view, not at
  // the view of the link.
  await map.focus()
  await page.keyboard.press('ArrowRight')
  await expect.poll(() => new URL(page.url()).searchParams.get('at')).not.toBe(at)
  const moved = new URL(page.url()).searchParams.get('at')
  await page.getByRole('button', { name: 'List', exact: true }).click()
  await expect(page).not.toHaveURL(/at=|view=map/)
  await page.getByRole('button', { name: 'Map', exact: true }).click()
  await expect(map.getByRole('button', { name: 'Can Sole' })).toBeVisible()
  await map.focus()
  await page.keyboard.press('ArrowUp')
  await expect(page).toHaveURL(/at=41\.\d+(,|%2C)2\.\d+(,|%2C)14$/)
  expect(new URL(page.url()).searchParams.get('at')?.split(',')[1]).toBe(moved?.split(',')[1])

  // A link opens at its position, not at the preferred location.
  await page.goto(`/restaurants?tags=${tag}&view=map&at=40.4168,-3.7038,13`)
  await expect(map.getByRole('button', { name: 'Casa Mayor' })).toBeVisible()
  await expect(map.getByRole('button', { name: 'Can Sole' })).toHaveCount(0)
  await page.getByRole('searchbox', { name: 'Search restaurants' }).fill('casa')
  await expect(page).toHaveURL(/search=casa/)
  await expect(page).toHaveURL(madridAt)

  // A position that is not valid gives the preferred location.
  await page.goto(`/restaurants?tags=${tag}&view=map&at=north`)
  await expect(map.getByRole('button', { name: 'Can Sole' })).toBeVisible()
  await expect(page).not.toHaveURL(/at=/)
})

test('a new location removes the position of the map from the URL', async ({ page, api, context }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('at-move')
  await api.createRestaurant(owner, { name: 'Can Sole', tags: [tag], location: barceloneta })
  await api.createRestaurant(owner, { name: 'Casa Mayor', tags: [tag], location: madrid })
  await context.grantPermissions(['geolocation'])
  await context.setGeolocation({ latitude: barceloneta.latitude, longitude: barceloneta.longitude })
  const map = page.getByRole('region', { name: 'Map of the restaurants' })

  await page.goto(`/restaurants?tags=${tag}&view=map&at=40.4168,-3.7038,13`)
  await expect(map.getByRole('button', { name: 'Casa Mayor' })).toBeVisible()
  await page.getByRole('button', { name: 'Set location' }).click()
  await page.getByRole('button', { name: 'Use my current location' }).click()
  await expect(map.getByRole('button', { name: 'Can Sole' })).toBeVisible()
  await expect(page).not.toHaveURL(/at=/)
})

test('a link with a list and a position does not frame the list', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const list = await api.createList(owner, 'Trip')
  const sole = await api.createRestaurant(owner, { name: 'Sole Trip', location: barceloneta })
  await api.addToList(owner, list.id, sole.id)
  const map = page.getByRole('region', { name: 'Map of the restaurants' })

  await page.goto(`/restaurants?list=${list.id}&view=map&at=40.4168,-3.7038,13`)
  await expect(page.getByText('0 restaurants on the map')).toBeVisible()
  // eslint-disable-next-line playwright/no-wait-for-timeout -- The test checks that the map does not move in this time.
  await page.waitForTimeout(1000)
  await expect(page.getByText('0 restaurants on the map')).toBeVisible()
  await expect(page).toHaveURL(madridAt)

  // Without a position, the map shows the list.
  await page.goto(`/restaurants?list=${list.id}&view=map`)
  await expect(map.getByRole('button', { name: 'Sole Trip' })).toBeVisible()
})

test('a click on the map sets the point of a location by hand', async ({ page, user: _user }) => {
  await page.goto('/restaurants/new')
  await page.getByRole('button', { name: 'Enter the location by hand' }).click()
  const map = page.getByRole('region', { name: 'Map to set the point' })
  await expect(map.getByRole('button', { name: 'The point of the restaurant' })).toHaveCount(0)

  await map.click({ position: { x: 200, y: 100 } })
  await expect(page.getByLabel('Latitude')).toHaveValue(/^41\.\d{1,6}$/)
  await expect(page.getByLabel('Longitude')).toHaveValue(/^2\.\d{1,6}$/)
  await expect(map.getByRole('button', { name: 'The point of the restaurant' })).toBeVisible()

  // A second click at the same place gives the same point. The map did not
  // move under the pointer.
  const latitude = await page.getByLabel('Latitude').inputValue()
  await map.click({ position: { x: 200, y: 100 } })
  await expect(page.getByLabel('Latitude')).toHaveValue(latitude)

  // The form saves the typed numbers.
  await page.getByLabel('Latitude').fill('-45.5')
  await page.getByLabel('Longitude').fill('-140.25')
  await page.getByLabel('Address').fill('Harbor Road 9, Testville')
  await page.getByLabel('Name').fill('Pinned')
  await page.getByRole('button', { name: 'Add restaurant' }).click()
  await expect(page.getByRole('heading', { name: 'Pinned', level: 1 })).toBeVisible()
  const pinnedMap = await restaurantMap(page, 'Pinned')
  await expect(pinnedMap.getByRole('button', { name: 'Pinned' })).toBeVisible()
})

test('a user shows the favorites on the map and sorts them', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('favorites')
  const sole = await api.createRestaurant(owner, { name: 'Sole Place', tags: [tag], location: barceloneta })
  const nou = await api.createRestaurant(owner, { name: 'Nou Place', tags: [tag], location: eixample })
  await api.createRestaurant(owner, { name: 'Other Place', tags: [tag], location: { ...barceloneta, longitude: 2.17 } })
  await api.addToList(user, 'favorites', sole.id)
  await api.addToList(user, 'favorites', nou.id)

  await page.goto(`/reviewers/${user.auth.user.id}`)
  await page.getByRole('region', { name: 'Lists' }).getByRole('link', { name: /Favorites/ }).click()
  await page.getByRole('navigation', { name: 'View' }).getByRole('link', { name: 'Map' }).click()
  await expect(page).toHaveURL(/\/restaurants\?list=[a-z0-9]{15}&view=map$/)
  await expect(page.getByText('List: Favorites')).toBeVisible()
  const map = page.getByRole('region', { name: 'Map of the restaurants' })
  await expect(map.getByRole('button', { name: 'Sole Place' })).toBeVisible()
  await expect(map.getByRole('button', { name: 'Nou Place' })).toBeVisible()
  await expect(map.getByRole('button', { name: 'Other Place' })).toHaveCount(0)

  await page.getByRole('button', { name: 'List', exact: true }).click()
  await page.getByRole('combobox', { name: 'Sort by' }).click()
  await page.getByRole('option', { name: 'Name' }).click()
  const names = page.getByRole('heading', { level: 2, name: / Place$/ })
  await expect(names).toHaveText(['Nou Place', 'Sole Place'])

  await page.getByRole('button', { name: 'Remove filter list favorites' }).click()
  await expect(page).not.toHaveURL(/list=/)
  await page.goto(`/restaurants?tags=${tag}&sort=name`)
  await expect(names).toHaveText(['Nou Place', 'Other Place', 'Sole Place'])
})

test('a list filter ignores the distance and frames the list on the map', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const sole = await api.createRestaurant(owner, { name: 'Sole Trip', location: barceloneta })
  const nou = await api.createRestaurant(owner, { name: 'Nou Trip', location: eixample })
  await api.addToList(user, 'favorites', sole.id)
  await api.addToList(user, 'favorites', nou.id)
  // Madrid is far from the restaurants of the list.
  await page.addInitScript(() => {
    localStorage.setItem('reviews-location', JSON.stringify({ latitude: 40.4168, longitude: -3.7038, label: 'Madrid', radius: 10000 }))
  })

  await page.goto(`/reviewers/${user.auth.user.id}`)
  await page.getByRole('region', { name: 'Lists' }).getByRole('link', { name: /Favorites/ }).click()
  await page.getByRole('link', { name: 'Sort and filter the restaurants' }).click()
  const names = page.getByRole('heading', { level: 2, name: / Trip$/ })
  await expect(page.getByRole('combobox', { name: 'Sort by' })).toHaveText('Nearest')
  await expect(names).toHaveText(['Nou Trip', 'Sole Trip'])

  await page.getByRole('button', { name: 'Favorite Nou Trip' }).click()
  await expect(names).toHaveText(['Sole Trip'])

  await page.getByRole('button', { name: 'Map', exact: true }).click()
  const map = page.getByRole('region', { name: 'Map of the restaurants' })
  await expect(map.getByRole('button', { name: 'Sole Trip' })).toBeVisible()
})

test.describe('from the lg breakpoint', () => {
  test.use({ viewport: { width: 1280, height: 800 } })

  test('the list and the map are side by side and mark each other', async ({ page, api }) => {
    const owner = await api.signUp('Joe')
    const tag = uniqueTag('side')
    const sole = await api.createRestaurant(owner, { name: 'Side Sole', tags: [tag], location: barceloneta })
    await api.createReview(owner, sole.id, { space: 8, service: 8, food: 8, price: 8, title: 'Good' })
    await api.createRestaurant(owner, { name: 'Side Nou', tags: [tag], location: eixample })

    await page.goto(`/restaurants?tags=${tag}`)
    await expect(page.getByRole('button', { name: 'Map', exact: true })).toHaveCount(0)
    const map = page.getByRole('region', { name: 'Map of the restaurants' })
    // A pin is a score stamp. A restaurant without reviews has an empty one.
    await expect(map.getByRole('button', { name: 'Side Sole, 8' })).toBeVisible()
    await expect(map.getByRole('button', { name: 'Side Nou, No reviews yet' })).toBeVisible()
    await expect(page.getByRole('heading', { level: 2 })).toHaveCount(2)

    // A hover marks the pin with the selected look. It does not press it.
    await page.getByRole('link', { name: 'Side Nou' }).hover()
    const nou = map.getByRole('button', { name: 'Side Nou, No reviews yet' })
    await expect(nou).toHaveClass(/map-stamp-selected/)
    await expect(nou).toHaveAttribute('aria-pressed', 'false')
    await map.getByRole('button', { name: 'Side Sole, 8' }).click()
    await expect(map.getByRole('button', { name: 'Side Sole, 8' })).toHaveAttribute('aria-pressed', 'true')
  })
})
