import type { Place, Restaurant } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import LocationField from '@/components/app/LocationField.vue'
import RestaurantDetails from '@/components/app/RestaurantDetails.vue'
import { restaurantsApi } from '@/lib/api'
import { normalizeWebsite, priceLevelLabel, telLink, websiteHost } from '@/lib/restaurantDetails'
import { clearSession, saveSession } from '@/lib/session'
import { checkOpeningHours, checkPhone, checkWebsite } from '@/lib/validation'
import RestaurantFormPage from '@/pages/RestaurantFormPage.vue'
import { expectAccessible } from './axe'

const user = { id: 'u5n1c8r3t6y0b2e', email: 'ana@example.com', name: 'Ana', approved: true, created: '2024-11-04T01:26:55.002Z', updated: '2024-11-04T01:26:55.002Z' }
const restaurant: Restaurant = {
  id: 'k3x9m2p7q1w8z4a',
  name: 'Can Solé',
  location: { latitude: 41.3766, longitude: 2.1893, address: 'Carrer de Sant Carles 4, 08003 Barcelona' },
  website: 'https://www.cansole.example/menu',
  phone: '+34 932 21 50 12',
  openingHours: 'Tu-Sa 13:00-16:00\nSu 13:00-16:00',
  priceLevel: 3,
  creator: { id: user.id, name: 'Ana' },
  reviewCount: 0,
  rating: null,
  cover: null,
  favorite: false,
  created: user.created,
  updated: user.updated,
}

describe('restaurant details helpers', () => {
  it('adds https to a website without a scheme', () => {
    expect(normalizeWebsite('  cansole.example ')).toBe('https://cansole.example')
    expect(normalizeWebsite('HTTP://cansole.example/A')).toBe('http://cansole.example/A')
    expect(normalizeWebsite('')).toBe('')
    expect(normalizeWebsite('javascript:alert(1)')).toBe('javascript:alert(1)')
    expect(normalizeWebsite('www.cansole.example:8080/menu')).toBe('https://www.cansole.example:8080/menu')
    expect(normalizeWebsite('localhost:3000')).toBe('https://localhost:3000')
  })

  it('checks the details with the rules of the API', () => {
    expect(checkWebsite('')).toBe('')
    expect(checkWebsite('https://cansole.example/a?b=c')).toBe('')
    for (const bad of ['javascript:alert(1)', 'https:///a', 'https://can sole.example', 'https://a.example/​', `https://${'a'.repeat(2048)}`]) {
      expect(checkWebsite(bad)).not.toBe('')
    }
    expect(checkPhone('')).toBe('')
    expect(checkPhone(' (030) 123-45/67 ')).toBe('')
    for (const bad of ['12', 'call 932', '1'.repeat(21)]) {
      expect(checkPhone(bad)).not.toBe('')
    }
    expect(checkOpeningHours('Mo-Fr 12:00-15:00\n\tSa 12:00-23:00')).toBe('')
    expect(checkOpeningHours('a'.repeat(501))).toBe('Use at most 500 characters.')
    expect(checkOpeningHours('Mo​')).not.toBe('')
  })

  it('makes the link texts', () => {
    expect(websiteHost('https://www.cansole.example/menu')).toBe('cansole.example')
    expect(websiteHost('not a url')).toBe('not a url')
    expect(telLink('+34 (932) 21-50.12')).toBe('tel:+34932215012')
    expect(priceLevelLabel(1)).toBe('Inexpensive')
    expect(priceLevelLabel(4)).toBe('Very expensive')
    expect(priceLevelLabel(undefined)).toBe('')
  })
})

describe('RestaurantDetails', () => {
  it('shows the details with links', async () => {
    const w = mount(RestaurantDetails, { props: { restaurant }, attachTo: document.body })
    const website = w.get('a[target=_blank]')
    expect(website.attributes('href')).toBe('https://www.cansole.example/menu')
    expect(website.attributes('rel')).toBe('nofollow ugc noopener noreferrer')
    expect(website.text()).toBe('cansole.example, opens in a new tab')
    expect(w.get('a[href^="tel:"]').attributes('href')).toBe('tel:+34932215012')
    expect(w.text()).toContain('Tu-Sa 13:00-16:00\nSu 13:00-16:00')
    expect(w.text()).toContain('Price level:Expensive')
    await expectAccessible(w.element)
    w.unmount()
  })

  it('shows nothing without details', () => {
    const w = mount(RestaurantDetails, {
      props: { restaurant: { ...restaurant, website: undefined, phone: undefined, openingHours: undefined, priceLevel: undefined } },
    })
    expect(w.find('dl').exists()).toBe(false)
  })
})

