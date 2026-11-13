import type { FilterDef, Option } from '@/components/ui/filter-bar'
import { FetchError, ResponseError } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import PagedList from '@/components/app/PagedList.vue'
import { searchRelevance, useListQuery } from '@/composables/useListQuery'
import { usePagedList } from '@/composables/usePagedList'

const defs: FilterDef[] = [
  { key: 'tags', label: 'Tags', type: 'tags', chip: (v) => v },
  { key: 'rating', label: 'Rating', type: 'select', options: [{ value: '4', label: '4 or more' }], chip: (v) => v },
]
const sortOptions: Option[] = [{ value: '-created', label: 'Newest' }]

async function queryAt(url: string, filterDefs: FilterDef[] | (() => FilterDef[]) = defs, relevance = false, keep: string[] = []) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
  await router.push(url)
  let result!: ReturnType<typeof useListQuery>
  mount(
    defineComponent({
      setup() {
        result = useListQuery(filterDefs, sortOptions, '-created', relevance ? searchRelevance : undefined, keep)
        return () => null
      },
    }),
    { global: { plugins: [router] } },
  )
  return Object.assign(result, { router })
}

describe('useListQuery', () => {
  it('ignores values that the filters do not allow', async () => {
    const q = await queryAt('/r?tags=italian,Bad,pasta&rating=abc&sort=-nope&other=1')
    expect(q.filters.value).toEqual({ tags: ['italian', 'pasta'] })
    expect(q.sort.value).toBe('-created')
  })

  it('keeps allowed values', async () => {
    const q = await queryAt('/r?rating=4')
    expect(q.filters.value).toEqual({ rating: '4' })
    expect(q.active.value).toBe(true)
  })

  it('keeps at most 20 different tags like the API', async () => {
    const tags = Array.from({ length: 25 }, (_, i) => `t${i}`)
    const q = await queryAt(`/r?tags=t0,t0,${tags.join(',')}`)
    expect(q.filters.value.tags).toEqual(tags.slice(0, 20))
  })
})

describe('useListQuery search filters', () => {
  it('trims the text and drops text that is too long', async () => {
    const search: FilterDef[] = [{ key: 'name', label: 'Name', type: 'search', chip: (v) => v }]
    expect((await queryAt('/r?name=%20lui%20', search)).filters.value).toEqual({ name: 'lui' })
    expect((await queryAt(`/r?name=${'x'.repeat(256)}`, search)).filters.value).toEqual({})
  })
})

describe('useListQuery relevance sort', () => {
  const search: FilterDef[] = [{ key: 'search', label: 'Search', type: 'search', chip: (v) => v }]

  it('sorts a search by relevance', async () => {
    const q = await queryAt('/r?search=pasta', search, true)
    expect(q.sort.value).toBe('relevance')
    expect(q.sortOptions.value.map((o) => o.value)).toEqual(['relevance', '-created'])
  })

  it('has no relevance sort without a search', async () => {
    const q = await queryAt('/r?sort=relevance', search, true)
    expect(q.sort.value).toBe('-created')
    expect(q.sortOptions.value.map((o) => o.value)).toEqual(['-created'])
  })
})

describe('useListQuery with other query keys', () => {
  it('keeps the keys of the page when a filter changes', async () => {
    const q = await queryAt('/r?view=map&other=1', defs, false, ['view'])
    q.filters.value = { rating: '4' }
    await flushPromises()
    expect(q.router.currentRoute.value.query).toEqual({ view: 'map', rating: '4' })
  })
})

describe('useListQuery with changing defs', () => {
  it('drops a filter when its definition goes away', async () => {
    const creator: FilterDef = { key: 'creator', label: 'Written by', type: 'select', options: [{ value: 'u1', label: 'You' }], chip: () => 'you' }
    const current = ref<FilterDef[]>([creator, ...defs])
    const q = await queryAt('/r?creator=u1', () => current.value)
    expect(q.filters.value).toEqual({ creator: 'u1' })
    current.value = defs
    expect(q.filters.value).toEqual({})
  })
})

describe('PagedList', () => {
  it('moves focus to the first new item after load more', async () => {
    const fetchPage = vi.fn(async (cursor?: string) =>
      cursor
        ? { items: [{ id: 'c' }], nextCursor: null }
        : { items: [{ id: 'a' }, { id: 'b' }], nextCursor: 'next' },
    )
    const list = usePagedList(fetchPage)
    await list.reload()
    const el = document.createElement('div')
    document.body.appendChild(el)
    const w = mount(PagedList, {
      props: { list, noun: 'items', skeletonClass: '', skeletonCount: 1 },
      slots: { item: ({ item }: { item: { id: string } }) => h('span', item.id), empty: () => h('p', 'none') },
      attachTo: el,
    })
    await w.get('button').trigger('click')
    await flushPromises()
    expect(fetchPage).toHaveBeenLastCalledWith('next')
    expect(document.activeElement?.id).toBe('items-c')
  })

  it('shows the reason of a 400 response without a retry button', async () => {
    const problem = { type: 'about:blank', title: 'Bad Request', status: 400, detail: 'The search can have at most 10 words.' }
    const list = usePagedList<{ id: string }>(async () => {
      throw new ResponseError(new Response(JSON.stringify(problem), { status: 400, headers: { 'Content-Type': 'application/problem+json' } }))
    })
    await list.reload()
    const w = mount(PagedList, {
      props: { list, noun: 'items', skeletonClass: '', skeletonCount: 1 },
      slots: { item: () => h('span'), empty: () => h('p', 'none') },
    })
    expect(w.text()).toContain('The search can have at most 10 words.')
    expect(w.text()).not.toContain('Check your connection')
    expect(w.find('button').exists()).toBe(false)
  })

  it('keeps a failed load more of an old list out of the new list', async () => {
    let failMore!: () => void
    const list = usePagedList(async (cursor?: string) => {
      if (cursor) {
        await new Promise<void>((_, reject) => {
          failMore = () => reject(new FetchError(new Error('offline')))
        })
      }
      return { items: [{ id: 'a' }], nextCursor: 'next' }
    })
    await list.reload()
    const more = list.loadMore('next')
    await list.reload()
    failMore()
    await more
    expect(list.moreFailed.value).toBe(false)
    expect(list.loadingMore.value).toBe(false)
  })
})
