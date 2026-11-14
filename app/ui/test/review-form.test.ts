import type { Restaurant } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { restaurantsApi } from '@/lib/api'
import ReviewFormPage from '@/pages/ReviewFormPage.vue'
import { expectAccessible } from './axe'

const restaurant: Restaurant = {
  id: 'k3x9m2p7q1w8z4a',
  name: 'Trattoria',
  location: null,
  creator: { id: 'u5n1c8r3t6y0b2e', name: 'Ana' },
  reviewCount: 0,
  rating: null,
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
    expect(groups.map((g) => g.attributes('aria-label'))).toEqual(['Space', 'Service', 'Food', 'Value', 'Cleanliness'])
    const price = groups[3]!.element
    expect(descriptionOf(price)).toBe(
      'Value for money, and whether the final bill matches the quality provided. 1 means poor value. 10 means great value.',
    )
    for (const g of groups) {
      expect(descriptionOf(g.element)).toMatch(/1 means .+\. 10 means .+\.$/)
    }
    await expectAccessible(w.element)
  })

  it('shows the overall score after the five scores have a value', async () => {
    const w = await mountForm()
    const overall = w.get('[role=status]')
    expect(overall.text()).toContain('It shows after you select the five scores.')
    const values = [7, 9, 8, 6, 8]
    for (const [i, group] of w.findAll('[role=radiogroup]').entries()) {
      await group.findAll('[role=radio]')[values[i]! - 1]!.trigger('click')
    }
    expect(overall.text()).toBe('Overall7.6')
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
