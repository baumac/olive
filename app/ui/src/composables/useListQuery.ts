import type { FilterDef, FilterValue, Option } from '@/components/ui/filter-bar'
import type { MaybeRefOrGetter } from 'vue'
import { computed, toValue } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { isTag, maxTags } from '@/lib/validation'

// RelevanceSort is the sort of a search filter. It is an option and the
// default only while the filter has a value.
export interface RelevanceSort {
  filter: string
  option: Option
}

// searchRelevance is the relevance sort of the API search filter.
export const searchRelevance: RelevanceSort = { filter: 'search', option: { value: 'relevance', label: 'Best match' } }

// useListQuery keeps the filters and the sort of a list in the URL query.
// It ignores values that defs do not allow. defs can change, for example
// when the signed in user changes.
export function useListQuery(
  defs: MaybeRefOrGetter<FilterDef[]>,
  baseSortOptions: Option[],
  baseDefaultSort: string,
  relevance?: RelevanceSort,
  // keep has the query keys of the page that a filter change does not remove.
  keep: string[] = [],
) {
  const route = useRoute()
  const router = useRouter()

  function first(v: unknown): string | undefined {
    const s = Array.isArray(v) ? v[0] : v
    return typeof s === 'string' && s !== '' ? s : undefined
  }

  function parse(def: FilterDef, raw: string): FilterValue {
    if (def.type === 'search') {
      const text = raw.trim()
      return text && Array.from(text).length <= 255 ? text : undefined
    }
    if (def.type === 'tags') {
      const tags = [...new Set(raw.split(',').filter(isTag))].slice(0, maxTags)
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
      if (sort.value !== defaultSort.value) {
        query.sort = sort.value
      }
      for (const key of keep) {
        const value = first(route.query[key])
        if (value !== undefined) {
          query[key] = value
        }
      }
      void router.replace({ query })
    },
  })

  const searching = computed(() => !!relevance && filters.value[relevance.filter] !== undefined)
  const sortOptions = computed(() => (searching.value && relevance ? [relevance.option, ...baseSortOptions] : baseSortOptions))
  const defaultSort = computed(() => (searching.value && relevance ? relevance.option.value : baseDefaultSort))

  const sort = computed<string>({
    get() {
      const raw = first(route.query.sort)
      return sortOptions.value.some((o) => o.value === raw) ? (raw as string) : defaultSort.value
    },
    set(next) {
      const query = { ...route.query }
      if (next === defaultSort.value) {
        delete query.sort
      } else {
        query.sort = next
      }
      void router.replace({ query })
    },
  })

  const active = computed(() => Object.keys(filters.value).length > 0)

  return { filters, sort, active, sortOptions }
}
