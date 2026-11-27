import type { TagSummary } from '@reviews/sdk'
import type { TagSuggestion } from '@/components/ui/tag-input'
import { tagsApi } from '@/lib/api'

export type TagKind = 'restaurants' | 'recipes'

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

export function tagCountText(tag: TagSummary, kind?: TagKind): string {
  const restaurants = plural(tag.restaurantCount, 'restaurant', 'restaurants')
  const recipes = plural(tag.recipeCount, 'recipe', 'recipes')
  if (kind === 'restaurants') {
    return restaurants
  }
  if (kind === 'recipes') {
    return recipes
  }
  return [tag.restaurantCount ? restaurants : '', tag.recipeCount ? recipes : ''].filter(Boolean).join(' and ')
}

function countOf(tag: TagSummary, kind: TagKind): number {
  return kind === 'restaurants' ? tag.restaurantCount : tag.recipeCount
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
