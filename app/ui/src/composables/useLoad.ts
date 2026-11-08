import type { Ref } from 'vue'
import { ref } from 'vue'
import { apiFailure } from '@/lib/api'

export interface LoadState<T> {
  data: Ref<T | null>
  loading: Ref<boolean>
  // failed is the status of a failed load. null means a network error.
  failed: Ref<{ status: number | null } | null>
  load: () => Promise<void>
}

export function useLoad<T>(loader: () => Promise<T>): LoadState<T> {
  const data = ref<T | null>(null) as Ref<T | null>
  const loading = ref(true)
  const failed = ref<{ status: number | null } | null>(null)

  async function load() {
    loading.value = true
    failed.value = null
    try {
      data.value = await loader()
    } catch (error) {
      failed.value = { status: (await apiFailure(error)).status }
    } finally {
      loading.value = false
    }
  }

  return { data, loading, failed, load }
}
