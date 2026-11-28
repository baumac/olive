import type { Restaurant, SavedList } from '@reviews/sdk'
import { FetchError } from '@reviews/sdk'
import { DOMWrapper, flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import FavoriteButton from '@/components/app/FavoriteButton.vue'
import JournalLists from '@/components/app/JournalLists.vue'
import SaveToListMenu from '@/components/app/SaveToListMenu.vue'
import { useListFilter } from '@/composables/useListFilter'
import { useListQuery } from '@/composables/useListQuery'
import { listsApi, recipesApi, restaurantsApi } from '@/lib/api'
import { forgetOwnLists } from '@/lib/lists'
import { resetMapConfig } from '@/lib/mapConfig'
import { clearSession, saveSession } from '@/lib/session'
import ListPage from '@/pages/ListPage.vue'
import { expectAccessible } from './axe'
import { rejectWith } from './problem'

const ana = { id: 'u5n1c8r3t6y0b2e', email: 'ana@example.com', name: 'Ana', approved: true, created: '2024-11-04T01:26:55.002Z', updated: '2024-11-04T01:26:55.002Z' }

function signIn() {
  saveSession({ expires: '2099-01-01T00:00:00.000Z', user: ana })
}

function list(kind: 'favorites' | 'custom', name: string, restaurantCount = 1): SavedList {
  return {
    id: kind === 'favorites' ? 'f1a6c3e8g5i2k9m' : 'l7d2f9h4j1k8m3p',
    name,
    kind,
    creator: { id: ana.id, name: 'Ana' },
    restaurantCount,
    recipeCount: 0,
    pantryItemCount: 0,
    created: '2024-11-04T01:26:55.002Z',
    updated: '2024-11-04T01:26:55.002Z',
  }
}

const restaurant: Restaurant = {
  id: 'k3x9m2p7q1w8z4a',
  name: "Luigi's Trattoria",
  location: { latitude: 41.3766, longitude: 2.1893, address: 'Carrer de Sant Carles 4' },
  creator: { id: ana.id, name: 'Ana' },
  reviewCount: 0,
  rating: null,
  cover: null,
  favorite: false,
  ownReviewCount: 0,
  created: '2024-11-04T01:26:55.002Z',
  updated: '2024-11-04T01:26:55.002Z',
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

afterEach(() => {
  vi.restoreAllMocks()
  clearSession()
  forgetOwnLists()
})

describe('FavoriteButton', () => {
  async function mountButton(favorite: boolean) {
    const router = await routerAt('/restaurants')
    const w = mount(FavoriteButton, {
      props: { restaurantId: restaurant.id, name: restaurant.name, favorite },
      global: { plugins: [router] },
      attachTo: attach(),
    })
    return { w, router }
  }

  it('sends a signed out user to the sign in page', async () => {
    const add = vi.spyOn(listsApi, 'addListRestaurant')
    const { w, router } = await mountButton(false)
    await w.get('button').trigger('click')
    await flushPromises()
    expect(add).not.toHaveBeenCalled()
    expect(router.currentRoute.value.path).toBe('/sign-in')
    expect(router.currentRoute.value.query.redirect).toBe('/restaurants')
  })

  it('adds and removes a favorite', async () => {
    signIn()
    const add = vi.spyOn(listsApi, 'addListRestaurant').mockResolvedValue()
    const remove = vi.spyOn(listsApi, 'removeListRestaurant').mockResolvedValue()
    const { w } = await mountButton(false)
    const button = w.get('button')
    expect(button.attributes('aria-label')).toBe("Favorite Luigi's Trattoria")
    expect(button.attributes('aria-pressed')).toBe('false')
    await button.trigger('click')
    await flushPromises()
    expect(add).toHaveBeenCalledWith({ listRef: 'favorites', restaurantId: restaurant.id })
    expect(button.attributes('aria-pressed')).toBe('true')
    expect(w.emitted('change')).toEqual([[true]])
    await button.trigger('click')
    await flushPromises()
    expect(remove).toHaveBeenCalledWith({ listRef: 'favorites', restaurantId: restaurant.id })
    expect(button.attributes('aria-pressed')).toBe('false')
    await expectAccessible(w.element)
  })

  it('changes back when the request fails', async () => {
    signIn()
    vi.spyOn(listsApi, 'removeListRestaurant').mockRejectedValue(rejectWith(500))
    const { w } = await mountButton(true)
    await w.get('button').trigger('click')
    await flushPromises()
    expect(w.get('button').attributes('aria-pressed')).toBe('true')
    expect(w.emitted('change')).toBeUndefined()
  })

  it('shows no favorite after a sign out', async () => {
    const { w } = await mountButton(true)
    expect(w.get('button').attributes('aria-pressed')).toBe('false')
  })
})

describe('FavoriteButton on a reused card', () => {
  it('shows the state of the new restaurant', async () => {
    signIn()
    vi.spyOn(listsApi, 'addListRestaurant').mockResolvedValue()
    const w = mount(FavoriteButton, {
      props: { restaurantId: restaurant.id, name: restaurant.name, favorite: false },
      global: { plugins: [await routerAt('/restaurants')] },
      attachTo: attach(),
    })
    await w.get('button').trigger('click')
    await flushPromises()
    expect(w.get('button').attributes('aria-pressed')).toBe('true')
    await w.setProps({ restaurantId: 'r9z8y7x6w5v4u3t', name: 'Can Sole', favorite: false })
    expect(w.get('button').attributes('aria-pressed')).toBe('false')
  })
})

describe('SaveToListMenu', () => {
  const body = () => document.body
  const checkbox = (name: string) =>
    Array.from(body().querySelectorAll('label'))
      .find((l) => l.textContent?.trim() === name)
      ?.querySelector<HTMLInputElement>('input[type=checkbox]')
  const alertText = () => Array.from(body().querySelectorAll('[role=alert]')).map((a) => a.textContent?.trim()).join(' ')

  async function openMenu() {
    const w = mount(SaveToListMenu, {
      props: { restaurantId: restaurant.id },
      global: { plugins: [await routerAt('/restaurants/k3x9m2p7q1w8z4a')] },
      attachTo: attach(),
    })
    menu = w
    await w.get('button').trigger('click')
    await flushPromises()
    return w
  }

  async function createList(name: string) {
    await new DOMWrapper(body().querySelector('#save-to-list-name')!).setValue(name)
    await new DOMWrapper(body().querySelector('form')!).trigger('submit')
    await flushPromises()
  }

  let menu: ReturnType<typeof mount> | undefined
  afterEach(() => {
    menu?.unmount()
    menu = undefined
  })

  it('shows the custom lists and changes one', async () => {
    signIn()
    const load = vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({
      items: [list('favorites', 'Favorites'), { ...list('custom', 'Date night'), hasRestaurant: false }],
    })
    const add = vi.spyOn(listsApi, 'addListRestaurant').mockResolvedValue()
    await openMenu()
    expect(load).toHaveBeenCalledWith({ reviewerId: ana.id, restaurantEq: restaurant.id })
    expect(checkbox('Favorites')).toBeUndefined()
    const box = checkbox('Date night')!
    expect(box.checked).toBe(false)
    box.checked = true
    box.dispatchEvent(new Event('change'))
    await flushPromises()
    expect(add).toHaveBeenCalledWith({ listRef: 'l7d2f9h4j1k8m3p', restaurantId: restaurant.id })
    expect(checkbox('Date night')!.checked).toBe(true)
  })

  it('changes a checkbox back when the request fails', async () => {
    signIn()
    vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [{ ...list('custom', 'Date night'), hasRestaurant: true }] })
    vi.spyOn(listsApi, 'removeListRestaurant').mockRejectedValue(rejectWith(500))
    await openMenu()
    const box = checkbox('Date night')!
    box.checked = false
    box.dispatchEvent(new Event('change'))
    await flushPromises()
    expect(checkbox('Date night')!.checked).toBe(true)
    expect(alertText()).toBe('Date night did not change. Select it to try again.')
  })

  it('creates a list with the restaurant', async () => {
    signIn()
    vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [] })
    const create = vi.spyOn(listsApi, 'createList').mockResolvedValue(list('custom', 'Date night', 0))
    const add = vi.spyOn(listsApi, 'addListRestaurant').mockResolvedValue()
    await openMenu()
    await createList(' Date night ')
    expect(create).toHaveBeenCalledWith({ savedListCreate: { name: 'Date night' } })
    expect(add).toHaveBeenCalledWith({ listRef: 'l7d2f9h4j1k8m3p', restaurantId: restaurant.id })
    expect(checkbox('Date night')!.checked).toBe(true)
    expect(body().querySelector<HTMLInputElement>('#save-to-list-name')!.value).toBe('')
  })

  it('tells that a list is full', async () => {
    signIn()
    vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [] })
    vi.spyOn(listsApi, 'createList').mockResolvedValue(list('custom', 'Date night', 0))
    vi.spyOn(listsApi, 'addListRestaurant').mockRejectedValue(rejectWith(409, { code: 'list_restaurant_limit' }))
    await openMenu()
    await createList('Date night')
    expect(alertText()).toBe('This list is full. Remove a restaurant first.')
  })

  it('keeps a new list when the add fails', async () => {
    signIn()
    vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [] })
    vi.spyOn(listsApi, 'createList').mockResolvedValue(list('custom', 'Date night', 0))
    vi.spyOn(listsApi, 'addListRestaurant').mockRejectedValue(rejectWith(500))
    await openMenu()
    await createList('Date night')
    expect(checkbox('Date night')!.checked).toBe(false)
    expect(alertText()).toBe('Date night did not change. Select it to try again.')
    expect(body().querySelector<HTMLInputElement>('#save-to-list-name')!.value).toBe('')
  })

  it('sends a signed out user to the sign in page', async () => {
    const load = vi.spyOn(listsApi, 'listReviewerLists')
    const router = await routerAt('/restaurants/k3x9m2p7q1w8z4a')
    const w = mount(SaveToListMenu, { props: { restaurantId: restaurant.id }, global: { plugins: [router] }, attachTo: attach() })
    await w.get('button').trigger('click')
    await flushPromises()
    expect(load).not.toHaveBeenCalled()
    expect(router.currentRoute.value.path).toBe('/sign-in')
  })
})

