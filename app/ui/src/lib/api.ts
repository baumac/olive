import type { Middleware, Problem } from '@reviews/sdk'
import { Configuration, FetchError, ListsApi, PlacesApi, ResponseError, RestaurantsApi, ReviewersApi, UsersApi } from '@reviews/sdk'
import { clearSession, markNotApproved, signedIn } from './session'

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

// The user operations set and delete the session cookie.
const usersPath = /\/v1\/users(\/|\?|$)/

const sessionMiddleware: Middleware = {
  // Without a session, requests leave out the cookie. A cookie that a failed
  // sign out left then has no effect.
  pre({ url, init }) {
    if (init.credentials !== undefined) {
      return Promise.resolve({ url, init })
    }
    const credentials: RequestCredentials = signedIn.value || usersPath.test(url) ? 'same-origin' : 'omit'
    return Promise.resolve({ url, init: { ...init, credentials } })
  },
  // A 401 on a request with the cookie ends the session. The response deleted
  // the cookie, so the second request has no token.
  async post({ fetch, url, init, response }) {
    if (response.status === 403 && (await problemCode(response)) === 'not_approved') {
      // The approval banner then tells the reason of the refused write.
      markNotApproved()
      return response
    }
    if (response.status !== 401 || init.credentials === 'omit' || usersPath.test(url)) {
      return response
    }
    endSession()
    return fetch(url, { ...init, credentials: 'omit' })
  },
}

async function problemCode(response: Response): Promise<string | undefined> {
  try {
    return ((await response.clone().json()) as Problem).code
  } catch {
    return undefined
  }
}

const configuration = new Configuration({
  basePath: '/v1',
  middleware: [sessionMiddleware],
})

export const restaurantsApi = new RestaurantsApi(configuration)
export const usersApi = new UsersApi(configuration)
export const reviewersApi = new ReviewersApi(configuration)
export const placesApi = new PlacesApi(configuration)
export const listsApi = new ListsApi(configuration)

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

// writeLimitText is the text of a 429 on a write. The limit of the user can
// wait up to one hour.
export const writeLimitText = 'You made many changes in a short time. Try again later.'

// retryText is the text for a failed write. fallback is the text for a
// failure without a known reason.
export function retryText(status: number | null, fallback = 'Try again in a moment.'): string {
  if (status === 429) {
    return writeLimitText
  }
  return status === 401 ? 'Sign in again, then try again.' : fallback
}

// fieldErrors maps the property of each 422 error to its API message. An
// array item, such as tags[2], maps to its array property. A nested
// property, such as location.address, maps to its object property.
export function fieldErrors(problem: Problem | null): Record<string, string> {
  const out: Record<string, string> = {}
  for (const e of problem?.errors ?? []) {
    const key = e.property.replace(/(\[\d+\]|\..*)$/, '')
    const message = e.message.charAt(0).toUpperCase() + e.message.slice(1)
    out[key] ??= message.endsWith('.') ? message : `${message}.`
  }
  return out
}
