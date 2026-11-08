import type { FilterDef, Option } from '@/components/ui/filter-bar'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import PagedList from '@/components/app/PagedList.vue'
import { useListQuery } from '@/composables/useListQuery'
import { usePagedList } from '@/composables/usePagedList'

const defs: FilterDef[] = [
  { key: 'tags', label: 'Tags', type: 'tags', chip: (v) => v },
  { key: 'rating', label: 'Rating', type: 'select', options: [{ value: '4', label: '4 or more' }], chip: (v) => v },
]
const sortOptions: Option[] = [{ value: '-created', label: 'Newest' }]

async function queryAt(url: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
  await router.push(url)
  let result!: ReturnType<typeof useListQuery>
  mount(
    defineComponent({
      setup() {
        result = useListQuery(defs, sortOptions, '-created')
        return () => null
      },
    }),
    { global: { plugins: [router] } },
  )
  return result
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
})
