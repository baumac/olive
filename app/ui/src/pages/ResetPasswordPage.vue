<script setup lang="ts">
import { CircleAlertIcon } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import NewPasswordFields from '@/components/app/NewPasswordFields.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, fieldErrors, usersApi } from '@/lib/api'
import { clearAllDrafts } from '@/lib/drafts'
import { takeEmailToken } from '@/lib/emailToken'
import { clearSession } from '@/lib/session'

usePageTitle('Set a new password')

const route = useRoute()
const router = useRouter()
const token = takeEmailToken(route, router)

const invalidLink = {
  title: 'This link does not work',
  description: 'The link is not complete, is too old, or was used before. Ask for a new link.',
}

const password = ref('')
const passwordError = ref('')
const loading = ref(false)
const linkFailed = ref(!token)
const error = ref<{ title: string; description: string } | null>(token ? null : invalidLink)
const fields = ref<InstanceType<typeof NewPasswordFields> | null>(null)

async function submit() {
  if (!fields.value?.validate()) {
    return
  }
  loading.value = true
  error.value = null
  passwordError.value = ''
  try {
    await usersApi.resetPassword({ passwordReset: { token, password: password.value } })
    // The reset made the session of this browser invalid.
    clearSession()
    clearAllDrafts()
    void router.replace('/sign-in')
    toast.success('Your password is changed', { description: 'Sign in with the new password.' })
  } catch (e) {
    const { status, problem } = await apiFailure(e)
    const api = fieldErrors(problem)
    if (status === 422 && api.password) {
      passwordError.value = 'Use 8 to 71 characters.'
    } else if (status === 422) {
      linkFailed.value = true
      error.value = invalidLink
    } else if (status === 429) {
      error.value = { title: 'Too many tries', description: 'Wait a minute and try again.' }
    } else {
      error.value = { title: 'Your password was not changed', description: 'Try again in a moment.' }
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
        <CardTitle as="h1" class="text-2xl">Set a new password</CardTitle>
        <CardDescription v-if="!linkFailed">You are signed out on all devices after the change.</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="grid gap-4" novalidate @submit.prevent="submit">
          <Alert v-if="error" variant="destructive">
            <CircleAlertIcon aria-hidden="true" />
            <AlertTitle>{{ error.title }}</AlertTitle>
            <AlertDescription>{{ error.description }}</AlertDescription>
          </Alert>
          <p v-if="linkFailed" class="text-muted-foreground text-center text-sm">
            <RouterLink to="/forgot-password" class="text-primary font-medium underline-offset-4 hover:underline">
              Ask for a new link
            </RouterLink>
          </p>
          <template v-else>
            <NewPasswordFields ref="fields" v-model="password" id-prefix="reset-password" :api-error="passwordError" />
            <Button type="submit" class="w-full" :disabled="loading">
              {{ loading ? 'Saving…' : 'Set the new password' }}
            </Button>
          </template>
        </form>
      </CardContent>
    </Card>
  </main>
</template>
