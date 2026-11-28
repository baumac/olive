import type { PantryItem, PantryItemReview, PantryItemVersion, TagSummary } from '@reviews/sdk'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import BottomNav from '@/components/app/BottomNav.vue'
import PantryItemCard from '@/components/app/PantryItemCard.vue'
import SaveToListMenu from '@/components/app/SaveToListMenu.vue'
import { listsApi, pantryApi, recipesApi, restaurantsApi, tagsApi } from '@/lib/api'
import { changedProperties, propertyLabel, undoChange, valueText } from '@/lib/history'
import { listCountText } from '@/lib/lists'
import { categoryLabel, pantryFilterParams, pantryItemMeta } from '@/lib/pantry'
import { pantryItemReviewFilterParams, reviewFilterDefsOf } from '@/lib/reviewFilters'
import { kindOf, kindOfQuery, parentNameOf, reviewPathOf } from '@/lib/reviewKinds'
import { overallOf, pantryItemScoreFields } from '@/lib/reviews'
import { clearSession, saveSession } from '@/lib/session'
import NewReviewPage from '@/pages/NewReviewPage.vue'
import PantryItemFormPage from '@/pages/PantryItemFormPage.vue'
import PantryItemPage from '@/pages/PantryItemPage.vue'
import PantryPage from '@/pages/PantryPage.vue'
import TagPage from '@/pages/TagPage.vue'
import { expectAccessible } from './axe'

const item: PantryItem = {
  id: 'p2k7m4q9s1v6x3z',
  name: 'Wehlener Sonnenuhr',
  category: 'wine',
  producer: 'Weingut Example',
  origin: 'Mosel, Germany',
  style: 'Riesling',
  description: 'Off-dry',
  tags: new Set(['white']),
  creator: { id: 'u5n1c8r3t6y0b2e', name: 'Ana' },
  reviewCount: 2,
  rating: { overall: 8.1, taste: 8.5, aroma: 8, texture: 7.5, value: 8.5 },
  cover: null,
  favorite: false,
  ownReviewCount: 1,
  created: '2026-11-23T18:20:11.410Z',
  updated: '2026-11-23T18:20:11.410Z',
}

const review: PantryItemReview = {
  id: 't4s8n2q6m0k3h7a',
  pantryItem: item.id,
  pantryItemName: item.name,
  date: '2026-11-23',
  overall: 8.25,
  taste: 9,
  aroma: 8,
  texture: 7,
  value: 9,
  title: 'A light white',
  creator: { id: 'u5n1c8r3t6y0b2e', name: 'Ana' },
  photos: [],
  created: '2026-11-23T19:02:11.120Z',
  updated: '2026-11-23T19:02:11.120Z',
}

const ana = { id: 'u5n1c8r3t6y0b2e', email: 'ana@example.com', name: 'Ana', approved: true, created: '2024-11-04T01:26:55.002Z', updated: '2024-11-04T01:26:55.002Z' }