describe('RestaurantFormPage details', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    clearSession()
    document.body.innerHTML = ''
  })

  async function mountForm(restaurantId?: string) {
    saveSession({ token: 't', expires: '2099-01-01T00:00:00.000Z', user })
    vi.spyOn(restaurantsApi, 'listRestaurants').mockResolvedValue({ items: [], nextCursor: null })
    vi.spyOn(restaurantsApi, 'listReviews').mockResolvedValue({ items: [], nextCursor: null })
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
    const w = mount(RestaurantFormPage, { props: { restaurantId }, global: { plugins: [router] }, attachTo: document.body })
    await flushPromises()
    return w
  }

  it('fills the empty details from the selected place and sends them', async () => {
    const create = vi.spyOn(restaurantsApi, 'createRestaurant').mockResolvedValue(restaurant)
    const w = await mountForm()
    await w.get('input[name=phone]').setValue('+34 600 00 00 00')
    const place: Place = {
      name: 'Can Solé',
      location: restaurant.location!,
      website: 'https://cansole.example',
      phone: '+34 932 21 50 12',
      openingHours: 'Tu-Sa 13:00-16:00',
    }
    w.getComponent(LocationField).vm.$emit('selected', place)
    w.getComponent(LocationField).vm.$emit('update:modelValue', place.location)
    await flushPromises()
    expect((w.get('input[name=name]').element as HTMLInputElement).value).toBe('Can Solé')
    expect((w.get('input[name=website]').element as HTMLInputElement).value).toBe('https://cansole.example')
    expect((w.get('input[name=phone]').element as HTMLInputElement).value).toBe('+34 600 00 00 00')
    expect((w.get('textarea[name=opening-hours]').element as HTMLTextAreaElement).value).toBe('Tu-Sa 13:00-16:00')
    await expectAccessible(w.element)

    await w.get('form').trigger('submit')
    await flushPromises()
    expect(create).toHaveBeenCalledWith({
      restaurantCreate: expect.objectContaining({
        website: 'https://cansole.example',
        phone: '+34 600 00 00 00',
        openingHours: 'Tu-Sa 13:00-16:00',
        priceLevel: undefined,
      }),
    })
  })

  it('shows the errors of the details and focuses the first', async () => {
    const create = vi.spyOn(restaurantsApi, 'createRestaurant')
    const w = await mountForm()
    w.getComponent(LocationField).vm.$emit('update:modelValue', restaurant.location)
    await w.get('input[name=name]').setValue('Can Solé')
    await w.get('input[name=website]').setValue('ftp://cansole.example')
    await w.get('input[name=phone]').setValue('call us')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(create).not.toHaveBeenCalled()
    expect(w.text()).toContain('Enter a web address, for example https://cansole.example.')
    expect(w.text()).toContain('Enter a phone number with 3 to 20 digits')
    expect(document.activeElement?.id).toBe('restaurant-website')
  })

  it('removes the cleared details of a restaurant', async () => {
    vi.spyOn(restaurantsApi, 'getRestaurant').mockResolvedValue(restaurant)
    const update = vi.spyOn(restaurantsApi, 'updateRestaurant').mockResolvedValue(restaurant)
    const w = await mountForm(restaurant.id)
    expect((w.get('input[name=website]').element as HTMLInputElement).value).toBe(restaurant.website)
    expect(w.get('#restaurant-price_level').text()).toBe('Expensive')
    await w.get('input[name=website]').setValue('')
    await w.get('input[name=phone]').setValue('')
    await w.get('textarea[name=opening-hours]').setValue(`${'a'.repeat(498)}\n\n\n`)
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(update).toHaveBeenCalledWith({
      restaurantId: restaurant.id,
      restaurantPatch: expect.objectContaining({ website: null, phone: null, openingHours: 'a'.repeat(498), priceLevel: 3 }),
    })
  })
})
