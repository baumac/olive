import { uniqueTag } from '../api'
import { expect, test } from '../fixtures'

test('a signed in user writes a review', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Can Sole' })

  await page.goto(`/restaurants/${restaurant.id}/reviews/new`)
  await expect(page.getByRole('tab')).toHaveCount(0)
  await page.getByRole('button', { name: 'Save review' }).click()
  await expect(page.getByText('Select a space score.')).toBeVisible()
  await expect(page.getByRole('radiogroup', { name: 'Value' })).toHaveAccessibleDescription(
    'Value for money, and whether the final bill matches the quality provided. 1 means poor value. 10 means great value. Select a value score.',
  )

  for (const [label, score] of [['Space', 7], ['Service', 9], ['Food', 8], ['Value', 6]] as const) {
    await page.getByRole('radiogroup', { name: label }).getByRole('radio', { name: `${score} out of 10` }).click()
  }
  await expect(page.getByRole('status').filter({ hasText: 'Overall' })).toHaveText('Overall7.5')
  await page.getByLabel('Visited on').fill('2024-03-01')
  await page.getByLabel('Title').fill('Great food, a bit expensive')
  await page.getByLabel('Feedback').fill('Friendly staff.\nGreat pasta.')
  await expect(page.getByLabel('Tags')).toHaveCount(0)
  await page.getByRole('button', { name: 'Save review' }).click()

  await expect(page.getByText('Review saved')).toBeVisible()
  const card = page.getByRole('region', { name: 'Reviews' }).getByRole('listitem').filter({ hasText: 'Great food, a bit expensive' })
  await expect(card.getByRole('img', { name: 'Overall: 7.5 out of 10' })).toBeVisible()
  await expect(card.getByRole('img')).toHaveCount(1)
  await card.getByRole('button', { name: 'Show scores of Great food, a bit expensive' }).click()
  await expect(card.getByRole('img', { name: 'Value: 6 out of 10' })).toBeVisible()
  await card.getByRole('button', { name: 'Hide scores of Great food, a bit expensive' }).click()
  await expect(card.getByRole('img')).toHaveCount(1)
  await expect(card).toContainText(user.name)
  await expect(card).toContainText('Visited on 1 Mar 2024')
  await expect(page.getByRole('link', { name: 'Write a review' })).toBeVisible()
})

test('a user writes a review of each visit', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Bao House' })
  await api.createReview(user, restaurant.id, { date: '2024-01-15', space: 5, service: 5, food: 5, price: 5, title: 'First visit' })

  await page.goto(`/restaurants/${restaurant.id}`)
  await page.getByRole('link', { name: 'Write a review' }).click()
  await expect(page).toHaveURL(new RegExp(`/restaurants/${restaurant.id}/reviews/new$`))
  await expect(page.getByLabel('Title')).toHaveValue('')
  await page.getByLabel('Visited on').fill('2024-06-01')
  for (const label of ['Space', 'Service', 'Food', 'Value']) {
    await page.getByRole('radiogroup', { name: label }).getByRole('radio', { name: '9 out of 10' }).click()
  }
  await page.getByLabel('Title').fill('Second visit')
  await page.getByRole('button', { name: 'Save review' }).click()

  await expect(page.getByText('Review saved')).toBeVisible()
  const titles = page.getByRole('region', { name: 'Reviews' }).getByRole('heading', { level: 3 })
  await expect(titles).toHaveText(['Second visit', 'First visit'])
  await expect(page.getByRole('region', { name: 'Reviews' }).getByRole('listitem').first()).toContainText('Visited on 1 Jun 2024')
  await expect(page.getByText('out of 10 · 2 reviews')).toBeVisible()

  await page.goto(`/restaurants/${restaurant.id}?sort=date`)
  await expect(titles).toHaveText(['First visit', 'Second visit'])
})

test('filters and score sorts select and order reviews', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Mixed' })
  const ana = await api.signUp('Ana')
  const bo = await api.signUp('Bo')
  const cy = await api.signUp('Cy')
  await api.createReview(ana, restaurant.id, { space: 2, service: 9, food: 2, price: 9, title: 'Poor food' })
  await api.createReview(bo, restaurant.id, { space: 5, service: 5, food: 10, price: 1, title: 'Best food' })
  await api.createReview(cy, restaurant.id, { space: 7, service: 9, food: 8, price: 6, title: 'Good food' })

  const titles = page.getByRole('heading', { level: 3 })

  await page.goto(`/restaurants/${restaurant.id}`)
  await expect(titles).toHaveCount(3)
  await expect(page.getByRole('searchbox', { name: 'Search reviews' })).toHaveCount(0)

  await page.goto(`/restaurants/${restaurant.id}?sort=-overall`)
  await expect(titles).toHaveText(['Good food', 'Poor food', 'Best food'])

  await page.goto(`/restaurants/${restaurant.id}?sort=overall`)
  await expect(titles).toHaveText(['Best food', 'Poor food', 'Good food'])

  // The sort control does not have this sort. A link with it still works.
  await page.goto(`/restaurants/${restaurant.id}?sort=-food`)
  await expect(titles).toHaveText(['Best food', 'Good food', 'Poor food'])

  await page.goto(`/restaurants/${restaurant.id}?food=8`)
  await expect(titles).toHaveText(['Good food', 'Best food'])
  await expect(page.getByText('Food: 8 or more')).toBeVisible()
  await page.getByRole('button', { name: 'Remove filter food 8 or more' }).click()
  await expect(titles).toHaveCount(3)
})