function signIn() {
  saveSession({ expires: '2099-01-01T00:00:00.000Z', user: ana })
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

// The pages update after the API answers. The DOM of the setup goes away
// after each test. The pages must go first.
enableAutoUnmount(afterEach)

afterEach(() => {
  vi.restoreAllMocks()
  clearSession()
})

describe('pantry helpers', () => {
  it('names the categories and the details', () => {
    expect(categoryLabel('cheese')).toBe('Cheese')
    expect(categoryLabel('cider')).toBe('cider')
    expect(pantryItemMeta(item)).toBe('Wine · Weingut Example · Mosel, Germany')
    expect(pantryItemMeta({ category: 'beer' })).toBe('Beer')
  })

  it('changes the pantry filters to the parameters of the SDK', () => {
    expect(pantryFilterParams({ search: 'riesling', tags: ['white'], overall: '8', list: 'l7d2f9h4j1k8m3p' }, 'wine')).toEqual({
      search: 'riesling',
      tagsHasAny: ['white'],
      overall: { gte: 8 },
      categoryEq: 'wine',
      listEq: 'l7d2f9h4j1k8m3p',
    })
    expect(pantryItemReviewFilterParams({ aroma: '7' })).toMatchObject({ aroma: { gte: 7 }, texture: undefined })
    expect(reviewFilterDefsOf('pantryItem').map((d) => d.key)).toEqual(['search', 'overall', 'taste', 'aroma', 'texture', 'value'])
  })

  it('gives the overall of the four tasting scores', () => {
    expect(overallOf({ taste: 9, aroma: 8, texture: 7, value: 9 }, pantryItemScoreFields)).toBe(8.3)
    expect(overallOf({ taste: 9, aroma: 8, texture: null, value: 9 }, pantryItemScoreFields)).toBeNull()
  })

  it('knows the kind and the paths of a tasting', () => {
    expect(kindOf(review)).toBe('pantryItem')
    expect(parentNameOf(review)).toBe('Wehlener Sonnenuhr')
    expect(reviewPathOf(review)).toBe(`/pantry/${item.id}/reviews/${review.id}`)
    expect(kindOfQuery('pantry')).toBe('pantryItem')
    expect(kindOfQuery('wines')).toBeUndefined()
  })

  it('names the pantry items of a list', () => {
    expect(listCountText({ restaurantCount: 0, recipeCount: 0, pantryItemCount: 2 })).toBe('2 pantry items')
    expect(listCountText({ restaurantCount: 1, recipeCount: 1, pantryItemCount: 1 })).toBe('1 restaurant, 1 recipe, 1 pantry item')
    expect(listCountText({ restaurantCount: 0, recipeCount: 0, pantryItemCount: 0 })).toBe('0 restaurants')
  })

  it('shows the history of a pantry item and undoes a category', () => {
    const versions: PantryItemVersion[] = [
      {
        id: 'v3p8n1t6r4s9q2a',
        number: 2,
        action: 'update',
        source: 'user',
        editor: { id: ana.id, name: 'Ana' },
        before: { category: 'wine', style: null },
        after: { category: 'beer', style: 'Pils' },
        created: '2026-11-24T10:12:40.120Z',
      },
    ]
    expect(changedProperties('pantryItem', versions[0]!)).toEqual(['category', 'style'])
    expect(propertyLabel('pantryItem', 'producer')).toBe('Producer')
    expect(valueText('category', 'beer')).toBe('Beer')
    expect(undoChange('pantryItem', versions, 2)?.values).toEqual({ category: 'wine', style: null })
  })
})

describe('PantryItemCard', () => {
  it('links the pantry item and shows its category and details', async () => {
    const w = mount(PantryItemCard, { props: { item }, global: { plugins: [await routerAt('/pantry')] }, attachTo: attach() })
    expect(w.get('h2 a').attributes('href')).toBe(`/pantry/${item.id}`)
    expect(w.text()).toContain('Wine · Weingut Example · Mosel, Germany')
    expect(w.text()).toContain('2 reviews')
    expect(w.find('button[aria-label="Favorite Wehlener Sonnenuhr"]').exists()).toBe(true)
    await expectAccessible(w.element)
  })
})

describe('BottomNav', () => {
  it('has the pantry and no journal', async () => {
    signIn()
    const w = mount(BottomNav, { global: { plugins: [await routerAt('/pantry/p2k7m4q9s1v6x3z')] }, attachTo: attach() })
    const links = w.findAll('a')
    expect(links.map((a) => a.text())).toEqual(['Table', 'Restaurants', 'Add review', 'Recipes', 'Pantry'])
    expect(links[4]!.attributes('aria-current')).toBe('page')
  })

  it('has four sections for a visitor', async () => {
    const w = mount(BottomNav, { global: { plugins: [await routerAt('/')] }, attachTo: attach() })
    expect(w.findAll('a').map((a) => a.text())).toEqual(['Table', 'Restaurants', 'Recipes', 'Pantry'])
  })
})

describe('PantryPage', () => {
  async function mountPage(path: string) {
    const list = vi.spyOn(pantryApi, 'listPantryItems').mockResolvedValue({ items: [item], nextCursor: null })
    const router = await routerAt(path)
    const w = mount(PantryPage, { global: { plugins: [router] }, attachTo: attach() })
    await flushPromises()
    return { w, router, list }
  }

  it('selects the category with a chip', async () => {
    const { w, router, list } = await mountPage('/pantry?sort=name')
    expect(list.mock.calls[0]![0]).toMatchObject({ sort: 'name', categoryEq: undefined })
    const chips = w.get('nav[aria-label=Category]').findAll('a')
    expect(chips.map((a) => a.text())).toEqual(['All', 'Wine', 'Beer', 'Cheese', 'Chocolate'])
    expect(chips[0]!.attributes('aria-current')).toBe('page')
    expect(chips[2]!.attributes('href')).toBe('/pantry?sort=name&category=beer')
    await router.push('/pantry?sort=name&category=beer')
    await flushPromises()
    expect(list.mock.calls.at(-1)![0]).toMatchObject({ sort: 'name', categoryEq: 'beer' })
    expect(w.get('nav[aria-label=Category]').findAll('a')[2]!.attributes('aria-current')).toBe('page')
    expect(w.get('h2 a').attributes('href')).toBe(`/pantry/${item.id}`)
    await expectAccessible(w.element)
  })

  it('ignores an unknown category', async () => {
    const { list } = await mountPage('/pantry?category=cider')
    expect(list.mock.calls[0]![0]).toMatchObject({ categoryEq: undefined })
  })
})

describe('PantryItemPage', () => {
  it('shows the category, the details, and the tastings', async () => {
    vi.spyOn(pantryApi, 'getPantryItem').mockResolvedValue(item)
    vi.spyOn(pantryApi, 'listPantryItemPhotos').mockResolvedValue({ items: [], nextCursor: null })
    const reviews = vi.spyOn(pantryApi, 'listPantryItemReviews').mockResolvedValue({ items: [review], nextCursor: null })
    const w = mount(PantryItemPage, { props: { pantryItemId: item.id }, global: { plugins: [await routerAt(`/pantry/${item.id}`)] }, attachTo: attach() })
    await flushPromises()
    expect(w.get('h1').text()).toBe(item.name)
    expect(w.text()).toContain('Wine')
    for (const detail of ['Weingut Example', 'Mosel, Germany', 'Riesling']) {
      expect(w.get('dl').text()).toContain(detail)
    }
    expect(w.text()).toContain('2 tastings')
    expect(w.text()).toContain('Aroma')
    expect(w.text()).toContain('Every tasting')
    expect(w.findAll('a').find((a) => a.text().includes('Add a tasting'))!.attributes('href')).toBe(`/pantry/${item.id}/reviews/new`)
    expect(w.findAll('a').find((a) => a.text() === 'white')!.attributes('href')).toBe('/tags/white')
    expect(reviews.mock.calls[0]![0]).toMatchObject({ pantryItemId: item.id })
    await expectAccessible(w.element)
  })
})

describe('PantryItemFormPage', () => {
  async function mountForm(path = '/pantry/new') {
    signIn()
    const router = await routerAt(path)
    const w = mount(PantryItemFormPage, { global: { plugins: [router] }, attachTo: attach() })
    await flushPromises()
    return { w, router }
  }

  it('needs a category', async () => {
    const create = vi.spyOn(pantryApi, 'createPantryItem')
    const { w } = await mountForm()
    await w.get('#pantry-item-name').setValue('Gouda')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(w.text()).toContain('Select a category.')
    expect(create).not.toHaveBeenCalled()
  })

  it('shows the pantry items of the same name', async () => {
    vi.useFakeTimers()
    const other = { ...item, id: 'w8n3b6c1d4f7g2h', name: 'Wehlener Sonnenuhr Spätlese' }
    const list = vi.spyOn(pantryApi, 'listPantryItems').mockResolvedValue({ items: [other, item], nextCursor: null })
    const { w } = await mountForm('/pantry/new?review=new')
    await w.get('#pantry-item-name').setValue(' wehlener sonnenuhr ')
    await vi.advanceTimersByTimeAsync(500)
    await flushPromises()
    vi.useRealTimers()
    expect(list.mock.calls[0]![0]).toMatchObject({ search: 'wehlener sonnenuhr' })
    expect(w.text()).toContain('Is it one of these?')
    const links = w.findAll('a').filter((a) => a.text() === item.name)
    expect(links.map((a) => a.attributes('href'))).toEqual([`/reviews/new?pantry_item=${item.id}`])
  })

  it('keeps a category that this UI does not know', async () => {
    const spirits = { ...item, category: 'spirits' } as unknown as PantryItem
    vi.spyOn(pantryApi, 'getPantryItem').mockResolvedValue(spirits)
    const update = vi.spyOn(pantryApi, 'updatePantryItem').mockResolvedValue({ ...spirits, name: 'Renamed' })
    signIn()
    const edit = mount(PantryItemFormPage, { props: { pantryItemId: item.id }, global: { plugins: [await routerAt(`/pantry/${item.id}/edit`)] }, attachTo: attach() })
    await flushPromises()
    expect(edit.get('#pantry-item-category').text()).toBe('spirits')
    await edit.get('#pantry-item-name').setValue('Renamed')
    await edit.get('form').trigger('submit')
    await flushPromises()
    expect(edit.text()).not.toContain('Select a category.')
    expect(update).toHaveBeenCalledWith({ pantryItemId: item.id, pantryItemPatch: { name: 'Renamed' } })
  })

  it('suggests the tags of the pantry', async () => {
    const tags: TagSummary[] = [{ name: 'raw-milk', restaurantCount: 1, recipeCount: 0, pantryItemCount: 2, created: '2026-11-27T09:12:44.318Z' }]
    const list = vi.spyOn(tagsApi, 'listTags').mockResolvedValue({ items: tags, nextCursor: null })
    const { w } = await mountForm()
    await w.get('#pantry-item-tags').setValue('raw')
    await flushPromises()
    await new Promise((r) => setTimeout(r, 300))
    await flushPromises()
    expect(list).toHaveBeenCalledWith({ kind: 'pantry', search: 'raw', limit: 8 })
    expect(w.text()).toContain('2 pantry items')
  })
})

describe('NewReviewPage', () => {
  it('starts with I tasted for kind=pantry', async () => {
    signIn()
    vi.spyOn(pantryApi, 'listPantryItems').mockResolvedValue({ items: [item], nextCursor: null })
    const w = mount(NewReviewPage, { props: { startKind: 'pantryItem' }, global: { plugins: [await routerAt('/reviews/new?kind=pantry')] }, attachTo: attach() })
    await flushPromises()
    expect(w.get('h2').text()).toBe('What did you have?')
    const choices = w.get('[role=group]').findAll('button')
    expect(choices.map((b) => [b.text(), b.attributes('aria-pressed')])).toEqual([
      ['I ate out', 'false'],
      ['I cooked', 'false'],
      ['I tasted', 'true'],
    ])
    expect(w.get('label[for=subject-search]').text()).toBe('Pantry item')
    expect(w.text()).toContain('Add a new pantry item')
  })
})

describe('SaveToListMenu', () => {
  it('asks for the lists with the pantry item', async () => {
    signIn()
    const lists = vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [] })
    const w = mount(SaveToListMenu, { props: { pantryItemId: item.id }, global: { plugins: [await routerAt(`/pantry/${item.id}`)] }, attachTo: attach() })
    await w.get('button').trigger('click')
    await flushPromises()
    expect(lists).toHaveBeenCalledWith({ reviewerId: ana.id, restaurantEq: undefined, recipeEq: undefined, pantryItemEq: item.id })
  })
})

