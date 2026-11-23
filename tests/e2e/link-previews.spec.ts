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
  await expect(page).toHaveTitle('Casa <Pepe> & "Co" · Sponga47 Food Reviews')
  expect(served).toContain('<title>Casa &lt;Pepe&gt; &amp; &#34;Co&#34; · Sponga47 Food Reviews</title>')
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
  await expect(page).toHaveTitle('Great grill · Sponga47 Food Reviews')
  expect(served).toContain('<title>Great grill · Sponga47 Food Reviews</title>')

  const file = await page.request.get(photo.urls.full)
  expect(file.status()).toBe(200)
  expect(file.headers()['content-type']).toBe('image/jpeg')
})

test('a missing restaurant gets 404 and the not found screen', async ({ page }) => {
  const response = await page.goto('/restaurants/aaaaaaaaaaaaaaa')
  expect(response?.status()).toBe(404)
  await expect(page.locator('head meta[property="og:title"]')).toHaveAttribute('content', 'Sponga47 Food Reviews')
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