test('the creator edits and deletes a review', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Edit me' })
  const review = await api.createReview(user, restaurant.id, { space: 3, service: 3, food: 3, price: 3, title: 'Okay' })

  await page.goto(`/restaurants/${restaurant.id}/reviews/${review.id}`)
  await page.getByRole('link', { name: 'Edit review' }).click()
  await expect(page.getByRole('radiogroup', { name: 'Food' }).getByRole('radio', { name: '3 out of 10' })).toBeChecked()
  await page.getByRole('radiogroup', { name: 'Food' }).getByRole('radio', { name: '9 out of 10' }).click()
  await page.getByRole('button', { name: 'Save review' }).click()
  await expect(page.getByRole('region', { name: 'Reviews' }).getByRole('img', { name: 'Overall: 4.5 out of 10' })).toBeVisible()
  await expect(page.getByText('edited')).toBeVisible()

  await page.getByRole('link', { name: 'Okay' }).click()
  await page.getByRole('button', { name: 'Delete review' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete review' }).click()
  await expect(page.getByText('Review deleted')).toBeVisible()
  await expect(page.getByText('No reviews yet')).toBeVisible()
})

test('my reviews lists the reviews of the user at all restaurants', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const first = await api.createRestaurant(owner, { name: 'First Place' })
  const second = await api.createRestaurant(owner, { name: 'Second Place' })
  await api.createReview(user, first.id, { date: '2024-01-01', space: 4, service: 4, food: 4, price: 4, title: 'Mine at first' })
  await api.createReview(user, second.id, { date: '2024-02-01', space: 2, service: 2, food: 2, price: 2, title: 'Mine at second' })
  await api.createReview(owner, first.id, { space: 5, service: 5, food: 5, price: 5, title: 'Not mine' })

  await page.goto('/restaurants')
  await page.getByRole('button', { name: `AN, account menu for ${user.name}` }).click()
  await page.getByRole('menuitem', { name: 'My reviews' }).click()

  await expect(page.getByRole('heading', { name: 'My reviews', level: 1 })).toBeVisible()
  await expect(page).toHaveURL(/\/my-reviews$/)
  await expect(page.getByRole('heading', { level: 3 })).toHaveText(['Mine at second', 'Mine at first'])
  await page.getByRole('link', { name: 'First Place' }).click()
  await expect(page.getByRole('heading', { name: 'First Place', level: 1 })).toBeVisible()
})

test('the header has the home link and the sections', async ({ page }) => {
  await page.goto('/restaurants')
  const header = page.getByRole('banner')
  await expect(header.getByRole('link', { name: 'Sponga47 Reviews, home' })).toBeVisible()
  await expect(header.getByRole('navigation')).toHaveCount(1)
  await expect(header.getByRole('navigation', { name: 'Sections' }).getByRole('link')).toHaveText(['Restaurants', 'Recipes'])
  await expect(header.getByRole('link', { name: 'Reviews', exact: true })).toHaveCount(0)
})

test('the list tells a signed out visitor what the site is', async ({ page }) => {
  await page.goto('/restaurants')
  const main = page.getByRole('main')
  await expect(main.getByText('Find a restaurant by its scores')).toBeVisible()
  await main.getByRole('link', { name: 'Sign up' }).click()
  await expect(page).toHaveURL(/\/sign-up$/)
})

test('my reviews needs an account', async ({ page }) => {
  await page.goto('/my-reviews')
  await expect(page).toHaveURL(/\/sign-in\?redirect=(%2F|\/)my-reviews$/)
})

test('the all reviews page does not exist', async ({ page }) => {
  await page.goto('/reviews')
  await expect(page.getByRole('heading', { name: 'We cannot find this page' })).toBeVisible()
})

test('the search box finds reviews by their text', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const first = await api.createRestaurant(owner, { name: 'Search Kitchen' })
  const second = await api.createRestaurant(owner, { name: 'Search Grill' })
  const word = uniqueTag('dish')
  await api.createReview(user, first.id, { space: 5, service: 5, food: 5, price: 5, title: `The ${word} was good` })
  await api.createReview(user, second.id, { space: 5, service: 5, food: 5, price: 5, title: 'Fine', feedback: `Try the ${word}.` })
  await api.createReview(owner, first.id, { space: 5, service: 5, food: 5, price: 5, title: `Not mine, also ${word}` })

  // A short list shows the search box only with a filter in the URL.
  await page.goto('/my-reviews?search=Fine')
  await page.getByRole('searchbox', { name: 'Search reviews' }).fill(word)
  await expect(page.getByRole('heading', { level: 3 })).toHaveText([`The ${word} was good`, 'Fine'])
})
