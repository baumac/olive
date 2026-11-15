import type { Router } from 'vue-router'
import { toast } from 'vue-sonner'
import { ResponseError } from '@reviews/sdk'
import { apiFailure, endSession, usersApi } from '@/lib/api'
import { clearSession, legacyToken, saveSession, sessionExpires, signedIn } from '@/lib/session'
import { clearAllDrafts } from '@/lib/drafts'
import { parseTimestamp } from '@/lib/format'

const day = 24 * 60 * 60 * 1000
const hour = 60 * 60 * 1000

// RefreshResult tells how a refresh ended. "ended" means that the server
// refused the token and the session is deleted.
export type RefreshResult = 'refreshed' | 'ended' | 'failed' | 'signed-out'

let refreshing: Promise<RefreshResult> | null = null

// refreshSession gets a new token and the current user. Calls at the same
// time share one request.
export function refreshSession(): Promise<RefreshResult> {
  refreshing ??= refresh().finally(() => (refreshing = null))
  return refreshing
}

async function refresh(): Promise<RefreshResult> {
  if (!signedIn.value) {
    return 'signed-out'
  }
  const token = legacyToken()
  try {
    // A bearer token with cookie=true moves the token into the cookie.
    const init = token ? { headers: { Authorization: `Bearer ${token}` } } : undefined
    saveSession(await usersApi.refreshToken({ cookie: true }, init))
    return 'refreshed'
  } catch (error) {
    const { status } = await apiFailure(error)
    if (status === 401) {
      endSession()
      return 'ended'
    }
    return 'failed'
  }
}

// startSessionRefresh refreshes the token at start and when less than one
// day is left before it expires.
export function startSessionRefresh() {
  void refreshSession()
  window.setInterval(() => {
    const expires = sessionExpires()
    if (expires && parseTimestamp(expires).getTime() - Date.now() < day) {
      void refreshSession()
    }
  }, hour)
}

export function signInPath(redirect: string, reason?: 'session-ended') {
  return { path: '/sign-in', query: { redirect, ...(reason ? { reason } : {}) } }
}

// revokeTokens makes all tokens of the user invalid on the server. It
// returns false when the server did not confirm it.
async function revokeTokens(expires: string | null, token: string | undefined): Promise<boolean> {
  // An expired token cannot sign out. The other tokens of the user can still be valid.
  if (expires && parseTimestamp(expires).getTime() <= Date.now()) {
    return false
  }
  try {
    // keepalive lets the request finish after the tab closes.
    const headers = token ? { Authorization: `Bearer ${token}` } : undefined
    await usersApi.signOut({ headers, keepalive: true })
    return true
  } catch (error) {
    // An invalid_token 401 means that a sign out already changed the token
    // key. A plain 401 means that the request had no cookie.
    return error instanceof ResponseError && error.response.status === 401 &&
      (error.response.headers.get('WWW-Authenticate') ?? '').includes('invalid_token')
  }
}

export function signOut(router: Router) {
  const hadSession = signedIn.value
  const expires = sessionExpires()
  const token = legacyToken()
  clearSession()
  clearAllDrafts()
  toast('Signed out')
  void router.push('/')
  if (hadSession) {
    void revokeTokens(expires, token).then((revoked) => {
      if (!revoked) {
        toast.warning('Signed out only in this browser', {
          description: 'The server did not answer. Other devices can stay signed in.',
        })
      }
    })
  }
}
