import type { Component } from 'vue'
import { ResponseError } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { usersApi } from '@/lib/api'
import { clearSession, currentUser, saveSession, signedIn } from '@/lib/session'
import AccountPage from '@/pages/AccountPage.vue'
import ConfirmEmailPage from '@/pages/ConfirmEmailPage.vue'
import ForgotPasswordPage from '@/pages/ForgotPasswordPage.vue'
import ResetPasswordPage from '@/pages/ResetPasswordPage.vue'
import SignUpPage from '@/pages/SignUpPage.vue'
import { expectAccessible } from './axe'

const user = { id: 'u5n1c8r3t6y0b2e', email: 'ana@example.com', name: 'Ana', approved: true, created: '2024-11-04T01:26:55.002Z', updated: '2024-11-04T01:26:55.002Z' }
const expires = '2099-01-01T00:00:00.000Z'

function problem(status: number, errors?: { property: string; message: string }[]) {
  const body = JSON.stringify({ type: 'about:blank', title: 'Error', status, errors })
  return new ResponseError(new Response(body, { status, headers: { 'Content-Type': 'application/problem+json' } }))
}

async function mountPage(page: Component, path: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
  await router.push(path)
  const el = document.createElement('div')
  document.body.appendChild(el)
  const w = mount(page, { global: { plugins: [router] }, attachTo: el })
  await flushPromises()
  return { w, router }
}

afterEach(() => {
  vi.restoreAllMocks()
  clearSession()
  document.body.innerHTML = ''
})

describe('SignUpPage', () => {
  it('tells the person to check the email and sends it again', async () => {
    const signUp = vi.spyOn(usersApi, 'signUp').mockResolvedValue()
    const { w } = await mountPage(SignUpPage, '/sign-up')
    await w.get('#sign-up-name').setValue('Ana')
    await w.get('#sign-up-email').setValue('ana@example.com')
    await w.get('#sign-up-password').setValue('correct-horse-battery')
    await w.get('#sign-up-confirm').setValue('correct-horse-battery')
    await w.get('form').trigger('submit')
    await flushPromises()

    const userCreate = { name: 'Ana', email: 'ana@example.com', password: 'correct-horse-battery' }
    expect(signUp).toHaveBeenCalledWith({ userCreate })
    expect(w.text()).toContain('Check your email')
    expect(w.text()).toContain('We sent an email to ana@example.com.')
    expect(signedIn.value).toBe(false)
    await expectAccessible(w.element)

    await w.get('button').trigger('click')
    await flushPromises()
    expect(signUp).toHaveBeenCalledTimes(2)
  })

  it('shows the 422 errors of the API next to the fields', async () => {
    vi.spyOn(usersApi, 'signUp').mockRejectedValue(problem(422, [{ property: 'email', message: 'must be an email address' }]))
    const { w } = await mountPage(SignUpPage, '/sign-up')
    await w.get('#sign-up-name').setValue('Ana')
    await w.get('#sign-up-email').setValue('ana@example.com')
    await w.get('#sign-up-password').setValue('correct-horse-battery')
    await w.get('#sign-up-confirm').setValue('correct-horse-battery')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(w.text()).toContain('Enter an email address, for example ana@example.com.')
    expect(w.text()).not.toContain('Check your email')
  })
})

describe('ConfirmEmailPage', () => {
  it('signs the person in with the token of the link', async () => {
    const confirm = vi.spyOn(usersApi, 'confirmEmail').mockResolvedValue({ expires, user })
    localStorage.setItem('reviews-sign-up-redirect', '/restaurants/new')
    const { w, router } = await mountPage(ConfirmEmailPage, '/confirm-email#token=abc.def')
    expect(confirm).not.toHaveBeenCalled()
    await w.get('button').trigger('click')
    await flushPromises()
    expect(confirm).toHaveBeenCalledWith({ emailConfirmation: { token: 'abc.def' }, cookie: true })
    expect(currentUser.value?.id).toBe(user.id)
    expect(router.currentRoute.value.fullPath).toBe('/restaurants/new')
    expect(localStorage.getItem('reviews-sign-up-redirect')).toBeNull()
  })

  it('tells about a link that does not work', async () => {
    vi.spyOn(usersApi, 'confirmEmail').mockRejectedValue(problem(422, [{ property: 'token', message: 'is not valid or is expired' }]))
    const { w, router } = await mountPage(ConfirmEmailPage, '/confirm-email#token=old')
    await w.get('button').trigger('click')
    await flushPromises()
    expect(w.text()).toContain('This link does not work')
    expect(router.currentRoute.value.hash).toBe('')
    expect(signedIn.value).toBe(false)
    await expectAccessible(w.element)
  })

  it('sends no request without a token', async () => {
    const confirm = vi.spyOn(usersApi, 'confirmEmail')
    const { w } = await mountPage(ConfirmEmailPage, '/confirm-email')
    expect(confirm).not.toHaveBeenCalled()
    expect(w.text()).toContain('This link does not work')
  })
})