describe('JournalLists', () => {
  async function mountLists(own: boolean) {
    const w = mount(JournalLists, { props: { reviewerId: ana.id, own }, global: { plugins: [await routerAt('/')] }, attachTo: attach() })
    await flushPromises()
    return w
  }

  it('links the lists and hides the form on the page of another user', async () => {
    vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [list('favorites', 'Favorites', 2), list('custom', 'Date night')] })
    vi.spyOn(restaurantsApi, 'listRestaurants').mockResolvedValue({ items: [], nextCursor: null })
    const w = await mountLists(false)
    expect(w.get('a[href="/lists/f1a6c3e8g5i2k9m"]').text()).toContain('See all 2')
    const links = w.get('section[aria-labelledby=lists-heading]').findAll('a')
    expect(links.map((a) => a.attributes('href'))).toEqual(['/lists/f1a6c3e8g5i2k9m', '/lists/l7d2f9h4j1k8m3p'])
    expect(links[0]!.text()).toContain('Favorites')
    expect(links[0]!.text()).toContain('2 restaurants')
    expect(links[1]!.text()).toContain('1 restaurant')
    expect(w.find('form').exists()).toBe(false)
    await expectAccessible(w.element)
  })

  it('creates a list after the favorites list', async () => {
    vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [list('favorites', 'Favorites')] })
    const create = vi.spyOn(listsApi, 'createList').mockResolvedValue(list('custom', 'Date night', 0))
    const w = await mountLists(true)
    await w.get('input').setValue(' Date night ')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(create).toHaveBeenCalledWith({ savedListCreate: { name: 'Date night' } })
    expect(w.get('section[aria-labelledby=lists-heading]').findAll('a').map((a) => a.attributes('href'))).toEqual(['/lists/f1a6c3e8g5i2k9m', '/lists/l7d2f9h4j1k8m3p'])
    expect((w.get('input').element as HTMLInputElement).value).toBe('')
    await expectAccessible(w.element)
  })

  it('tells that the name is taken', async () => {
    vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [list('favorites', 'Favorites')] })
    vi.spyOn(listsApi, 'createList').mockRejectedValue(rejectWith(409, { code: 'list_name_taken' }))
    const w = await mountLists(true)
    await w.get('input').setValue('favorites')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(w.get('[role=alert]').text()).toBe('You have a list with this name.')
  })
})

