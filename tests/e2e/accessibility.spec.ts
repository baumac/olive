import type { Page } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import type { TestApi, TestUser } from '../api'
import { uniqueTag } from '../api'
import { expect, test } from '../fixtures'

const wcag = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa']

async function expectNoViolations(page: Page) {
  const results = await new AxeBuilder({ page }).withTags(wcag).analyze()
  const violations = results.violations.map(
    (v) => `${v.id} on ${page.url()}: ${v.nodes.map((n) => `${n.target.join(' ')} (${n.failureSummary?.split('\n').slice(1, 2).join('').trim()})`).join('; ')}`,
  )
  expect(violations).toEqual([])
}

// The open animation changes the colors. axe runs after it.
async function waitForAnimations(page: Page) {
  await page.waitForFunction(() => document.getAnimations().every((a) => a.playState !== 'running'))
}

async function restaurantWithReview(api: TestApi, owner: TestUser) {
  const restaurant = await api.createRestaurant(owner, {
    name: uniqueTag("Luigi's"),
    description: 'A family-run Italian restaurant.',
    tags: ['italian'],
  })
  await api.createReview(owner, restaurant.id, { space: 7, service: 9, food: 8, price: 6, title: 'Great food' })
  return restaurant
}

async function pantryItemWithReview(api: TestApi, owner: TestUser) {
  const item = await api.createPantryItem(owner, {
    name: uniqueTag('Kabinett'),
    category: 'wine',
    producer: 'Weingut Example',
    origin: 'Mosel, Germany',
    style: 'Riesling',
    tags: ['white'],
  })
  await api.createPantryItemReview(owner, item.id, { taste: 9, aroma: 8, texture: 7, value: 9, title: 'Fresh and light' })
  return item
}

async function recipeWithReview(api: TestApi, owner: TestUser) {
  const recipe = await api.createRecipe(owner, {
    name: uniqueTag('Shakshuka'),
    description: 'Eggs in a spicy sauce.',
    tags: ['breakfast'],
    ingredients: ['6 eggs', '1 can of tomatoes'],
    steps: ['Cook the sauce.', 'Add the eggs.'],
    totalTime: 30,
  })
  await api.createRecipeReview(owner, recipe.id, { taste: 9, ease: 8, instructions: 8, value: 8, title: 'Quick and good' })
  return recipe
}

async function restaurantWithTwoReviews(api: TestApi) {
  const restaurant = await restaurantWithReview(api, await api.signUp('Joe'))
  const reviewer = await api.signUp('Ana')
  await api.createReview(reviewer, restaurant.id, { space: 4, service: 4, food: 4, price: 4, title: 'Good' })
  return restaurant
}

async function reviewWithPhotos(api: TestApi, user: TestUser) {
  const restaurant = await api.createRestaurant(user, { name: uniqueTag('Photo Spot') })
  const review = await api.createReview(user, restaurant.id, {
    space: 6, service: 6, food: 6, price: 6, title: 'With photos',
  })
  await api.addPhoto(user, review, 'menu.png', 'The menu')
  await api.addPhoto(user, review, 'gps-rotated.jpg')
  return { restaurant, review }
}

