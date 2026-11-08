import { uniqueTag } from '../api'
import { expect, test } from '../fixtures'

// The noCspViolations fixture checks each step. This test opens the menus,
// popovers, dialogs, and toasts. These set styles at runtime.
test('the UI works under the Content Security Policy', async ({ page, api, user }) => {
  const response = await page.goto('/restaurants')
  expect(response?.headers()['content-security-policy']).toContain("script-src 'self'")

  await page.getByRole('button', { name: /^Theme:/ }).click()
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
  await page.getByRole('option', { name: 'Name, A to Z' }).click()

  const restaurant = await api.createRestaurant(user, { name: 'Strict Place' })
  await page.goto(`/restaurants/${restaurant.id}/reviews/new`)
  await page.getByRole('tab', { name: 'Joc de cartes' }).click()
  for (const label of ['Space', 'Service', 'Food', 'Price']) {
    await page.getByRole('radiogroup', { name: label }).getByRole('radio', { name: '7 out of 10' }).click()
  }
  await page.getByLabel('Title').fill('Fine')
  await page.getByRole('button', { name: 'Save review' }).click()
  await expect(page.getByText('Review saved')).toBeVisible()

  await page.getByRole('button', { name: 'Filters' }).click()
  await page.getByRole('combobox', { name: 'Kind' }).click()
  await page.getByRole('option', { name: 'Basic' }).click()
  await expect(page.getByText('Kind: Basic')).toBeVisible()
  await page.keyboard.press('Escape')

  // A restaurant with reviews cannot be deleted. The UI shows an error toast.
  await page.getByRole('button', { name: 'Delete restaurant' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete restaurant' }).click()
  await expect(page.getByText('Restaurant not deleted')).toBeVisible()

  await page.getByRole('button', { name: `AN, account menu for ${user.name}` }).click()
  await expect(page.getByRole('menuitem', { name: 'Sign out' })).toBeVisible()
})
