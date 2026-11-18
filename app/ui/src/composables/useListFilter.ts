import type { SavedList } from '@reviews/sdk'
import type { FilterDef, Option } from '@/components/ui/filter-bar'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { firstQueryValue } from '@/composables/useListQuery'
import { apiFailure, listsApi } from '@/lib/api'
import { ownLists } from '@/lib/lists'
import { currentUser } from '@/lib/session'

const listId = /^[a-z0-9]{15}$/
const retryDelays = [2000, 10000]

// useListFilter gives the list filter of the restaurants page or the recipes
// page. Its options are the lists of the signed in user and the list of the
// URL. The filter is null without options. noun names the items in plural.
export function useListFilter(noun = 'restaurants') {
  const route = useRoute()
  const own = ref<SavedList[]>([])
  const linked = ref<SavedList>()
  const unknown = ref<string>()
  let retryTimer: ReturnType<typeof setTimeout> | undefined
  onBeforeUnmount(() => clearTimeout(retryTimer))

  watch(
    () => currentUser.value?.id,
    async (id) => {
      own.value = []
      if (!id) {
        return
      }
      try {
        const lists = await ownLists(id)
        if (currentUser.value?.id === id) {
          own.value = lists
        }
      } catch {
        // The filter then has only the list of the URL.
      }
    },
    { immediate: true },
  )

  const linkedId = computed(() => {
    const raw = firstQueryValue(route.query.list)
    return raw && listId.test(raw) ? raw : undefined
  })

  async function loadLinked(id: string, attempt: number) {
    try {
      const list = await listsApi.getList({ listId: id })
      if (linkedId.value === id) {
        linked.value = list
      }
    } catch (error) {
      const { status } = await apiFailure(error).catch(() => ({ status: null }))
      if (linkedId.value !== id) {
        return
      }
      if (status === 404) {
        unknown.value = id
      } else if (attempt < retryDelays.length) {
        retryTimer = setTimeout(() => void loadLinked(id, attempt + 1), retryDelays[attempt])
      }
    }
  }

  watch(
    linkedId,
    (id) => {
      clearTimeout(retryTimer)
      if (linked.value?.id === id || own.value.some((l) => l.id === id)) {
        return
      }
      linked.value = undefined
      unknown.value = undefined
      if (id) {
        void loadLinked(id, 0)
      }
    },
    { immediate: true },
  )

  function linkedLabel(id: string): string {
    if (linked.value?.id === id) {
      return `${linked.value.name} by ${linked.value.creator.name}`
    }
    return unknown.value === id ? 'Unknown list' : 'A list'
  }

  const options = computed<Option[]>(() => {
    const out = own.value.map((l) => ({ value: l.id, label: l.name }))
    const id = linkedId.value
    if (id && !out.some((o) => o.value === id)) {
      // The option exists before getList answers. useListQuery then keeps
      // the filter of the URL.
      out.push({ value: id, label: linkedLabel(id) })
    }
    return out
  })

  const favoritesId = computed(() => own.value.find((l) => l.kind === 'favorites')?.id)

  const def = computed<FilterDef | null>(() => {
    if (options.value.length === 0) {
      return null
    }
    const label = (value: string) => options.value.find((o) => o.value === value)?.label ?? 'A list'
    return {
      key: 'list',
      label: 'List',
      type: 'select',
      options: options.value,
      help: `Shows only the ${noun} of the list.`,
      chip: (v) => `List: ${label(v)}`,
    }
  })

  return { def, favoritesId }
}
