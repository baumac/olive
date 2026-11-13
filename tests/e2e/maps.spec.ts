import { uniqueTag } from '../api'
import { expect, test } from '../fixtures'

// The fake server of photon.ts gives the map tiles. The noCspViolations
// fixture checks that the browser loads them under the policy.
const barceloneta = { latitude: 41.3766, longitude: 2.1893, address: 'Carrer de Sant Carles 4, Barcelona' }
const eixample = { latitude: 41.39, longitude: 2.16, address: 'Carrer Nou 1, Barcelona' }

test('the restaurant page shows the location on a map', async ({ page, api }) => {
  const restaurant = await api.createRestaurant(await api.signUp('Joe'), { name: 'Can Sole', location: barceloneta })

  const tiles = page.waitForResponse((res) => res.url().includes('/tiles/') && res.ok())
  await page.goto(`/restaurants/${restaurant.id}`)
  const map = page.getByRole('region', { name: 'Map of Can Sole' })
  await expect(map).toBeVisible()
  await tiles
  await expect(map.getByRole('button', { name: 'Can Sole' })).toBeVisible()
  await expect(map.getByRole('link', { name: '© OpenStreetMap contributors' })).toHaveAttribute('href', 'https://www.openstreetmap.org/copyright')
  await expect(page.getByRole('link', { name: 'Open in OpenStreetMap' })).toHaveAttribute(
    'href',
    'https://www.openstreetmap.org/?mlat=41.3766&mlon=2.1893#map=18/41.3766/2.1893',
  )
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

  const map = page.getByRole('region', { name: 'Map of the restaurants' }).first()
  await expect(page.getByText('2 restaurants on the map')).toBeVisible()
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

test('a click on the map sets the point of a location by hand', async ({ page, user: _user }) => {
  await page.goto('/restaurants/new')
  await page.getByRole('button', { name: 'Enter the location by hand' }).click()
  const map = page.getByRole('region', { name: 'Map to set the point' })
  await expect(map.getByRole('button', { name: 'The point of the restaurant' })).toHaveCount(0)

  await map.click({ position: { x: 200, y: 100 } })
  await expect(page.getByLabel('Latitude')).toHaveValue(/^41\.\d{1,6}$/)
  await expect(page.getByLabel('Longitude')).toHaveValue(/^2\.\d{1,6}$/)
  await expect(map.getByRole('button', { name: 'The point of the restaurant' })).toBeVisible()

  // The typed numbers move the pin, and the form saves the point.
  await page.getByLabel('Latitude').fill('-45.5')
  await page.getByLabel('Longitude').fill('-140.25')
  await page.getByLabel('Address').fill('Harbor Road 9, Testville')
  await page.getByLabel('Name').fill('Pinned')
  await page.getByRole('button', { name: 'Add restaurant' }).click()
  await expect(page.getByRole('heading', { name: 'Pinned', level: 1 })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Map of Pinned' }).getByRole('button', { name: 'Pinned' })).toBeVisible()
})
