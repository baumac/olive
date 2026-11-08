import type { FilterDef, FilterValue, Option } from '@/components/ui/filter-bar'

export const restaurantFilterDefs: FilterDef[] = [
  { key: 'tags', label: 'Tags', type: 'tags', help: 'Shows restaurants with at least one of the tags.', chip: (v) => `Tag: ${v}` },
  {
    key: 'overall',
    label: 'Overall',
    type: 'select',
    options: [9, 8, 7, 6, 5].map((n) => ({ value: String(n), label: `${n} or more` })),
    help: 'Restaurants without reviews do not match.',
    chip: (v) => `Overall: ${v} or more`,
  },
]

export const restaurantSortOptions: Option[] = [
  { value: '-created', label: 'Newest' },
  { value: 'created', label: 'Oldest' },
  { value: '-overall', label: 'Highest rated' },
  { value: 'overall', label: 'Lowest rated' },
  { value: 'name', label: 'Name, A to Z' },
  { value: '-name', label: 'Name, Z to A' },
  { value: '-updated', label: 'Recently updated' },
]

// restaurantFilterParams changes the filters to the query parameters of the
// SDK.
export function restaurantFilterParams(f: Record<string, FilterValue>) {
  const tags = f.tags as string[] | undefined
  const overall = Number(f.overall)
  return {
    tagsHasAny: tags?.length ? tags : undefined,
    overall: f.overall !== undefined && Number.isFinite(overall) ? { gte: overall } : undefined,
  }
}
