import { apiFailure, failureText, fieldErrors } from './api'

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
