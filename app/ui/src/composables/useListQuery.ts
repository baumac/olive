import type { FilterDef, FilterValue, Option } from '@/components/ui/filter-bar'
import type { MaybeRefOrGetter } from 'vue'
import { computed, toValue } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { isTag } from '@/lib/validation'

// useListQuery keeps the filters and the sort of a list in the URL query.
// It ignores values that defs do not allow. defs can change, for example
// when the signed in user changes.
export function useListQuery(defs: MaybeRefOrGetter<FilterDef[]>, sortOptions: Option[], defaultSort: string) {
  const route = useRoute()
  const router = useRouter()

  function first(v: unknown): string | undefined {
    const s = Array.isArray(v) ? v[0] : v
    return typeof s === 'string' && s !== '' ? s : undefined
  }

  function parse(def: FilterDef, raw: string): FilterValue {
    if (def.type === 'tags') {
      const tags = raw.split(',').filter(isTag)
      return tags.length ? tags : undefined
    }
    return def.options?.some((o) => o.value === raw) ? raw : undefined
  }

  const filters = computed<Record<string, FilterValue>>({
    get() {
      const out: Record<string, FilterValue> = {}
      for (const def of toValue(defs)) {
        const raw = first(route.query[def.key])
        const value = raw === undefined ? undefined : parse(def, raw)
        if (value !== undefined) {
          out[def.key] = value
        }
      }
      return out
    },
    set(next) {
      const query: Record<string, string> = {}
      for (const [key, value] of Object.entries(next)) {
        if (Array.isArray(value)) {
          query[key] = value.join(',')
        } else if (value) {
          query[key] = value
        }
      }
      if (sort.value !== defaultSort) {
        query.sort = sort.value
      }
      void router.replace({ query })
    },
  })

  const sort = computed<string>({
    get() {
      const raw = first(route.query.sort)
      return sortOptions.some((o) => o.value === raw) ? (raw as string) : defaultSort
    },
    set(next) {
      const query = { ...route.query }
      if (next === defaultSort) {
        delete query.sort
      } else {
        query.sort = next
      }
      void router.replace({ query })
    },
  })

  const active = computed(() => Object.keys(filters.value).length > 0)

  return { filters, sort, active }
}
