<script setup lang="ts">
import { CircleAlertIcon } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { FormControl, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { PasswordInput } from '@/components/ui/password-input'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, fieldErrors, usersApi } from '@/lib/api'
import { clearAllDrafts } from '@/lib/drafts'
import { takeEmailToken, takeRedirect } from '@/lib/emailToken'
import { currentUser, saveSession } from '@/lib/session'

usePageTitle('Confirm your email address')

const route = useRoute()
const router = useRouter()
const token = takeEmailToken(route, router)

type Failure = 'invalid' | 'limit' | 'other'
const failure = ref<Failure | null>(token ? null : 'invalid')
const loading = ref(false)
const password = ref('')
const passwordError = ref('')
const passwordInput = ref<InstanceType<typeof PasswordInput> | null>(null)

const failureText: Record<Failure, { title: string; description: string }> = {
  invalid: {
    title: 'This link does not work',
    description: 'The link is not complete, is too old, or was used before. Sign in to get a new link.',
  },
  limit: { title: 'Too many tries', description: 'Wait a minute, then open the link again.' },
  other: { title: 'Your email address was not confirmed', description: 'Open the link again in a moment.' },
}

// The link alone does not confirm the account. Another person can sign up
// with this email address and set the password. See decision 00003.
async function confirm() {
  passwordError.value = password.value === '' ? 'Enter your password.' : ''
  if (passwordError.value) {
    passwordInput.value?.focus()
    return
  }
  loading.value = true
  try {
    const auth = await usersApi.confirmEmail({ emailConfirmation: { token, password: password.value }, cookie: true })
    if (currentUser.value && currentUser.value.id !== auth.user.id) {
      clearAllDrafts()
    }
    saveSession(auth)
    void router.replace(takeRedirect())
    const next = auth.user.approved ? '' : ' An administrator must approve your account before you can add data.'
    toast.success('Welcome to Sponga47 Food', { description: `You are signed in as ${auth.user.name}.${next}` })
  } catch (e) {
    const { status, problem } = await apiFailure(e)
    if (status === 429 && problem?.code === 'wrong_password_limit') {
      passwordError.value = 'Too many wrong passwords. Try again later.'
    } else if (status === 422 && fieldErrors(problem).password) {
      passwordError.value = 'The password is not correct. Enter the password of your sign up.'
      passwordInput.value?.focus()
    } else {
      failure.value = status === 422 || status === 400 ? 'invalid' : status === 429 ? 'limit' : 'other'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main id="main" class="mx-auto w-full max-w-sm px-4 pt-8 pb-12">
    <Card>
      <CardHeader>
        <CardTitle as="h1" class="text-2xl">Confirm your email address</CardTitle>
        <CardDescription v-if="!failure">
          Enter the password of your sign up to finish your account and to sign in.
        </CardDescription>
      </CardHeader>
      <CardContent v-if="!failure">
        <form class="grid gap-4" novalidate @submit.prevent="confirm">
          <FormItem id="confirm-email-password" :error="passwordError">
            <FormLabel>Password</FormLabel>
            <FormControl>
              <PasswordInput
                ref="passwordInput"
                v-model="password"
                name="password"
                autocomplete="current-password"
                required
              />
            </FormControl>
            <FormMessage />
            <p class="text-sm">
              <RouterLink
                to="/forgot-password"
                class="text-primary font-medium underline-offset-4 hover:underline"
              >
                Forgot your password?
              </RouterLink>
            </p>
          </FormItem>
          <Button type="submit" class="w-full" :disabled="loading">
            {{ loading ? 'Confirming…' : 'Confirm email address' }}
          </Button>
        </form>
      </CardContent>
      <CardContent v-if="failure" class="grid gap-4">
        <Alert variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertTitle>{{ failureText[failure].title }}</AlertTitle>
          <AlertDescription>{{ failureText[failure].description }}</AlertDescription>
        </Alert>
        <p class="text-muted-foreground text-center text-sm">
          <RouterLink to="/sign-in" class="text-primary font-medium underline-offset-4 hover:underline">
            Go to sign in
          </RouterLink>
        </p>
      </CardContent>
    </Card>
  </main>
</template>
