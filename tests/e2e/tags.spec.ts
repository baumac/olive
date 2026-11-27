import { uniqueTag } from '../api'
import { expect, selectPlace, test } from '../fixtures'

test('the tag input suggests the tags of other users and adds a new tag', async ({ page, api, user: _user }) => {
  const other = await api.signUp('Eva')
  const known = uniqueTag('wood-fired')
  const fresh = uniqueTag('wood-oven')
  await api.createRestaurant(other, { name: uniqueTag('Forno'), tags: [known] })

  await page.goto('/restaurants/new')
  await selectPlace(page, uniqueTag('pizzeria'))
  await page.getByLabel('Name').fill('Pizzeria Sol')
  const tags = page.getByLabel('Tags')
  await tags.fill(known.slice(0, -2))
  await expect(page.getByRole('option', { name: known })).toContainText('1 restaurant')
  await tags.press('ArrowDown')
  await tags.press('Enter')
  await expect(page.getByRole('button', { name: `Remove ${known}` })).toBeVisible()

  await tags.fill(fresh)
  await page.getByRole('option', { name: `Add new tag ‘${fresh}’` }).click()
  await expect(page.getByRole('button', { name: `Remove ${fresh}` })).toBeVisible()
  await page.getByRole('button', { name: 'Add restaurant' }).click()

  await expect(page.getByRole('heading', { name: 'Pizzeria Sol', level: 1 })).toBeVisible()
  await page.getByRole('list', { name: 'Tags' }).getByRole('link', { name: known }).click()
  await expect(page).toHaveURL(new RegExp(`/tags/${known}$`))
  await expect(page.getByRole('heading', { name: known, level: 1 })).toBeVisible()
  await expect(page.getByText('2 restaurants')).toBeVisible()
})

test('the tag input changes free text to the stored form of a tag', async ({ page, api, user: _user }) => {
  const other = await api.signUp('Eva')
  const known = uniqueTag('wood-fired')
  const fresh = uniqueTag('wood-oven')
  await api.createRestaurant(other, { name: uniqueTag('Forno'), tags: [known] })

  await page.goto('/restaurants/new')
  await selectPlace(page, uniqueTag('pizzeria'))
  await page.getByLabel('Name').fill('Pizzeria Luna')
  const tags = page.getByLabel('Tags')
  await tags.fill(known.toUpperCase().replaceAll('-', ' '))
  await expect(page.getByRole('option', { name: known })).toContainText('1 restaurant')
  await expect(page.getByRole('option', { name: /Add new tag/ })).toHaveCount(0)
  await tags.press('Enter')
  await expect(page.getByRole('button', { name: `Remove ${known}` })).toBeVisible()

  await tags.fill(`${fresh.replaceAll('-', '_')}!`)
  await expect(page.getByRole('option', { name: `Add new tag ‘${fresh}’` })).toBeVisible()
  await tags.press('Enter')
  await expect(page.getByRole('button', { name: `Remove ${fresh}` })).toBeVisible()
  await page.getByRole('button', { name: 'Add restaurant' }).click()

  await expect(page.getByRole('heading', { name: 'Pizzeria Luna', level: 1 })).toBeVisible()
  const list = page.getByRole('list', { name: 'Tags' })
  await expect(list.getByRole('link', { name: known })).toBeVisible()
  await expect(list.getByRole('link', { name: fresh })).toBeVisible()

  await page.goto(`/tags/${encodeURIComponent(known.toUpperCase().replaceAll('-', ' '))}`)
  await expect(page).toHaveURL(new RegExp(`/tags/${known}$`))
  await expect(page.getByRole('heading', { name: known, level: 1 })).toBeVisible()
  await expect(page.getByText('2 restaurants')).toBeVisible()
})

test('the tag page shows the restaurants and the recipes of a tag', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('brunch')
  const restaurant = await api.createRestaurant(owner, { name: uniqueTag('Cafe'), tags: [tag] })
  const recipe = await api.createRecipe(owner, { name: uniqueTag('Pancakes'), tags: [tag] })

  await page.goto(`/tags/${tag}`)
  await expect(page.getByText('1 restaurant and 1 recipe')).toBeVisible()
  await expect(page.getByRole('heading', { name: restaurant.name, level: 3 })).toBeVisible()
  await expect(page.getByRole('heading', { name: recipe.name, level: 3 })).toBeVisible()

  await page.getByRole('navigation', { name: 'Kind of the tagged things' }).getByRole('link', { name: 'Recipes' }).click()
  await expect(page).toHaveURL(/\?kind=recipes$/)
  await expect(page.getByRole('heading', { name: 'Restaurants', level: 2 })).toHaveCount(0)
  await expect(page.getByRole('heading', { name: recipe.name, level: 3 })).toBeVisible()

  await page.getByRole('link', { name: 'Sort and filter the recipes' }).click()
  await expect(page.getByText(`Tag: ${tag}`)).toBeVisible()
  await expect(page.getByRole('heading', { name: recipe.name, level: 2 })).toBeVisible()
})

test('the tag filter suggests only existing tags', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('tapas')
  const restaurant = await api.createRestaurant(owner, { name: uniqueTag('Bar'), tags: [tag] })

  await page.goto('/restaurants')
  await page.getByRole('button', { name: 'Filters' }).click()
  const tags = page.getByLabel('Tags')
  await tags.fill(`${tag}-none`)
  await expect(page.getByRole('option')).toHaveCount(0)
  await tags.fill(tag.toUpperCase())
  await page.getByRole('option', { name: tag }).click()
  await expect(page.getByText(`Tag: ${tag}`)).toBeVisible()
  await expect(page.getByRole('heading', { name: restaurant.name, level: 2 })).toBeVisible()
})

test('an unknown tag has the not found page', async ({ page }) => {
  await page.goto(`/tags/${uniqueTag('none')}`)
  await expect(page.getByRole('heading', { name: 'We cannot find this page' })).toBeVisible()
  await page.goto('/tags/Not%20A%20Tag')
  await expect(page.getByRole('heading', { name: 'We cannot find this page' })).toBeVisible()
  await page.goto('/tags/%21%21%21')
  await expect(page.getByRole('heading', { name: 'We cannot find this page' })).toBeVisible()
})