describe('ListPage', () => {
  async function mountPage(data: SavedList, items: Restaurant[] = [restaurant]) {
    vi.spyOn(listsApi, 'getList').mockResolvedValue(data)
    vi.spyOn(restaurantsApi, 'listRestaurants').mockResolvedValue({ items, nextCursor: null })
    vi.spyOn(recipesApi, 'listRecipes').mockResolvedValue({ items: [], nextCursor: null })
    const router = await routerAt(`/lists/${data.id}`)
    const w = mount(ListPage, { props: { listId: data.id }, global: { plugins: [router] }, attachTo: attach() })
    await flushPromises()
    return { w, router }
  }
  const buttons = (w: Awaited<ReturnType<typeof mountPage>>['w']) => w.findAll('button').map((b) => b.text())

  it('shows a list of another user without the owner actions', async () => {
    const { w } = await mountPage(list('custom', 'Date night'))
    expect(w.get('h1').text()).toBe('Date night')
    expect(w.text()).toContain('A list by Ana')
    expect(w.text()).toContain('1 restaurant ·')
    expect(w.get('h3 a').text()).toBe("Luigi's Trattoria")
    expect(buttons(w)).not.toContain('Rename list')
    expect(buttons(w)).not.toContain('Delete list')
    expect(buttons(w)).not.toContain('Remove')
    expect(document.title).toBe('Date night by Ana · Olive')
    await expectAccessible(w.element)
  })

  it('lets the owner rename a list and remove a restaurant', async () => {
    signIn()
    const { w } = await mountPage(list('custom', 'Date night'))
    expect(buttons(w)).toEqual(expect.arrayContaining(['Rename list', 'Delete list', 'Remove']))

    const update = vi.spyOn(listsApi, 'updateList').mockResolvedValue(list('custom', 'Dinner'))
    await w.findAll('button').find((b) => b.text() === 'Rename list')!.trigger('click')
    await w.get('#list-name').setValue('Dinner')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(update).toHaveBeenCalledWith({ listId: 'l7d2f9h4j1k8m3p', savedListPatch: { name: 'Dinner' } })
    expect(w.get('h1').text()).toBe('Dinner')
    expect(w.find('form').exists()).toBe(false)

    const remove = vi.spyOn(listsApi, 'removeListRestaurant').mockResolvedValue()
    await w.get(`button[aria-label="Remove Luigi's Trattoria from the list"]`).trigger('click')
    await flushPromises()
    expect(remove).toHaveBeenCalledWith({ listRef: 'l7d2f9h4j1k8m3p', restaurantId: restaurant.id })
    expect(w.find('h3').exists()).toBe(false)
    expect(w.text()).toContain('No restaurants yet')
  })

  it('keeps the favorites list fixed for its owner', async () => {
    signIn()
    const { w } = await mountPage(list('favorites', 'Favorites'), [{ ...restaurant, favorite: true }])
    expect(buttons(w)).not.toContain('Rename list')
    expect(buttons(w)).not.toContain('Delete list')
    expect(buttons(w)).not.toContain('Remove')

    vi.spyOn(listsApi, 'removeListRestaurant').mockResolvedValue()
    await w.get('button[aria-pressed=true]').trigger('click')
    await flushPromises()
    expect(w.find('h3').exists()).toBe(false)
  })

  it('links the restaurants page with the list filter', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ tiles: 'https://t/{z}/{x}/{y}.png' }))))
    resetMapConfig()
    const { w } = await mountPage(list('custom', 'Date night'))
    vi.unstubAllGlobals()
    resetMapConfig()
    const links = Object.fromEntries(w.findAll('a').map((a) => [a.text(), a.attributes('href')]))
    expect(links['Sort and filter the restaurants']).toBe('/restaurants?list=l7d2f9h4j1k8m3p')
    expect(links['Map']).toBe('/restaurants?list=l7d2f9h4j1k8m3p&view=map')
  })

  it('has no filter links for an empty list', async () => {
    const { w } = await mountPage(list('custom', 'Date night', 0), [])
    expect(w.text()).not.toContain('Sort and filter')
    expect(w.find('nav[aria-label=View]').exists()).toBe(false)
  })

  it('shows the not found page for an unknown list', async () => {
    vi.spyOn(listsApi, 'getList').mockRejectedValue(rejectWith(404))
    vi.spyOn(restaurantsApi, 'listRestaurants').mockRejectedValue(rejectWith(404))
    vi.spyOn(recipesApi, 'listRecipes').mockRejectedValue(rejectWith(404))
    const w = mount(ListPage, { props: { listId: 'aaaaaaaaaaaaaaa' }, global: { plugins: [await routerAt('/lists/aaaaaaaaaaaaaaa')] }, attachTo: attach() })
    await flushPromises()
    expect(w.text()).toContain('We cannot find this page')
  })
})

