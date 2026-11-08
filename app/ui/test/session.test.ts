import { afterEach, describe, expect, it, vi } from 'vitest'
import { refreshSession } from '@/composables/useAuth'
import { handleSessionEnded, restaurantsApi } from '@/lib/api'
import { clearSession, saveSession, signedIn } from '@/lib/session'

const user = { id: 'u5n1c8r3t6y0b2e', email: 'ana@example.com', name: 'Ana', created: '', updated: '' }

afterEach(() => {
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

  it('keeps the session in localStorage', () => {
    saveSession({ token: 't', expires: '2099-01-01 00:00:00.000Z', user })
    expect(JSON.parse(localStorage.getItem('reviews-session')!).user.name).toBe('Ana')
  })
})
