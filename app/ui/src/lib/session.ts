import type { AuthToken, User } from '@reviews/sdk'
import { computed, ref } from 'vue'
import { parseTimestamp } from './format'

const storageKey = 'reviews-session'

// Session has no token. The token is in an HttpOnly cookie. See decision
// 00003.
export interface Session {
  expires: string
  // issued is the browser time when the UI got the token. A session of an
  // older build can miss it.
  issued?: string
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
export const pendingApproval = computed(() => session.value?.user.approved === false)

export function sessionExpires(): string | null {
  return session.value?.expires ?? null
}

// refreshDue is true when half of the time of the token is over. A session
// without an issued value is due.
export function refreshDue(): boolean {
  const current = session.value
  if (!current) {
    return false
  }
  if (!current.issued) {
    return true
  }
  const issued = parseTimestamp(current.issued).getTime()
  const expires = parseTimestamp(current.expires).getTime()
  return expires - Date.now() < (expires - issued) / 2
}

export function saveSession(auth: AuthToken) {
  store({ expires: auth.expires, issued: new Date().toISOString(), user: auth.user })
}

function store(next: Session) {
  session.value = next
  try {
    localStorage.setItem(storageKey, JSON.stringify(session.value))
  } catch {
    // The session then lasts until the page closes.
  }
}

// saveUser replaces the stored user after a change of the account. An equal
// user makes no write. Each write sends a storage event to the other tabs.
export function saveUser(user: User) {
  const current = session.value
  if (current && JSON.stringify(current.user) !== JSON.stringify(user)) {
    store({ ...current, user })
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
  const current = session.value
  if (current?.user.approved) {
    store({ ...current, user: { ...current.user, approved: false } })
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
