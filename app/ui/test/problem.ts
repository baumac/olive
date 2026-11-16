import type { Problem } from '@reviews/sdk'
import { ResponseError } from '@reviews/sdk'

// rejectWith makes the error of an API call that failed with a problem.
export function rejectWith(status: number, fields: Partial<Problem> = {}) {
  const body = JSON.stringify({ type: 'about:blank', title: 'Error', status, ...fields })
  return new ResponseError(new Response(body, { status, headers: { 'Content-Type': 'application/problem+json' } }))
}
