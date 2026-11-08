// Drafts keep what a person typed when a request needs a new sign in.

const prefix = 'reviews-draft:'

export function saveDraft(key: string, value: unknown) {
  try {
    sessionStorage.setItem(prefix + key, JSON.stringify(value))
  } catch {
    // Without storage, the draft is lost after sign in.
  }
}

export function takeDraft<T>(key: string): T | null {
  try {
    const raw = sessionStorage.getItem(prefix + key)
    sessionStorage.removeItem(prefix + key)
    return raw ? (JSON.parse(raw) as T) : null
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