describe('TagPage', () => {
  const tag: TagSummary = { name: 'aged', restaurantCount: 0, recipeCount: 0, pantryItemCount: 1, created: '2026-11-27T09:12:44.318Z' }

  it('shows the pantry section of a tag', async () => {
    vi.spyOn(tagsApi, 'getTag').mockResolvedValue(tag)
    const restaurants = vi.spyOn(restaurantsApi, 'listRestaurants').mockResolvedValue({ items: [], nextCursor: null })
    const recipes = vi.spyOn(recipesApi, 'listRecipes').mockResolvedValue({ items: [], nextCursor: null })
    const pantry = vi.spyOn(pantryApi, 'listPantryItems').mockResolvedValue({ items: [item], nextCursor: null })
    const w = mount(TagPage, { props: { name: 'aged' }, global: { plugins: [await routerAt('/tags/aged?kind=pantry')] }, attachTo: attach() })
    await flushPromises()
    expect(w.text()).toContain('1 pantry item')
    expect(w.findAll('h2').map((h) => h.text())).toEqual(['Pantry'])
    expect(pantry.mock.calls[0]![0]).toMatchObject({ tagsHas: 'aged', sort: '-overall' })
    expect(restaurants).not.toHaveBeenCalled()
    expect(recipes).not.toHaveBeenCalled()
    expect(w.findAll('a').find((a) => a.text().includes('Sort and filter'))!.attributes('href')).toBe('/pantry?tags=aged')
    expect(w.get('h3 a').attributes('href')).toBe(`/pantry/${item.id}`)
  })
})
