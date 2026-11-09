import type { Ref } from 'vue'
import { ref } from 'vue'
import { apiFailure } from '@/lib/api'

export interface Page<T> {
  items: T[]
  nextCursor: string | null
}

export interface ListFailure {
  // status is null for a network error.
  status: number | null
  // detail is the reason of a 400 response, such as a search with too many
  // words. Another try with the same filters gets the same response.
  detail: string | null
}

export interface PagedList<T> {
  items: Ref<T[]>
  cursor: Ref<string | null>
  loading: Ref<boolean>
  loadingMore: Ref<boolean>
  failed: Ref<ListFailure | null>
  moreFailed: Ref<boolean>
  reload: () => Promise<void>
  loadMore: (cursor: string) => Promise<void>
}

// usePagedList loads the first page and appends the next pages. A new
// first load drops the cursor. A cursor is valid only for the filters and
// sort that made it.
export function usePagedList<T>(fetchPage: (cursor?: string) => Promise<Page<T>>): PagedList<T> {
  const items = ref<T[]>([]) as Ref<T[]>
  const cursor = ref<string | null>(null)
  const loading = ref(true)
  const loadingMore = ref(false)
  const failed = ref<ListFailure | null>(null)
  const moreFailed = ref(false)
  let generation = 0

  async function reload() {
    const g = ++generation
    loading.value = true
    loadingMore.value = false
    failed.value = null
    moreFailed.value = false
    try {
      const page = await fetchPage()
      if (g !== generation) {
        return
      }
      items.value = page.items
      cursor.value = page.nextCursor
    } catch (error) {
      if (g === generation) {
        const { status, problem } = await apiFailure(error)
        failed.value = { status, detail: status === 400 ? (problem?.detail ?? null) : null }
      }
    } finally {
      if (g === generation) {
        loading.value = false
      }
    }
  }

  async function loadMore(next: string) {
    const g = generation
    loadingMore.value = true
    moreFailed.value = false
    try {
      const page = await fetchPage(next)
      if (g !== generation) {
        return
      }
      items.value = [...items.value, ...page.items]
      cursor.value = page.nextCursor
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

  return { items, cursor, loading, loadingMore, failed, moreFailed, reload, loadMore }
}
