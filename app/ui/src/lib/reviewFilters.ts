import type { FilterDef, FilterValue, Option } from '@/components/ui/filter-bar'

// The review filters and sorts of the restaurant page and the my reviews page.

function scoreOptions(max: number): Option[] {
  return Array.from({ length: max }, (_, i) => max - i).map((n) => ({ value: String(n), label: `${n} or more` }))
}

export const reviewFilterDefs: FilterDef[] = [
  { key: 'search', label: 'Search reviews', type: 'search', placeholder: 'Search reviews', chip: (v) => `Search: ${v}` },
  { key: 'tags', label: 'Tags', type: 'tags', help: 'Shows reviews with at least one of the tags.', chip: (v) => `Tag: ${v}` },
  { key: 'food', label: 'Food', type: 'select', options: scoreOptions(10), chip: (v) => `Food: ${v} or more` },
  { key: 'service', label: 'Service', type: 'select', options: scoreOptions(10), chip: (v) => `Service: ${v} or more` },
  { key: 'space', label: 'Space', type: 'select', options: scoreOptions(10), chip: (v) => `Space: ${v} or more` },
  { key: 'price', label: 'Price', type: 'select', options: scoreOptions(10), chip: (v) => `Price: ${v} or more` },
  { key: 'cleanliness', label: 'Cleanliness', type: 'select', options: scoreOptions(10), chip: (v) => `Cleanliness: ${v} or more` },
]

export const reviewSortOptions: Option[] = [
  { value: '-created', label: 'Newest' },
  { value: 'created', label: 'Oldest' },
  { value: '-food', label: 'Best food' },
  { value: '-service', label: 'Best service' },
  { value: '-space', label: 'Best space' },
  { value: '-price', label: 'Best value for money' },
  { value: '-cleanliness', label: 'Cleanest' },
  { value: 'title', label: 'Title, A to Z' },
]

function atLeast(filters: Record<string, FilterValue>, key: string) {
  const v = Number(filters[key])
  return Number.isInteger(v) && v > 0 ? { gte: v } : undefined
}

// reviewFilterParams changes the filters to the query parameters of the SDK.
export function reviewFilterParams(f: Record<string, FilterValue>) {
  const tags = f.tags as string[] | undefined
  return {
    search: typeof f.search === 'string' ? f.search : undefined,
    tagsHasAny: tags?.length ? tags : undefined,
    food: atLeast(f, 'food'),
    service: atLeast(f, 'service'),
    space: atLeast(f, 'space'),
    price: atLeast(f, 'price'),
    cleanliness: atLeast(f, 'cleanliness'),
  }
}
