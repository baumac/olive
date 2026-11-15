<script setup lang="ts">
import { CircleAlertIcon, MailIcon } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { FormControl, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, fieldErrors, usersApi } from '@/lib/api'
import { checkEmail } from '@/lib/validation'

usePageTitle('Forgot your password')

const email = ref('')
const emailError = ref('')
const loading = ref(false)
const error = ref<{ title: string; description: string } | null>(null)
const sent = ref(false)

const fieldId = 'forgot-password-email'
onMounted(() => document.getElementById(fieldId)?.focus())

async function submit() {
  emailError.value = checkEmail(email.value)
  if (emailError.value) {
    document.getElementById(fieldId)?.focus()
    return
  }
  loading.value = true
  error.value = null
  try {
    await usersApi.requestPasswordReset({ passwordResetRequest: { email: email.value } })
    sent.value = true
  } catch (e) {
    const { status, problem } = await apiFailure(e)
    if (status === 422) {
      emailError.value = fieldErrors(problem).email ?? 'Enter an email address, for example ana@example.com.'
    } else if (status === 429) {
      error.value = { title: 'Too many tries', description: 'Wait a minute and try again.' }
    } else {
      error.value = { title: 'The email was not sent', description: 'Try again in a moment.' }
    }
  } finally {
    loading.value = false
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
          If {{ email }} has an account, we sent an email to it. Open the link in it to set a new password. The link
          works for 30 minutes.
        </CardDescription>
      </CardHeader>
      <CardContent class="text-muted-foreground text-center text-sm">
        <RouterLink to="/sign-in" class="text-primary font-medium underline-offset-4 hover:underline">
          Go to sign in
        </RouterLink>
      </CardContent>
    </Card>
    <Card v-else>
      <CardHeader>
        <CardTitle as="h1" class="text-2xl">Forgot your password?</CardTitle>
        <CardDescription>Enter your email address. We send you a link to set a new password.</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="grid gap-4" novalidate @submit.prevent="submit">
          <Alert v-if="error" variant="destructive">
            <CircleAlertIcon aria-hidden="true" />
            <AlertTitle>{{ error.title }}</AlertTitle>
            <AlertDescription>{{ error.description }}</AlertDescription>
          </Alert>
          <FormItem :id="fieldId" :error="emailError">
            <FormLabel>Email</FormLabel>
            <FormControl>
              <Input v-model="email" type="email" name="email" autocomplete="username" maxlength="255" required />
            </FormControl>
            <FormMessage />
          </FormItem>
          <Button type="submit" class="w-full" :disabled="loading">
            {{ loading ? 'Sending…' : 'Send the link' }}
          </Button>
        </form>
      </CardContent>
      <CardFooter class="text-muted-foreground justify-center text-sm">
        <RouterLink to="/sign-in" class="text-primary font-medium underline-offset-4 hover:underline">
          Back to sign in
        </RouterLink>
      </CardFooter>
    </Card>
  </main>
</template>
