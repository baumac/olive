import { expect, signIn, test } from '../fixtures'

test('another user undoes a wrong change of a restaurant', async ({ page, api }) => {
  const ana = await api.signUp('Ana')
  const eva = await api.signUp('Eva')
  const joe = await api.signUp('Joe')
  const r = await api.createRestaurant(ana, { name: 'Can Solé', description: 'Small tapas bar', phone: '931 000 000' })
  await api.updateRestaurant(eva, r.id, { description: null, tags: new Set(['tapas']) })
  await signIn(page, joe)

  await page.goto(`/restaurants/${r.id}`)
  await page.getByRole('link', { name: 'History' }).click()
  await expect(page.getByRole('heading', { name: 'History of Can Solé', level: 1 })).toBeVisible()
  const latest = page.getByRole('listitem').filter({ has: page.getByRole('heading', { name: 'Version 2' }) })
  await expect(latest.getByText('Eva')).toBeVisible()
  await expect(latest.getByText('Small tapas bar')).toBeVisible()
  await expect(page.getByRole('listitem').filter({ hasText: 'Added the restaurant.' })).toBeVisible()

  await latest.getByRole('link', { name: 'Undo version 2' }).click()
  await expect(page.getByText('Undo of version 2')).toBeVisible()
  await expect(page.getByLabel('Description')).toHaveValue('Small tapas bar')
  await page.getByRole('button', { name: 'Save restaurant' }).click()
  await expect(page.getByRole('heading', { name: 'Can Solé', level: 1 })).toBeVisible()
  await expect(page.getByText('Small tapas bar')).toBeVisible()

  await page.getByRole('link', { name: 'History' }).click()
  const undo = page.getByRole('listitem').filter({ has: page.getByRole('heading', { name: 'Version 3' }) })
  await expect(undo.getByText('Joe')).toBeVisible()
  await expect(undo.getByText('Description')).toBeVisible()
})

test('an undo keeps a property that a newer version changed', async ({ page, api }) => {
  const ana = await api.signUp('Ana')
  const joe = await api.signUp('Joe')
  const r = await api.createRestaurant(ana, { name: 'Bar Pinotxo', phone: '931 000 000', website: 'https://pinotxo.example' })
  await api.updateRestaurant(joe, r.id, { phone: '931 111 111', website: null })
  await api.updateRestaurant(ana, r.id, { phone: '931 222 222' })
  await signIn(page, joe)

  await page.goto(`/restaurants/${r.id}/history`)
  await page.getByRole('link', { name: 'Undo version 2' }).click()
  await expect(page.getByText('Ana changed the phone again in version 3. The form keeps the current value.')).toBeVisible()
  await expect(page.getByLabel('Phone')).toHaveValue('931 222 222')
  await expect(page.getByLabel('Website')).toHaveValue('https://pinotxo.example')
  await page.getByRole('button', { name: 'Save restaurant' }).click()
  await expect(page.getByRole('heading', { name: 'Bar Pinotxo', level: 1 })).toBeVisible()
  await expect(page.getByRole('link', { name: 'pinotxo.example' })).toBeVisible()
})

test('a user undoes a change of the steps of a recipe', async ({ page, api, user }) => {
  const joe = await api.signUp('Joe')
  const recipe = await api.createRecipe(user, { name: 'Dal', ingredients: ['1 cup of lentils'], steps: ['Rinse the lentils.', 'Cook them.'] })
  await api.updateRecipe(joe, recipe.id, { steps: ['Cook them.'] })

  await page.goto(`/recipes/${recipe.id}/history`)
  await page.getByRole('link', { name: 'Undo version 2' }).click()
  await expect(page.getByText('Undo of version 2')).toBeVisible()
  await page.getByRole('button', { name: 'Save recipe' }).click()
  await expect(page.getByRole('heading', { name: 'Dal', level: 1 })).toBeVisible()
  await expect(page.getByText('Rinse the lentils.')).toBeVisible()
})

test('a signed out visitor sees no history', async ({ page, api }) => {
  const ana = await api.signUp('Ana')
  const r = await api.createRestaurant(ana, { name: 'Quimet' })

  await page.goto(`/restaurants/${r.id}`)
  await expect(page.getByRole('heading', { name: 'Quimet', level: 1 })).toBeVisible()
  await expect(page.getByRole('link', { name: 'History' })).toHaveCount(0)
  await page.goto(`/restaurants/${r.id}/history`)
  await expect(page).toHaveURL(/\/sign-in/)
})
