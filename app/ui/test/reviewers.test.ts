import type { Restaurant, RestaurantReview, SavedList } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { type RouteLocationNormalized, createMemoryHistory, createRouter } from 'vue-router'
import ReviewCard from '@/components/app/ReviewCard.vue'
import { listsApi, restaurantsApi, reviewersApi } from '@/lib/api'
import ReviewerPage from '@/pages/ReviewerPage.vue'
import { router as appRouter } from '@/router'
import { expectAccessible } from './axe'
import { rejectWith } from './problem'

const review: RestaurantReview = {
  id: 'r1b8c5e2g9i6k3m',
  restaurant: 'k3x9m2p7q1w8z4a',
  restaurantName: "Luigi's Trattoria",
  overall: 7.6,
  space: 7,
  service: 9,
  food: 8,
  price: 6,
  cleanliness: 8,
  title: 'Great food, a bit expensive',
  creator: { id: 'u5n1c8r3t6y0b2e', name: 'Ana' },
  photos: [],
  created: '2024-11-06T19:02:11.120Z',
  updated: '2024-11-06T19:02:11.120Z',
}

const restaurant: Restaurant = {
  id: 'k3x9m2p7q1w8z4a',
  name: "Luigi's Trattoria",
  location: null,
  creator: { id: 'u2g6k0m4o8q1s5w', name: 'Joe' },
  reviewCount: 0,
  rating: null,
  cover: null,
  favorite: false,
  created: '2024-11-04T01:26:55.002Z',
  updated: '2024-11-04T01:26:55.002Z',
}

const favorites: SavedList = {
  id: 'f1a6c3e8g5i2k9m',
  name: 'Favorites',
  kind: 'favorites',
  creator: { id: 'u5n1c8r3t6y0b2e', name: 'Ana' },
  restaurantCount: 2,
  recipeCount: 0,
  created: '2024-11-04T01:26:55.002Z',
  updated: '2024-11-04T01:26:55.002Z',
}

function attach() {
  const el = document.createElement('div')
  document.body.appendChild(el)
  return el
}

describe('ReviewCard', () => {
  it('shows the overall score first and links the reviewer', async () => {
    const w = mount(ReviewCard, { props: { review, showCreator: true }, attachTo: attach() })
    const labels = w.findAll('[role=img]').map((e) => e.attributes('aria-label'))
    expect(labels).toEqual(['Overall: 7.6 out of 10'])
    const link = w.findAll('a').find((a) => a.text() === 'Ana')
    expect(link?.attributes('href')).toBe('/reviewers/u5n1c8r3t6y0b2e')
    await expectAccessible(w.element)
  })

  it('shows the five scores after a click', async () => {
    const w = mount(ReviewCard, { props: { review }, attachTo: attach() })
    const button = w.get('button[aria-expanded]')
    expect(button.text()).toBe('Show scores of Great food, a bit expensive')
    await button.trigger('click')
    expect(button.attributes('aria-expanded')).toBe('true')
    const labels = w.findAll('ul [role=img]').map((e) => e.attributes('aria-label'))
    expect(labels).toEqual([
      'Space: 7 out of 10',
      'Service: 9 out of 10',
      'Food: 8 out of 10',
      'Value: 6 out of 10',
      'Cleanliness: 8 out of 10',
    ])
    await expectAccessible(w.element)
  })

  it('does not name the reviewer in a list of one reviewer', () => {
    const w = mount(ReviewCard, { props: { review } })
    expect(w.text()).not.toContain('Ana')
  })
})

async function mountPage(reviewerId: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
  await router.push(`/reviewers/${reviewerId}`)
  const w = mount(ReviewerPage, { props: { reviewerId }, global: { plugins: [router] }, attachTo: attach() })
  await flushPromises()
  return w
}

