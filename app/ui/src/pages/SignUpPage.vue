<script setup lang="ts">
import type { SignUpError } from '@/components/ui/sign-up-form'
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { SignUpForm } from '@/components/ui/sign-up-form'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, fieldErrors, usersApi } from '@/lib/api'
import { saveSession } from '@/lib/session'
import { safeRedirect } from '@/lib/validation'

usePageTitle('Create an account')

const route = useRoute()
const router = useRouter()

const redirect = computed(() => safeRedirect(route.query.redirect))
const signInPath = computed(() => router.resolve({ path: '/sign-in', query: { redirect: redirect.value } }).fullPath)

const loading = ref(false)
const error = ref<SignUpError | null>(null)
const errors = ref<Record<string, string>>({})

const uiText: Record<string, string> = {
  name: 'Enter your name.',
  email: 'Enter an email address, for example ana@example.com.',
  password: 'Use 8 to 71 characters.',
}

async function submit(userCreate: { name: string; email: string; password: string }) {
  loading.value = true
  error.value = null
  try {
    const auth = await usersApi.signUp({ userCreate, cookie: true })
    saveSession(auth)
    void router.replace(redirect.value)
    toast.success('Welcome to Sponga47 Reviews', { description: `You are signed in as ${auth.user.name}.` })
  } catch (e) {
    const { status, problem } = await apiFailure(e)
    if (status === 409) {
      error.value = {
        title: 'An account with this email already exists',
        description: 'Sign in with this email instead.',
        signIn: true,
      }
    } else if (status === 422) {
      const api = fieldErrors(problem)
      errors.value = Object.fromEntries(Object.entries(api).map(([k, m]) => [k, uiText[k] ?? m]))
    } else if (status === 429) {
      error.value = { title: 'Too many tries', description: 'Wait a minute and try again.' }
    } else {
      error.value = { title: 'Your account was not created', description: 'Try again in a moment.' }
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main id="main" class="mx-auto w-full max-w-sm px-4 pt-8 pb-12">
    <SignUpForm :loading="loading" :error="error" :errors="errors" :sign-in-path="signInPath" @submit="submit" />
  </main>
</template>
