<script setup lang="ts">
import type { SignUpError } from '@/components/ui/sign-up-form'
import type { UserCreate } from '@reviews/sdk'
import { MailIcon } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { SignUpForm } from '@/components/ui/sign-up-form'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, fieldErrors, usersApi } from '@/lib/api'
import { keepRedirect } from '@/lib/emailToken'
import { safeRedirect } from '@/lib/validation'

usePageTitle('Create an account')

const route = useRoute()
const router = useRouter()

const redirect = computed(() => safeRedirect(route.query.redirect))
const signInPath = computed(() => router.resolve({ path: '/sign-in', query: { redirect: redirect.value } }).fullPath)

const loading = ref(false)
const error = ref<SignUpError | null>(null)
const errors = ref<Record<string, string>>({})
// sent has the values of the sign up that the API accepted.
const sent = ref<UserCreate | null>(null)

const uiText: Record<string, string> = {
  name: 'Enter your name.',
  email: 'Enter an email address, for example ana@example.com.',
  password: 'Use 8 to 71 characters.',
}

// send returns the HTTP status of a refused sign up, or 0.
async function send(userCreate: UserCreate): Promise<number | null> {
  loading.value = true
  try {
    await usersApi.signUp({ userCreate })
    keepRedirect(redirect.value)
    sent.value = userCreate
    return 0
  } catch (e) {
    const { status, problem } = await apiFailure(e)
    if (status === 422) {
      const api = fieldErrors(problem)
      errors.value = Object.fromEntries(Object.entries(api).map(([k, m]) => [k, uiText[k] ?? m]))
    }
    return status
  } finally {
    loading.value = false
  }
}

async function submit(userCreate: UserCreate) {
  error.value = null
  const status = await send(userCreate)
  if (status === 429) {
    error.value = { title: 'Too many tries', description: 'Wait a minute and try again.' }
  } else if (status !== 0 && status !== 422) {
    error.value = { title: 'Your account was not created', description: 'Try again in a moment.' }
  }
}

async function sendAgain() {
  if (!sent.value) {
    return
  }
  const status = await send(sent.value)
  if (status === 0) {
    toast.success('We sent the email again')
  } else {
    toast.error('The email was not sent', { description: status === 429 ? 'Wait a minute and try again.' : 'Try again in a moment.' })
  }
}
</script>

<template>
  <main id="main" class="mx-auto w-full max-w-sm px-4 pt-8 pb-12">
    <Card v-if="sent" role="status">
      <CardHeader>
        <MailIcon class="text-muted-foreground size-8" aria-hidden="true" />
        <CardTitle as="h1" class="text-2xl">Check your email</CardTitle>
        <CardDescription>
          We sent an email to {{ sent.email }}. Open the link in it to finish your account. The link works for one day.
        </CardDescription>
      </CardHeader>
      <CardContent class="grid gap-3 text-sm">
        <p class="text-muted-foreground">
          If you already have an account with this email address, the email tells you how to sign in.
        </p>
        <Button type="button" variant="outline" :disabled="loading" @click="sendAgain">Send the email again</Button>
        <p class="text-muted-foreground text-center">
          <RouterLink :to="signInPath" class="text-primary font-medium underline-offset-4 hover:underline">
            Go to sign in
          </RouterLink>
        </p>
      </CardContent>
    </Card>
    <SignUpForm v-else :loading="loading" :error="error" :errors="errors" :sign-in-path="signInPath" @submit="submit" />
  </main>
</template>
