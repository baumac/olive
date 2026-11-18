import type { FilterDef, FilterValue, Option } from '@/components/ui/filter-bar'
import type { ReviewKind } from '@/lib/reviewKinds'
import { recipeScoreFields } from '@/lib/reviews'

// The review filters and sorts of the restaurant and recipe pages and the
// reviewer pages.

function scoreOptions(max: number): Option[] {
  return Array.from({ length: max }, (_, i) => max - i).map((n) => ({ value: String(n), label: `${n} or more` }))
}

export const reviewFilterDefs: FilterDef[] = [
  { key: 'search', label: 'Search reviews', type: 'search', placeholder: 'Search reviews', chip: (v) => `Search: ${v}` },
  { key: 'overall', label: 'Overall', type: 'select', options: scoreOptions(10), chip: (v) => `Overall: ${v} or more` },
  { key: 'food', label: 'Food', type: 'select', options: scoreOptions(10), chip: (v) => `Food: ${v} or more` },
  { key: 'service', label: 'Service', type: 'select', options: scoreOptions(10), chip: (v) => `Service: ${v} or more` },
  { key: 'space', label: 'Space', type: 'select', options: scoreOptions(10), chip: (v) => `Space: ${v} or more` },
  { key: 'price', label: 'Value', type: 'select', options: scoreOptions(10), chip: (v) => `Value: ${v} or more` },
  { key: 'cleanliness', label: 'Cleanliness', type: 'select', options: scoreOptions(10), chip: (v) => `Cleanliness: ${v} or more` },
]

export const reviewSortOptions: Option[] = [
  { value: '-created', label: 'Newest' },
  { value: '-overall', label: 'Highest rated' },
  { value: 'overall', label: 'Lowest rated' },
]

// oldReviewSortOptions are the sorts that only a link can select.
export const oldReviewSortOptions: Option[] = [
  { value: 'created', label: 'Oldest' },
  { value: '-food', label: 'Best food' },
  { value: '-service', label: 'Best service' },
  { value: '-space', label: 'Best space' },
  { value: '-price', label: 'Best value' },
  { value: '-cleanliness', label: 'Cleanest' },
  { value: 'title', label: 'Title, A to Z' },
]

// recipeReviewFilterDefs have the scores of recipe reviews. See decision
// 00047.
export const recipeReviewFilterDefs: FilterDef[] = [
  reviewFilterDefs[0]!,
  reviewFilterDefs[1]!,
  ...recipeScoreFields.map((f): FilterDef => ({
    key: f.name,
    label: f.label,
    type: 'select',
    options: scoreOptions(10),
    chip: (v) => `${f.label}: ${v} or more`,
  })),
]

export const oldRecipeReviewSortOptions: Option[] = [
  { value: 'created', label: 'Oldest' },
  { value: '-taste', label: 'Best taste' },
  { value: '-ease', label: 'Easiest' },
  { value: '-instructions', label: 'Clearest steps' },
  { value: '-value', label: 'Best value' },
  { value: 'title', label: 'Title, A to Z' },
]

export function reviewFilterDefsOf(kind: ReviewKind): FilterDef[] {
  return kind === 'recipe' ? recipeReviewFilterDefs : reviewFilterDefs
}

export function oldReviewSortOptionsOf(kind: ReviewKind): Option[] {
  return kind === 'recipe' ? oldRecipeReviewSortOptions : oldReviewSortOptions
}

function atLeast(filters: Record<string, FilterValue>, key: string) {
  const v = Number(filters[key])
  return Number.isInteger(v) && v > 0 ? { gte: v } : undefined
}

// reviewFilterParams changes the filters to the query parameters of the SDK.
export function reviewFilterParams(f: Record<string, FilterValue>) {
  return {
    search: typeof f.search === 'string' ? f.search : undefined,
    overall: atLeast(f, 'overall'),
    food: atLeast(f, 'food'),
    service: atLeast(f, 'service'),
    space: atLeast(f, 'space'),
    price: atLeast(f, 'price'),
    cleanliness: atLeast(f, 'cleanliness'),
  }
}

export function recipeReviewFilterParams(f: Record<string, FilterValue>) {
  return {
    search: typeof f.search === 'string' ? f.search : undefined,
    overall: atLeast(f, 'overall'),
    taste: atLeast(f, 'taste'),
    ease: atLeast(f, 'ease'),
    instructions: atLeast(f, 'instructions'),
    value: atLeast(f, 'value'),
  }
}

export function reviewFilterParamsOf(kind: ReviewKind, f: Record<string, FilterValue>): Record<string, unknown> {
  return kind === 'recipe' ? recipeReviewFilterParams(f) : reviewFilterParams(f)
}
