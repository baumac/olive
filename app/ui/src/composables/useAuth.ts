import type { Router } from 'vue-router'
import { toast } from 'vue-sonner'
import { ResponseError } from '@reviews/sdk'
import { apiFailure, endSession, usersApi } from '@/lib/api'
import { clearSession, refreshDue, saveSession, saveUser, sessionExpires, signedIn } from '@/lib/session'
import { clearAllDrafts } from '@/lib/drafts'
import { parseTimestamp } from '@/lib/format'

const hour = 60 * 60 * 1000

// CheckResult tells how a check ended. "ended" means that the server
// refused the token and the session is deleted.
export type CheckResult = 'checked' | 'ended' | 'failed' | 'signed-out'

let checking: Promise<CheckResult> | null = null

// checkSession gets the current user from the server. Calls at the same
// time share one request.
export function checkSession(): Promise<CheckResult> {
  checking ??= check().finally(() => (checking = null))
  return checking
}

// check gets a new token only when half of the token time is over. A visit
// then keeps the session alive. Otherwise it gets the user without a new
// token.
async function check(): Promise<CheckResult> {
  if (!signedIn.value) {
    return 'signed-out'
  }
  try {
    if (refreshDue()) {
      saveSession(await usersApi.refreshToken({ cookie: true }))
    } else {
      saveUser(await usersApi.getCurrentUser())
    }
    return 'checked'
  } catch (error) {
    const { status } = await apiFailure(error)
    if (status === 401) {
      endSession()
      return 'ended'
    }
    return 'failed'
  }
}

// startSessionCheck checks the session at start and each hour. It returns
// a function that stops the hourly check.
export function startSessionCheck(): () => void {
  void checkSession()
  const timer = window.setInterval(() => {
    if (refreshDue()) {
      void checkSession()
    }
  }, hour)
  return () => window.clearInterval(timer)
}

export function signInPath(redirect: string, reason?: 'session-ended') {
  return { path: '/sign-in', query: { redirect, ...(reason ? { reason } : {}) } }
}

// revokeTokens makes all tokens of the user invalid on the server. It
// returns false when the server did not confirm it.
async function revokeTokens(expires: string | null): Promise<boolean> {
  // An expired token cannot sign out. The other tokens of the user can still be valid.
  if (expires && parseTimestamp(expires).getTime() <= Date.now()) {
    return false
  }
  try {
    // keepalive lets the request finish after the tab closes.
    await usersApi.signOut({ keepalive: true })
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
  clearSession()
  clearAllDrafts()
  toast('Signed out')
  void router.push('/')
  if (hadSession) {
    void revokeTokens(expires).then((revoked) => {
      if (!revoked) {
        toast.warning('Signed out only in this browser', {
          description: 'The server did not answer. Other devices can stay signed in.',
        })
      }
    })
  }
}
