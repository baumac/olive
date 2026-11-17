import { randomLocation, uniqueTag } from '../api'
import { expect, selectPlace, signIn, test } from '../fixtures'

test('the list shows restaurants, filters by tag, and loads more', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('batch')
  await Promise.all(
    Array.from({ length: 21 }, (_, i) =>
      api.createRestaurant(owner, { name: `Place ${String(i).padStart(2, '0')}`, tags: [tag] }),
    ),
  )

  await page.goto(`/restaurants?tags=${tag}&sort=name`)
  await expect(page.getByRole('heading', { name: 'Restaurants', level: 1 })).toBeVisible()
  await expect(page.getByText(`Tag: ${tag}`)).toBeVisible()
  const cards = page.getByRole('heading', { level: 2, name: /^Place / })
  await expect(cards).toHaveCount(20)
  await expect(page.getByRole('status')).toHaveText('20 restaurants shown, more to load')

  await page.getByRole('button', { name: 'Load more restaurants' }).click()
  await expect(cards).toHaveCount(21)
  await expect(page.getByRole('button', { name: 'Load more restaurants' })).toHaveCount(0)
  await expect(page.getByRole('listitem').filter({ hasText: 'Place 20' })).toBeFocused()
})

test('a signed in user adds, edits, and deletes a restaurant', async ({ page, user }) => {
  await page.goto('/restaurants')
  await page.getByRole('link', { name: 'Add restaurant' }).first().click()

  // The name of the selected place goes into the empty name field.
  const place = uniqueTag('taqueria')
  await selectPlace(page, place)
  await expect(page.getByLabel('Name')).toHaveValue(place)
  await page.getByLabel('Name').fill('Taqueria Sol')
  await page.getByLabel('Description').fill('Tacos al pastor.')
  await page.getByLabel('Tags').fill('mexican')
  await page.getByLabel('Tags').press('Enter')
  await page.getByRole('button', { name: 'Add restaurant' }).click()

  await expect(page.getByRole('heading', { name: 'Taqueria Sol', level: 1 })).toBeVisible()
  await expect(page.getByText('Restaurant added')).toBeVisible()
  await expect(page.getByText(`Added by ${user.name}`)).toBeVisible()
  await expect(page.getByText('Test Street 4, 08003 Testville, Testland')).toBeVisible()

  await page.getByRole('link', { name: 'Edit restaurant' }).click()
  await expect(page.getByLabel('Name')).toHaveValue('Taqueria Sol')
  await page.getByLabel('Name').fill('Taqueria Sol y Luna')
  await page.getByRole('button', { name: 'Save restaurant' }).click()
  await expect(page.getByRole('heading', { name: 'Taqueria Sol y Luna', level: 1 })).toBeVisible()

  await page.getByRole('button', { name: 'Delete restaurant' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete restaurant' }).click()
  await expect(page).toHaveURL(/\/restaurants$/)
  await expect(page.getByText('Restaurant deleted')).toBeVisible()
})

test('a tag can have the letters of all scripts', async ({ page, user: _user }) => {
  const tag = uniqueTag('café-ラーメン')
  await page.goto('/restaurants/new')
  await selectPlace(page, uniqueTag('kissa'))
  await page.getByLabel('Name').fill('Kissa Ramen')
  // A decomposed é becomes the composed form of the API.
  await page.getByLabel('Tags').fill(tag.normalize('NFD'))
  await page.getByLabel('Tags').press('Enter')
  await page.getByRole('button', { name: 'Add restaurant' }).click()
  await expect(page.getByRole('list', { name: 'Tags' }).getByText(tag, { exact: true })).toBeVisible()

  await page.goto(`/restaurants?tags=${encodeURIComponent(tag)}`)
  await expect(page.getByText(`Tag: ${tag}`)).toBeVisible()
  await expect(page.getByRole('heading', { level: 2, name: 'Kissa Ramen' })).toBeVisible()
})

test('the place search fills the details, and the creator changes and removes them', async ({ page, user: _user }) => {
  await page.goto('/restaurants/new')
  await page.getByLabel('Phone').fill('+34 600 11 22 33')
  await selectPlace(page, uniqueTag('details'))
  // The typed phone number stays. The empty fields get the tags of the place.
  await expect(page.getByLabel('Website')).toHaveValue('https://details.example')
  await expect(page.getByLabel('Phone')).toHaveValue('+34 600 11 22 33')
  await expect(page.getByLabel('Opening hours')).toHaveValue('Tu-Sa 13:00-16:00\nSu 13:00-16:00')
  await page.getByLabel('Price level').click()
  await page.getByRole('option', { name: 'Moderate' }).click()
  await page.getByRole('button', { name: 'Add restaurant' }).click()

  const details = page.getByRole('definition')
  await expect(page.getByText('Restaurant added')).toBeVisible()
  await expect(page.getByRole('link', { name: /^details\.example ?, opens in a new tab$/ })).toHaveAttribute('href', 'https://details.example')
  await expect(page.getByRole('link', { name: '+34 600 11 22 33' })).toHaveAttribute('href', 'tel:+34600112233')
  await expect(details.filter({ hasText: 'Tu-Sa 13:00-16:00' })).toBeVisible()
  await expect(details.filter({ hasText: 'Moderate' })).toBeVisible()

  await page.getByRole('link', { name: 'Edit restaurant' }).click()
  await page.getByLabel('Website').fill('www.details.example/menu')
  await page.getByLabel('Phone').fill('')
  await page.getByLabel('Opening hours').fill('')
  await page.getByLabel('Price level').click()
  await page.getByRole('option', { name: 'Not set' }).click()
  await page.getByRole('button', { name: 'Save restaurant' }).click()

  await expect(page.getByText('Restaurant saved')).toBeVisible()
  await expect(page.getByRole('link', { name: /^details\.example ?, opens in a new tab$/ })).toHaveAttribute('href', 'https://www.details.example/menu')
  await expect(details).toHaveCount(1)
})

test('only the creator sees edit and delete', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const restaurant = await api.createRestaurant(owner, { name: 'Can Sole' })
  const other = await api.signUp('Ana')
  await signIn(page, other)

  await page.goto(`/restaurants/${restaurant.id}`)
  await expect(page.getByRole('heading', { name: 'Can Sole', level: 1 })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Edit restaurant' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Delete restaurant' })).toHaveCount(0)

  await page.goto(`/restaurants/${restaurant.id}/edit`)
  await expect(page.getByRole('heading', { name: 'You cannot open this page' })).toBeVisible()
})

