import { uniqueTag } from '../api'
import { expect, test } from '../fixtures'

test('a signed in user writes a review', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Can Sole' })

  await page.goto(`/restaurants/${restaurant.id}/reviews/new`)
  await expect(page.getByRole('tab')).toHaveCount(0)
  await page.getByRole('button', { name: 'Save review' }).click()
  await expect(page.getByText('Select a space score.')).toBeVisible()

  for (const [label, score] of [['Space', 7], ['Service', 9], ['Food', 8], ['Price', 6], ['Cleanliness', 5]] as const) {
    await page.getByRole('radiogroup', { name: label }).getByRole('radio', { name: `${score} out of 10` }).click()
  }
  await page.getByLabel('Title').fill('Great food, a bit expensive')
  await page.getByLabel('Feedback').fill('Friendly staff.\nGreat pasta.')
  await page.getByRole('button', { name: 'Save review' }).click()

  await expect(page.getByText('Review saved')).toBeVisible()
  const card = page.getByRole('region', { name: 'Reviews' }).getByRole('listitem').filter({ hasText: 'Great food, a bit expensive' })
  await expect(card.getByRole('img', { name: 'Food: 8 out of 10' })).toBeVisible()
  await expect(card.getByRole('img', { name: 'Cleanliness: 5 out of 10' })).toBeVisible()
  await expect(card).toContainText(user.name)
  await expect(page.getByRole('link', { name: 'Edit your review' })).toBeVisible()
})

test('write a review opens the edit form for a user with a review', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Bao House' })
  const review = await api.createReview(user, restaurant.id, { space: 5, service: 5, food: 5, price: 5, cleanliness: 5, title: 'Fine' })

  await page.goto(`/restaurants/${restaurant.id}/reviews/new`)
  await expect(page).toHaveURL(new RegExp(`/reviews/${review.id}/edit$`))
  await expect(page.getByRole('heading', { name: 'Edit review' })).toBeVisible()
  await expect(page.getByLabel('Title')).toHaveValue('Fine')
})

test('filters and score sorts select and order reviews', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Mixed' })
  const ana = await api.signUp('Ana')
  const bo = await api.signUp('Bo')
  const cy = await api.signUp('Cy')
  await api.createReview(ana, restaurant.id, { space: 2, service: 2, food: 2, price: 9, cleanliness: 9, title: 'Poor food' })
  await api.createReview(bo, restaurant.id, { space: 5, service: 5, food: 10, price: 1, cleanliness: 1, title: 'Best food' })
  await api.createReview(cy, restaurant.id, { space: 7, service: 9, food: 8, price: 6, cleanliness: 6, title: 'Good food' })

  const titles = page.getByRole('heading', { level: 3 })

  await page.goto(`/restaurants/${restaurant.id}?sort=-food`)
  await expect(titles).toHaveText(['Best food', 'Good food', 'Poor food'])

  await page.goto(`/restaurants/${restaurant.id}?sort=-price`)
  await expect(titles).toHaveText(['Poor food', 'Good food', 'Best food'])

  await page.goto(`/restaurants/${restaurant.id}?food=8`)
  await expect(titles).toHaveText(['Good food', 'Best food'])
  await expect(page.getByText('Food: 8 or more')).toBeVisible()
  await page.getByRole('button', { name: 'Remove filter food 8 or more' }).click()
  await expect(titles).toHaveCount(3)
})

test('the creator edits and deletes a review', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Edit me' })
  const review = await api.createReview(user, restaurant.id, { space: 3, service: 3, food: 3, price: 3, cleanliness: 3, title: 'Okay' })

  await page.goto(`/restaurants/${restaurant.id}/reviews/${review.id}`)
  await page.getByRole('link', { name: 'Edit review' }).click()
  await expect(page.getByRole('radiogroup', { name: 'Food' }).getByRole('radio', { name: '3 out of 10' })).toBeChecked()
  await page.getByRole('radiogroup', { name: 'Food' }).getByRole('radio', { name: '9 out of 10' }).click()
  await page.getByRole('button', { name: 'Save review' }).click()
  await expect(page.getByRole('region', { name: 'Reviews' }).getByRole('img', { name: 'Food: 9 out of 10' })).toBeVisible()
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
  await api.createReview(user, first.id, { space: 4, service: 4, food: 4, price: 4, cleanliness: 4, title: 'Mine at first' })
  await api.createReview(user, second.id, { space: 2, service: 2, food: 2, price: 2, cleanliness: 2, title: 'Mine at second' })
  await api.createReview(owner, first.id, { space: 5, service: 5, food: 5, price: 5, cleanliness: 5, title: 'Not mine' })

  await page.goto('/restaurants')
  await page.getByRole('button', { name: `AN, account menu for ${user.name}` }).click()
  await page.getByRole('menuitem', { name: 'My reviews' }).click()

  await expect(page.getByRole('heading', { name: 'My reviews', level: 1 })).toBeVisible()
  await expect(page).toHaveURL(/\/my-reviews$/)
  await expect(page.getByRole('heading', { level: 3 })).toHaveText(['Mine at second', 'Mine at first'])
  await page.getByRole('link', { name: 'First Place' }).click()
  await expect(page.getByRole('heading', { name: 'First Place', level: 1 })).toBeVisible()
})

test('the main navigation has no reviews link', async ({ page }) => {
  await page.goto('/restaurants')
  const nav = page.getByRole('navigation', { name: 'Main' })
  await expect(nav.getByRole('link', { name: 'Restaurants' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Reviews' })).toHaveCount(0)
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
  await api.createReview(user, first.id, { space: 5, service: 5, food: 5, price: 5, cleanliness: 5, title: `The ${word} was good` })
  await api.createReview(user, second.id, { space: 5, service: 5, food: 5, price: 5, cleanliness: 5, title: 'Fine', feedback: `Try the ${word}.` })
  await api.createReview(owner, first.id, { space: 5, service: 5, food: 5, price: 5, cleanliness: 5, title: `Not mine, also ${word}` })

  await page.goto('/my-reviews')
  await page.getByRole('searchbox', { name: 'Search reviews' }).fill(word)
  await expect(page.getByRole('heading', { level: 3 })).toHaveText([`The ${word} was good`, 'Fine'])
})
