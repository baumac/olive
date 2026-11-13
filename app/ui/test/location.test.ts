import type { Location, Place, Restaurant } from '@reviews/sdk'
import { ResponseError } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import LocationField from '@/components/app/LocationField.vue'
import { fieldErrors, placesApi, restaurantsApi } from '@/lib/api'
import { saveDraft } from '@/lib/drafts'
import { clearSession, saveSession } from '@/lib/session'
import RestaurantFormPage from '@/pages/RestaurantFormPage.vue'
import { parseCoordinate, sameLocation } from '@/lib/places'
import { expectAccessible } from './axe'

const canSole: Place = {
  name: 'Can Solé',
  location: { latitude: 41.3766, longitude: 2.1893, address: 'Carrer de Sant Carles 4, 08003 Barcelona', placeId: 'osm:N1' },
}
const house: Place = { location: { latitude: 41.377, longitude: 2.19, address: 'Carrer de Sant Carles 6, 08003 Barcelona' } }

function mountField(props: Record<string, unknown> = {}) {
  const el = document.createElement('div')
  document.body.appendChild(el)
  return mount(LocationField, { props: { id: 'location', ...props }, attachTo: el })
}

describe('places', () => {
  it('compares locations with their place ID', () => {
    expect(sameLocation(null, null)).toBe(true)
    expect(sameLocation(canSole.location, null)).toBe(false)
    expect(sameLocation(canSole.location, { ...canSole.location })).toBe(true)
    expect(sameLocation(canSole.location, { ...canSole.location, placeId: undefined })).toBe(false)
    expect(sameLocation(house.location, { ...house.location, placeId: undefined })).toBe(true)
  })

  it('reads coordinates with a point or a comma', () => {
    expect(parseCoordinate(' 41.3766 ', 90)).toBe(41.3766)
    expect(parseCoordinate('-2,5', 180)).toBe(-2.5)
    expect(parseCoordinate('0', 90)).toBe(0)
    for (const bad of ['', '91', '1e2', '41.', 'north', '1.2.3']) {
      expect(parseCoordinate(bad, 90)).toBeNull()
    }
  })

  it('maps the errors of a location to the location', () => {
    const problem = {
      type: 'about:blank',
      title: 'Unprocessable Content',
      status: 422,
      errors: [
        { property: 'location.address', message: 'must have 1 to 255 characters' },
        { property: 'location.latitude', message: 'must be between -90 and 90' },
      ],
    }
    expect(fieldErrors(problem)).toEqual({ location: 'Must have 1 to 255 characters.' })
  })
})

