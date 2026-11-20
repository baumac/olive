import type { Restaurant, RestaurantPhoto } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import RestaurantCard from '@/components/app/RestaurantCard.vue'
import PhotoGallery from '@/components/app/PhotoGallery.vue'
import { CoverPhoto } from '@/components/ui/cover-photo'
import { restaurantsApi } from '@/lib/api'
import { expectAccessible } from './axe'

function photo(id: string, review: string, name: string, alt: string | null = null): RestaurantPhoto {
  const base = `/v1/restaurants/k3x9m2p7q1w8z4a/reviews/${review}/photos/${id}`
  return {
    id,
    alt,
    width: 800,
    height: 600,
    urls: { full: `${base}/full`, thumb: `${base}/thumb` },
    created: '2024-11-06T19:04:30.811Z',
    review,
    creator: { id: `u${review}`, name },
  }
}

const cover = photo('p4h7j1l5n9q2s6v', 'r1b8c5e2g9i6k3m', 'Ana', 'A plate of carbonara')
const second = photo('p1a2b3c4d5e6f7g', 'r9z8y7x6w5v4u3t', 'Joe')

function restaurantWith(c: RestaurantPhoto | null): Restaurant {
  return {
    id: 'k3x9m2p7q1w8z4a',
    name: 'Trattoria',
    location: null,
    creator: { id: 'u5n1c8r3t6y0b2e', name: 'Ana' },
    reviewCount: 2,
    rating: null,
    cover: c,
    favorite: false,
    ownReviewCount: 0,
    created: '2024-11-06T19:02:11.120Z',
    updated: '2024-11-06T19:02:11.120Z',
  }
}

function attach() {
  const el = document.createElement('div')
  document.body.appendChild(el)
  return el
}

async function routerAt(path: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
  await router.push(path)
  return router
}

describe('CoverPhoto', () => {
  it('shows the thumb in a card and the full image in the hero', () => {
    const card = mount(CoverPhoto, { props: { photo: cover, size: 'card' } })
    expect(card.get('img').attributes()).toMatchObject({ src: cover.urls.thumb, alt: '', loading: 'lazy', width: '800', height: '600' })
    const hero = mount(CoverPhoto, { props: { photo: cover, size: 'hero' } })
    expect(hero.get('img').attributes()).toMatchObject({
      src: cover.urls.full,
      srcset: `${cover.urls.thumb} 480w, ${cover.urls.full} 800w`,
    })
  })

  it('shows the placeholder without a photo and after a failed load', async () => {
    const empty = mount(CoverPhoto, { props: { photo: null, size: 'card' } })
    expect(empty.find('img').exists()).toBe(false)
    expect(empty.find('svg').exists()).toBe(true)
    const failed = mount(CoverPhoto, { props: { photo: cover, size: 'card' } })
    await failed.get('img').trigger('error')
    expect(failed.find('img').exists()).toBe(false)
  })
})

describe('RestaurantCard', () => {
  it('shows the cover above the name of the link', async () => {
    const w = mount(RestaurantCard, { props: { restaurant: restaurantWith(cover) }, global: { plugins: [await routerAt('/')] }, attachTo: attach() })
    expect(w.get('img').attributes('src')).toBe(cover.urls.thumb)
    expect(w.get('a').text()).toBe('Trattoria')
    await expectAccessible(w.element)
  })
})

describe('PhotoGallery', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    document.body.innerHTML = ''
  })

  it('shows nothing and loads nothing without a cover', async () => {
    const list = vi.spyOn(restaurantsApi, 'listRestaurantPhotos')
    const w = mount(PhotoGallery, { props: { parent: restaurantWith(null) }, global: { plugins: [await routerAt('/')] } })
    await flushPromises()
    expect(w.find('section').exists()).toBe(false)
    expect(list).not.toHaveBeenCalled()
  })

  it('lists the photos and links each photo to its review', async () => {
    const list = vi.spyOn(restaurantsApi, 'listRestaurantPhotos').mockResolvedValue({ items: [cover, second], nextCursor: null })
    const w = mount(PhotoGallery, { props: { parent: restaurantWith(cover) }, global: { plugins: [await routerAt('/')] }, attachTo: attach() })
    await flushPromises()
    expect(list).toHaveBeenCalledWith({ restaurantId: 'k3x9m2p7q1w8z4a', limit: 15, cursor: undefined })
    expect(w.get('h2').text()).toBe('Photos')
    const buttons = w.findAll('ul[aria-label="Photos of Trattoria"] button')
    expect(buttons.map((b) => b.attributes('aria-label'))).toEqual(['A plate of carbonara', 'Photo 2 of 2'])
    await expectAccessible(w.element)

    await buttons[1]!.trigger('click')
    await flushPromises()
    const dialog = document.querySelector('[role=dialog]') as HTMLElement
    const link = dialog.querySelector('a')
    expect(dialog.textContent).toContain('From the review by Joe')
    expect(link?.getAttribute('href')).toBe('/restaurants/k3x9m2p7q1w8z4a/reviews/r9z8y7x6w5v4u3t')
    await expectAccessible(dialog)
  })

  it('opens the cover before the gallery loads', async () => {
    vi.spyOn(restaurantsApi, 'listRestaurantPhotos').mockReturnValue(new Promise(() => {}))
    const w = mount(PhotoGallery, { props: { parent: restaurantWith(cover) }, global: { plugins: [await routerAt('/')] }, attachTo: attach() })
    ;(w.vm as unknown as { openCover: () => void }).openCover()
    await flushPromises()
    const dialog = document.querySelector('[role=dialog]') as HTMLElement
    expect(dialog.textContent).toContain('Photo 1 of 1')
    expect(dialog.textContent).toContain('From the review by Ana')
  })

  it('opens the photo of the cover when the gallery order changed', async () => {
    vi.spyOn(restaurantsApi, 'listRestaurantPhotos').mockResolvedValue({ items: [second, cover], nextCursor: null })
    const w = mount(PhotoGallery, { props: { parent: restaurantWith(cover) }, global: { plugins: [await routerAt('/')] }, attachTo: attach() })
    await flushPromises()
    ;(w.vm as unknown as { openCover: () => void }).openCover()
    await flushPromises()
    const dialog = document.querySelector('[role=dialog]') as HTMLElement
    expect(dialog.textContent).toContain('Photo 2 of 2')
    expect(dialog.querySelector('img')?.getAttribute('src')).toBe(cover.urls.full)
  })
})
