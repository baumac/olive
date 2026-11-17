import type { FilterDef, FilterValue, Option } from '@/components/ui/filter-bar'
import type { PriceLevelRange } from '@reviews/sdk'
import { priceLevelLabel } from '@/lib/restaurantDetails'

// priceLevelRanges are the ranges of the price level filter. A link has
// the value, such as 1-2 for moderate or less.
const priceLevelRanges = new Map<string, { label: string; range: PriceLevelRange }>(
  [
    [1, 1],
    [1, 2],
    [1, 3],
    [2, 4],
    [3, 4],
    [4, 4],
  ].map(([low, high]) => {
    if (low === high) {
      return [String(low), { label: priceLevelLabel(low), range: { eq: low } }]
    }
    const label = low === 1 ? `${priceLevelLabel(high)} or less` : `${priceLevelLabel(low)} or more`
    return [`${low}-${high}`, { label, range: { gte: low, lte: high } }]
  }),
)

export const restaurantFilterDefs: FilterDef[] = [
  { key: 'search', label: 'Search restaurants', type: 'search', placeholder: 'Search restaurants', chip: (v) => `Search: ${v}` },
  { key: 'tags', label: 'Tags', type: 'tags', help: 'Shows restaurants with at least one of the tags.', chip: (v) => `Tag: ${v}` },
  {
    key: 'overall',
    label: 'Overall',
    type: 'select',
    options: [9, 8, 7, 6, 5].map((n) => ({ value: String(n), label: `${n} or more` })),
    help: 'Restaurants without reviews do not match.',
    chip: (v) => `Overall: ${v} or more`,
  },
  {
    key: 'price_level',
    label: 'Price level',
    type: 'select',
    options: [...priceLevelRanges].map(([value, { label }]) => ({ value, label })),
    help: 'Restaurants without a price level do not match.',
    chip: (v) => `Price level: ${priceLevelRanges.get(v)?.label ?? v}`,
  },
]

export const restaurantSortOptions: Option[] = [
  { value: '-overall', label: 'Highest rated' },
  { value: '-created', label: 'Newest' },
  { value: 'name', label: 'Name' },
]

// nearestSort needs the near parameter of the API.
export const nearestSort: Option = { value: 'distance', label: 'Nearest' }

// oldRestaurantSortOptions are the sorts that only a link can select.
export const oldRestaurantSortOptions: Option[] = [
  { value: 'created', label: 'Oldest' },
  { value: 'overall', label: 'Lowest rated' },
  { value: '-name', label: 'Name, Z to A' },
  { value: '-updated', label: 'Recently updated' },
]

// restaurantFilterParams changes the filters to the query parameters of the
// SDK.
export function restaurantFilterParams(f: Record<string, FilterValue>) {
  const tags = f.tags as string[] | undefined
  const overall = Number(f.overall)
  return {
    search: typeof f.search === 'string' ? f.search : undefined,
    tagsHasAny: tags?.length ? tags : undefined,
    overall: f.overall !== undefined && Number.isFinite(overall) ? { gte: overall } : undefined,
    listEq: typeof f.list === 'string' ? f.list : undefined,
    priceLevel: typeof f.price_level === 'string' ? priceLevelRanges.get(f.price_level)?.range : undefined,
  }
}
