import { uniqueTag } from '../api'
import { expect, signIn, test } from '../fixtures'

test('the header links the restaurants and the recipes', async ({ page }) => {
  await page.goto('/restaurants')
  const sections = page.getByRole('banner').getByRole('navigation', { name: 'Sections' })
  await expect(sections.getByRole('link', { name: 'Restaurants' })).toHaveAttribute('aria-current', 'page')
  await sections.getByRole('link', { name: 'Recipes' }).click()
  await expect(page).toHaveURL(/\/recipes$/)
  await expect(page.getByRole('heading', { name: 'Recipes', level: 1 })).toBeAttached()
  await expect(sections.getByRole('link', { name: 'Recipes' })).toHaveAttribute('aria-current', 'page')
})

test('a signed in user adds, edits, and deletes a recipe', async ({ page, user: _ }) => {
  const name = uniqueTag('Shakshuka')
  await page.goto('/recipes')
  await page.getByRole('link', { name: 'Add recipe' }).click()
  await expect(page.getByRole('heading', { name: 'Add a recipe' })).toBeVisible()

  await page.getByRole('button', { name: 'Add recipe' }).click()
  await expect(page.getByText('Enter at least one ingredient.')).toBeVisible()

  await page.getByLabel('Name').fill(name)
  await page.getByLabel('Ingredients').fill('6 eggs\n1 can of tomatoes\n\n1 onion')
  await page.getByLabel('Step 1', { exact: true }).fill('Cook the onion.')
  await page.getByRole('button', { name: 'Add step' }).click()
  await expect(page.getByLabel('Step 2', { exact: true })).toBeFocused()
  await page.getByLabel('Step 2', { exact: true }).fill('Add the tomatoes and the eggs.')
  await page.getByLabel('Total time in minutes').fill('30')
  await page.getByLabel('Servings').fill('3')
  await page.getByRole('button', { name: 'Add recipe' }).click()

  await expect(page.getByText('Recipe added', { exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Ingredients' }).getByRole('listitem')).toHaveText(['6 eggs', '1 can of tomatoes', '1 onion'])
  await expect(page.getByRole('region', { name: 'Steps' }).getByRole('listitem')).toHaveText(['Cook the onion.', 'Add the tomatoes and the eggs.'])
  await expect(page.getByText('30 min')).toBeVisible()
  await expect(page.getByText('3 servings')).toBeVisible()

  await page.getByRole('link', { name: 'Edit recipe' }).click()
  await page.getByRole('button', { name: 'Move step 2 up' }).click()
  await page.getByLabel('Servings').fill('')
  await page.getByRole('button', { name: 'Save recipe' }).click()
  await expect(page.getByText('Recipe saved', { exact: true })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Steps' }).getByRole('listitem')).toHaveText(['Add the tomatoes and the eggs.', 'Cook the onion.'])
  await expect(page.getByText('3 servings')).toHaveCount(0)

  await page.getByRole('button', { name: 'Delete recipe' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click()
  await expect(page.getByText('Recipe deleted', { exact: true })).toBeVisible()
  await expect(page).toHaveURL(/\/recipes$/)
})

test('another user changes each field of a recipe but cannot delete it', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const recipe = await api.createRecipe(owner, { name: 'Dal', servings: 2 })
  await api.createRecipeReview(owner, recipe.id, { taste: 8, ease: 8, instructions: 8, value: 8, title: 'Good' })
  const other = await api.signUp('Ana')
  await signIn(page, other)

  await page.goto(`/recipes/${recipe.id}`)
  await expect(page.getByRole('heading', { name: 'Dal', level: 1 })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Delete recipe' })).toHaveCount(0)
  await page.getByRole('link', { name: 'Edit recipe' }).click()

  await page.getByLabel('Name').fill('Tarka dal')
  await page.getByLabel('Ingredients').fill('1 cup of lentils')
  await page.getByLabel('Servings').fill('4')
  await page.getByRole('button', { name: 'Save recipe' }).click()

  await expect(page.getByText('Recipe saved', { exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Tarka dal', level: 1 })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Ingredients' }).getByRole('listitem')).toHaveText(['1 cup of lentils'])
  await expect(page.getByText('4 servings')).toBeVisible()
})

test('a second recipe with the same source opens the first one', async ({ page, api, user: _ }) => {
  const owner = await api.signUp('Joe')
  const source = `https://recipes.example/${uniqueTag('dal')}`
  const first = await api.createRecipe(owner, { name: 'Dal', source })

  await page.goto('/recipes/new')
  await page.getByLabel('Name').fill('My dal')
  await page.getByLabel('Ingredients').fill('Lentils')
  await page.getByLabel('Step 1', { exact: true }).fill('Boil.')
  await page.getByLabel('Source').fill(`${source.replace('recipes.example', 'RECIPES.example')}#top`)
  await page.getByRole('button', { name: 'Add recipe' }).click()
  await expect(page.getByText('This recipe is already here')).toBeVisible()
  await page.getByRole('link', { name: 'Open the recipe' }).click()
  await expect(page).toHaveURL(new RegExp(`/recipes/${first.id}$`))
})

test('a signed in user reviews a recipe', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const recipe = await api.createRecipe(owner, { name: uniqueTag('Hummus') })

  await page.goto(`/recipes/${recipe.id}`)
  await page.getByRole('link', { name: 'Write a review' }).first().click()
  await expect(page.getByRole('heading', { name: 'Write a review' })).toBeVisible()
  await expect(page.getByRole('radiogroup', { name: 'Ease' })).toHaveAccessibleDescription(
    'How easy the recipe is to make. A high score is easy. 1 means hard. 10 means easy.',
  )
  for (const [label, score] of [['Taste', 9], ['Ease', 8], ['Instructions', 8], ['Value', 8]] as const) {
    await page.getByRole('radiogroup', { name: label }).getByRole('radio', { name: `${score} out of 10` }).click()
  }
  await expect(page.getByRole('status').filter({ hasText: 'Overall' })).toHaveText('Overall8.3')
  await page.getByLabel('Title').fill('Smooth and quick')
  await page.getByRole('button', { name: 'Save review' }).click()

  await expect(page.getByText('Review saved')).toBeVisible()
  await expect(page.getByRole('region', { name: 'Average scores' }).getByRole('img', { name: 'Taste: 9 out of 10' })).toBeVisible()
  const card = page.getByRole('region', { name: 'Reviews' }).getByRole('listitem').filter({ hasText: 'Smooth and quick' })
  await expect(card).toContainText(user.name)
  await card.getByRole('link', { name: 'Smooth and quick' }).click()
  await expect(page.getByRole('heading', { name: 'Smooth and quick', level: 1 })).toBeVisible()
  await expect(page.getByRole('link', { name: recipe.name })).toHaveAttribute('href', `/recipes/${recipe.id}`)

  await page.goto('/my-reviews')
  await expect(page.getByRole('heading', { name: 'Smooth and quick' })).toHaveCount(0)
  await page.getByRole('navigation', { name: 'Reviews of' }).getByRole('link', { name: 'Recipes' }).click()
  await expect(page).toHaveURL(/\/my-reviews\?kind=recipes$/)
  await expect(page.getByRole('heading', { name: 'Smooth and quick' })).toBeVisible()
  await expect(page.getByRole('link', { name: recipe.name })).toBeVisible()

  await page.goto(`/recipes/${recipe.id}/reviews/new`)
  await page.getByLabel('Cooked on').fill('2024-02-10')
  for (const label of ['Taste', 'Ease', 'Instructions', 'Value']) {
    await page.getByRole('radiogroup', { name: label }).getByRole('radio', { name: '6 out of 10' }).click()
  }
  await page.getByLabel('Title').fill('Too much garlic')
  await page.getByRole('button', { name: 'Save review' }).click()
  await expect(page.getByText('Review saved')).toBeVisible()
  const titles = page.getByRole('region', { name: 'Reviews' }).getByRole('heading', { level: 3 })
  await expect(titles).toHaveText(['Smooth and quick', 'Too much garlic'])
  await expect(page.getByRole('region', { name: 'Reviews' }).getByRole('listitem').last()).toContainText('Cooked on 10 Feb 2024')
})

test('the recipe page shows the photos of its reviews', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const recipe = await api.createRecipe(owner, { name: uniqueTag('Bread') })
  const review = await api.createRecipeReview(owner, recipe.id, { taste: 8, ease: 6, instructions: 7, value: 9, title: 'Crusty' })
  await api.addRecipePhoto(owner, review, 'menu.png')

  await page.goto(`/recipes/${recipe.id}`)
  await expect(page.getByRole('button', { name: 'Open the cover photo' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Photos' })).toBeVisible()
  await page.getByRole('button', { name: 'Open the cover photo' }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog.getByRole('link', { name: 'review by Joe' })).toHaveAttribute('href', `/recipes/${recipe.id}/reviews/${review.id}`)
})

test('the recipes page searches, filters, and sorts', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('dinner')
  await api.createRecipe(owner, { name: 'Quick dal', tags: [tag], ingredients: ['1 cup of lentils'], totalTime: 20 })
  await api.createRecipe(owner, { name: 'Slow stew', tags: [tag], ingredients: ['1 kg of beef'], totalTime: 180 })
  await api.createRecipe(owner, { name: 'Salad', tags: [tag], ingredients: ['1 cucumber'] })

  const names = page.getByRole('heading', { level: 2 })
  await page.goto(`/recipes?tags=${tag}&sort=total_time`)
  await expect(names).toHaveText(['Quick dal', 'Slow stew', 'Salad'])

  await page.goto(`/recipes?tags=${tag}&total_time=30`)
  await expect(names).toHaveText(['Quick dal'])
  await expect(page.getByText('Total time: 30 min or less')).toBeVisible()

  await page.goto(`/recipes?tags=${tag}&search=beef`)
  await expect(names).toHaveText(['Slow stew'])
})

test('a recipe goes into the favorites and a list', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const recipe = await api.createRecipe(owner, { name: uniqueTag('Pancakes') })
  const list = await api.createList(user, uniqueTag('Brunch'))

  await page.goto(`/recipes/${recipe.id}`)
  await page.getByRole('button', { name: `Favorite ${recipe.name}` }).click()
  await expect(page.getByRole('button', { name: `Favorite ${recipe.name}` })).toHaveAttribute('aria-pressed', 'true')
  await page.getByRole('button', { name: 'Save to list' }).click()
  await page.getByRole('checkbox', { name: list.name }).check()
  await expect(page.getByRole('checkbox', { name: list.name })).toBeChecked()
  await page.keyboard.press('Escape')

  await page.goto(`/lists/${list.id}`)
  await expect(page.getByText('1 recipe')).toBeVisible()
  await page.getByRole('navigation', { name: 'Items of the list' }).getByRole('link', { name: 'Recipes' }).click()
  await expect(page.getByRole('heading', { name: recipe.name })).toBeVisible()
  await page.getByRole('button', { name: `Remove ${recipe.name} from the list` }).click()
  await expect(page.getByText('No recipes yet')).toBeVisible()

  await page.goto(`/reviewers/${user.auth.user.id}`)
  await expect(page.getByRole('link', { name: /Favorites/ })).toContainText('1 recipe')
})

test('the header fits on a 320px screen', async ({ page, user: _ }) => {
  await page.setViewportSize({ width: 320, height: 640 })
  await page.goto('/recipes')
  await expect(page.getByRole('banner').getByRole('link', { name: 'Recipes' })).toBeVisible()
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
  expect(overflow).toBe(0)
  const menu = await page.getByRole('banner').getByRole('button').last().boundingBox()
  expect(menu && menu.x + menu.width).toBeLessThanOrEqual(320)
})
