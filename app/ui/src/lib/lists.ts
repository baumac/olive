import { apiFailure, fieldErrors } from './api'

// listFailureText gives the text for a failed create, rename, or change of
// a list. fallback is the text for a failure without a known reason.
export async function listFailureText(error: unknown, fallback: string): Promise<string> {
  const { status, problem } = await apiFailure(error)
  if (status === 409) {
    return problem?.property === 'name' ? 'You have a list with this name.' : (problem?.detail ?? fallback)
  }
  if (status === 422) {
    return fieldErrors(problem).name ?? 'The name is not valid.'
  }
  return status === 401 ? 'Sign in again, then try again.' : fallback
}