// Each screen has its own test. The tests then run in parallel.
for (const colorScheme of ['light', 'dark'] as const) {
  test.describe(`in the ${colorScheme} theme`, () => {
    test.use({ colorScheme })

    test('the password screens have no axe violations', async ({ page }) => {
      await page.goto('/forgot-password')
      await expect(page.getByRole('heading', { name: 'Forgot your password?' })).toBeVisible()
      await expectNoViolations(page)

      await page.goto('/reset-password#token=not-a-token')
      await expect(page.getByRole('heading', { name: 'Set a new password' })).toBeVisible()
      await expectNoViolations(page)
    })

    // The user fixture signs the page in. Its value is not used.
    test('the account screens have no axe violations', async ({ page, user: _ }) => {
      await page.goto('/account')
      await expect(page.getByRole('heading', { name: 'Account', level: 1 })).toBeVisible()
      await expectNoViolations(page)
      await page.getByRole('button', { name: 'Delete account' }).click()
      await expect(page.getByRole('alertdialog')).toBeVisible()
      await waitForAnimations(page)
      await expectNoViolations(page)
      await page.keyboard.press('Escape')
    })

    test('the restaurant list has no axe violations', async ({ page, api }) => {
      await restaurantWithTwoReviews(api)
      await page.goto('/restaurants')
      await expect(page.getByRole('heading', { name: 'Restaurants', level: 1 })).toBeVisible()
      await expect(page.getByRole('status')).not.toHaveText('Loading restaurants')
      await expectNoViolations(page)
    })

    test('the restaurant page has no axe violations', async ({ page, api }) => {
      const restaurant = await restaurantWithTwoReviews(api)
      await page.goto(`/restaurants/${restaurant.id}`)
      await expect(page.getByRole('heading', { name: 'Good' })).toBeVisible()
      await expectNoViolations(page)
    })

    test('the sign in and not found screens have no axe violations', async ({ page }) => {
      await page.goto('/sign-in')
      await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()
      await expectNoViolations(page)

      await page.goto('/nothing-here')
      await expect(page.getByRole('heading', { name: 'We cannot find this page' })).toBeVisible()
      await expectNoViolations(page)
    })

    test('the review form has no axe violations', async ({ page, api, user }) => {
      const restaurant = await api.createRestaurant(user, { name: uniqueTag('Form Place') })
      const review = await api.createReview(user, restaurant.id, { space: 7, service: 9, food: 8, price: 6, title: 'Great food' })
      await page.goto(`/restaurants/${restaurant.id}/reviews/new`)
      await expect(page.getByRole('heading', { name: 'Add a visit' })).toBeVisible()
      await expectNoViolations(page)
      await page.getByRole('button', { name: 'Save review' }).click()
      await expect(page.getByText('Select a space score.')).toBeVisible()
      await expectNoViolations(page)

      await page.goto(`/restaurants/${restaurant.id}/reviews/${review.id}/edit`)
      await expect(page.getByRole('heading', { name: 'Edit review' })).toBeVisible()
      await expectNoViolations(page)
    })

    test('the review photos and the photo dialog have no axe violations', async ({ page, api, user }) => {
      const { review } = await reviewWithPhotos(api, user)
      await page.goto(`/restaurants/${review.restaurant}/reviews/${review.id}`)
      await expect(page.getByRole('button', { name: 'The menu' })).toBeVisible()
      await expectNoViolations(page)
      await page.getByRole('button', { name: 'The menu' }).click()
      await expect(page.getByRole('dialog').getByRole('heading', { name: 'Photo 1 of 2' })).toBeVisible()
      await waitForAnimations(page)
      await expectNoViolations(page)
      await page.keyboard.press('Escape')
    })

    test('the gallery and the photo editor have no axe violations', async ({ page, api, user }) => {
      const { restaurant, review } = await reviewWithPhotos(api, user)
      await page.goto(`/restaurants/${restaurant.id}`)
      await expect(page.getByRole('list', { name: `Photos of ${restaurant.name}` })).toBeVisible()
      await expectNoViolations(page)

      await page.goto(`/restaurants/${restaurant.id}/reviews/${review.id}/edit`)
      await expect(page.getByRole('heading', { name: 'Photos' })).toBeVisible()
      await expectNoViolations(page)
    })

    test('the map view has no axe violations', async ({ page, api, user }) => {
      const restaurant = await restaurantWithReview(api, user)
      await page.goto(`/restaurants?view=map&search=${encodeURIComponent(restaurant.name)}`)
      await expect(page.getByRole('region', { name: 'Map of the restaurants' })).toBeVisible()
      await expect(page.getByRole('status')).not.toHaveText('Loading restaurants')
      await expectNoViolations(page)
    })

    test('the restaurant form has no axe violations', async ({ page, user: _ }) => {
      await page.goto('/restaurants/new')
      await page.getByLabel('Location').fill(`axe ${colorScheme}`)
      await expect(page.getByRole('list', { name: 'Places' })).toBeVisible()
      await expectNoViolations(page)
      await page.getByRole('button', { name: 'Enter the location by hand' }).click()
      await expect(page.getByLabel('Latitude')).toBeVisible()
      await expectNoViolations(page)
    })

    test('the tag picker and the tag page have no axe violations', async ({ page, api, user }) => {
      const tag = uniqueTag('axe')
      await api.createRestaurant(user, { name: uniqueTag('Tagged'), tags: [tag] })
      await page.goto('/restaurants/new')
      await page.getByLabel('Tags').fill(tag)
      await expect(page.getByRole('option', { name: tag })).toBeVisible()
      await expectNoViolations(page)
      await page.goto(`/tags/${encodeURIComponent(tag)}`)
      await expect(page.getByRole('heading', { name: tag, level: 1 })).toBeVisible()
      await expect(page.getByRole('heading', { level: 3 })).toHaveCount(1)
      await expectNoViolations(page)
    })

    test('the journal and the table have no axe violations', async ({ page, api, user }) => {
      await restaurantWithReview(api, user)
      await page.goto('/my-reviews')
      await expect(page.getByRole('heading', { name: user.name, level: 1 })).toBeVisible()
      await expect(page.getByRole('heading', { name: 'Your journal' })).toBeVisible()
      await expect(page.getByLabel('Loading reviews')).toHaveCount(0)
      await expectNoViolations(page)

      await page.goto('/')
      await expect(page.getByRole('heading', { name: 'On the table', level: 1 })).toBeVisible()
      await expect(page.getByRole('status')).not.toHaveText('Loading entries')
      await expectNoViolations(page)

      await page.goto('/reviews/new')
      await expect(page.getByRole('heading', { name: 'What did you have?' })).toBeVisible()
      await expectNoViolations(page)
    })

    test('the recipe list has no axe violations', async ({ page, api }) => {
      await recipeWithReview(api, await api.signUp('Joe'))
      await page.goto('/recipes')
      await expect(page.getByRole('heading', { name: 'Recipes', level: 1 })).toBeAttached()
      await expect(page.getByRole('status')).not.toHaveText('Loading recipes')
      await expectNoViolations(page)
    })

    test('the recipe page has no axe violations', async ({ page, api }) => {
      const recipe = await recipeWithReview(api, await api.signUp('Joe'))
      await page.goto(`/recipes/${recipe.id}`)
      await expect(page.getByRole('heading', { name: 'Quick and good' })).toBeVisible()
      await expectNoViolations(page)
    })

    test('the recipe form has no axe violations', async ({ page, user: _ }) => {
      await page.goto('/recipes/new')
      await page.getByRole('button', { name: 'Add recipe' }).click()
      await expect(page.getByText('Enter at least one ingredient.')).toBeVisible()
      await page.getByRole('button', { name: 'Add step' }).click()
      // The first remove button fades in from the disabled opacity.
      await expect(page.getByRole('button', { name: 'Remove step 1' })).toBeEnabled()
      await waitForAnimations(page)
      await expectNoViolations(page)
    })

    test('the pantry has no axe violations', async ({ page, api }) => {
      await pantryItemWithReview(api, await api.signUp('Joe'))
      await page.goto('/pantry?category=wine')
      await expect(page.getByRole('heading', { name: 'Pantry', level: 1 })).toBeAttached()
      await expect(page.getByRole('status')).not.toHaveText('Loading pantry items')
      await expectNoViolations(page)
    })

    test('the pantry item page has no axe violations', async ({ page, api }) => {
      const item = await pantryItemWithReview(api, await api.signUp('Joe'))
      await page.goto(`/pantry/${item.id}`)
      await expect(page.getByRole('heading', { name: 'Fresh and light' })).toBeVisible()
      await expectNoViolations(page)
    })

    test('the pantry item form has no axe violations', async ({ page, user: _ }) => {
      await page.goto('/pantry/new')
      await page.getByRole('button', { name: 'Add pantry item' }).click()
      await expect(page.getByText('Select a category.')).toBeVisible()
      await waitForAnimations(page)
      await expectNoViolations(page)
    })

    test('the reviewer page has no axe violations', async ({ page, api, user: _ }) => {
      const owner = await api.signUp('Joe')
      await restaurantWithReview(api, owner)
      await page.goto(`/reviewers/${owner.auth.user.id}`)
      await expect(page.getByRole('heading', { name: 'Joe', level: 1 })).toBeVisible()
      await expect(page.getByLabel('Loading reviews')).toHaveCount(0)
      await expectNoViolations(page)
    })
  })
}

