import type { Review } from '@reviews/sdk'
import { ResponseError } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import ReviewCard from '@/components/app/ReviewCard.vue'
import { reviewersApi } from '@/lib/api'
import ReviewerPage from '@/pages/ReviewerPage.vue'
import { expectAccessible } from './axe'

const review: Review = {
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
  created: '2024-11-06 19:02:11.120Z',
  updated: '2024-11-06 19:02:11.120Z',
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

function rejectWith(status: number) {
  const problem = { type: 'about:blank', title: 'Error', status, detail: 'The user does not exist.' }
  return new ResponseError(new Response(JSON.stringify(problem), { status, headers: { 'Content-Type': 'application/problem+json' } }))
}

async function mountPage(reviewerId: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
  await router.push(`/reviewers/${reviewerId}`)
  const w = mount(ReviewerPage, { props: { reviewerId }, global: { plugins: [router] }, attachTo: attach() })
  await flushPromises()
  return w
}

describe('ReviewerPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('lists the reviews of the reviewer', async () => {
    vi.spyOn(reviewersApi, 'getReviewer').mockResolvedValue({ id: 'u5n1c8r3t6y0b2e', name: 'Ana', reviewCount: 1 })
    const list = vi.spyOn(reviewersApi, 'listReviewerReviews').mockResolvedValue({ items: [review], nextCursor: null })
    const w = await mountPage('u5n1c8r3t6y0b2e')
    expect(w.get('h1').text()).toBe('Ana')
    expect(w.text()).toContain('1 review')
    expect(w.find('[role=search], input[type=search]').exists()).toBe(false)
    expect(w.findAll('a').some((a) => a.text() === 'Ana')).toBe(false)
    expect(w.text()).toContain('Great food, a bit expensive')
    expect(w.text()).toContain("Luigi's Trattoria")
    expect(list.mock.calls[0]![0]).toMatchObject({ reviewerId: 'u5n1c8r3t6y0b2e', sort: '-created' })
    expect(document.title).toBe('Ana · Sponga47 Reviews')
    await expectAccessible(w.element)
  })

  it('names the reviewer when there are no reviews', async () => {
    vi.spyOn(reviewersApi, 'getReviewer').mockResolvedValue({ id: 'u2g6k0m4o8q1s5w', name: 'Joe', reviewCount: 0 })
    vi.spyOn(reviewersApi, 'listReviewerReviews').mockResolvedValue({ items: [], nextCursor: null })
    const w = await mountPage('u2g6k0m4o8q1s5w')
    expect(w.text()).toContain('No reviews yet')
    expect(w.text()).toContain('Joe has not reviewed a restaurant yet.')
  })

  it.each([
    ['an unknown', 404, 'aaaaaaaaaaaaaaa'],
    ['a malformed', 400, 'BAD'],
  ])('shows the not found page for %s reviewer ID', async (_, status, reviewerId) => {
    vi.spyOn(reviewersApi, 'getReviewer').mockRejectedValue(rejectWith(status))
    const list = vi.spyOn(reviewersApi, 'listReviewerReviews')
    const w = await mountPage(reviewerId)
    expect(w.text()).toContain('We cannot find this page')
    expect(w.text()).toContain('The page may have been deleted, or the link has a typo.')
    expect(list).not.toHaveBeenCalled()
  })
})
