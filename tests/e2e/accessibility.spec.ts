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
      await api.createReview(reviewer, restaurant.id, { kind: 'basic', rating: 4, title: 'Good', tags: ['pasta'] })
      await api.createReview(owner, restaurant.id, {
        kind: 'joc_de_cartes', space: 7, service: 9, food: 8, price: 6, title: 'Joc',
      })

      await page.goto('/restaurants')
      await expect(page.getByRole('heading', { name: 'Restaurants', level: 1 })).toBeVisible()
      await expect(page.getByRole('status')).not.toHaveText('Loading restaurants')
      await expectNoViolations(page)

      await page.goto(`/restaurants/${restaurant.id}`)
      await expect(page.getByRole('heading', { name: 'Good' })).toBeVisible()
      await expectNoViolations(page)

      await page.goto('/reviews')
      await expect(page.getByRole('heading', { name: 'Reviews', level: 1 })).toBeVisible()
      await expect(page.getByRole('status')).not.toHaveText('Loading reviews')
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
