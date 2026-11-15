<script setup lang="ts">
import { CircleAlertIcon } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, usersApi } from '@/lib/api'
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

const failureText: Record<Failure, { title: string; description: string }> = {
  invalid: {
    title: 'This link does not work',
    description: 'The link is not complete, is too old, or was used before. Sign in to get a new link.',
  },
  limit: { title: 'Too many tries', description: 'Wait a minute, then open the link again.' },
  other: { title: 'Your email address was not confirmed', description: 'Open the link again in a moment.' },
}

// The person starts the confirmation with the button. A mail scanner that
// only opens the link then does not use the token.
async function confirm() {
  loading.value = true
  try {
    const auth = await usersApi.confirmEmail({ emailConfirmation: { token }, cookie: true })
    if (currentUser.value && currentUser.value.id !== auth.user.id) {
      clearAllDrafts()
    }
    saveSession(auth)
    void router.replace(takeRedirect())
    const next = auth.user.approved ? '' : ' An administrator must approve your account before you can add data.'
    toast.success('Welcome to Sponga47 Reviews', { description: `You are signed in as ${auth.user.name}.${next}` })
  } catch (e) {
    const { status } = await apiFailure(e)
    failure.value = status === 422 || status === 400 ? 'invalid' : status === 429 ? 'limit' : 'other'
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
        <CardDescription v-if="!failure">Select the button to finish your account and to sign in.</CardDescription>
      </CardHeader>
      <CardContent v-if="!failure">
        <Button type="button" class="w-full" :disabled="loading" @click="confirm">
          {{ loading ? 'Confirming…' : 'Confirm email address' }}
        </Button>
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