test('a restaurant with a review of another user keeps its name and cannot be deleted', async ({ page, api, user }) => {
  const restaurant = await api.createRestaurant(user, { name: "Luigi's" })
  const reviewer = await api.signUp('Joe')
  await api.createReview(reviewer, restaurant.id, { space: 9, service: 9, food: 9, price: 9, cleanliness: 9, title: 'Great' })

  await page.goto(`/restaurants/${restaurant.id}/edit`)
  await expect(page.getByLabel('Name')).not.toBeEditable()
  await expect(page.getByText('The name cannot change because other people have reviewed this restaurant.')).toBeVisible()
  await expect(page.getByText('The location cannot change because other people have reviewed this restaurant.')).toBeVisible()
  await expect(page.getByText('Other people have reviewed this restaurant. You cannot delete it.')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Enter the location by hand' })).toHaveCount(0)
  await expect(page.getByLabel('Location')).not.toBeEditable()
  await page.getByLabel('Description').fill('Family run')
  await page.getByRole('button', { name: 'Save restaurant' }).click()

  await expect(page).toHaveURL(new RegExp(`/restaurants/${restaurant.id}$`))
  await expect(page.getByText('Family run')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Edit restaurant' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Delete restaurant' })).toHaveCount(0)
})

test('a review that arrives after the form opens blocks the rename', async ({ page, api, user }) => {
  const restaurant = await api.createRestaurant(user, { name: "Luigi's" })
  await page.goto(`/restaurants/${restaurant.id}/edit`)
  await expect(page.getByLabel('Name')).toHaveValue("Luigi's")

  const reviewer = await api.signUp('Joe')
  await api.createReview(reviewer, restaurant.id, { space: 9, service: 9, food: 9, price: 9, cleanliness: 9, title: 'Great' })
  await page.getByLabel('Name').fill("Luigi's Trattoria")
  await page.getByRole('button', { name: 'Save restaurant' }).click()
  await expect(page.getByText('The name cannot change because other people have reviewed this restaurant.')).toBeVisible()
  await expect(page).toHaveURL(new RegExp(`/restaurants/${restaurant.id}/edit$`))
})

