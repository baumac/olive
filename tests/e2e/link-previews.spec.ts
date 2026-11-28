import { expect, test } from '../fixtures'
import { appURL } from '../playwright.config'

// The app server writes these tags into index.html. See decision 00002.
test('a restaurant page has its own link preview', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Casa <Pepe> & "Co"', description: 'Tapas\nand wine.' })
  const path = `/restaurants/${restaurant.id}`
  const served = await (await page.request.get(path)).text()

  const response = await page.goto(path)
  expect(response?.status()).toBe(200)
  const head = page.locator('head')
  await expect(head.locator('meta[property="og:title"]')).toHaveAttribute('content', 'Casa <Pepe> & "Co"')
  await expect(head.locator('meta[name="description"]')).toHaveAttribute('content', 'Tapas and wine.')
  await expect(head.locator('link[rel="canonical"]')).toHaveCount(0)
  await expect(head.locator('meta[property="og:url"]')).toHaveAttribute('content', `${appURL}${path}`)
  await expect(head.locator('meta[property="og:image"]')).toHaveAttribute('content', /\/og-image\.png$/)
  await expect(page.getByRole('heading', { name: 'Casa <Pepe> & "Co"' })).toBeVisible()
  await expect(page).toHaveTitle('Casa <Pepe> & "Co" · Olive')
  expect(served).toContain('<title>Casa &lt;Pepe&gt; &amp; &#34;Co&#34; · Olive</title>')
})

test('a review page shows its first photo in the link preview', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Preview Grill' })
  const review = await api.createReview(user, restaurant.id, {
    space: 8, service: 8, food: 8, price: 8, title: 'Great grill', feedback: 'Juicy ribs.',
  })
  const photo = await api.addPhoto(user, review, 'gps-rotated.jpg', 'Ribs on a plate')
  const path = `/restaurants/${restaurant.id}/reviews/${review.id}`
  const served = await (await page.request.get(path)).text()

  const response = await page.goto(path)
  expect(response?.status()).toBe(200)
  const head = page.locator('head')
  await expect(head.locator('meta[property="og:title"]')).toHaveAttribute('content', 'Great grill · Preview Grill')
  await expect(head.locator('meta[property="og:description"]')).toHaveAttribute('content', 'Juicy ribs.')
  await expect(head.locator('meta[property="og:type"]')).toHaveAttribute('content', 'article')
  await expect(head.locator('meta[property="og:image"]')).toHaveAttribute('content', `${appURL}${photo.urls.full}`)
  await expect(head.locator('meta[property="og:image:width"]')).toHaveAttribute('content', String(photo.width))
  await expect(head.locator('meta[property="og:image:alt"]')).toHaveAttribute('content', 'Ribs on a plate')
  await expect(page).toHaveTitle('Great grill · Olive')
  expect(served).toContain('<title>Great grill · Olive</title>')

  const file = await page.request.get(photo.urls.full)
  expect(file.status()).toBe(200)
  expect(file.headers()['content-type']).toBe('image/jpeg')
})

test('a missing restaurant gets 404 and the not found screen', async ({ page }) => {
  const response = await page.goto('/restaurants/aaaaaaaaaaaaaaa')
  expect(response?.status()).toBe(404)
  await expect(page.locator('head meta[property="og:title"]')).toHaveAttribute('content', 'Olive, a food journal')
  await expect(page.getByRole('heading', { name: 'We cannot find this page' })).toBeVisible()
})

test('a recipe page has its own link preview', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const recipe = await api.createRecipe(owner, { name: 'Preview dal', description: 'Red lentils.', servings: 4 })
  const path = `/recipes/${recipe.id}`

  const response = await page.goto(path)
  expect(response?.status()).toBe(200)
  const head = page.locator('head')
  await expect(head.locator('meta[property="og:title"]')).toHaveAttribute('content', 'Preview dal')
  await expect(head.locator('meta[name="description"]')).toHaveAttribute('content', 'Red lentils.')
  await expect(head.locator('link[rel="canonical"]')).toHaveCount(0)
  await expect(page.getByRole('heading', { name: 'Preview dal', level: 1 })).toBeVisible()
})

// mondayInCET gives the Monday of the week of now plus the days in the zone
// of the link previews. See decision 00013.
function mondayInCET(days = 0): string {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: 'CET', year: 'numeric', month: '2-digit', day: '2-digit', weekday: 'short' })
    .formatToParts(new Date())
  const part = (type: string) => parts.find((p) => p.type === type)!.value
  const today = new Date(Date.UTC(Number(part('year')), Number(part('month')) - 1, Number(part('day'))))
  const weekday = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].indexOf(part('weekday'))
  today.setUTCDate(today.getUTCDate() - weekday + days)
  return today.toISOString().slice(0, 10)
}

test('a week page has its own link preview', async ({ page, api, user }) => {
  const quiet = await (await page.request.get('/weeks/2001-01-03')).text()
  expect(quiet).toContain('<title>Week of 1 January 2001 · Olive</title>')
  expect(quiet).toContain('<meta property="og:type" content="article" />')
  expect(quiet).toContain('<meta name="description" content="A quiet week on Olive." />')
  expect(quiet).toContain(`<meta property="og:url" content="${appURL}/weeks/2001-01-01" />`)

  const restaurant = await api.createRestaurant(user, { name: 'Week Preview Grill' })
  // All projects of one run share one app. The counts then include the
  // entries of other tests.
  const description = /<meta name="description" content="So far, \d+ entr(y|ies) from \d+ (person|people): [^"]*\d+ visits?[^"]*\." \/>/
  await expect(async () => {
    const monday = mondayInCET()
    await api.createReview(user, restaurant.id, { space: 8, service: 8, food: 8, price: 8, title: 'This week' })
    const current = await (await page.request.get(`/weeks/${monday}`)).text()
    const next = await page.request.get(`/weeks/${mondayInCET(7)}`)
    // A run across Monday 00:00 in CET tries again in the new week.
    expect(mondayInCET()).toBe(monday)
    expect(current).toMatch(description)
    expect(next.status()).toBe(404)
  }).toPass()

  for (const path of ['/weeks/2026-02-30', '/weeks/1899-12-25']) {
    const response = await page.request.get(path)
    expect(response.status(), path).toBe(404)
    expect(await response.text()).toContain('<meta property="og:title" content="Olive, a food journal" />')
  }
})
