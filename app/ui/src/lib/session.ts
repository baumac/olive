import type { AuthToken, User } from '@reviews/sdk'
import { computed, ref } from 'vue'

const storageKey = 'reviews-session'

// Session has no token. The token is in an HttpOnly cookie. See decision
// 00027.
export interface Session {
  expires: string
  user: User
  // token is only in a session that an older version stored.
  token?: string
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
// A session from before decision 00034 has no approved value. Its user was
// approved by the migration.
export const pendingApproval = computed(() => session.value?.user.approved === false)

// legacyToken returns the token of a session from an older version. The
// first refresh moves it into the cookie.
export function legacyToken(): string | undefined {
  return session.value?.token
}

export function sessionExpires(): string | null {
  return session.value?.expires ?? null
}

export function saveSession(auth: AuthToken) {
  session.value = { expires: auth.expires, user: auth.user }
  try {
    localStorage.setItem(storageKey, JSON.stringify(session.value))
  } catch {
    // The session then lasts until the page closes.
  }
}

// saveUser replaces the stored user after a change of the account.
export function saveUser(user: User) {
  if (session.value) {
    saveSession({ expires: session.value.expires, user })
  }
}

// watchOtherTabs keeps the session equal to the session of the other tabs.
// It runs onUserChange when another tab changes the signed in user. The
// browser sends the storage event only to the other tabs.
export function watchOtherTabs(onUserChange: (before: User | null, after: User | null) => void): () => void {
  const listener = (event: StorageEvent) => {
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
  }
  window.addEventListener('storage', listener)
  return () => window.removeEventListener('storage', listener)
}

// markNotApproved changes the stored user after the API refused a write
// with the code not_approved. A superuser took the approval back.
export function markNotApproved() {
  if (session.value?.user.approved) {
    saveSession({ expires: session.value.expires, user: { ...session.value.user, approved: false } })
  }
}

export function clearSession() {
  session.value = null
  try {
    localStorage.removeItem(storageKey)
  } catch {
    // Nothing to remove.
  }
}
