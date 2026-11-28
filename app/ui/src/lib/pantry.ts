import type { PantryItem, PantryItemCategory } from '@reviews/sdk'
import { BeerIcon, ShoppingBasketIcon, WineIcon } from 'lucide-vue-next'
import type { FilterDef, FilterValue, Option } from '@/components/ui/filter-bar'
import { tagSuggester } from '@/lib/tags'

// The categories, the filters, and the sorts of the pantry. See decision
// 00012.

export interface Category {
  value: PantryItemCategory
  label: string
  icon: typeof ShoppingBasketIcon
}

// Lucide has no icon for cheese or chocolate. They use the icon of the
// pantry.
export const categories: Category[] = [
  { value: 'wine', label: 'Wine', icon: WineIcon },
  { value: 'beer', label: 'Beer', icon: BeerIcon },
  { value: 'cheese', label: 'Cheese', icon: ShoppingBasketIcon },
  { value: 'chocolate', label: 'Chocolate', icon: ShoppingBasketIcon },
]

export const pantryIcon = ShoppingBasketIcon

export function isCategory(value: unknown): value is PantryItemCategory {
  return categories.some((c) => c.value === value)
}

// categoryLabel shows an unknown category of a newer API as its text.
export function categoryLabel(value: string): string {
  return categories.find((c) => c.value === value)?.label ?? value
}

export function categoryIcon(value: string | undefined): typeof ShoppingBasketIcon {
  return categories.find((c) => c.value === value)?.icon ?? ShoppingBasketIcon
}

// pantryItemMeta reads like "Wine · Weingut Example · Mosel, Germany".
export function pantryItemMeta(item: Pick<PantryItem, 'category' | 'producer' | 'origin'>): string {
  return [categoryLabel(item.category), item.producer, item.origin].filter(Boolean).join(' · ')
}

export const pantryFilterDefs: FilterDef[] = [
  { key: 'search', label: 'Search the pantry', type: 'search', placeholder: 'Search the pantry', chip: (v) => `Search: ${v}` },
  {
    key: 'tags',
    label: 'Tags',
    type: 'tags',
    help: 'Shows pantry items with at least one of the tags.',
    suggest: tagSuggester('pantry', { onlyUsed: true }),
    chip: (v) => `Tag: ${v}`,
  },
  {
    key: 'overall',
    label: 'Overall',
    type: 'select',
    options: [9, 8, 7, 6, 5].map((n) => ({ value: String(n), label: `${n} or more` })),
    help: 'Pantry items without tastings do not match.',
    chip: (v) => `Overall: ${v} or more`,
  },
]

export const pantrySortOptions: Option[] = [
  { value: '-overall', label: 'Highest rated' },
  { value: '-created', label: 'Newest' },
  { value: 'name', label: 'Name' },
]

// oldPantrySortOptions are the sorts that only a link can select.
export const oldPantrySortOptions: Option[] = [
  { value: 'created', label: 'Oldest' },
  { value: 'overall', label: 'Lowest rated' },
  { value: '-name', label: 'Name, Z to A' },
  { value: '-updated', label: 'Recently updated' },
]

// pantryFilterParams changes the filters and the category to the query
// parameters of the SDK.
export function pantryFilterParams(f: Record<string, FilterValue>, category: PantryItemCategory | undefined) {
  const tags = f.tags as string[] | undefined
  const overall = Number(f.overall)
  return {
    search: typeof f.search === 'string' ? f.search : undefined,
    tagsHasAny: tags?.length ? tags : undefined,
    overall: f.overall !== undefined && Number.isFinite(overall) ? { gte: overall } : undefined,
    categoryEq: category,
    listEq: typeof f.list === 'string' ? f.list : undefined,
  }
}
