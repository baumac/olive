import { uniqueTag } from '../api'
import { expect, test } from '../fixtures'

test('a signed in user writes a basic review', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: "Luigi's" })

  await page.goto(`/restaurants/${restaurant.id}/reviews/new`)
  await page.getByRole('radiogroup', { name: 'Rating' }).getByRole('radio', { name: '4 out of 5' }).click()
  await page.getByLabel('Title').fill('Best carbonara in town')
  await page.getByLabel('Feedback').fill('Friendly staff.\nGreat pasta.')
  await page.getByRole('button', { name: 'Save review' }).click()

  await expect(page.getByText('Review saved')).toBeVisible()
  const card = page.getByRole('region', { name: 'Reviews' }).getByRole('listitem').filter({ hasText: 'Best carbonara in town' })
  await expect(card.getByRole('img', { name: 'Rating: 4 out of 5' })).toBeVisible()
  await expect(card).toContainText(user.name)
  await expect(page.getByRole('link', { name: 'Edit your review' })).toBeVisible()
})

test('a signed in user writes a joc de cartes review', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Can Sole' })

  await page.goto(`/restaurants/${restaurant.id}/reviews/new`)
  await page.getByRole('tab', { name: 'Joc de cartes' }).click()
  await page.getByRole('button', { name: 'Save review' }).click()
  await expect(page.getByText('Select a space score.')).toBeVisible()

  for (const [label, score] of [['Space', 7], ['Service', 9], ['Food', 8], ['Price', 6]] as const) {
    await page.getByRole('radiogroup', { name: label }).getByRole('radio', { name: `${score} out of 10` }).click()
  }
  await page.getByLabel('Title').fill('Great food, a bit expensive')
  await page.getByRole('button', { name: 'Save review' }).click()

  const card = page.getByRole('region', { name: 'Reviews' }).getByRole('listitem').filter({ hasText: 'Great food, a bit expensive' })
  await expect(card.getByRole('img', { name: 'Food: 8 out of 10' })).toBeVisible()
  await expect(card).toContainText(user.name)
})

test('write a review opens the edit form for a user with a review', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Bao House' })
  const review = await api.createReview(user, restaurant.id, { kind: 'basic', rating: 3, title: 'Fine' })

  await page.goto(`/restaurants/${restaurant.id}/reviews/new`)
  await expect(page).toHaveURL(new RegExp(`/reviews/${review.id}/edit$`))
  await expect(page.getByRole('heading', { name: 'Edit review' })).toBeVisible()
  await expect(page.getByLabel('Title')).toHaveValue('Fine')
})

test('filters select reviews and score sorts put other kinds last', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Mixed' })
  const ana = await api.signUp('Ana')
  const bo = await api.signUp('Bo')
  const cy = await api.signUp('Cy')
  await api.createReview(ana, restaurant.id, { kind: 'basic', rating: 2, title: 'Two stars' })
  await api.createReview(bo, restaurant.id, { kind: 'basic', rating: 5, title: 'Five stars' })
  await api.createReview(cy, restaurant.id, {
    kind: 'joc_de_cartes', space: 7, service: 9, food: 8, price: 6, title: 'Joc review',
  })

  const titles = page.getByRole('heading', { level: 3 })

  await page.goto(`/restaurants/${restaurant.id}?sort=-rating`)
  await expect(titles).toHaveText(['Five stars', 'Two stars', 'Joc review'])

  await page.goto(`/restaurants/${restaurant.id}?sort=rating`)
  await expect(titles).toHaveText(['Two stars', 'Five stars', 'Joc review'])

  await page.goto(`/restaurants/${restaurant.id}?rating=4`)
  await expect(titles).toHaveText(['Five stars'])
  await expect(page.getByText('Rating: 4 or more')).toBeVisible()

  await page.goto(`/restaurants/${restaurant.id}?kind=joc_de_cartes`)
  await expect(titles).toHaveText(['Joc review'])
  await page.getByRole('button', { name: 'Remove filter kind joc de cartes' }).click()
  await expect(titles).toHaveCount(3)
})

test('the creator edits and deletes a review', async ({ page, api, user }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Edit me' })
  const review = await api.createReview(user, restaurant.id, { kind: 'basic', rating: 3, title: 'Okay' })

  await page.goto(`/restaurants/${restaurant.id}/reviews/${review.id}`)
  await page.getByRole('link', { name: 'Edit review' }).click()
  await expect(page.getByText('Basic', { exact: true })).toBeVisible()
  await expect(page.getByRole('tab')).toHaveCount(0)
  await page.getByRole('radio', { name: '5 out of 5' }).click()
  await page.getByRole('button', { name: 'Save review' }).click()
  await expect(page.getByRole('img', { name: 'Rating: 5 out of 5' })).toBeVisible()
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
  await api.createReview(user, first.id, { kind: 'basic', rating: 4, title: 'Mine at first' })
  await api.createReview(user, second.id, { kind: 'basic', rating: 2, title: 'Mine at second' })
  await api.createReview(owner, first.id, { kind: 'basic', rating: 5, title: 'Not mine' })

  await page.goto('/restaurants')
  await page.getByRole('button', { name: `AN, account menu for ${user.name}` }).click()
  await page.getByRole('menuitem', { name: 'My reviews' }).click()

  await expect(page.getByRole('heading', { name: 'My reviews', level: 1 })).toBeVisible()
  await expect(page.getByText('Written by: you')).toBeVisible()
  await expect(page.getByRole('heading', { level: 3 })).toHaveText(['Mine at second', 'Mine at first'])
  await page.getByRole('link', { name: 'First Place' }).click()
  await expect(page.getByRole('heading', { name: 'First Place', level: 1 })).toBeVisible()
})

test('the reviews link lists reviews of all restaurants', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Any Place' })
  const tag = uniqueTag('all')
  await api.createReview(owner, restaurant.id, { kind: 'basic', rating: 3, title: 'Seen on all reviews', tags: [tag] })

  await page.goto('/restaurants')
  await page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Reviews' }).click()
  await expect(page.getByRole('heading', { name: 'Reviews', level: 1 })).toBeVisible()
  await expect(page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Reviews', exact: true })).toHaveAttribute(
    'aria-current',
    'page',
  )

  await page.goto(`/reviews?tags=${tag}`)
  const card = page.getByRole('listitem').filter({ hasText: 'Seen on all reviews' })
  await expect(card.getByRole('link', { name: 'Any Place' })).toBeVisible()
})
