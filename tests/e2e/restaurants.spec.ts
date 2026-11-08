import { uniqueTag } from '../api'
import { expect, signIn, test } from '../fixtures'

test('the list shows restaurants, filters by tag, and loads more', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('batch')
  await Promise.all(
    Array.from({ length: 21 }, (_, i) =>
      api.createRestaurant(owner, { name: `Place ${String(i).padStart(2, '0')}`, tags: [tag] }),
    ),
  )

  await page.goto(`/restaurants?tags=${tag}&sort=name`)
  await expect(page.getByRole('heading', { name: 'Restaurants', level: 1 })).toBeVisible()
  await expect(page.getByText(`Tag: ${tag}`)).toBeVisible()
  const cards = page.getByRole('heading', { level: 2, name: /^Place / })
  await expect(cards).toHaveCount(20)
  await expect(page.getByRole('status')).toHaveText('20 restaurants shown, more to load')

  await page.getByRole('button', { name: 'Load more restaurants' }).click()
  await expect(cards).toHaveCount(21)
  await expect(page.getByRole('button', { name: 'Load more restaurants' })).toHaveCount(0)
  await expect(page.getByRole('listitem').filter({ hasText: 'Place 20' })).toBeFocused()
})

test('a signed in user adds, edits, and deletes a restaurant', async ({ page, user }) => {
  await page.goto('/restaurants')
  await page.getByRole('link', { name: 'Add restaurant' }).first().click()

  await page.getByLabel('Name').fill('Taqueria Sol')
  await page.getByLabel('Description').fill('Tacos al pastor.')
  await page.getByLabel('Tags').fill('mexican')
  await page.getByLabel('Tags').press('Enter')
  await page.getByRole('button', { name: 'Add restaurant' }).click()

  await expect(page.getByRole('heading', { name: 'Taqueria Sol', level: 1 })).toBeVisible()
  await expect(page.getByText('Restaurant added')).toBeVisible()
  await expect(page.getByText(`Added by ${user.name}`)).toBeVisible()

  await page.getByRole('link', { name: 'Edit restaurant' }).click()
  await expect(page.getByLabel('Name')).toHaveValue('Taqueria Sol')
  await page.getByLabel('Name').fill('Taqueria Sol y Luna')
  await page.getByRole('button', { name: 'Save restaurant' }).click()
  await expect(page.getByRole('heading', { name: 'Taqueria Sol y Luna', level: 1 })).toBeVisible()

  await page.getByRole('button', { name: 'Delete restaurant' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete restaurant' }).click()
  await expect(page).toHaveURL(/\/restaurants$/)
  await expect(page.getByText('Restaurant deleted')).toBeVisible()
})

test('only the creator sees edit and delete', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Can Sole' })
  const other = await api.signUp('Ana')
  await signIn(page, other)

  await page.goto(`/restaurants/${restaurant.id}`)
  await expect(page.getByRole('heading', { name: 'Can Sole', level: 1 })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Edit restaurant' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Delete restaurant' })).toHaveCount(0)

  await page.goto(`/restaurants/${restaurant.id}/edit`)
  await expect(page.getByRole('heading', { name: 'You cannot open this page' })).toBeVisible()
})

test('a restaurant with reviews keeps its name and cannot be deleted', async ({ page, api, user }) => {
  const restaurant = await api.createRestaurant(user, { name: "Luigi's" })
  const reviewer = await api.signUp('Joe')
  await api.createReview(reviewer, restaurant.id, { space: 9, service: 9, food: 9, price: 9, cleanliness: 9, title: 'Great' })

  await page.goto(`/restaurants/${restaurant.id}/edit`)
  await page.getByLabel('Name').fill("Luigi's Trattoria")
  await page.getByRole('button', { name: 'Save restaurant' }).click()
  await expect(page.getByText('The name cannot change because the restaurant has reviews.')).toBeVisible()

  await page.goto(`/restaurants/${restaurant.id}`)
  await page.getByRole('button', { name: 'Delete restaurant' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete restaurant' }).click()
  await expect(page.getByText('Restaurant not deleted')).toBeVisible()
  await expect(page.getByText('A restaurant with reviews cannot be deleted.', { exact: true })).toBeVisible()
})
