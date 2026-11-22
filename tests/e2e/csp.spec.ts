import { photoFile, uniqueTag } from '../api'
import { expect, test } from '../fixtures'

// The noCspViolations fixture checks each step. This test opens the menus,
// popovers, dialogs, and toasts. These set styles at runtime.
test('the UI works under the Content Security Policy', async ({ page, api, user }) => {
  const response = await page.goto('/restaurants')
  expect(response?.headers()['content-security-policy']).toContain("script-src 'self'")

  await page.getByRole('button', { name: /account menu/ }).click()
  await page.getByRole('menuitemradio', { name: 'Dark' }).click()
  await expect(page.locator('html')).toHaveClass(/dark/)

  const tag = uniqueTag('csp')
  await Promise.all(
    Array.from({ length: 21 }, (_, i) => api.createRestaurant(user, { name: `Csp ${String(i).padStart(2, '0')}`, tags: [tag] })),
  )
  await page.goto(`/restaurants?tags=${tag}`)
  await page.getByRole('button', { name: 'Load more restaurants' }).click()
  await expect(page.getByRole('heading', { level: 2, name: /^Csp / })).toHaveCount(21)

  await page.getByRole('button', { name: 'Filters' }).click()
  await expect(page.getByLabel('Tags')).toBeVisible()
  await page.keyboard.press('Escape')
  await page.getByRole('combobox', { name: 'Sort by' }).click()
  await page.getByRole('option', { name: 'Name' }).click()

  const restaurant = await api.createRestaurant(user, { name: 'Strict Place' })
  await page.goto(`/restaurants/${restaurant.id}/reviews/new`)
  for (const label of ['Space', 'Service', 'Food', 'Value']) {
    await page.getByRole('radiogroup', { name: label }).getByRole('radio', { name: '7 out of 10' }).click()
  }
  await page.getByLabel('Title').fill('Fine')
  await page.getByRole('button', { name: 'Save review' }).click()
  await expect(page.getByText('Review saved')).toBeVisible()

  // A short review list shows the filters only with a sort in the URL.
  await page.goto(`/restaurants/${restaurant.id}?sort=overall`)
  await page.getByRole('button', { name: 'Filters' }).click()
  await page.getByRole('combobox', { name: 'Food' }).click()
  await page.getByRole('option', { name: '7 or more' }).click()
  await expect(page.getByText('Food: 7 or more')).toBeVisible()
  await expect(page.getByRole('listbox')).toBeHidden()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog', { name: 'Filters' })).toBeHidden()

  // The page does not know the new review of another user. The delete gets
  // 409. The UI then shows an error toast.
  const reviewer = await api.signUp('Joe')
  await api.createReview(reviewer, restaurant.id, { space: 7, service: 7, food: 7, price: 7, title: 'Late' })
  await page.getByRole('button', { name: 'Delete restaurant' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete restaurant' }).click()
  await expect(page.getByText('Restaurant not deleted')).toBeVisible()

  await page.getByRole('button', { name: `AN, account menu for ${user.name}` }).click()
  await expect(page.getByRole('menuitem', { name: 'Sign out' })).toBeVisible()
})

// The upload form shows a blob: preview. The viewer shows the full image.
test('the photos work under the Content Security Policy', async ({ page, api, user }) => {
  const restaurant = await api.createRestaurant(user, { name: 'Photo Policy' })
  const review = await api.createReview(user, restaurant.id, { space: 7, service: 7, food: 7, price: 7, title: 'Policy' })
  await api.addPhoto(user, review, 'menu.png', 'A menu')

  await page.goto(`/restaurants/${restaurant.id}/reviews/${review.id}/edit`)
  await page.getByLabel('Add photo').setInputFiles({ name: 'dish.jpg', mimeType: 'image/jpeg', buffer: photoFile('gps-rotated.jpg') })
  await expect(page.getByRole('img', { name: 'The selected photo' })).toHaveJSProperty('complete', true)
  await page.getByRole('button', { name: 'Upload photo' }).click()
  await expect(page.getByText('Photo added')).toBeVisible()

  await page.getByRole('button', { name: 'A menu' }).click()
  await expect(page.getByRole('dialog').getByRole('img', { name: 'A menu' })).toHaveJSProperty('complete', true)
  await page.getByRole('button', { name: 'Next photo' }).click()
  await expect(page.getByRole('dialog').getByRole('heading', { name: 'Photo 2 of 2' })).toBeVisible()

  await page.goto(`/restaurants/${restaurant.id}`)
  await expect(page.getByRole('button', { name: 'Open the cover photo: A menu' }).locator('img')).toHaveJSProperty('complete', true)
  await expect(page.getByRole('list', { name: 'Photos of Photo Policy' }).getByRole('button')).toHaveCount(2)
})
