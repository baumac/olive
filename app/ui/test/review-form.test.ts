import type { Restaurant } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { restaurantsApi } from '@/lib/api'
import { localDay } from '@/lib/format'
import ReviewFormPage from '@/pages/ReviewFormPage.vue'
import { expectAccessible } from './axe'

const restaurant: Restaurant = {
  id: 'k3x9m2p7q1w8z4a',
  name: 'Trattoria',
  location: null,
  creator: { id: 'u5n1c8r3t6y0b2e', name: 'Ana' },
  reviewCount: 0,
  rating: null,
  cover: null,
  favorite: false,
  ownReviewCount: 0,
  created: '2024-11-06T19:02:11.120Z',
  updated: '2024-11-06T19:02:11.120Z',
}

async function mountForm() {
  vi.spyOn(restaurantsApi, 'getRestaurant').mockResolvedValue(restaurant)
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
  const el = document.createElement('div')
  document.body.appendChild(el)
  const w = mount(ReviewFormPage, { props: { restaurantId: restaurant.id }, global: { plugins: [router] }, attachTo: el })
  await flushPromises()
  return w
}

function descriptionOf(group: Element): string {
  const ids = group.getAttribute('aria-describedby')?.split(' ') ?? []
  const parts = ids.flatMap((id) => {
    const el = document.getElementById(id)
    const paragraphs = el?.querySelectorAll('p')
    return paragraphs?.length ? [...paragraphs] : el ? [el] : []
  })
  return parts
    .map((p) => {
      const copy = p.cloneNode(true) as Element
      copy.querySelectorAll('[aria-hidden=true]').forEach((hidden) => hidden.remove())
      return copy.textContent?.replace(/\s+/g, ' ').trim()
    })
    .join(' ')
}

describe('ReviewFormPage scores', () => {
  afterEach(() => vi.restoreAllMocks())

  it('links the meaning of each score and its scale to the group', async () => {
    const w = await mountForm()
    const groups = w.findAll('[role=radiogroup]')
    expect(groups.map((g) => g.attributes('aria-label'))).toEqual(['Space', 'Service', 'Food', 'Value'])
    const price = groups[3]!.element
    expect(descriptionOf(price)).toBe(
      'Value for money, and whether the final bill matches the quality provided. 1 means poor value. 10 means great value.',
    )
    for (const g of groups) {
      expect(descriptionOf(g.element)).toMatch(/1 means .+\. 10 means .+\.$/)
    }
    await expectAccessible(w.element)
  })

  it('shows the overall score after the four scores have a value', async () => {
    const w = await mountForm()
    const overall = w.get('[role=status]')
    expect(overall.text()).toContain('It shows after you select each score.')
    const values = [7, 9, 8, 6]
    for (const [i, group] of w.findAll('[role=radiogroup]').entries()) {
      await group.findAll('[role=radio]')[values[i]! - 1]!.trigger('click')
    }
    expect(overall.text()).toBe('Overall7.5')
  })

  it('keeps the help text linked when a score has an error', async () => {
    const w = await mountForm()
    await w.get('form').trigger('submit')
    const space = w.get('[role=radiogroup][aria-label=Space]')
    expect(space.attributes('aria-describedby')).toBe('review-space-description review-space-ends review-space-message')
    expect(descriptionOf(space.element)).toContain('Select a space score.')
    expect(descriptionOf(space.element)).toContain('1 means uncomfortable.')
  })
})

describe('ReviewFormPage date', () => {
  afterEach(() => vi.restoreAllMocks())

  function mockCreate(date: string) {
    return vi.spyOn(restaurantsApi, 'createRestaurantReview').mockResolvedValue({
      id: 'r1b8c5e2g9i6k3m', restaurant: restaurant.id, restaurantName: restaurant.name, date,
      overall: 7, space: 7, service: 7, food: 7, price: 7, title: 'Second dinner',
      creator: restaurant.creator!, photos: [], created: '2024-11-06T19:02:11.120Z', updated: '2024-11-06T19:02:11.120Z',
    })
  }

  async function fillScoresAndTitle(w: Awaited<ReturnType<typeof mountForm>>) {
    for (const group of w.findAll('[role=radiogroup]')) {
      await group.findAll('[role=radio]')[6]!.trigger('click')
    }
    await w.get('input[name=title]').setValue('Second dinner')
  }

  it('starts at today and sends the date with the review', async () => {
    const list = vi.spyOn(restaurantsApi, 'listRestaurantReviews')
    const create = mockCreate('2024-03-01')
    const w = await mountForm()
    expect(list).not.toHaveBeenCalled()
    const date = w.get<HTMLInputElement>('input[name=date]')
    expect(w.get('label[for=review-date]').text()).toBe('Visited on')
    expect(date.element.value).toBe(localDay())
    expect(date.attributes('max')).toBe(localDay())
    await date.setValue('2024-03-01')
    await fillScoresAndTitle(w)
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(create.mock.calls[0]![0].restaurantReviewCreate).toMatchObject({ date: '2024-03-01', title: 'Second dinner' })
  })

  it('allows the new date after midnight', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2024, 10, 4, 23, 50))
    const create = mockCreate('2024-11-05')
    try {
      const w = await mountForm()
      expect(w.get<HTMLInputElement>('input[name=date]').element.value).toBe('2024-11-04')
      vi.setSystemTime(new Date(2024, 10, 5, 0, 10))
      const date = w.get('input[name=date]')
      await date.trigger('focus')
      expect(date.attributes('max')).toBe('2024-11-05')
      await date.setValue('2024-11-05')
      await fillScoresAndTitle(w)
      await w.get('form').trigger('submit')
      await flushPromises()
      expect(w.text()).not.toContain('Enter a date from 1 Jan 1900 to today.')
      expect(create.mock.calls[0]![0].restaurantReviewCreate).toMatchObject({ date: '2024-11-05' })
    } finally {
      vi.useRealTimers()
    }
  })

  it('refuses a date after today', async () => {
    const create = vi.spyOn(restaurantsApi, 'createRestaurantReview')
    const w = await mountForm()
    await w.get('input[name=date]').setValue('2999-01-01')
    await fillScoresAndTitle(w)
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(create).not.toHaveBeenCalled()
    expect(w.text()).toContain('Enter a date from 1 Jan 1900 to today.')
    expect(document.activeElement).toBe(w.get('input[name=date]').element)
  })
})