describe('ReviewerPage', () => {
  beforeEach(() => {
    vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [favorites] })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('lists the reviews of the reviewer', async () => {
    vi.spyOn(reviewersApi, 'getReviewer').mockResolvedValue({
      id: 'u5n1c8r3t6y0b2e',
      name: 'Ana',
      restaurantReviewCount: 1,
      restaurantAverageOverall: 7.6,
      restaurantCount: 0,
      recipeReviewCount: 0,
      recipeAverageOverall: null,
      recipeCount: 0,
      listCount: 1,
    })
    const list = vi.spyOn(reviewersApi, 'listReviewerRestaurantReviews').mockResolvedValue({ items: [review], nextCursor: null })
    const restaurants = vi.spyOn(restaurantsApi, 'listRestaurants')
    const w = await mountPage('u5n1c8r3t6y0b2e')
    expect(w.get('h1').text()).toBe('Ana')
    expect(w.find('[aria-label="Average restaurant score: 7.6 out of 10"]').exists()).toBe(true)
    expect(w.text()).toContain('average of 1 restaurant review')
    expect(w.text()).not.toContain('added')
    expect(restaurants).not.toHaveBeenCalled()
    expect(w.find('[role=search], input[type=search]').exists()).toBe(false)
    expect(w.findAll('a').some((a) => a.text() === 'Ana')).toBe(false)
    expect(w.text()).toContain('Great food, a bit expensive')
    expect(w.text()).toContain("Luigi's Trattoria")
    expect(list.mock.calls[0]![0]).toMatchObject({ reviewerId: 'u5n1c8r3t6y0b2e', sort: '-created' })
    expect(document.title).toBe('Ana · Sponga47 Reviews')
    await expectAccessible(w.element)
  })

  it('names the reviewer when there are no reviews', async () => {
    vi.spyOn(reviewersApi, 'getReviewer').mockResolvedValue({
      id: 'u2g6k0m4o8q1s5w',
      name: 'Joe',
      restaurantReviewCount: 0,
      restaurantAverageOverall: null,
      restaurantCount: 0,
      recipeReviewCount: 0,
      recipeAverageOverall: null,
      recipeCount: 0,
      listCount: 1,
    })
    vi.spyOn(reviewersApi, 'listReviewerRestaurantReviews').mockResolvedValue({ items: [], nextCursor: null })
    const w = await mountPage('u2g6k0m4o8q1s5w')
    expect(w.text()).toContain('No reviews yet')
    expect(w.find('[aria-label^="Average score"]').exists()).toBe(false)
    expect(w.text()).toContain('Joe has not reviewed a restaurant yet.')
  })

  it('lists the restaurants that the reviewer added', async () => {
    vi.spyOn(reviewersApi, 'getReviewer').mockResolvedValue({
      id: 'u2g6k0m4o8q1s5w',
      name: 'Joe',
      restaurantReviewCount: 0,
      restaurantAverageOverall: null,
      restaurantCount: 2,
      recipeReviewCount: 0,
      recipeAverageOverall: null,
      recipeCount: 0,
      listCount: 1,
    })
    vi.spyOn(reviewersApi, 'listReviewerRestaurantReviews').mockResolvedValue({ items: [], nextCursor: null })
    const list = vi.spyOn(restaurantsApi, 'listRestaurants').mockResolvedValue({ items: [restaurant], nextCursor: null })
    const w = await mountPage('u2g6k0m4o8q1s5w')
    expect(w.text()).toContain('No restaurant reviews yet · 2 restaurants added')
    expect(w.findAll('h2').map((h) => h.text())).toEqual(['Lists', 'Restaurants added', 'Reviews'])
    expect(w.get('h3 a').text()).toBe("Luigi's Trattoria")
    expect(w.get('h3 a').attributes('href')).toBe('/restaurants/k3x9m2p7q1w8z4a')
    expect(list.mock.calls[0]![0]).toMatchObject({ creatorEq: 'u2g6k0m4o8q1s5w', sort: '-created', limit: 3 })
    await expectAccessible(w.element)
  })

  it('tells that the restaurants are gone when the list is empty', async () => {
    vi.spyOn(reviewersApi, 'getReviewer').mockResolvedValue({
      id: 'u2g6k0m4o8q1s5w',
      name: 'Joe',
      restaurantReviewCount: 0,
      restaurantAverageOverall: null,
      restaurantCount: 1,
      recipeReviewCount: 0,
      recipeAverageOverall: null,
      recipeCount: 0,
      listCount: 1,
    })
    vi.spyOn(reviewersApi, 'listReviewerRestaurantReviews').mockResolvedValue({ items: [], nextCursor: null })
    vi.spyOn(restaurantsApi, 'listRestaurants').mockResolvedValue({ items: [], nextCursor: null })
    const w = await mountPage('u2g6k0m4o8q1s5w')
    expect(w.text()).toContain('These restaurants are not available now.')
  })

  it('offers a retry and no filter hint when the restaurants get 400', async () => {
    vi.spyOn(reviewersApi, 'getReviewer').mockResolvedValue({
      id: 'u2g6k0m4o8q1s5w',
      name: 'Joe',
      restaurantReviewCount: 0,
      restaurantAverageOverall: null,
      restaurantCount: 1,
      recipeReviewCount: 0,
      recipeAverageOverall: null,
      recipeCount: 0,
      listCount: 1,
    })
    vi.spyOn(reviewersApi, 'listReviewerRestaurantReviews').mockResolvedValue({ items: [], nextCursor: null })
    const list = vi.spyOn(restaurantsApi, 'listRestaurants').mockRejectedValueOnce(rejectWith(400))
    const w = await mountPage('u2g6k0m4o8q1s5w')
    expect(w.text()).toContain('Restaurants did not load')
    expect(w.text()).not.toContain('Change the search or the filters.')
    list.mockResolvedValue({ items: [restaurant], nextCursor: null })
    await w.findAll('button').find((b) => b.text() === 'Try again')!.trigger('click')
    await flushPromises()
    expect(w.get('h3 a').text()).toBe("Luigi's Trattoria")
  })

  it.each([
    ['an unknown', 404, 'aaaaaaaaaaaaaaa'],
    ['a malformed', 400, 'BAD'],
  ])('shows the not found page for %s reviewer ID', async (_, status, reviewerId) => {
    vi.spyOn(reviewersApi, 'getReviewer').mockRejectedValue(rejectWith(status))
    const list = vi.spyOn(reviewersApi, 'listReviewerRestaurantReviews')
    const w = await mountPage(reviewerId)
    expect(w.text()).toContain('We cannot find this page')
    expect(w.text()).toContain('The page may have been deleted, or the link has a typo.')
    expect(list).not.toHaveBeenCalled()
  })
})

describe('router scroll', () => {
  const scroll = appRouter.options.scrollBehavior!
  const at = (path: string) => ({ path }) as RouteLocationNormalized

  it('keeps the position when only the query changes', () => {
    expect(scroll(at('/reviewers/u2g6k0m4o8q1s5w'), at('/reviewers/u2g6k0m4o8q1s5w'), null)).toBe(false)
  })

  it('goes to the top of a new path and back to a saved position', () => {
    expect(scroll(at('/recipes'), at('/reviewers/u2g6k0m4o8q1s5w'), null)).toEqual({ top: 0 })
    expect(scroll(at('/recipes'), at('/restaurants'), { left: 0, top: 300 })).toEqual({ left: 0, top: 300 })
  })
})
