import { FetchError } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import ApprovalBanner from '@/components/app/ApprovalBanner.vue'
import FavoriteButton from '@/components/app/FavoriteButton.vue'
import SaveToListMenu from '@/components/app/SaveToListMenu.vue'
import { listsApi, usersApi } from '@/lib/api'
import { clearSession, currentUser, markNotApproved, pendingApproval, saveSession } from '@/lib/session'
import { router } from '@/router'
import { expectAccessible } from './axe'
import { rejectWith } from './problem'

const user = { id: 'u5n1c8r3t6y0b2e', email: 'ana@example.com', name: 'Ana', approved: false, created: '2024-11-04T01:26:55.002Z', updated: '2024-11-04T01:26:55.002Z' }
const expires = '2099-01-01T00:00:00.000Z'

function attach() {
  const el = document.createElement('div')
  document.body.appendChild(el)
  return el
}

afterEach(() => {
  vi.restoreAllMocks()
  clearSession()
})

describe('ApprovalBanner', () => {
  it('shows nothing for a visitor and for an approved user', () => {
    const w = mount(ApprovalBanner, { attachTo: attach() })
    expect(w.text()).toBe('')
    saveSession({ expires, user: { ...user, approved: true } })
    expect(w.text()).toBe('')
  })

  it('shows nothing for a session without the approved value', async () => {
    localStorage.setItem('reviews-session', JSON.stringify({ expires, user: { id: user.id, email: user.email, name: user.name } }))
    window.dispatchEvent(new StorageEvent('storage', { key: 'reviews-session' }))
    expect(pendingApproval.value).toBe(false)
  })

  it('tells a new user to wait and gets the approval', async () => {
    saveSession({ expires, user })
    const w = mount(ApprovalBanner, { attachTo: attach() })
    expect(w.text()).toContain('Your account waits for approval')
    await expectAccessible(w.element)

    const refresh = vi.spyOn(usersApi, 'refreshToken').mockResolvedValue({ expires, user })
    await w.get('button').trigger('click')
    await flushPromises()
    expect(refresh).toHaveBeenCalledWith({ cookie: true }, undefined)
    expect(w.text()).toContain('Your account waits for approval')

    refresh.mockResolvedValue({ expires, user: { ...user, approved: true } })
    await w.get('button').trigger('click')
    await flushPromises()
    expect(w.text()).toBe('')
    expect(currentUser.value?.approved).toBe(true)
  })
})

describe('ApprovalBanner after a refresh that did not work', () => {
  it('tells that the check did not work', async () => {
    saveSession({ expires, user })
    const w = mount(ApprovalBanner, { attachTo: attach() })
    vi.spyOn(usersApi, 'refreshToken').mockRejectedValue(new FetchError(new Error('offline')))
    const info = vi.spyOn(toast, 'info')
    const error = vi.spyOn(toast, 'error')
    await w.get('button').trigger('click')
    await flushPromises()
    expect(error).toHaveBeenCalledWith('The check did not work', expect.anything())
    expect(info).not.toHaveBeenCalled()
    expect(w.text()).toContain('Your account waits for approval')
  })

  it('does not tell an approval when the session ended', async () => {
    saveSession({ expires, user })
    const w = mount(ApprovalBanner, { attachTo: attach() })
    vi.spyOn(usersApi, 'refreshToken').mockRejectedValue(rejectWith(401))
    const success = vi.spyOn(toast, 'success')
    await w.get('button').trigger('click')
    await flushPromises()
    expect(success).not.toHaveBeenCalled()
    expect(currentUser.value).toBeNull()
  })
})

describe('the route guard', () => {
  // The router keeps its route between tests. Each test starts on a page.
  async function onPage() {
    saveSession({ expires, user: { ...user, approved: true } })
    await router.push('/my-reviews')
  }

  it('keeps a user who waits on the page', async () => {
    await onPage()
    saveSession({ expires, user })
    const refresh = vi.spyOn(usersApi, 'refreshToken')
    const info = vi.spyOn(toast, 'info')
    for (const path of ['/restaurants/new', '/restaurants/k3x9m2p7q1w8z4a/edit', '/restaurants/k3x9m2p7q1w8z4a/reviews/new', '/restaurants/k3x9m2p7q1w8z4a/reviews/r1b8c5e2g9i6k3m/edit']) {
      await router.push(path)
      expect(router.currentRoute.value.path).toBe('/my-reviews')
    }
    expect(info).toHaveBeenCalledTimes(4)
    expect(refresh).not.toHaveBeenCalled()
  })

  it('lets an approved user open a form', async () => {
    await onPage()
    await router.push('/restaurants/new')
    expect(router.currentRoute.value.path).toBe('/restaurants/new')
  })

  it('marks the user after the API refuses a write', () => {
    saveSession({ expires, user: { ...user, approved: true } })
    markNotApproved()
    expect(pendingApproval.value).toBe(true)
  })
})

describe('SaveToListMenu of a user who waits for approval', () => {
  it('does not open', async () => {
    saveSession({ expires, user })
    const load = vi.spyOn(listsApi, 'listReviewerLists')
    const memory = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
    await memory.push('/restaurants/k3x9m2p7q1w8z4a')
    const w = mount(SaveToListMenu, { props: { restaurantId: 'k3x9m2p7q1w8z4a' }, global: { plugins: [memory] }, attachTo: attach() })
    await w.get('button').trigger('click')
    await flushPromises()
    expect(load).not.toHaveBeenCalled()
    expect(document.body.querySelector('#save-to-list-name')).toBeNull()
    w.unmount()
  })
})

describe('FavoriteButton of a user who waits for approval', () => {
  it('sends no request', async () => {
    saveSession({ expires, user })
    const add = vi.spyOn(listsApi, 'addListRestaurant')
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
    await router.push('/restaurants')
    const w = mount(FavoriteButton, {
      props: { restaurantId: 'k3x9m2p7q1w8z4a', name: 'Trattoria', favorite: false },
      global: { plugins: [router] },
      attachTo: attach(),
    })
    await w.get('button').trigger('click')
    await flushPromises()
    expect(add).not.toHaveBeenCalled()
    expect(w.get('button').attributes('aria-pressed')).toBe('false')
    expect(router.currentRoute.value.path).toBe('/restaurants')
  })
})
