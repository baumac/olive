import type { Middleware, Problem } from '@reviews/sdk'
import { Configuration, FetchError, ResponseError, RestaurantsApi, ReviewersApi, UsersApi } from '@reviews/sdk'
import { clearSession, sessionToken } from './session'

type SessionEndedHandler = () => void

let onSessionEnded: SessionEndedHandler = () => {}

export function handleSessionEnded(handler: SessionEndedHandler) {
  onSessionEnded = handler
}

// endSession deletes the session and runs the session ended handler.
export function endSession() {
  clearSession()
  onSessionEnded()
}

// A 401 on a request with a token means that the token is not valid. The
// middleware deletes the session and sends the request again without the
// token. A read then works. A write gets a 401 without a token.
const sessionMiddleware: Middleware = {
  async post({ fetch, url, init, response }) {
    const headers = new Headers(init.headers)
    if (response.status !== 401 || !headers.has('Authorization') || url.includes('/users/tokens')) {
      return response
    }
    endSession()
    headers.delete('Authorization')
    return fetch(url, { ...init, headers })
  },
}

const configuration = new Configuration({
  basePath: '/v1',
  accessToken: () => sessionToken(),
  middleware: [sessionMiddleware],
})

export const restaurantsApi = new RestaurantsApi(configuration)
export const usersApi = new UsersApi(configuration)
export const reviewersApi = new ReviewersApi(configuration)

export interface ApiFailure {
  status: number | null
  problem: Problem | null
  // retryAfter is the Retry-After header in seconds, or null without it.
  retryAfter: number | null
}

// apiFailure gives the status and the Problem Details body of a failed
// call. A network error has a null status.
export async function apiFailure(error: unknown): Promise<ApiFailure> {
  if (error instanceof ResponseError) {
    let problem: Problem | null
    try {
      problem = (await error.response.clone().json()) as Problem
    } catch {
      problem = null
    }
    const retryAfter = Number.parseInt(error.response.headers.get('Retry-After') ?? '', 10)
    return { status: error.response.status, problem, retryAfter: Number.isNaN(retryAfter) ? null : retryAfter }
  }
  if (error instanceof FetchError) {
    return { status: null, problem: null, retryAfter: null }
  }
  throw error
}

// fieldErrors maps the property of each 422 error to its API message. An
// array item, such as tags[2], maps to its array property.
export function fieldErrors(problem: Problem | null): Record<string, string> {
  const out: Record<string, string> = {}
  for (const e of problem?.errors ?? []) {
    const key = e.property.replace(/\[\d+\]$/, '')
    const message = e.message.charAt(0).toUpperCase() + e.message.slice(1)
    out[key] ??= message.endsWith('.') ? message : `${message}.`
  }
  return out
}
