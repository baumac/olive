import { expect, test } from '../fixtures'

test('a user favorites a restaurant from a card and sees it in the favorites list', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const kitchen = await api.createRestaurant(owner, { name: 'Heart Kitchen' })
  await api.createRestaurant(owner, { name: 'Heart Grill' })

  await page.goto(`/reviewers/${owner.auth.user.id}`)
  const heart = page.getByRole('button', { name: 'Favorite Heart Kitchen' })
  await expect(heart).toHaveAttribute('aria-pressed', 'false')
  await heart.click()
  await expect(heart).toHaveAttribute('aria-pressed', 'true')
  await expect(page).toHaveURL(new RegExp(`/reviewers/${owner.auth.user.id}$`))

  await page.reload()
  await expect(page.getByRole('button', { name: 'Favorite Heart Kitchen' })).toHaveAttribute('aria-pressed', 'true')
  await expect(page.getByRole('button', { name: 'Favorite Heart Grill' })).toHaveAttribute('aria-pressed', 'false')

  await page.goto(`/restaurants/${kitchen.id}`)
  await expect(page.getByRole('button', { name: 'Favorite Heart Kitchen' })).toHaveAttribute('aria-pressed', 'true')

  await page.getByRole('button', { name: /account menu/ }).click()
  await page.getByRole('menuitem', { name: 'My lists' }).click()
  await expect(page).toHaveURL(new RegExp(`/reviewers/${user.auth.user.id}$`))
  await page.getByRole('region', { name: 'Lists' }).getByRole('link', { name: /Favorites.*1 restaurant/ }).click()

  await expect(page.getByRole('heading', { name: 'Favorites', level: 1 })).toBeVisible()
  await expect(page.getByRole('heading', { level: 3 })).toHaveText(['Heart Kitchen'])
  await expect(page.getByRole('button', { name: 'Delete list' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Rename list' })).toHaveCount(0)

  await page.getByRole('button', { name: 'Favorite Heart Kitchen' }).click()
  await expect(page.getByText('Select the heart of a restaurant to add it here.')).toBeVisible()
})

test('a user makes a list, renames it, and deletes it', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const bistro = await api.createRestaurant(owner, { name: 'List Bistro' })

  await page.goto(`/restaurants/${bistro.id}`)
  await page.getByRole('button', { name: 'Save to list' }).click()
  await expect(page.getByText('You have no lists yet. Make the first one.')).toBeVisible()
  await page.getByLabel('Name of a new list').fill('Date night')
  await page.getByRole('button', { name: 'Create', exact: true }).click()
  await expect(page.getByRole('checkbox', { name: 'Date night' })).toBeChecked()
  await page.getByLabel('Name of a new list').fill('date NIGHT')
  await page.getByRole('button', { name: 'Create', exact: true }).click()
  await expect(page.getByRole('alert').filter({ hasText: 'You have a list with this name.' })).toBeVisible()
  await page.keyboard.press('Escape')

  await page.goto(`/reviewers/${user.auth.user.id}`)
  await page.getByRole('region', { name: 'Lists' }).getByRole('link', { name: /Date night.*1 restaurant/ }).click()
  await expect(page.getByRole('heading', { name: 'Date night', level: 1 })).toBeVisible()
  await expect(page.getByRole('heading', { level: 3 })).toHaveText(['List Bistro'])

  await page.getByRole('button', { name: 'Rename list' }).click()
  await page.getByLabel('Name of the list').fill('Dinner')
  await page.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Dinner', level: 1 })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Rename list' })).toBeFocused()

  await page.getByRole('button', { name: 'Remove List Bistro from the list' }).click()
  await expect(page.getByText('Open a restaurant and select Save to list to add it here.')).toBeVisible()

  await page.getByRole('button', { name: 'Delete list' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete list' }).click()
  await expect(page).toHaveURL(new RegExp(`/reviewers/${user.auth.user.id}$`))
  await expect(page.getByRole('region', { name: 'Lists' }).getByRole('link')).toHaveCount(1)
})

test('a signed out visitor reads a list and cannot change it', async ({ page, api }) => {
  const ana = await api.signUp('Ana')
  const cafe = await api.createRestaurant(ana, { name: 'Public Cafe' })
  const list = await api.createList(ana, 'Brunch')
  await api.addToList(ana, list.id, cafe.id)

  await page.goto(`/lists/${list.id}`)
  await expect(page.getByRole('heading', { name: 'Brunch', level: 1 })).toBeVisible()
  await expect(page).toHaveTitle(/^Brunch by Ana · /)
  await expect(page.getByRole('heading', { level: 3 })).toHaveText(['Public Cafe'])
  await expect(page.getByRole('button', { name: 'Rename list' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: /Remove .* from the list/ })).toHaveCount(0)
  await page.getByRole('main').getByRole('link', { name: 'Ana', exact: true }).click()
  await expect(page).toHaveURL(new RegExp(`/reviewers/${ana.auth.user.id}$`))
  await expect(page.getByRole('region', { name: 'Lists' }).getByLabel('Name of a new list')).toHaveCount(0)

  await page.goto(`/lists/${list.id}`)
  await page.getByRole('button', { name: 'Favorite Public Cafe' }).click()
  await expect(page).toHaveURL(/\/sign-in\?redirect=/)
})

test('an unknown list ID has the not found page', async ({ page }) => {
  await page.goto('/lists/aaaaaaaaaaaaaaa')
  await expect(page.getByRole('heading', { name: 'We cannot find this page' })).toBeVisible()
})
