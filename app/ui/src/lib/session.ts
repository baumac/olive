import type { AuthToken, User } from '@reviews/sdk'
import { computed, ref } from 'vue'

const storageKey = 'reviews-session'

export interface Session {
  token: string
  expires: string
  user: User
}

function load(): Session | null {
  try {
    const raw = localStorage.getItem(storageKey)
    return raw ? (JSON.parse(raw) as Session) : null
  } catch {
    return null
  }
}

const session = ref<Session | null>(load())

export const currentUser = computed(() => session.value?.user ?? null)
export const signedIn = computed(() => session.value !== null)

export function sessionToken(): string {
  return session.value?.token ?? ''
}

export function sessionExpires(): string | null {
  return session.value?.expires ?? null
}

export function saveSession(auth: AuthToken) {
  session.value = { token: auth.token, expires: auth.expires, user: auth.user }
  try {
    localStorage.setItem(storageKey, JSON.stringify(session.value))
  } catch {
    // The session then lasts until the page closes.
  }
}

// watchOtherTabs keeps the session equal to the session of the other tabs.
// It runs onUserChange when another tab changes the signed in user. The
// browser sends the storage event only to the other tabs.
export function watchOtherTabs(onUserChange: (before: User | null, after: User | null) => void) {
  window.addEventListener('storage', (event) => {
    // A null key means that all keys were removed.
    if (event.key !== null && event.key !== storageKey) {
      return
    }
    const before = currentUser.value
    session.value = load()
    const after = currentUser.value
    if (before?.id !== after?.id) {
      onUserChange(before, after)
    }
  })
}

export function clearSession() {
  session.value = null
  try {
    localStorage.removeItem(storageKey)
  } catch {
    // Nothing to remove.
  }
}
