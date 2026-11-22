import type { FilterDef, FilterValue, Option } from '@/components/ui/filter-bar'
import { formatMinutes } from '@/lib/format'

// The filters and the sorts of the recipes page. See decision 00004.

const totalTimes = [15, 30, 45, 60, 120]

export const recipeFilterDefs: FilterDef[] = [
  { key: 'search', label: 'Search recipes', type: 'search', placeholder: 'Search recipes', chip: (v) => `Search: ${v}` },
  { key: 'tags', label: 'Tags', type: 'tags', help: 'Shows recipes with at least one of the tags.', chip: (v) => `Tag: ${v}` },
  {
    key: 'overall',
    label: 'Overall',
    type: 'select',
    options: [9, 8, 7, 6, 5].map((n) => ({ value: String(n), label: `${n} or more` })),
    help: 'Recipes without reviews do not match.',
    chip: (v) => `Overall: ${v} or more`,
  },
  {
    key: 'total_time',
    label: 'Total time',
    type: 'select',
    options: totalTimes.map((n) => ({ value: String(n), label: `${formatMinutes(n)} or less` })),
    help: 'Recipes without a total time do not match.',
    chip: (v) => `Total time: ${formatMinutes(Number(v))} or less`,
  },
]

export const recipeSortOptions: Option[] = [
  { value: '-overall', label: 'Highest rated' },
  { value: '-created', label: 'Newest' },
  { value: 'total_time', label: 'Quickest' },
  { value: 'name', label: 'Name' },
]

// oldRecipeSortOptions are the sorts that only a link can select.
export const oldRecipeSortOptions: Option[] = [
  { value: 'created', label: 'Oldest' },
  { value: 'overall', label: 'Lowest rated' },
  { value: '-total_time', label: 'Longest' },
  { value: '-name', label: 'Name, Z to A' },
  { value: '-updated', label: 'Recently updated' },
]

// recipeFilterParams changes the filters to the query parameters of the SDK.
export function recipeFilterParams(f: Record<string, FilterValue>) {
  const tags = f.tags as string[] | undefined
  const overall = Number(f.overall)
  const totalTime = Number(f.total_time)
  return {
    search: typeof f.search === 'string' ? f.search : undefined,
    tagsHasAny: tags?.length ? tags : undefined,
    overall: f.overall !== undefined && Number.isFinite(overall) ? { gte: overall } : undefined,
    totalTime: f.total_time !== undefined && Number.isInteger(totalTime) ? { lte: totalTime } : undefined,
    listEq: typeof f.list === 'string' ? f.list : undefined,
  }
}
