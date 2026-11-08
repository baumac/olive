import type { Router } from 'vue-router'
import { toast } from 'vue-sonner'
import { apiFailure, endSession, usersApi } from '@/lib/api'
import { clearSession, saveSession, sessionExpires, signedIn } from '@/lib/session'
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

export function signOut(router: Router) {
  clearSession()
  toast('Signed out')
  void router.push('/')
}