describe('ForgotPasswordPage', () => {
  it('checks the email address and sends the request', async () => {
    const request = vi.spyOn(usersApi, 'requestPasswordReset').mockResolvedValue()
    const { w } = await mountPage(ForgotPasswordPage, '/forgot-password')
    await w.get('form').trigger('submit')
    expect(request).not.toHaveBeenCalled()
    expect(w.text()).toContain('Enter your email address.')
    await expectAccessible(w.element)

    await w.get('#forgot-password-email').setValue('ana@example.com')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(request).toHaveBeenCalledWith({ passwordResetRequest: { email: 'ana@example.com' } })
    expect(w.text()).toContain('If ana@example.com has an account')
  })

  it('tells about the rate limit', async () => {
    vi.spyOn(usersApi, 'requestPasswordReset').mockRejectedValue(problem(429))
    const { w } = await mountPage(ForgotPasswordPage, '/forgot-password')
    await w.get('#forgot-password-email').setValue('ana@example.com')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(w.text()).toContain('Too many tries')
  })
})

describe('ResetPasswordPage', () => {
  it('checks the two passwords and sets the new one', async () => {
    const reset = vi.spyOn(usersApi, 'resetPassword').mockResolvedValue()
    saveSession({ expires, user })
    const { w, router } = await mountPage(ResetPasswordPage, '/reset-password#token=abc.def')
    expect(router.currentRoute.value.hash).toBe('')

    await w.get('#reset-password-password').setValue('better-horse-battery')
    await w.get('#reset-password-confirm').setValue('other-horse-battery')
    await w.get('form').trigger('submit')
    expect(reset).not.toHaveBeenCalled()
    expect(w.text()).toContain('The passwords do not match.')
    await expectAccessible(w.element)

    await w.get('#reset-password-confirm').setValue('better-horse-battery')
    expect(w.text()).not.toContain('The passwords do not match.')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(reset).toHaveBeenCalledWith({ passwordReset: { token: 'abc.def', password: 'better-horse-battery' } })
    expect(signedIn.value).toBe(false)
    expect(router.currentRoute.value.fullPath).toBe('/sign-in')
  })

  it('offers a new link when the token is too old', async () => {
    vi.spyOn(usersApi, 'resetPassword').mockRejectedValue(problem(422, [{ property: 'token', message: 'is not valid or is expired' }]))
    const { w } = await mountPage(ResetPasswordPage, '/reset-password#token=old')
    await w.get('#reset-password-password').setValue('better-horse-battery')
    await w.get('#reset-password-confirm').setValue('better-horse-battery')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(w.text()).toContain('This link does not work')
    expect(w.text()).toContain('Ask for a new link')
    expect(w.find('#reset-password-password').exists()).toBe(false)
  })
})

describe('AccountPage', () => {
  it('changes the name of the stored user', async () => {
    saveSession({ expires, user })
    const update = vi.spyOn(usersApi, 'updateUser').mockResolvedValue({ ...user, name: 'Ana Maria' })
    const { w } = await mountPage(AccountPage, '/account')
    await expectAccessible(w.element)
    await w.get('#account-name').setValue('Ana Maria')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(update).toHaveBeenCalledWith({ userPatch: { name: 'Ana Maria' } })
    expect(currentUser.value?.name).toBe('Ana Maria')
  })

  it('shows a wrong current password next to its field', async () => {
    saveSession({ expires, user })
    const change = vi.spyOn(usersApi, 'changePassword').mockRejectedValueOnce(problem(422, [{ property: 'current_password', message: 'is not correct' }]))
    const { w } = await mountPage(AccountPage, '/account')
    const form = w.findAll('form')[1]!
    await form.trigger('submit')
    expect(w.text()).toContain('Enter your current password.')

    await w.get('#account-current-password').setValue('wrong-password')
    await w.get('#account-new-password').setValue('better-horse-battery')
    await w.get('#account-new-confirm').setValue('better-horse-battery')
    await form.trigger('submit')
    await flushPromises()
    expect(w.text()).toContain('The password is not correct.')

    change.mockResolvedValue({ expires: '2099-02-01T00:00:00.000Z', user })
    await form.trigger('submit')
    await flushPromises()
    expect(change).toHaveBeenLastCalledWith({ passwordChange: { currentPassword: 'wrong-password', password: 'better-horse-battery' }, cookie: true })
    expect((w.get('#account-current-password').element as HTMLInputElement).value).toBe('')
  })
})
