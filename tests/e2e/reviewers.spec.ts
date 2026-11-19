import { expect, test } from '../fixtures'

test('a review card shows the overall score and links the reviewer page', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const ana = await api.signUp('Ana')
  const first = await api.createRestaurant(owner, { name: 'Overall Kitchen' })
  const second = await api.createRestaurant(owner, { name: 'Overall Grill' })
  await api.createReview(ana, first.id, { space: 7, service: 9, food: 8, price: 6, cleanliness: 8, title: 'Ana likes it' })
  await api.createReview(ana, second.id, { space: 2, service: 2, food: 2, price: 2, cleanliness: 2, title: 'Ana does not' })
  await api.createReview(owner, first.id, { space: 5, service: 5, food: 5, price: 5, cleanliness: 5, title: 'Joe at his place' })

  await page.goto(`/restaurants/${first.id}`)
  const card = page.getByRole('listitem').filter({ hasText: 'Ana likes it' })
  await expect(card.getByRole('img', { name: 'Overall: 7.6 out of 10' })).toBeVisible()
  await card.getByRole('link', { name: 'Ana', exact: true }).click()

  await expect(page).toHaveURL(new RegExp(`/reviewers/${ana.auth.user.id}$`))
  await expect(page.getByRole('heading', { name: 'Ana', level: 1 })).toBeVisible()
  await expect(page.getByRole('img', { name: 'Average restaurant score: 4.8 out of 10' })).toBeVisible()
  await expect(page.getByText('average of 2 restaurant reviews', { exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Restaurants added' })).toHaveCount(0)
  const titles = page.getByRole('heading', { level: 3 })
  await expect(titles).toHaveText(['Ana does not', 'Ana likes it'])
  await expect(page.locator('body')).not.toContainText(ana.email)
  await expect(page.getByRole('main').getByRole('link', { name: 'Ana', exact: true })).toHaveCount(0)

  await page.goto(`/reviewers/${ana.auth.user.id}?sort=-overall`)
  await expect(titles).toHaveText(['Ana likes it', 'Ana does not'])
  await page.goto(`/reviewers/${ana.auth.user.id}?overall=5`)
  await expect(titles).toHaveText(['Ana likes it'])
  await expect(page.getByText('Overall: 5 or more')).toBeVisible()
})

test('a reviewer page names a reviewer without reviews', async ({ page, api }) => {
  const sam = await api.signUp('Sam')
  await page.goto(`/reviewers/${sam.auth.user.id}`)
  await expect(page.getByRole('heading', { name: 'Sam', level: 1 })).toBeVisible()
  await expect(page.getByText('Sam has not reviewed a restaurant yet.')).toBeVisible()
})

test('a reviewer page lists the restaurants or the recipes that the reviewer added above the reviews', async ({ page, api }) => {
  const joe = await api.signUp('Joe')
  const ana = await api.signUp('Ana')
  const first = await api.createRestaurant(joe, { name: 'Joe Kitchen' })
  await Promise.all([
    ...['Joe Grill', 'Joe Diner', 'Joe Bar'].map((name) => api.createRestaurant(joe, { name })),
    api.createRecipe(joe, { name: 'Joe Dal' }),
  ])
  await api.createRestaurant(ana, { name: 'Ana Bistro' })
  await api.createReview(ana, first.id, { space: 7, service: 9, food: 8, price: 6, cleanliness: 8, title: 'Ana likes it' })

  await page.goto(`/reviewers/${joe.auth.user.id}`)
  await expect(page.getByText('No restaurant reviews yet · 4 restaurants added', { exact: true })).toBeVisible()
  const headings = page.getByRole('heading', { level: 2 })
  await expect(headings).toHaveText(['Lists', 'Restaurants added', 'Reviews'])
  const section = page.getByRole('region', { name: 'Restaurants added' })
  const cards = section.getByRole('heading', { level: 3 })
  await expect(cards).toHaveCount(3)
  await section.getByRole('button', { name: 'Load more restaurants' }).click()
  await expect(cards).toHaveCount(4)
  await expect(section.getByRole('button', { name: 'Load more restaurants' })).toHaveCount(0)
  await expect(section.getByRole('link', { name: 'Joe Kitchen' })).toHaveAttribute('href', `/restaurants/${first.id}`)

  await page.getByRole('navigation', { name: 'Restaurants or recipes' }).getByRole('link', { name: 'Recipes' }).click()
  await expect(headings).toHaveText(['Lists', 'Recipes added', 'Reviews'])
  await expect(page.getByRole('region', { name: 'Recipes added' }).getByRole('link', { name: 'Joe Dal' })).toBeVisible()
})

for (const id of ['aaaaaaaaaaaaaaa', 'BAD']) {
  test(`the reviewer ID ${id} has the not found page`, async ({ page }) => {
    await page.goto(`/reviewers/${id}`)
    await expect(page.getByRole('heading', { name: 'We cannot find this page' })).toBeVisible()
  })
}
