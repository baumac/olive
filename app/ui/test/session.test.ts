import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Router } from 'vue-router'
import { toast } from 'vue-sonner'
import { refreshSession, signOut } from '@/composables/useAuth'
import { handleSessionEnded, restaurantsApi } from '@/lib/api'
import { saveDraft, takeDraft } from '@/lib/drafts'
import { clearSession, currentUser, saveSession, signedIn, watchOtherTabs } from '@/lib/session'

const user = { id: 'u5n1c8r3t6y0b2e', email: 'ana@example.com', name: 'Ana', created: '', updated: '' }

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  clearSession()
})

describe('session', () => {
  it('deletes a token that the API refuses and reads again without it', async () => {
    saveSession({ token: 'old-token', expires: '2099-01-01 00:00:00.000Z', user })
    const ended = vi.fn()
    handleSessionEnded(ended)

    const sent: (string | null)[] = []
    vi.stubGlobal('fetch', async (_url: string, init: RequestInit) => {
      const auth = new Headers(init.headers).get('Authorization')
      sent.push(auth)
      if (auth) {
        return new Response(JSON.stringify({ type: 'about:blank', title: 'Unauthorized', status: 401 }), {
          status: 401,
          headers: { 'Content-Type': 'application/problem+json' },
        })
      }
      return new Response(JSON.stringify({ items: [], next_cursor: null }), {
        headers: { 'Content-Type': 'application/json' },
      })
    })

    const page = await restaurantsApi.listRestaurants({})
    expect(page.items).toEqual([])
    expect(sent).toEqual(['Bearer old-token', null])
    expect(signedIn.value).toBe(false)
    expect(ended).toHaveBeenCalledOnce()
  })

  it('ends the session when the token refresh gets 401', async () => {
    saveSession({ token: 'old-token', expires: '2099-01-01 00:00:00.000Z', user })
    const ended = vi.fn()
    handleSessionEnded(ended)
    vi.stubGlobal('fetch', async () =>
      new Response(JSON.stringify({ type: 'about:blank', title: 'Unauthorized', status: 401 }), {
        status: 401,
        headers: { 'Content-Type': 'application/problem+json' },
      }),
    )
    await refreshSession()
    expect(signedIn.value).toBe(false)
    expect(ended).toHaveBeenCalledOnce()
  })

  it('gives a draft only to the user who saved it', () => {
    saveSession({ token: 't', expires: '2099-01-01 00:00:00.000Z', user })
    saveDraft('review:x:new', { title: 'Mine' })
    saveSession({ token: 't2', expires: '2099-01-01 00:00:00.000Z', user: { ...user, id: 'joe0000000000000' } })
    expect(takeDraft('review:x:new')).toBeNull()

    saveSession({ token: 't', expires: '2099-01-01 00:00:00.000Z', user })
    saveDraft('review:x:new', { title: 'Mine' })
    expect(takeDraft('review:x:new')).toEqual({ title: 'Mine' })
  })

  it('sign out sends the token to the server and deletes the session', async () => {
    saveSession({ token: 'old-token', expires: '2099-01-01 00:00:00.000Z', user })
    const sent: { url: string; method?: string; auth: string | null }[] = []
    vi.stubGlobal('fetch', async (url: string, init: RequestInit) => {
      sent.push({ url, method: init.method, auth: new Headers(init.headers).get('Authorization') })
      return new Response(null, { status: 204 })
    })
    const push = vi.fn()

    signOut({ push } as unknown as Router)
    expect(signedIn.value).toBe(false)
    expect(push).toHaveBeenCalledWith('/')
    await vi.waitFor(() => {
      expect(sent).toEqual([{ url: '/v1/users/tokens', method: 'DELETE', auth: 'Bearer old-token' }])
    })
  })

  it('sign out does not send an expired token and warns', async () => {
    saveSession({ token: 'old-token', expires: '2000-01-01 00:00:00.000Z', user })
    const fetch = vi.fn()
    vi.stubGlobal('fetch', fetch)
    const warning = vi.spyOn(toast, 'warning')

    signOut({ push: vi.fn() } as unknown as Router)
    expect(signedIn.value).toBe(false)
    await vi.waitFor(() => expect(warning).toHaveBeenCalledOnce())
    expect(fetch).not.toHaveBeenCalled()
  })

  it('follows the session of the other tabs', ({ onTestFinished }) => {
    const changes: (string | null)[][] = []
    onTestFinished(watchOtherTabs((before, after) => changes.push([before?.name ?? null, after?.name ?? null])))
    const fromOtherTab = (value: string | null) => {
      if (value === null) {
        localStorage.removeItem('reviews-session')
      } else {
        localStorage.setItem('reviews-session', value)
      }
      window.dispatchEvent(new StorageEvent('storage', { key: 'reviews-session', newValue: value }))
    }
    const stored = (token: string) => JSON.stringify({ token, expires: '2099-01-01 00:00:00.000Z', user })

    fromOtherTab(stored('t1'))
    expect(currentUser.value?.name).toBe('Ana')
    // A refresh in the other tab changes only the token.
    fromOtherTab(stored('t2'))
    fromOtherTab(null)
    expect(signedIn.value).toBe(false)
    window.dispatchEvent(new StorageEvent('storage', { key: 'other-key' }))
    expect(changes).toEqual([
      [null, 'Ana'],
      ['Ana', null],
    ])
  })

  it('keeps the session in localStorage', () => {
    saveSession({ token: 't', expires: '2099-01-01 00:00:00.000Z', user })
    expect(JSON.parse(localStorage.getItem('reviews-session')!).user.name).toBe('Ana')
  })
})
