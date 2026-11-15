import type { Restaurant } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import LocationMenu from '@/components/app/LocationMenu.vue'
import RestaurantCard from '@/components/app/RestaurantCard.vue'
import { defaultCenter } from '@/lib/places'
import {
  currentPosition,
  distanceMeters,
  formatDistance,
  homePoint,
  loadPreferredLocation,
  preferredLocation,
  setPreferredLocation,
} from '@/lib/preferredLocation'
import { expectAccessible } from './axe'

const barceloneta = { latitude: 41.3766, longitude: 2.1893 }

function stubGeolocation(result: { latitude: number; longitude: number } | { code: number }) {
  vi.stubGlobal('navigator', {
    ...navigator,
    userAgent: navigator.userAgent,
    geolocation: {
      getCurrentPosition(ok: (p: unknown) => void, fail: (e: unknown) => void) {
        if ('code' in result) {
          fail({ code: result.code, PERMISSION_DENIED: 1 })
        } else {
          ok({ coords: result })
        }
      },
    },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
  setPreferredLocation(null)
})

describe('preferred location', () => {
  it('stays in the storage of the browser', () => {
    setPreferredLocation({ ...barceloneta, label: 'Barceloneta', radius: 5000 })
    expect(homePoint()).toEqual(barceloneta)
    preferredLocation.value = null
    loadPreferredLocation()
    expect(preferredLocation.value).toEqual({ ...barceloneta, label: 'Barceloneta', radius: 5000 })

    setPreferredLocation(null)
    loadPreferredLocation()
    expect(preferredLocation.value).toBeNull()
    expect(homePoint()).toEqual(defaultCenter)
  })

  it('ignores a stored value that is not a location', () => {
    for (const bad of ['{', '"text"', '{"latitude":91,"longitude":2,"label":"A"}', '{"latitude":1,"longitude":"2","label":"A"}', '{"latitude":1,"longitude":2}']) {
      localStorage.setItem('reviews-location', bad)
      loadPreferredLocation()
      expect(preferredLocation.value).toBeNull()
    }
    // A radius that the API does not allow becomes any distance.
    localStorage.setItem('reviews-location', '{"latitude":1,"longitude":2,"label":"A","radius":999999}')
    loadPreferredLocation()
    expect(preferredLocation.value).toEqual({ latitude: 1, longitude: 2, label: 'A', radius: null })
  })

  it('computes and formats a distance', () => {
    expect(distanceMeters(barceloneta, barceloneta)).toBe(0)
    // Barcelona to Madrid is about 507 km.
    expect(Math.round(distanceMeters(barceloneta, { latitude: 40.4168, longitude: -3.7038 }) / 1000)).toBe(507)
    expect(Math.round(distanceMeters({ latitude: 0, longitude: 179.9999 }, { latitude: 0, longitude: -179.9999 }))).toBe(22)
    expect(formatDistance(40)).toBe('Less than 100 m')
    expect(formatDistance(430)).toBe('450 m')
    expect(formatDistance(980)).toBe('1.0 km')
    expect(formatDistance(1240)).toBe('1.2 km')
    expect(formatDistance(9960)).toBe('10 km')
    expect(formatDistance(12400)).toBe('12 km')
  })

  it('rounds the position of the device', async () => {
    stubGeolocation({ latitude: 41.376612, longitude: 2.189349 })
    await expect(currentPosition()).resolves.toEqual({ latitude: 41.377, longitude: 2.189 })
    stubGeolocation({ code: 1 })
    await expect(currentPosition()).rejects.toBe('denied')
    stubGeolocation({ code: 2 })
    await expect(currentPosition()).rejects.toBe('unavailable')
    vi.stubGlobal('isSecureContext', false)
    await expect(currentPosition()).rejects.toBe('insecure')
  })
})

describe('RestaurantCard', () => {
  it('shows the distance from a point', () => {
    const restaurant = {
      id: 'r1',
      name: 'Can Sole',
      tags: [],
      reviewCount: 0,
      rating: null,
      cover: null,
      favorite: false,
      location: { latitude: 41.39, longitude: 2.16, address: 'Carrer Nou 1' },
    } as unknown as Restaurant
    const w = mount(RestaurantCard, { props: { restaurant, from: barceloneta } })
    expect(w.text()).toContain('Distance: 2.9 km')
    expect(mount(RestaurantCard, { props: { restaurant } }).text()).not.toContain('Distance')
  })
})

describe('LocationMenu', () => {
  function open() {
    const el = document.createElement('div')
    document.body.appendChild(el)
    const w = mount(LocationMenu, { attachTo: el })
    return w.get('button').trigger('click').then(() => w)
  }
  const button = (name: string) => [...document.body.querySelectorAll('button')].find((b) => b.textContent?.trim() === name)

  it('sets the location of the device', async () => {
    stubGeolocation({ latitude: 41.376612, longitude: 2.189349 })
    const w = await open()
    expect(w.get('button').text()).toBe('Set location')
    await expectAccessible(document.body)
    button('Use my current location')!.click()
    await flushPromises()
    expect(preferredLocation.value).toEqual({ latitude: 41.377, longitude: 2.189, label: 'Current location', radius: 10000 })
    expect(w.get('button').attributes('aria-label')).toBe('Location: Current location')

    await w.get('button').trigger('click')
    button('Clear location')!.click()
    await flushPromises()
    expect(preferredLocation.value).toBeNull()
  })

  it('ignores a position that comes after another location', async () => {
    let give: (p: unknown) => void = () => {}
    vi.stubGlobal('navigator', { geolocation: { getCurrentPosition: (ok: (p: unknown) => void) => (give = ok) } })
    await open()
    button('Use my current location')!.click()
    setPreferredLocation({ latitude: 40.4168, longitude: -3.7038, label: 'Madrid', radius: 5000 })
    await flushPromises()
    // A click on Clear location is a new choice of the person.
    button('Clear location')!.click()
    give({ coords: barceloneta })
    await flushPromises()
    expect(preferredLocation.value).toBeNull()
  })

  it('tells why the browser gives no location', async () => {
    stubGeolocation({ code: 1 })
    await open()
    button('Use my current location')!.click()
    await flushPromises()
    expect(document.body.querySelector('[role=alert]')?.textContent).toContain('Allow the location for this site')
    expect(preferredLocation.value).toBeNull()
  })
})
