import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Router } from 'vue-router'
import { toast } from 'vue-sonner'
import { checkSession, signOut, startSessionCheck } from '@/composables/useAuth'
import { handleSessionEnded, restaurantsApi, usersApi } from '@/lib/api'
import { saveDraft, takeDraft } from '@/lib/drafts'
import { clearSession, currentUser, saveSession, signedIn, watchOtherTabs } from '@/lib/session'

const user = { id: 'u5n1c8r3t6y0b2e', email: 'ana@example.com', name: 'Ana', approved: true, created: '', updated: '' }

const hour = 60 * 60 * 1000
const day = 24 * hour

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  clearSession()
})

describe('session', () => {
  const expires = '2099-01-01T00:00:00.000Z'
  const unauthorized = () =>
    new Response(JSON.stringify({ type: 'about:blank', title: 'Unauthorized', status: 401 }), {
      status: 401,
      headers: { 'Content-Type': 'application/problem+json' },
    })
  const emptyList = () =>
    new Response(JSON.stringify({ items: [], next_cursor: null }), { headers: { 'Content-Type': 'application/json' } })
  const authToken = () =>
    new Response(JSON.stringify({ expires, user }), { headers: { 'Content-Type': 'application/json' } })
  const userResponse = (u = user) =>
    new Response(JSON.stringify(u), { headers: { 'Content-Type': 'application/json' } })
  // settle lets the pending fetch calls finish. The fake timers do not change
  // setTimeout.
  const settle = () => new Promise((resolve) => setTimeout(resolve, 10))
  const sendTo = (sent: string[]) => async (url: string) => {
    sent.push(url)
    return url.endsWith('/users/me') ? userResponse() : authToken()
  }

  // fromOtherTab writes the session as another tab does. The browser then
  // sends a storage event to this tab.
  const fromOtherTab = (value: string | null) => {
    if (value === null) {
      localStorage.removeItem('reviews-session')
    } else {
      localStorage.setItem('reviews-session', value)
    }
    window.dispatchEvent(new StorageEvent('storage', { key: 'reviews-session', newValue: value }))
  }

  it('ends a session that the API refuses and reads again without the cookie', async () => {
    saveSession({ expires, user })
    const ended = vi.fn()
    handleSessionEnded(ended)

    const sent: (RequestCredentials | undefined)[] = []
    vi.stubGlobal('fetch', async (_url: string, init: RequestInit) => {
      sent.push(init.credentials)
      return init.credentials === 'omit' ? emptyList() : unauthorized()
    })

    const page = await restaurantsApi.listRestaurants({})
    expect(page.items).toEqual([])
    expect(sent).toEqual(['same-origin', 'omit'])
    expect(signedIn.value).toBe(false)
    expect(ended).toHaveBeenCalledOnce()
  })

  it('sends no cookie without a session, except to the user operations', async () => {
    const sent: { url: string; credentials?: RequestCredentials }[] = []
    vi.stubGlobal('fetch', async (url: string, init: RequestInit) => {
      sent.push({ url, credentials: init.credentials })
      return url.includes('/users') ? authToken() : emptyList()
    })

    await restaurantsApi.listRestaurants({})
    await usersApi.signIn({ credentials: { email: 'ana@example.com', password: 'correct-horse-battery' }, cookie: true })
    expect(sent).toEqual([
      { url: '/v1/restaurants', credentials: 'omit' },
      { url: '/v1/users/tokens?cookie=true', credentials: 'same-origin' },
    ])
  })

  it('ends the session when the check gets 401', async () => {
    saveSession({ expires, user })
    const ended = vi.fn()
    handleSessionEnded(ended)
    vi.stubGlobal('fetch', async () => unauthorized())
    expect(await checkSession()).toBe('ended')
    expect(signedIn.value).toBe(false)
    expect(ended).toHaveBeenCalledOnce()
  })

  it('gets the user without a new token before half of the token time is over', async () => {
    saveSession({ expires, user })
    const sent: { url: string; method?: string }[] = []
    vi.stubGlobal('fetch', async (url: string, init: RequestInit) => {
      sent.push({ url, method: init.method })
      return userResponse({ ...user, name: 'Ana Maria' })
    })

    expect(await checkSession()).toBe('checked')
    expect(sent).toEqual([{ url: '/v1/users/me', method: 'GET' }])
    expect(currentUser.value?.name).toBe('Ana Maria')
    expect(JSON.parse(localStorage.getItem('reviews-session')!).expires).toBe(expires)
  })

  it('refreshes the token on a visit after half of the token time', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    saveSession({ expires: new Date(Date.now() + 7 * day).toISOString(), user })
    const sent: string[] = []
    vi.stubGlobal('fetch', sendTo(sent))

    vi.setSystemTime(Date.now() + 3 * day)
    await checkSession()
    vi.setSystemTime(Date.now() + 1 * day)
    await checkSession()
    expect(sent).toEqual(['/v1/users/me', '/v1/users/tokens/refresh?cookie=true'])
    expect(JSON.parse(localStorage.getItem('reviews-session')!).expires).toBe(expires)
  })

  it('does not write an unchanged user', async () => {
    saveSession({ expires, user })
    vi.stubGlobal('fetch', async () => userResponse())
    const write = vi.spyOn(Storage.prototype, 'setItem')

    await checkSession()
    expect(write).not.toHaveBeenCalled()
  })

  it('refreshes the token each hour after half of the token time', async ({ onTestFinished }) => {
    vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] })
    saveSession({ expires: new Date(Date.now() + 7 * day).toISOString(), user })
    const sent: string[] = []
    vi.stubGlobal('fetch', sendTo(sent))

    onTestFinished(startSessionCheck())
    await settle()
    vi.advanceTimersByTime(3 * day)
    await settle()
    expect(sent).toEqual(['/v1/users/me'])

    vi.advanceTimersByTime(day)
    await settle()
    expect(sent).toEqual(['/v1/users/me', '/v1/users/tokens/refresh?cookie=true'])
  })

  it('gives a draft only to the user who saved it', () => {
    saveSession({ expires, user })
    saveDraft('review:x:new', { title: 'Mine' })
    saveSession({ expires, user: { ...user, id: 'joe0000000000000' } })
    expect(takeDraft('review:x:new')).toBeNull()

    saveSession({ expires, user })
    saveDraft('review:x:new', { title: 'Mine' })
    expect(takeDraft('review:x:new')).toEqual({ title: 'Mine' })
  })

  it('sign out sends the cookie to the server and deletes the session', async () => {
    saveSession({ expires, user })
    const sent: { url: string; method?: string; credentials?: RequestCredentials; keepalive?: boolean }[] = []
    vi.stubGlobal('fetch', async (url: string, init: RequestInit) => {
      sent.push({ url, method: init.method, credentials: init.credentials, keepalive: init.keepalive })
      return new Response(null, { status: 204 })
    })
    const push = vi.fn()

    signOut({ push } as unknown as Router)
    expect(signedIn.value).toBe(false)
    expect(push).toHaveBeenCalledWith('/')
    await vi.waitFor(() => {
      expect(sent).toEqual([{ url: '/v1/users/tokens', method: 'DELETE', credentials: 'same-origin', keepalive: true }])
    })
  })

  it('sign out warns only when a 401 does not say that the token is invalid', async () => {
    for (const [header, warns] of [
      ['Bearer error="invalid_token"', false],
      ['Bearer', true],
    ] as const) {
      saveSession({ expires, user })
      vi.stubGlobal('fetch', async () => {
        const res = unauthorized()
        res.headers.set('WWW-Authenticate', header)
        return res
      })
      const warning = vi.spyOn(toast, 'warning').mockClear()
      signOut({ push: vi.fn() } as unknown as Router)
      if (warns) {
        await vi.waitFor(() => expect(warning).toHaveBeenCalledOnce())
      } else {
        await new Promise((resolve) => setTimeout(resolve, 20))
        expect(warning).not.toHaveBeenCalled()
      }
    }
  })

  it('sign out does not send an expired session and warns', async () => {
    saveSession({ expires: '2000-01-01T00:00:00.000Z', user })
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
    const stored = (expires: string) => JSON.stringify({ expires, user })

    fromOtherTab(stored('2099-01-01T00:00:00.000Z'))
    expect(currentUser.value?.name).toBe('Ana')
    // A refresh in the other tab changes only the expiry.
    fromOtherTab(stored('2099-01-02T00:00:00.000Z'))
    fromOtherTab(null)
    expect(signedIn.value).toBe(false)
    window.dispatchEvent(new StorageEvent('storage', { key: 'other-key' }))
    expect(changes).toEqual([
      [null, 'Ana'],
      ['Ana', null],
    ])
  })

  it('keeps the session in localStorage without a token', () => {
    saveSession({ token: 't', expires, user })
    const stored = JSON.parse(localStorage.getItem('reviews-session')!)
    expect(stored.user.name).toBe('Ana')
    expect(stored).not.toHaveProperty('token')
  })
})
