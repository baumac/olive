import type { SavedList } from '@reviews/sdk'
import { apiFailure, failureText, fieldErrors, listsApi } from './api'
import { formatCount } from './format'

// listCountText names the restaurants, the recipes, and the pantry items of
// a list. A list without items has 0 restaurants.
export function listCountText(list: Pick<SavedList, 'restaurantCount' | 'recipeCount' | 'pantryItemCount'>): string {
  const parts = [
    list.restaurantCount ? formatCount(list.restaurantCount, 'restaurant') : '',
    list.recipeCount ? formatCount(list.recipeCount, 'recipe') : '',
    list.pantryItemCount ? formatCount(list.pantryItemCount, 'pantry item') : '',
  ].filter(Boolean)
  return parts.length ? parts.join(', ') : formatCount(0, 'restaurant')
}

// itemCountOf counts the items of all kinds of a list.
export function itemCountOf(list: Pick<SavedList, 'restaurantCount' | 'recipeCount' | 'pantryItemCount'>): number {
  return list.restaurantCount + list.recipeCount + list.pantryItemCount
}

let cached: { userId: string; lists: Promise<SavedList[]> } | undefined

// ownLists gets the lists of a user one time per page load. A failed
// request is not kept.
export function ownLists(userId: string): Promise<SavedList[]> {
  if (cached?.userId !== userId) {
    const lists = listsApi.listReviewerLists({ reviewerId: userId }).then((page) => page.items)
    lists.catch(() => {
      if (cached?.lists === lists) {
        cached = undefined
      }
    })
    cached = { userId, lists }
  }
  return cached.lists
}

// forgetOwnLists must follow each create, rename, and delete of a list.
export function forgetOwnLists() {
  cached = undefined
}

// listFailureText gives the text for a failed create, rename, or change of
// a list. fallback is the text for a failure without a known reason.
export async function listFailureText(error: unknown, fallback: string): Promise<string> {
  const failure = await apiFailure(error)
  switch (failure.problem?.code) {
    case 'list_name_taken':
      return 'You have a list with this name.'
    case 'list_limit':
      return 'You have the maximum number of lists. Delete a list first.'
    case 'list_restaurant_limit':
      return 'This list is full. Remove a restaurant first.'
    case 'list_recipe_limit':
      return 'This list is full. Remove a recipe first.'
    case 'list_pantry_item_limit':
      return 'This list is full. Remove a pantry item first.'
    case 'favorites_list':
      return 'The favorites list cannot be renamed or deleted.'
    case 'not_creator':
      return 'Only the person who made the list can change it.'
  }
  if (failure.status === 422) {
    return fieldErrors(failure.problem).name ?? 'The name is not valid.'
  }
  return failureText(failure, fallback)
}
