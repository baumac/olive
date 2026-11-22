import type { Recipe, RecipeVersion, Restaurant, RestaurantVersion } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { recipesApi, restaurantsApi } from '@/lib/api'
import { changedProperties, findUndo, undoChange, undoNumber, valueText } from '@/lib/history'
import { clearSession, saveSession } from '@/lib/session'
import HistoryPage from '@/pages/HistoryPage.vue'
import RecipeFormPage from '@/pages/RecipeFormPage.vue'
import RestaurantFormPage from '@/pages/RestaurantFormPage.vue'
import { expectAccessible } from './axe'

const ana = { id: 'u5n1c8r3t6y0b2e', name: 'Ana' }
const joe = { id: 'u2g6k0m4o8q1s5w', name: 'Joe' }
const eva = { id: 'u9e3v7a1b5c8d2f', name: 'Eva' }

const location = { latitude: 41.4, longitude: 2.2, address: 'Carrer 1' }

const restaurant: Restaurant = {
  id: 'k3x9m2p7q1w8z4a',
  name: 'Can Solé',
  description: undefined,
  tags: new Set(['tapas']),
  location,
  phone: '931 111 111',
  creator: ana,
  reviewCount: 0,
  rating: null,
  cover: null,
  favorite: false,
  ownReviewCount: 0,
  created: '2024-11-04T01:26:55.002Z',
  updated: '2024-11-06T01:26:55.002Z',
}

// version has the values that the SDK makes. The SDK sets each property.
// An unchanged property is undefined.
function version(number: number, editor: typeof ana | null, before: Partial<RestaurantVersion['before']>, after: Partial<RestaurantVersion['after']>): RestaurantVersion {
  return {
    id: `v${number}`.padEnd(15, '0'),
    number,
    action: number === 1 ? 'create' : 'update',
    source: editor ? 'user' : 'admin',
    editor,
    before,
    after,
    created: '2024-11-05T10:12:40.120Z',
  }
}

// The newest version comes first, as in the API.
const versions: RestaurantVersion[] = [
  version(4, eva, { phone: '931 000 000' }, { phone: '931 111 111' }),
  version(3, joe, { description: 'Small tapas bar', tags: null, phone: '931 222 222' }, { description: null, tags: new Set(['tapas']), phone: '931 000 000' }),
  version(2, null, { location: null }, { location }),
  version(1, ana, {}, { name: 'Can Solé', phone: '931 222 222' }),
]

async function routerAt(path: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
  await router.push(path)
  return router
}

function attach() {
  const el = document.createElement('div')
  document.body.appendChild(el)
  return el
}

function signIn() {
  saveSession({
    expires: '2099-01-01T00:00:00.000Z',
    user: { ...joe, email: 'joe@example.com', approved: true, created: '2024-11-04T01:26:55.002Z', updated: '2024-11-04T01:26:55.002Z' },
  })
}

afterEach(() => {
  vi.restoreAllMocks()
  clearSession()
  document.body.innerHTML = ''
})

