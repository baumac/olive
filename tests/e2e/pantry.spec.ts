import { uniqueTag } from '../api'
import { expect, test } from '../fixtures'

test('the sections link the pantry', async ({ page }) => {
  await page.goto('/pantry')
  const sections = page.getByRole('navigation', { name: 'Sections' })
  await expect(sections.getByRole('link', { name: 'Pantry' })).toHaveAttribute('aria-current', 'page')
  await expect(page.getByRole('heading', { name: 'Pantry', level: 1 })).toBeAttached()
})

test('a signed in user adds, edits, and deletes a pantry item', async ({ page, user: _ }) => {
  const name = uniqueTag('Kabinett')
  await page.goto('/pantry')
  // An empty pantry has a second link in its empty state.
  await page.getByRole('link', { name: 'Add pantry item' }).first().click()
  await expect(page.getByRole('heading', { name: 'Add a pantry item' })).toBeVisible()

  await page.getByLabel('Name').fill(name)
  await page.getByRole('button', { name: 'Add pantry item' }).click()
  await expect(page.getByText('Select a category.')).toBeVisible()

  await page.getByLabel('Category').click()
  await page.getByRole('option', { name: 'Wine' }).click()
  await page.getByLabel('Producer').fill('Weingut Example')
  await page.getByLabel('Origin').fill('Mosel, Germany')
  await page.getByLabel('Style').fill('Riesling')
  await page.getByRole('button', { name: 'Add pantry item' }).click()

  await expect(page.getByText('Pantry item added', { exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible()
  const details = page.getByRole('definition')
  await expect(details).toHaveText(['Weingut Example', 'Mosel, Germany', 'Riesling'])
  await expect(page.getByRole('heading', { name: 'Every tasting', level: 2 })).toBeVisible()

  await page.getByRole('link', { name: 'Edit pantry item' }).click()
  await page.getByLabel('Category').click()
  await page.getByRole('option', { name: 'Beer' }).click()
  await page.getByLabel('Style').fill('')
  await page.getByRole('button', { name: 'Save pantry item' }).click()
  await expect(page.getByText('Pantry item saved', { exact: true })).toBeVisible()
  await expect(page.getByText('Beer', { exact: true })).toBeVisible()
  await expect(details).toHaveText(['Weingut Example', 'Mosel, Germany'])

  await page.getByRole('link', { name: 'History' }).click()
  await expect(page.getByRole('heading', { name: `History of ${name}`, level: 1 })).toBeVisible()
  await expect(page.getByText('Category', { exact: true }).first()).toBeVisible()
  await page.getByRole('link', { name: 'Back to the pantry item' }).click()

  await page.getByRole('button', { name: 'Delete pantry item' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click()
  await expect(page.getByText('Pantry item deleted', { exact: true })).toBeVisible()
  await expect(page).toHaveURL(/\/pantry$/)
})

test('the pantry item form shows the pantry items of the same name', async ({ page, api, user: _ }) => {
  const owner = await api.signUp('Joe')
  const name = uniqueTag('Comte')
  const item = await api.createPantryItem(owner, { name, category: 'cheese', producer: 'Fromagerie' })

  await page.goto('/pantry/new')
  await page.getByLabel('Name').fill(name.toUpperCase())
  await expect(page.getByText('Is it one of these?')).toBeVisible()
  await page.getByRole('link', { name: item.name }).click()
  await expect(page).toHaveURL(new RegExp(`/pantry/${item.id}$`))
})

test('the pantry item form suggests the tags of the pantry', async ({ page, api, user: _ }) => {
  const other = await api.signUp('Eva')
  const known = uniqueTag('raw-milk')
  await api.createPantryItem(other, { name: uniqueTag('Brie'), category: 'cheese', tags: [known] })

  await page.goto('/pantry/new')
  await page.getByLabel('Name').fill(uniqueTag('Camembert'))
  const tags = page.getByLabel('Tags')
  await tags.fill(known.slice(0, -2))
  await expect(page.getByRole('option', { name: known })).toContainText('1 pantry item')
  await tags.press('ArrowDown')
  await tags.press('Enter')
  await expect(page.getByRole('button', { name: `Remove ${known}` })).toBeVisible()
  await page.getByLabel('Category').click()
  await page.getByRole('option', { name: 'Cheese' }).click()
  await page.getByRole('button', { name: 'Add pantry item' }).click()

  await expect(page.getByText('Pantry item added', { exact: true })).toBeVisible()
  await page.getByRole('list', { name: 'Tags' }).getByRole('link', { name: known }).click()
  await expect(page).toHaveURL(new RegExp(`/tags/${known}$`))
  await expect(page.getByText('2 pantry items')).toBeVisible()
})

test('the tag page shows the pantry items of a tag', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('aged')
  const restaurant = await api.createRestaurant(owner, { name: uniqueTag('Fromagerie'), tags: [tag] })
  const recipe = await api.createRecipe(owner, { name: uniqueTag('Fondue'), tags: [tag] })
  const item = await api.createPantryItem(owner, { name: uniqueTag('Gouda'), category: 'cheese', tags: [tag] })

  await page.goto(`/tags/${tag}`)
  await expect(page.getByText('1 restaurant, 1 recipe, and 1 pantry item')).toBeVisible()
  await expect(page.getByRole('heading', { level: 2 })).toHaveText(['Restaurants', 'Recipes', 'Pantry'])
  await expect(page.getByRole('heading', { name: restaurant.name, level: 3 })).toBeVisible()
  await expect(page.getByRole('heading', { name: recipe.name, level: 3 })).toBeVisible()
  await expect(page.getByRole('heading', { name: item.name, level: 3 })).toBeVisible()

  await page.getByRole('navigation', { name: 'Kind of the tagged things' }).getByRole('link', { name: 'Pantry' }).click()
  await expect(page).toHaveURL(/\?kind=pantry$/)
  await expect(page.getByRole('heading', { level: 2 })).toHaveText(['Pantry'])

  await page.getByRole('link', { name: 'Sort and filter the pantry items' }).click()
  await expect(page.getByText(`Tag: ${tag}`)).toBeVisible()
  await expect(page.getByRole('heading', { name: item.name, level: 2 })).toBeVisible()
})

test('the category chips filter the pantry', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('picnic')
  const wine = await api.createPantryItem(owner, { name: uniqueTag('Rose'), category: 'wine', tags: [tag] })
  const beer = await api.createPantryItem(owner, { name: uniqueTag('Pils'), category: 'beer', tags: [tag] })

  await page.goto(`/pantry?tags=${tag}`)
  await expect(page.getByRole('heading', { name: wine.name, level: 2 })).toBeVisible()
  await expect(page.getByRole('heading', { name: beer.name, level: 2 })).toBeVisible()
  await page.getByRole('navigation', { name: 'Category' }).getByRole('link', { name: 'Beer' }).click()
  await expect(page).toHaveURL(new RegExp(`/pantry\\?tags=${tag}&category=beer$`))
  await expect(page.getByRole('heading', { name: beer.name, level: 2 })).toBeVisible()
  await expect(page.getByRole('heading', { name: wine.name, level: 2 })).toHaveCount(0)
  await expect(page.getByText(`Tag: ${tag}`)).toBeVisible()
})

test('a user writes a tasting of a new pantry item from Add review', async ({ page, user: _ }) => {
  await page.goto('/reviews/new?kind=pantry')
  await expect(page.getByRole('heading', { name: 'What did you have?' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'I tasted' })).toHaveAttribute('aria-pressed', 'true')
  await page.getByRole('button', { name: 'Add a new pantry item' }).click()
  await expect(page).toHaveURL(/\/pantry\/new\?review=new$/)

  const name = uniqueTag('Dark 70')
  await page.getByLabel('Name').fill(name)
  await page.getByLabel('Category').click()
  await page.getByRole('option', { name: 'Chocolate' }).click()
  await page.getByRole('button', { name: 'Add pantry item' }).click()
  await expect(page).toHaveURL(/\/reviews\/new\?pantry_item=/)
  await expect(page.getByRole('main').getByText(name, { exact: true })).toBeVisible()
  await expect(page.getByLabel('Tasted on')).toBeVisible()

  for (const label of ['Taste', 'Aroma', 'Texture', 'Value']) {
    await page.getByRole('radiogroup', { name: label }).getByRole('radio', { name: '8 out of 10' }).click()
  }
  await page.getByLabel('Title').fill('Bitter and smooth')
  await page.getByRole('button', { name: 'Save review' }).click()
  await expect(page.getByText('Review saved')).toBeVisible()
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Every tasting' }).getByRole('listitem').filter({ hasText: 'Bitter and smooth' })).toBeVisible()

  await page.goto('/?kind=pantry')
  await expect(page.getByRole('navigation', { name: 'Show' }).getByRole('link', { name: 'Tasted' })).toHaveAttribute('aria-current', 'page')
  await expect(page.getByRole('link', { name: `${name}: Bitter and smooth` })).toBeVisible()
})