describe('useListFilter', () => {
  async function filterAt(path: string) {
    const router = await routerAt(path)
    let filter!: ReturnType<typeof useListFilter>
    let query!: ReturnType<typeof useListQuery>
    mount(
      defineComponent({
        setup() {
          filter = useListFilter()
          query = useListQuery(() => (filter.def.value ? [filter.def.value] : []), [], '-overall')
          return () => null
        },
      }),
      { global: { plugins: [router] } },
    )
    await flushPromises()
    return { def: filter.def, favoritesId: filter.favoritesId, query }
  }

  it('has no filter for a signed out visitor without a list in the URL', async () => {
    const { def } = await filterAt('/restaurants')
    expect(def.value).toBeNull()
  })

  it('has the lists of the signed in user', async () => {
    signIn()
    const lists = vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [list('favorites', 'Favorites'), list('custom', 'Date night')] })
    const { def, query, favoritesId } = await filterAt('/restaurants?list=f1a6c3e8g5i2k9m')
    expect(lists).toHaveBeenCalledWith({ reviewerId: ana.id })
    expect(favoritesId.value).toBe('f1a6c3e8g5i2k9m')
    expect(def.value?.options).toEqual([
      { value: 'f1a6c3e8g5i2k9m', label: 'Favorites' },
      { value: 'l7d2f9h4j1k8m3p', label: 'Date night' },
    ])
    expect(query.filters.value).toEqual({ list: 'f1a6c3e8g5i2k9m' })
    expect(def.value?.chip('f1a6c3e8g5i2k9m')).toBe('List: Favorites')
  })

  it('keeps the list of a link before and after it loads', async () => {
    let answer!: (l: SavedList) => void
    vi.spyOn(listsApi, 'getList').mockReturnValue(new Promise((resolve) => (answer = resolve)))
    const { def, query } = await filterAt('/restaurants?list=l7d2f9h4j1k8m3p')
    expect(query.filters.value).toEqual({ list: 'l7d2f9h4j1k8m3p' })
    expect(def.value?.chip('l7d2f9h4j1k8m3p')).toBe('List: A list')
    answer({ ...list('custom', 'Date night'), creator: { id: 'u2g6k0m4o8q1s5w', name: 'Joe' } })
    await flushPromises()
    expect(def.value?.chip('l7d2f9h4j1k8m3p')).toBe('List: Date night by Joe')
  })

  it('gets the lists of the user one time until a list changes', async () => {
    signIn()
    const lists = vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [list('favorites', 'Favorites')] })
    await filterAt('/restaurants')
    await filterAt('/restaurants')
    expect(lists).toHaveBeenCalledTimes(1)
    forgetOwnLists()
    await filterAt('/restaurants')
    expect(lists).toHaveBeenCalledTimes(2)
  })

  it('names a list that does not exist', async () => {
    vi.spyOn(listsApi, 'getList').mockRejectedValue(rejectWith(404))
    const { def } = await filterAt('/restaurants?list=aaaaaaaaaaaaaaa')
    expect(def.value?.chip('aaaaaaaaaaaaaaa')).toBe('List: Unknown list')
  })

  it('tries a failed list again', async () => {
    vi.useFakeTimers()
    try {
      const get = vi.spyOn(listsApi, 'getList').mockRejectedValueOnce(new FetchError(new Error('offline'))).mockResolvedValue(list('custom', 'Date night'))
      const { def } = await filterAt('/restaurants?list=l7d2f9h4j1k8m3p')
      expect(def.value?.chip('l7d2f9h4j1k8m3p')).toBe('List: A list')
      await vi.advanceTimersByTimeAsync(2000)
      await flushPromises()
      expect(get).toHaveBeenCalledTimes(2)
      expect(def.value?.chip('l7d2f9h4j1k8m3p')).toBe('List: Date night by Ana')
    } finally {
      vi.useRealTimers()
    }
  })

  it('ignores a malformed list ID', async () => {
    const get = vi.spyOn(listsApi, 'getList')
    const { def } = await filterAt('/restaurants?list=BAD')
    expect(def.value).toBeNull()
    expect(get).not.toHaveBeenCalled()
  })
})
