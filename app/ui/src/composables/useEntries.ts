import type { Ref } from 'vue'
import { nextTick, ref } from 'vue'
import type { ListFailure, Page } from '@/composables/usePagedList'
import { apiFailure } from '@/lib/api'
import { type EntryOrder, type EntrySource, takeMerged } from '@/lib/entries'
import type { AnyReview } from '@/lib/reviewKinds'

export type EntryFetch = (cursor?: string) => Promise<Page<AnyReview>>

export interface Entries {
  items: Ref<AnyReview[]>
  hasMore: Ref<boolean>
  loading: Ref<boolean>
  loadingMore: Ref<boolean>
  failed: Ref<ListFailure | null>
  moreFailed: Ref<boolean>
  reload: () => Promise<void>
  loadMore: () => Promise<void>
}

// useEntries merges the pages of one or more review lists in the order of
// their sort. Each source keeps its own cursor. See decision 00009.
export function useEntries(sources: () => EntryFetch[], order: () => EntryOrder): Entries {
  const items = ref<AnyReview[]>([]) as Ref<AnyReview[]>
  const hasMore = ref(false)
  const loading = ref(true)
  const loadingMore = ref(false)
  const failed = ref<ListFailure | null>(null)
  const moreFailed = ref(false)
  let fetchers: EntryFetch[] = []
  let states: EntrySource[] = []
  let generation = 0

  function update(append: boolean) {
    const next = takeMerged(states, order())
    items.value = append ? [...items.value, ...next] : next
    hasMore.value = states.some((s) => s.buffer.length > 0 || s.cursor !== null)
  }

  async function reload() {
    const g = ++generation
    loading.value = true
    loadingMore.value = false
    failed.value = null
    moreFailed.value = false
    fetchers = sources()
    try {
      const pages = await Promise.all(fetchers.map((f) => f()))
      if (g !== generation) {
        return
      }
      states = pages.map((p) => ({ buffer: [...p.items], cursor: p.nextCursor }))
      update(false)
    } catch (error) {
      const { status, problem } = await apiFailure(error)
      if (g === generation) {
        failed.value = { status, detail: status === 400 ? (problem?.detail ?? null) : null }
      }
    } finally {
      if (g === generation) {
        loading.value = false
      }
    }
  }

  async function loadMore() {
    const g = generation
    loadingMore.value = true
    moreFailed.value = false
    try {
      await Promise.all(
        states.map(async (s, i) => {
          if (s.buffer.length === 0 && s.cursor !== null) {
            const page = await fetchers[i]!(s.cursor)
            if (g === generation) {
              s.buffer.push(...page.items)
              s.cursor = page.nextCursor
            }
          }
        }),
      )
      if (g === generation) {
        update(true)
      }
    } catch {
      if (g === generation) {
        moreFailed.value = true
      }
    } finally {
      if (g === generation) {
        loadingMore.value = false
      }
    }
  }

  return { items, hasMore, loading, loadingMore, failed, moreFailed, reload, loadMore }
}

// loadMoreAndFocus moves the focus to the first new entry after the load.
// A keyboard user then goes on from there. idOf gives the element ID.
export async function loadMoreAndFocus(entries: Entries, idOf: (review: AnyReview) => string) {
  const start = entries.items.value.length
  await entries.loadMore()
  await nextTick()
  const first = entries.items.value[start]
  if (first) {
    document.getElementById(idOf(first))?.focus()
  }
}