describe('LocationField', () => {
  // Vue ignores a click with a time before its listener. The clock therefore
  // stays real.
  beforeEach(() => vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] }))
  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('searches after a delay and selects a place', async () => {
    const search = vi.spyOn(placesApi, 'searchPlaces').mockResolvedValue({ items: [canSole, house] })
    const w = mountField()
    const input = w.get('input[type=search]')

    await input.setValue('ca')
    await vi.advanceTimersByTimeAsync(1000)
    expect(search).not.toHaveBeenCalled()

    await input.setValue('can')
    await input.setValue('can sole')
    await vi.advanceTimersByTimeAsync(1000)
    await flushPromises()
    expect(search).toHaveBeenCalledTimes(1)
    expect(search).toHaveBeenCalledWith({ search: 'can sole', latitude: 41.3874, longitude: 2.1686 })
    expect(w.get('[role=status]').text()).toBe('2 places found')
    const buttons = w.findAll('ul button')
    expect(buttons.map((b) => b.text())).toEqual(['Can SoléCarrer de Sant Carles 4, 08003 Barcelona', 'Carrer de Sant Carles 6, 08003 Barcelona'])
    vi.useRealTimers()
    await expectAccessible(w.element)

    await buttons[0]!.trigger('click')
    expect(w.emitted('update:modelValue')).toEqual([[canSole.location]])
    expect(w.emitted('named')).toEqual([['Can Solé']])
    await w.setProps({ modelValue: canSole.location })
    expect(w.text()).toContain('Selected location: Carrer de Sant Carles 4, 08003 Barcelona')
    expect(w.findAll('ul button')).toHaveLength(0)
  })

  it('opens the fields by hand when the search fails', async () => {
    vi.spyOn(placesApi, 'searchPlaces').mockRejectedValue(new ResponseError(new Response('{}', { status: 503 })))
    const w = mountField()
    await w.get('input[type=search]').setValue('can sole')
    await vi.advanceTimersByTimeAsync(1000)
    await flushPromises()
    expect(w.text()).toContain('The place search does not answer.')

    await w.get('input[name=address]').setValue(' Carrer Nou 1 ')
    await w.get('input[name=latitude]').setValue('41,38')
    expect(w.emitted('update:modelValue')).toBeUndefined()
    await w.get('input[name=longitude]').setValue('2.19')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([{ address: 'Carrer Nou 1', latitude: 41.38, longitude: 2.19 }])
    await w.get('input[name=longitude]').setValue('200')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([null])
    vi.useRealTimers()
    await expectAccessible(w.element)
  })

  it('keeps the place ID when the fields by hand do not change', async () => {
    const w = mountField({ modelValue: canSole.location })
    await w.get('button').trigger('click')
    expect((w.get('input[name=address]').element as HTMLInputElement).value).toBe(canSole.location.address)
    expect(w.emitted('update:modelValue')).toBeUndefined()
    await w.get('input[name=address]').setValue('Carrer Nou 1')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([{ address: 'Carrer Nou 1', latitude: 41.3766, longitude: 2.1893 }])
  })

  it('shows only the location when it is locked', async () => {
    const w = mountField({ modelValue: canSole.location, locked: true, describedBy: 'reason' })
    const input = w.get('input')
    expect(w.findAll('input, button')).toHaveLength(1)
    expect(input.attributes('id')).toBe('location')
    expect(input.attributes('readonly')).toBeDefined()
    expect(input.attributes('aria-describedby')).toBe('reason')
    expect((input.element as HTMLInputElement).value).toBe(canSole.location.address)
  })

  it('names the address field only Address', async () => {
    const w = mountField()
    await w.get('button').trigger('click')
    const input = w.get('input[name=address]')
    expect(input.attributes('id')).toBe('location')
    expect(document.getElementById(input.attributes('aria-labelledby')!)?.textContent).toBe('Address')
  })

  it('keeps the search after a 429', async () => {
    vi.spyOn(placesApi, 'searchPlaces').mockRejectedValue(
      new ResponseError(new Response('{}', { status: 429, headers: { 'Retry-After': '12' } })),
    )
    const w = mountField()
    await w.get('input[type=search]').setValue('can sole')
    await vi.advanceTimersByTimeAsync(1000)
    await flushPromises()
    expect(w.text()).toContain('Too many searches. Try again in 12 seconds.')
    expect(w.find('input[type=search]').exists()).toBe(true)
    expect(w.find('input[name=address]').exists()).toBe(false)
  })

  it('stops searching after a 401', async () => {
    vi.spyOn(placesApi, 'searchPlaces').mockRejectedValue(new ResponseError(new Response('{}', { status: 401 })))
    const w = mountField()
    await w.get('input[type=search]').setValue('can sole')
    await vi.advanceTimersByTimeAsync(1000)
    await flushPromises()
    expect(w.text()).not.toContain('Searching')
    expect(w.find('input[name=address]').exists()).toBe(false)
  })
})

describe('RestaurantFormPage drafts', () => {
  const user = { id: 'u5n1c8r3t6y0b2e', email: 'ana@example.com', name: 'Ana', created: '2024-11-04 01:26:55.002Z', updated: '2024-11-04 01:26:55.002Z' }
  const restaurant: Restaurant = {
    id: 'k3x9m2p7q1w8z4a',
    name: 'Old Place',
    location: null,
    creator: { id: user.id, name: 'Ana' },
    reviewCount: 2,
    rating: null,
    created: user.created,
    updated: user.updated,
  }

  afterEach(() => {
    vi.restoreAllMocks()
    clearSession()
  })

  async function mountWithDraft(stored: Location | null) {
    saveSession({ token: 't', expires: '2099-01-01 00:00:00.000Z', user })
    vi.spyOn(restaurantsApi, 'getRestaurant').mockResolvedValue({ ...restaurant, location: stored })
    vi.spyOn(restaurantsApi, 'listReviews').mockResolvedValue({ items: [], nextCursor: null })
    vi.spyOn(restaurantsApi, 'listRestaurants').mockResolvedValue({ items: [], nextCursor: null })
    saveDraft(`restaurant:${restaurant.id}`, { name: 'Renamed', description: 'Draft', tags: [], location: canSole.location })
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
    const w = mount(RestaurantFormPage, { props: { restaurantId: restaurant.id }, global: { plugins: [router] }, attachTo: document.body })
    await flushPromises()
    return w
  }

  it('keeps the drafted location of a restaurant without one', async () => {
    const w = await mountWithDraft(null)
    expect((w.get('input[name=name]').element as HTMLInputElement).value).toBe('Old Place')
    expect(w.text()).toContain('Selected location: Carrer de Sant Carles 4, 08003 Barcelona')
    w.unmount()
  })

  it('keeps the stored location when it is locked', async () => {
    const w = await mountWithDraft(house.location)
    expect((w.get('input[name=address]').element as HTMLInputElement).value).toBe(house.location.address)
    w.unmount()
  })
})
