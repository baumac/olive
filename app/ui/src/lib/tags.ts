import type { TagSummary } from '@reviews/sdk'
import type { TagSuggestion } from '@/components/ui/tag-input'
import { tagsApi } from '@/lib/api'

export type TagKind = 'restaurants' | 'recipes' | 'pantry'

// suggestionLimit keeps the list of suggestions short. See decision 00011.
const suggestionLimit = 8
// maxSearch is the maxLength of the search of listTags in code points.
const maxSearch = 63

export function tagPath(name: string): string {
  return `/tags/${encodeURIComponent(name)}`
}

function plural(n: number, one: string, many: string): string {
  return `${n} ${n === 1 ? one : many}`
}

const tagKinds: TagKind[] = ['restaurants', 'recipes', 'pantry']

function countText(tag: TagSummary, kind: TagKind): string {
  switch (kind) {
    case 'restaurants':
      return plural(tag.restaurantCount, 'restaurant', 'restaurants')
    case 'recipes':
      return plural(tag.recipeCount, 'recipe', 'recipes')
    case 'pantry':
      return plural(tag.pantryItemCount, 'pantry item', 'pantry items')
  }
}

// joinNames reads like "a", "a and b", or "a, b, and c".
export function joinNames(names: string[]): string {
  if (names.length < 3) {
    return names.join(' and ')
  }
  return `${names.slice(0, -1).join(', ')}, and ${names[names.length - 1]}`
}

// tagCountText reads like "4 restaurants, 1 recipe, and 2 pantry items".
export function tagCountText(tag: TagSummary, kind?: TagKind): string {
  if (kind) {
    return countText(tag, kind)
  }
  return joinNames(tagKinds.filter((k) => countOf(tag, k) > 0).map((k) => countText(tag, k)))
}

function countOf(tag: TagSummary, kind: TagKind): number {
  switch (kind) {
    case 'restaurants':
      return tag.restaurantCount
    case 'recipes':
      return tag.recipeCount
    case 'pantry':
      return tag.pantryItemCount
  }
}

// tagSuggester gets the most used tags of kind that match the text. The API
// puts a tag with the exact name first, also when only the other kind has
// it. onlyUsed drops such a tag. A filter needs it.
export function tagSuggester(kind: TagKind, { onlyUsed = false }: { onlyUsed?: boolean } = {}) {
  return async (text: string): Promise<TagSuggestion[]> => {
    const search = Array.from(text).slice(0, maxSearch).join('').trim()
    const page = await tagsApi.listTags({ kind, search: search || undefined, limit: suggestionLimit })
    const tags = onlyUsed ? page.items.filter((t) => countOf(t, kind) > 0) : page.items
    return tags.map((t) => ({ name: t.name, label: tagCountText(t, kind) }))
  }
}