test('a restaurant needs a location', async ({ page, user: _user }) => {
  await page.goto('/restaurants/new')
  await page.getByLabel('Name').fill('No Place')
  await page.getByRole('button', { name: 'Add restaurant' }).click()
  await expect(page.getByText('Search for the place and select it, or enter the location by hand.')).toBeVisible()
  await expect(page.getByLabel('Location')).toBeFocused()
  await expect(page).toHaveURL(/\/restaurants\/new$/)
})

test('a second restaurant at a place shows the first one', async ({ page, user: _user }) => {
  const place = uniqueTag('duplicate')
  await page.goto('/restaurants/new')
  await selectPlace(page, place)
  await page.getByLabel('Name').fill('First Diner')
  await page.getByRole('button', { name: 'Add restaurant' }).click()
  await expect(page.getByRole('heading', { name: 'First Diner', level: 1 })).toBeVisible()
  const first = new URL(page.url()).pathname

  await page.goto('/restaurants/new')
  await selectPlace(page, place)
  const hint = page.getByRole('link', { name: 'First Diner' })
  await expect(page.getByText('Is it one of these?')).toBeVisible()
  await expect(hint).toHaveAttribute('href', first)

  await page.getByLabel('Name').fill('Second Diner')
  await page.getByRole('button', { name: 'Add restaurant' }).click()
  await expect(page.getByText('This restaurant is already here')).toBeVisible()
  await expect(page.getByText('Another restaurant has this place.')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Open the restaurant' })).toHaveAttribute('href', first)
})

test('a user enters a location by hand', async ({ page, api, user: _user }) => {
  const location = randomLocation()
  const neighbor = await api.createRestaurant(await api.signUp('Joe'), { name: 'Neighbor', location })

  await page.goto('/restaurants/new')
  await page.getByLabel('Location').fill('nothing here')
  await expect(page.getByText('No places found. Change the search, or enter the location by hand.')).toBeVisible()
  await page.getByRole('button', { name: 'Enter the location by hand' }).click()
  await page.getByLabel('Address').fill('Harbor Road 2, Testville')
  await page.getByLabel('Latitude').fill(String(location.latitude))
  await page.getByLabel('Longitude').fill(String(location.longitude))

  await expect(page.getByText('Is it one of these?')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Neighbor' })).toHaveAttribute('href', `/restaurants/${neighbor.id}`)

  await page.getByLabel('Name').fill('By Hand')
  await page.getByRole('button', { name: 'Add restaurant' }).click()
  await expect(page.getByRole('heading', { name: 'By Hand', level: 1 })).toBeVisible()
  await expect(page.getByText('Harbor Road 2, Testville')).toBeVisible()
})

test('the creator renames and deletes a restaurant that has only the own review', async ({ page, api, user }) => {
  const restaurant = await api.createRestaurant(user, { name: 'Luigis' })
  await api.createReview(user, restaurant.id, { space: 9, service: 9, food: 9, price: 9, cleanliness: 9, title: 'Great' })

  await page.goto(`/restaurants/${restaurant.id}/edit`)
  await page.getByLabel('Name').fill("Luigi's")
  await page.getByRole('button', { name: 'Save restaurant' }).click()
  await expect(page.getByRole('heading', { name: "Luigi's", level: 1 })).toBeVisible()

  await page.getByRole('button', { name: 'Delete restaurant' }).click()
  const dialog = page.getByRole('alertdialog')
  await expect(dialog).toContainText('Your review and its photos will be deleted too.')
  await dialog.getByRole('button', { name: 'Delete restaurant' }).click()
  await expect(page.getByText('Restaurant deleted')).toBeVisible()
  await expect(page).toHaveURL(/\/restaurants$/)

  await page.goto('/my-reviews')
  await expect(page.getByText('You have no reviews yet')).toBeVisible()
})

test('restaurants show the average scores of their reviews', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('rated')
  const rated = await api.createRestaurant(owner, { name: 'Rated Place', tags: [tag] })
  await api.createRestaurant(owner, { name: 'New Place', tags: [tag] })
  const ana = await api.signUp('Ana')
  await api.createReview(owner, rated.id, { space: 7, service: 8, food: 9, price: 6, cleanliness: 8, title: 'Good' })
  await api.createReview(ana, rated.id, { space: 8, service: 9, food: 10, price: 5, cleanliness: 7, title: 'Great' })

  await page.goto(`/restaurants?tags=${tag}&sort=name`)
  const ratedCard = page.getByRole('listitem').filter({ hasText: 'Rated Place' })
  await expect(ratedCard.getByRole('img', { name: 'Overall: 7.7 out of 10' })).toBeVisible()
  await expect(ratedCard).toContainText('2 reviews')
  await expect(page.getByRole('listitem').filter({ hasText: 'New Place' })).toContainText('No reviews yet')

  await page.getByRole('link', { name: 'Rated Place' }).click()
  const averages = page.getByRole('region', { name: 'Average scores' })
  await expect(averages.getByRole('img', { name: 'Overall: 7.7 out of 10' })).toBeVisible()
  await expect(averages.getByRole('img', { name: 'Food: 9.5 out of 10' })).toBeVisible()
  await expect(averages.getByRole('img', { name: 'Value: 5.5 out of 10' })).toBeVisible()
  await expect(averages).toContainText('out of 10 · 2 reviews')
})

test('restaurants filter and sort by the overall rating', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const tag = uniqueTag('overall')
  const top = await api.createRestaurant(owner, { name: 'Top Place', tags: [tag] })
  const fine = await api.createRestaurant(owner, { name: 'Fine Place', tags: [tag] })
  await api.createRestaurant(owner, { name: 'Unrated Place', tags: [tag] })
  await api.createReview(owner, top.id, { space: 9, service: 9, food: 9, price: 9, cleanliness: 9, title: 'Top' })
  await api.createReview(owner, fine.id, { space: 6, service: 6, food: 6, price: 6, cleanliness: 6, title: 'Fine' })
  const names = page.getByRole('heading', { level: 2, name: / Place$/ })

  // The highest rated restaurants are first without a sort in the link.
  await page.goto(`/restaurants?tags=${tag}`)
  await expect(page.getByRole('combobox', { name: 'Sort by' })).toHaveText('Highest rated')
  await expect(names).toHaveText(['Top Place', 'Fine Place', 'Unrated Place'])

  await page.goto(`/restaurants?tags=${tag}&overall=8`)
  await expect(names).toHaveText(['Top Place'])
  await expect(page.getByText('Overall: 8 or more')).toBeVisible()
  await page.getByRole('button', { name: 'Remove filter overall 8 or more' }).click()
  await expect(names).toHaveCount(3)
})