test('the skip link is the first stop of the keyboard', async ({ page }) => {
  await page.goto('/restaurants')
  await page.keyboard.press('Tab')
  const skip = page.getByRole('link', { name: 'Skip to content' })
  await expect(skip).toBeFocused()
  await expect(skip).toBeVisible()
})

test('the sign in form has a keyboard order', async ({ page }) => {
  await page.goto('/sign-in')
  await expect(page.getByLabel('Email')).toBeFocused()
  await page.keyboard.press('Tab')
  await expect(page.getByLabel('Password', { exact: true })).toBeFocused()
  await page.keyboard.press('Tab')
  await expect(page.getByRole('button', { name: 'Show password' })).toBeFocused()
  await page.keyboard.press('Tab')
  await expect(page.getByRole('link', { name: 'Forgot your password?' })).toBeFocused()
  await page.keyboard.press('Tab')
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeFocused()
})

test('Escape closes the delete dialog', async ({ page, api, user }) => {
  const restaurant = await api.createRestaurant(user, { name: 'Close me' })
  await page.goto(`/restaurants/${restaurant.id}`)
  await page.getByRole('button', { name: 'Delete restaurant' }).click()
  await expect(page.getByRole('alertdialog')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Cancel' })).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('alertdialog')).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Delete restaurant' })).toBeFocused()
})
