<script setup lang="ts">
import type { SignInError } from '@/components/ui/sign-in-form'
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { SignInForm } from '@/components/ui/sign-in-form'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, fieldErrors, usersApi } from '@/lib/api'
import { saveSession } from '@/lib/session'
import { safeRedirect } from '@/lib/validation'

usePageTitle('Sign in')

const route = useRoute()
const router = useRouter()

const redirect = computed(() => safeRedirect(route.query.redirect))
const notice = computed(() => (route.query.reason === 'session-ended' ? 'session-ended' : null))
const signUpPath = computed(() => ({ path: '/sign-up', query: { redirect: redirect.value } }))

const loading = ref(false)
const error = ref<SignInError | null>(null)
const errors = ref<Record<string, string>>({})
const form = ref<InstanceType<typeof SignInForm> | null>(null)

async function submit(credentials: { email: string; password: string }) {
  loading.value = true
  error.value = null
  errors.value = {}
  try {
    saveSession(await usersApi.signIn({ credentials }))
    void router.replace(redirect.value)
  } catch (e) {
    const { status, problem } = await apiFailure(e)
    if (status === 401) {
      error.value = { title: 'The email or password is not correct', description: 'Check both and try again.' }
      await form.value?.clearPassword()
    } else if (status === 422) {
      errors.value = fieldErrors(problem)
    } else if (status === 429) {
      error.value = { title: 'Too many tries', description: 'Wait a minute and try again.' }
    } else {
      error.value = { title: 'Sign in did not work', description: 'Try again in a moment.' }
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main id="main" class="mx-auto w-full max-w-sm px-4 pt-8 pb-12">
    <SignInForm
      ref="form"
      :loading="loading"
      :error="error"
      :field-errors="errors"
      :notice="notice"
      :sign-up-path="router.resolve(signUpPath).fullPath"
      @submit="submit"
    />
  </main>
</template>
