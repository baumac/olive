import type { RouteLocationNormalizedLoaded, Router } from 'vue-router'
import { safeRedirect } from './validation'

const redirectKey = 'reviews-sign-up-redirect'

// keepRedirect stores the path to open after the confirmation. The link of
// the email opens a new tab, so the path is in localStorage.
export function keepRedirect(path: string) {
  try {
    localStorage.setItem(redirectKey, path)
  } catch {
    // The confirmation then opens the default path.
  }
}

// takeRedirect returns the stored path one time.
export function takeRedirect(): string {
  try {
    const path = localStorage.getItem(redirectKey)
    localStorage.removeItem(redirectKey)
    return safeRedirect(path)
  } catch {
    return safeRedirect(null)
  }
}

// takeEmailToken returns the token of a link from an email. The link has it
// after "#token=". It then removes the token from the address and from the
// browser history.
export function takeEmailToken(route: RouteLocationNormalizedLoaded, router: Router): string {
  const token = new URLSearchParams(route.hash.slice(1)).get('token') ?? ''
  if (route.hash) {
    void router.replace({ path: route.path, query: route.query })
  }
  return token
}
