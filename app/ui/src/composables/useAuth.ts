import type { Router } from 'vue-router'
import { toast } from 'vue-sonner'
import { apiFailure, endSession, usersApi } from '@/lib/api'
import { clearSession, saveSession, sessionExpires, sessionToken, signedIn } from '@/lib/session'
import { clearAllDrafts } from '@/lib/drafts'
import { parseTimestamp } from '@/lib/format'

const day = 24 * 60 * 60 * 1000
const hour = 60 * 60 * 1000

export async function refreshSession() {
  if (!signedIn.value) {
    return
  }
  try {
    saveSession(await usersApi.refreshToken())
  } catch (error) {
    const { status } = await apiFailure(error)
    if (status === 401) {
      endSession()
    }
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
async function revokeTokens(token: string): Promise<boolean> {
  try {
    // keepalive lets the request finish after the tab closes.
    await usersApi.signOut({ headers: { Authorization: `Bearer ${token}` }, keepalive: true })
    return true
  } catch (error) {
    // A 401 means that the token is already not valid.
    return (await apiFailure(error)).status === 401
  }
}

export function signOut(router: Router) {
  const token = sessionToken()
  clearSession()
  clearAllDrafts()
  toast('Signed out')
  void router.push('/')
  if (token) {
    void revokeTokens(token).then((revoked) => {
      if (!revoked) {
        toast.warning('Signed out only in this browser', {
          description: 'The server did not answer. Other devices can stay signed in.',
        })
      }
    })
  }
}