describe('history', () => {
  it('lists the changed properties in the order of the form', () => {
    expect(changedProperties('restaurant', versions[1]!)).toEqual(['description', 'tags', 'phone'])
    expect(changedProperties('restaurant', versions[3]!)).toEqual(['name', 'phone'])
  })

  it('gives the text of a value', () => {
    expect(valueText('tags', new Set(['a', 'b']))).toBe('a, b')
    expect(valueText('location', location)).toBe('Carrer 1')
    expect(valueText('priceLevel', 2)).toBe('Moderate')
    expect(valueText('totalTime', 90)).toBe('1 hr 30 min')
    expect(valueText('steps', ['Cook.'])).toEqual(['Cook.'])
    expect(valueText('description', null)).toBeNull()
  })

  it('undoes the properties of one version', () => {
    expect(undoChange('restaurant', versions, 3)).toEqual({
      number: 3,
      values: { description: 'Small tapas bar', tags: null },
      kept: [{ property: 'phone', number: 4, editor: eva }],
    })
  })

  it('does not clear a location or undo a create', () => {
    expect(undoChange('restaurant', versions, 2)).toEqual({ number: 2, values: {}, kept: [] })
    expect(undoChange('restaurant', versions, 1)).toBeNull()
    expect(undoChange('restaurant', versions, 9)).toBeNull()
  })

  it('loads the pages down to the version', async () => {
    const list = vi
      .spyOn(restaurantsApi, 'listRestaurantVersions')
      .mockResolvedValueOnce({ items: versions.slice(0, 2), nextCursor: 'next' })
      .mockResolvedValueOnce({ items: versions.slice(2), nextCursor: null })
    const undo = await findUndo('restaurant', restaurant.id, 2)
    expect(list).toHaveBeenCalledTimes(2)
    expect(list.mock.calls[1]![0]).toEqual({ restaurantId: restaurant.id, limit: 100, cursor: 'next' })
    expect(undo?.number).toBe(2)
  })

  it('reads the undo query', () => {
    expect(undoNumber('3')).toBe(3)
    expect(undoNumber(undefined)).toBeNull()
    expect(undoNumber('x')).toBeNull()
    expect(undoNumber('0')).toBeNull()
  })

  it('shows the changes with the editor and an undo link', async () => {
    signIn()
    vi.spyOn(restaurantsApi, 'getRestaurant').mockResolvedValue(restaurant)
    vi.spyOn(restaurantsApi, 'listRestaurantVersions').mockResolvedValue({ items: versions, nextCursor: null })
    const w = mount(HistoryPage, {
      props: { kind: 'restaurant', parentId: restaurant.id },
      global: { plugins: [await routerAt(`/restaurants/${restaurant.id}/history`)] },
      attachTo: attach(),
    })
    await flushPromises()
    expect(w.get('h1').text()).toBe('History of Can Solé')
    const cards = w.findAll('li')
    expect(cards).toHaveLength(4)
    expect(cards[1]!.text()).toContain('Joe')
    expect(cards[1]!.text()).toContain('Small tapas bar')
    expect(cards[1]!.text()).toContain('None')
    expect(cards[1]!.get('a[aria-label="Undo version 3"]').attributes('href')).toBe(`/restaurants/${restaurant.id}/edit?undo=3`)
    expect(cards[2]!.text()).toContain('An admin')
    expect(cards[3]!.text()).toContain('Added the restaurant.')
    expect(cards[3]!.find('a[aria-label="Undo version 1"]').exists()).toBe(false)
    await expectAccessible(w.element)
    w.unmount()
  })

  it('tells that the history starts with the next change', async () => {
    signIn()
    vi.spyOn(restaurantsApi, 'getRestaurant').mockResolvedValue(restaurant)
    vi.spyOn(restaurantsApi, 'listRestaurantVersions').mockResolvedValue({ items: [], nextCursor: null })
    const w = mount(HistoryPage, {
      props: { kind: 'restaurant', parentId: restaurant.id },
      global: { plugins: [await routerAt(`/restaurants/${restaurant.id}/history`)] },
    })
    await flushPromises()
    expect(w.text()).toContain('No changes yet')
    w.unmount()
  })

  it('fills the restaurant form with the undo and sends only the undone values', async () => {
    signIn()
    vi.spyOn(restaurantsApi, 'getRestaurant').mockResolvedValue(restaurant)
    vi.spyOn(restaurantsApi, 'listRestaurantVersions').mockResolvedValue({ items: versions, nextCursor: null })
    const update = vi.spyOn(restaurantsApi, 'updateRestaurant').mockResolvedValue(restaurant)
    const w = mount(RestaurantFormPage, {
      props: { restaurantId: restaurant.id },
      global: { plugins: [await routerAt(`/restaurants/${restaurant.id}/edit?undo=3`)] },
      attachTo: attach(),
    })
    await flushPromises()
    expect(w.text()).toContain('Undo of version 3')
    expect(w.text()).toContain('Eva changed the phone again in version 4. The form keeps the current value.')
    expect((w.get('textarea[name=description]').element as HTMLTextAreaElement).value).toBe('Small tapas bar')
    expect((w.get('input[name=phone]').element as HTMLInputElement).value).toBe('931 111 111')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(update.mock.calls[0]![0].restaurantPatch).toEqual({ description: 'Small tapas bar', tags: null, location: undefined })
    w.unmount()
  })

  it('tells when a version has nothing to undo', async () => {
    signIn()
    vi.spyOn(restaurantsApi, 'getRestaurant').mockResolvedValue(restaurant)
    vi.spyOn(restaurantsApi, 'listRestaurantVersions').mockResolvedValue({ items: versions, nextCursor: null })
    const w = mount(RestaurantFormPage, {
      props: { restaurantId: restaurant.id },
      global: { plugins: [await routerAt(`/restaurants/${restaurant.id}/edit?undo=2`)] },
    })
    await flushPromises()
    expect(w.text()).toContain('Nothing to undo in version 2')
    expect(w.text()).toContain('The form has the current values.')
    expect(w.text()).not.toContain('Check the values, then save.')
    w.unmount()
  })

  it('tells when a version cannot be undone', async () => {
    signIn()
    vi.spyOn(restaurantsApi, 'getRestaurant').mockResolvedValue(restaurant)
    vi.spyOn(restaurantsApi, 'listRestaurantVersions').mockRejectedValue(new TypeError('Failed to fetch'))
    const w = mount(RestaurantFormPage, {
      props: { restaurantId: restaurant.id },
      global: { plugins: [await routerAt(`/restaurants/${restaurant.id}/edit?undo=3`)] },
    })
    await flushPromises()
    expect(w.text()).toContain('This change cannot be undone here')
    expect((w.get('input[name=phone]').element as HTMLInputElement).value).toBe('931 111 111')
    w.unmount()
  })

  it('fills the recipe form with the old steps', async () => {
    signIn()
    const recipe: Recipe = {
      id: 'c4r8e2p6i0s3t7a',
      name: 'Dal',
      ingredients: ['lentils'],
      steps: ['Rinse.', 'Cook.'],
      creator: ana,
      reviewCount: 0,
      rating: null,
      cover: null,
      favorite: false,
      ownReviewCount: 0,
      created: '2024-11-04T01:26:55.002Z',
      updated: '2024-11-06T01:26:55.002Z',
    }
    const recipeVersion: RecipeVersion = {
      id: 'v20000000000000',
      number: 2,
      action: 'update',
      source: 'user',
      editor: joe,
      before: { steps: ['Cook.'], servings: 2 },
      after: { steps: ['Rinse.', 'Cook.'], servings: null },
      created: '2024-11-05T10:12:40.120Z',
    }
    vi.spyOn(recipesApi, 'getRecipe').mockResolvedValue(recipe)
    vi.spyOn(recipesApi, 'listRecipeVersions').mockResolvedValue({ items: [recipeVersion], nextCursor: null })
    const update = vi.spyOn(recipesApi, 'updateRecipe').mockResolvedValue(recipe)
    const w = mount(RecipeFormPage, {
      props: { recipeId: recipe.id },
      global: { plugins: [await routerAt(`/recipes/${recipe.id}/edit?undo=2`)] },
      attachTo: attach(),
    })
    await flushPromises()
    expect(w.findAll('[id^=recipe-step-]').filter((e) => e.element.tagName === 'TEXTAREA')).toHaveLength(1)
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(update.mock.calls[0]![0].recipePatch).toEqual({ steps: ['Cook.'], servings: 2 })
    w.unmount()
  })
})