test('the search box finds restaurants by name, description, and tags', async ({ page, api }) => {
  const owner = await api.signUp('Joe')
  const word = uniqueTag('find')
  await api.createRestaurant(owner, { name: 'Quiet Corner', description: `Known for ${word} dishes` })
  await api.createRestaurant(owner, { name: `${word.toUpperCase()} Café` })
  await api.createRestaurant(owner, { name: 'Somewhere else' })
  const names = page.getByRole('heading', { level: 2 })
  const box = page.getByRole('searchbox', { name: 'Search restaurants' })

  await page.goto('/restaurants')
  await box.fill(word)
  await expect(page).toHaveURL(new RegExp(`search=${word}`))
  await expect(names).toHaveText([`${word.toUpperCase()} Café`, 'Quiet Corner'])
  await expect(page.getByRole('combobox', { name: 'Sort by' })).toHaveText('Best match')
  await expect(page.getByText(`Search: ${word}`)).toBeVisible()

  await box.fill(`${word} cafe`)
  await expect(names).toHaveText([`${word.toUpperCase()} Café`])

  await page.getByRole('button', { name: `Remove filter search ${word} cafe` }).click()
  await expect(box).toHaveValue('')
  await expect(page).not.toHaveURL(/search=/)
  await expect(page.getByRole('combobox', { name: 'Sort by' })).toHaveText('Highest rated')
})
