import type { Page } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { expect, signIn, test } from '../fixtures'

const wcag = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa']

async function expectNoViolations(page: Page) {
  const results = await new AxeBuilder({ page }).withTags(wcag).analyze()
  const violations = results.violations.map(
    (v) => `${v.id} on ${page.url()}: ${v.nodes.map((n) => `${n.target.join(' ')} (${n.failureSummary?.split('\n').slice(1, 2).join('').trim()})`).join('; ')}`,
  )
  expect(violations).toEqual([])
}

for (const colorScheme of ['light', 'dark'] as const) {
  test.describe(`in the ${colorScheme} theme`, () => {
    test.use({ colorScheme })

    test('the main screens have no axe violations', async ({ page, api }) => {
      const owner = await api.signUp('Joe')
      const restaurant = await api.createRestaurant(owner, {
        name: "Luigi's",
        description: 'A family-run Italian restaurant.',
        tags: ['italian'],
      })
      const reviewer = await api.signUp('Ana')
      await api.createReview(reviewer, restaurant.id, { space: 4, service: 4, food: 4, price: 4, cleanliness: 4, title: 'Good' })
      await api.createReview(owner, restaurant.id, { space: 7, service: 9, food: 8, price: 6, cleanliness: 6, title: 'Great food' })

      await page.goto('/restaurants')
      await expect(page.getByRole('heading', { name: 'Restaurants', level: 1 })).toBeVisible()
      await expect(page.getByRole('status')).not.toHaveText('Loading restaurants')
      await expectNoViolations(page)

      await page.goto(`/restaurants/${restaurant.id}`)
      await expect(page.getByRole('heading', { name: 'Good' })).toBeVisible()
      await expectNoViolations(page)

      await page.goto('/sign-in')
      await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()
      await expectNoViolations(page)

      await page.goto('/nothing-here')
      await expect(page.getByRole('heading', { name: 'We cannot find this page' })).toBeVisible()
      await expectNoViolations(page)

      await signIn(page, reviewer)
      await page.goto(`/restaurants/${restaurant.id}/reviews/new`)
      await expect(page.getByRole('heading', { name: 'Edit review' })).toBeVisible()
      await expectNoViolations(page)

      const photoReview = await api.createReview(reviewer, (await api.createRestaurant(owner, { name: 'Photo Spot' })).id, {
        space: 6, service: 6, food: 6, price: 6, cleanliness: 6, title: 'With photos',
      })
      await api.addPhoto(reviewer, photoReview, 'menu.png', 'The menu')
      await api.addPhoto(reviewer, photoReview, 'gps-rotated.jpg')
      await page.goto(`/restaurants/${photoReview.restaurant}/reviews/${photoReview.id}`)
      await expect(page.getByRole('button', { name: 'The menu' })).toBeVisible()
      await expectNoViolations(page)
      await page.getByRole('button', { name: 'The menu' }).click()
      await expect(page.getByRole('dialog').getByRole('heading', { name: 'Photo 1 of 2' })).toBeVisible()
      // The open animation changes the colors. axe runs after it.
      await page.waitForFunction(() => document.getAnimations().every((a) => a.playState !== 'running'))
      await expectNoViolations(page)
      await page.keyboard.press('Escape')

      await page.goto(`/restaurants/${photoReview.restaurant}`)
      await expect(page.getByRole('list', { name: 'Photos of Photo Spot' })).toBeVisible()
      await expectNoViolations(page)

      await page.goto(`/restaurants/${photoReview.restaurant}/reviews/${photoReview.id}/edit`)
      await expect(page.getByRole('heading', { name: 'Photos' })).toBeVisible()
      await expectNoViolations(page)

      await page.goto(`/restaurants?view=map&search=${encodeURIComponent("Luigi's")}`)
      await expect(page.getByRole('region', { name: 'Map of the restaurants' })).toBeVisible()
      await expect(page.getByRole('status')).not.toHaveText('Loading restaurants')
      await expectNoViolations(page)

      await page.goto('/restaurants/new')
      await page.getByLabel('Location').fill(`axe ${colorScheme}`)
      await expect(page.getByRole('list', { name: 'Places' })).toBeVisible()
      await expectNoViolations(page)
      await page.getByRole('button', { name: 'Enter the location by hand' }).click()
      await expect(page.getByLabel('Latitude')).toBeVisible()
      await expectNoViolations(page)

      await page.goto('/my-reviews')
      await expect(page.getByRole('heading', { name: 'My reviews', level: 1 })).toBeVisible()
      await expect(page.getByRole('status')).not.toHaveText('Loading reviews')
      await expectNoViolations(page)

      await page.goto(`/reviewers/${owner.auth.user.id}`)
      await expect(page.getByRole('heading', { name: 'Joe', level: 1 })).toBeVisible()
      await expect(page.getByRole('status')).not.toHaveText('Loading reviews')
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
