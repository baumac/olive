import type { FilterDef, FilterValue, Option } from '@/components/ui/filter-bar'
import { kindLabels } from './reviews'

// The review filters and sorts of the restaurant page and the reviews page.

function scoreOptions(max: number): Option[] {
  return Array.from({ length: max }, (_, i) => max - i).map((n) => ({ value: String(n), label: `${n} or more` }))
}

const joc = 'Matches only Joc de cartes reviews.'

export const reviewFilterDefs: FilterDef[] = [
  { key: 'tags', label: 'Tags', type: 'tags', help: 'Shows reviews with at least one of the tags.', chip: (v) => `Tag: ${v}` },
  {
    key: 'kind',
    label: 'Kind',
    type: 'select',
    options: Object.entries(kindLabels).map(([value, label]) => ({ value, label })),
    chip: (v) => `Kind: ${kindLabels[v] ?? v}`,
  },
  { key: 'rating', label: 'Rating', type: 'select', options: scoreOptions(5), help: 'Matches only Basic reviews.', chip: (v) => `Rating: ${v} or more` },
  { key: 'food', label: 'Food', type: 'select', options: scoreOptions(10), help: joc, chip: (v) => `Food: ${v} or more` },
  { key: 'service', label: 'Service', type: 'select', options: scoreOptions(10), help: joc, chip: (v) => `Service: ${v} or more` },
  { key: 'space', label: 'Space', type: 'select', options: scoreOptions(10), help: joc, chip: (v) => `Space: ${v} or more` },
  { key: 'price', label: 'Price', type: 'select', options: scoreOptions(10), help: joc, chip: (v) => `Price: ${v} or more` },
]

export const reviewSortOptions: Option[] = [
  { value: '-created', label: 'Newest' },
  { value: 'created', label: 'Oldest' },
  { value: '-rating', label: 'Highest rating' },
  { value: 'rating', label: 'Lowest rating' },
  { value: '-food', label: 'Best food' },
  { value: '-service', label: 'Best service' },
  { value: '-space', label: 'Best space' },
  { value: '-price', label: 'Best value for money' },
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
    tagsHasAny: tags?.length ? tags : undefined,
    kindIn: typeof f.kind === 'string' ? [f.kind] : undefined,
    rating: atLeast(f, 'rating'),
    food: atLeast(f, 'food'),
    service: atLeast(f, 'service'),
    space: atLeast(f, 'space'),
    price: atLeast(f, 'price'),
    creatorEq: typeof f.creator === 'string' ? f.creator : undefined,
  }
}
