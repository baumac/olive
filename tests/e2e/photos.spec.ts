import { photoFile, uniqueTag } from '../api'
import { expect, signIn, test } from '../fixtures'

const scores = { space: 7, service: 8, food: 9, price: 6, cleanliness: 8 }

test('a user adds a photo with alt text to a review', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Photo Place' })
  const review = await api.createReview(user, restaurant.id, { ...scores, title: 'Pasta night' })

  await page.goto(`/restaurants/${restaurant.id}/reviews/${review.id}/edit`)
  await page.getByLabel('Add photo').setInputFiles({ name: 'pasta.jpg', mimeType: 'image/jpeg', buffer: photoFile('gps-rotated.jpg') })
  await expect(page.getByRole('img', { name: 'The selected photo' })).toBeVisible()
  await page.getByLabel('Description').fill('A plate of carbonara')
  await page.getByRole('button', { name: 'Upload photo' }).click()
  await expect(page.getByText('Photo added')).toBeVisible()

  await page.goto(`/restaurants/${restaurant.id}`)
  const card = page.getByRole('region', { name: 'Reviews' }).getByRole('listitem').filter({ hasText: 'Pasta night' })
  await expect(card.getByRole('button', { name: 'A plate of carbonara' })).toBeVisible()
})

// The user fixture signs in the page.
test('a user adds a photo on the create form of a review', async ({ page, api, user: _user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'New Photos' })

  await page.goto(`/restaurants/${restaurant.id}/reviews/new`)
  for (const label of ['Space', 'Service', 'Food', 'Value', 'Cleanliness']) {
    await page.getByRole('radiogroup', { name: label }).getByRole('radio', { name: '7 out of 10' }).click()
  }
  await page.getByLabel('Title').fill('With a menu')
  await page.getByLabel('Add photo').setInputFiles({ name: 'menu.png', mimeType: 'image/png', buffer: photoFile('menu.png') })
  await page.getByLabel('Description').fill('The menu')
  await page.getByRole('button', { name: 'Add to review' }).click()
  await expect(page.getByRole('list', { name: 'Photos to add' }).getByRole('img', { name: 'The menu' })).toBeVisible()
  await page.getByRole('button', { name: 'Save review' }).click()

  await expect(page.getByText('Review saved')).toBeVisible()
  await expect(page).toHaveURL(new RegExp(`/restaurants/${restaurant.id}$`))
  const card = page.getByRole('region', { name: 'Reviews' }).getByRole('listitem').filter({ hasText: 'With a menu' })
  await expect(card.getByRole('button', { name: 'The menu' })).toBeVisible()
})

test('the viewer moves between photos and closes with Escape', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Viewer Place' })
  const review = await api.createReview(owner, restaurant.id, { ...scores, title: 'Two photos' })
  await api.addPhoto(owner, review, 'gps-rotated.jpg', 'The first dish')
  await api.addPhoto(owner, review, 'menu.png')

  await page.goto(`/restaurants/${restaurant.id}`)
  const reviews = page.getByRole('region', { name: 'Reviews' })
  await reviews.getByRole('button', { name: 'The first dish' }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog.getByRole('heading', { name: 'Photo 1 of 2' })).toBeVisible()
  await expect(dialog.getByRole('img', { name: 'The first dish' })).toBeVisible()
  await page.keyboard.press('ArrowRight')
  await expect(dialog.getByRole('heading', { name: 'Photo 2 of 2' })).toBeVisible()
  await expect(dialog.getByRole('img', { name: 'Photo 2 of 2' })).toBeVisible()
  await expect(dialog.getByRole('button', { name: 'Delete photo' })).toHaveCount(0)
  await page.keyboard.press('Escape')
  await expect(dialog).toBeHidden()
  await expect(reviews.getByRole('button', { name: 'Photo 2 of 2' })).toBeFocused()
})

