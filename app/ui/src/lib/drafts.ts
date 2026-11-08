import { currentUser } from './session'

// Drafts keep what a person typed when a request needs a new sign in. Each
// draft has the ID of its user. Another user who signs in on the same tab
// does not get it.

const prefix = 'reviews-draft:'

interface Stored<T> {
  user: string | null
  value: T
}

export function saveDraft(key: string, value: unknown) {
  try {
    const stored: Stored<unknown> = { user: currentUser.value?.id ?? null, value }
    sessionStorage.setItem(prefix + key, JSON.stringify(stored))
  } catch {
    // Without storage, the draft is lost after sign in.
  }
}

export function takeDraft<T>(key: string): T | null {
  try {
    const raw = sessionStorage.getItem(prefix + key)
    sessionStorage.removeItem(prefix + key)
    if (!raw) {
      return null
    }
    const stored = JSON.parse(raw) as Stored<T>
    return stored.user === (currentUser.value?.id ?? null) ? stored.value : null
  } catch {
    return null
  }
}

export function clearDraft(key: string) {
  try {
    sessionStorage.removeItem(prefix + key)
  } catch {
    // Nothing to remove.
  }
}

export function clearAllDrafts() {
  try {
    for (const key of Object.keys(sessionStorage)) {
      if (key.startsWith(prefix)) {
        sessionStorage.removeItem(key)
      }
    }
  } catch {
    // Nothing to remove.
  }
}