test('the restaurant card and page show the cover and the gallery', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  // The projects share one app. The search then finds only this restaurant.
  const name = uniqueTag('Gallery Place')
  const restaurant = await api.createRestaurant(owner, { name })
  const best = await api.createReview(owner, restaurant.id, { space: 9, service: 9, food: 9, price: 9, cleanliness: 9, title: 'Great' })
  const cover = await api.addPhoto(owner, best, 'gps-rotated.jpg', 'The best dish')
  const other = await api.signUp('Bo')
  const worse = await api.createReview(other, restaurant.id, { ...scores, title: 'Fine' })
  await api.addPhoto(other, worse, 'menu.png')

  await page.goto(`/restaurants?search=${encodeURIComponent(name)}`)
  const card = page.getByRole('listitem').filter({ has: page.getByRole('link', { name }) })
  await expect(card.locator('img')).toHaveAttribute('src', cover.urls.thumb)
  await expect(card.locator('img')).toHaveJSProperty('complete', true)

  await page.goto(`/restaurants/${restaurant.id}`)
  const coverButton = page.getByRole('button', { name: 'Open the cover photo: The best dish' })
  await expect(coverButton.locator('img')).toHaveAttribute('src', cover.urls.full)
  const gallery = page.getByRole('list', { name: `Photos of ${name}` }).getByRole('button')
  await expect(gallery).toHaveCount(2)
  await expect(gallery.first()).toHaveAccessibleName('The best dish')

  await coverButton.click()
  const dialog = page.getByRole('dialog')
  await expect(dialog.getByRole('heading', { name: 'Photo 1 of 2' })).toBeVisible()
  await expect(dialog).toContainText('From the review by Joe')
  await page.keyboard.press('ArrowRight')
  await expect(dialog).toContainText('From the review by Bo')
  await page.keyboard.press('Escape')
  await expect(coverButton).toBeFocused()

  await gallery.nth(1).click()
  await dialog.getByRole('link', { name: 'review by Bo' }).click()
  await expect(page).toHaveURL(new RegExp(`/restaurants/${restaurant.id}/reviews/${worse.id}$`))
})

test('a restaurant without photos has no cover and no gallery', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Plain Place' })
  await api.createReview(owner, restaurant.id, { ...scores, title: 'No pictures' })

  await page.goto(`/restaurants/${restaurant.id}`)
  await expect(page.getByRole('heading', { name: 'Reviews' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Photos' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: /cover photo/ })).toHaveCount(0)
})

test('the creator deletes a photo', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Delete Place' })
  const review = await api.createReview(user, restaurant.id, { ...scores, title: 'One photo' })
  const photo = await api.addPhoto(user, review, 'menu.png', 'An old menu')

  await page.goto(`/restaurants/${restaurant.id}/reviews/${review.id}`)
  await page.getByRole('button', { name: 'An old menu' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete photo' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete photo' }).click()
  await expect(page.getByText('Photo deleted')).toBeVisible()
  await expect(page.getByRole('button', { name: 'An old menu' })).toHaveCount(0)
  expect((await page.request.get(photo.urls.full)).status()).toBe(404)
})

test('another user cannot add or delete photos of a review', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Other Place' })
  const review = await api.createReview(owner, restaurant.id, { ...scores, title: 'Not yours' })
  await api.addPhoto(owner, review, 'menu.png', 'The menu of Joe')
  const other = await api.signUp('Bo')
  await signIn(page, other)

  await page.goto(`/restaurants/${restaurant.id}/reviews/${review.id}`)
  await page.getByRole('button', { name: 'The menu of Joe' }).click()
  await expect(page.getByRole('dialog').getByRole('img', { name: 'The menu of Joe' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Delete photo' })).toHaveCount(0)
  await expect(page.getByLabel('Add photo')).toHaveCount(0)
  const error = await api.addPhoto(other, review, 'menu.png').catch((e: { response: Response }) => e)
  expect('response' in error && error.response.status).toBe(403)
})

test('an anonymous visitor sees the photos and their files', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Open Place' })
  const review = await api.createReview(owner, restaurant.id, { ...scores, title: 'Public photo' })
  const photo = await api.addPhoto(owner, review, 'gps-rotated.jpg', 'A sunny table')

  await page.goto(`/restaurants/${restaurant.id}/reviews/${review.id}`)
  await expect(page.getByRole('button', { name: 'A sunny table' }).locator('img')).toHaveJSProperty('naturalWidth', 80)

  const file = await page.request.get(photo.urls.full)
  expect(file.status()).toBe(200)
  expect(file.headers()['content-type']).toBe('image/jpeg')
  expect(file.headers()['cache-control']).toBe('public, max-age=31536000, immutable')
  expect((await file.body()).includes('Exif')).toBe(false)
  // The fixture has the orientation "rotate 90". The API turns the pixels.
  expect([photo.width, photo.height]).toEqual([80, 160])
})

test('a deleted review takes its photos with it', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Gone Place' })
  const review = await api.createReview(owner, restaurant.id, { ...scores, title: 'Soon gone' })
  const photo = await api.addPhoto(owner, review, 'menu.png')
  expect((await page.request.get(photo.urls.thumb)).status()).toBe(200)
  await api.deleteReview(owner, review)
  expect((await page.request.get(photo.urls.thumb)).status()).toBe(404)
})
