<script setup lang="ts">
import { CircleAlertIcon, InfoIcon } from 'lucide-vue-next'
import { nextTick, onMounted, ref } from 'vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { FormControl, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { PasswordInput } from '@/components/ui/password-input'
import { checkEmail } from '@/lib/validation'

export interface SignInError {
  title: string
  description: string
}

const props = defineProps<{
  loading: boolean
  error: SignInError | null
  fieldErrors?: Record<string, string>
  notice: 'session-ended' | null
  signUpPath: string
  initialEmail?: string
}>()

const emit = defineEmits<{ submit: [value: { email: string; password: string }] }>()

const email = ref(props.initialEmail ?? '')
const password = ref('')
const errors = ref<Record<string, string>>({})
const emailInput = ref<InstanceType<typeof Input> | null>(null)
const passwordInput = ref<InstanceType<typeof PasswordInput> | null>(null)

function focusEmail() {
  ;(emailInput.value?.$el as HTMLInputElement | undefined)?.focus()
}

onMounted(() => (email.value ? passwordInput.value?.focus() : focusEmail()))

function submit() {
  errors.value = {
    email: checkEmail(email.value),
    password: password.value === '' ? 'Enter your password.' : '',
  }
  if (errors.value.email) {
    focusEmail()
    return
  }
  if (errors.value.password) {
    passwordInput.value?.focus()
    return
  }
  emit('submit', { email: email.value, password: password.value })
}

// clearPassword runs after a wrong email or password.
async function clearPassword() {
  password.value = ''
  await nextTick()
  passwordInput.value?.focus()
}

defineExpose({ clearPassword })
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle as="h1" class="text-2xl">Sign in</CardTitle>
      <CardDescription>Sign in to add restaurants and write reviews.</CardDescription>
    </CardHeader>
    <CardContent>
      <form class="grid gap-4" novalidate @submit.prevent="submit">
        <Alert v-if="notice === 'session-ended' && !error">
          <InfoIcon aria-hidden="true" />
          <AlertTitle>Your session has ended</AlertTitle>
          <AlertDescription>Sign in again to continue.</AlertDescription>
        </Alert>
        <Alert v-if="error" variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertTitle>{{ error.title }}</AlertTitle>
          <AlertDescription>{{ error.description }}</AlertDescription>
        </Alert>
        <FormItem id="sign-in-email" :error="errors.email || fieldErrors?.email">
          <FormLabel>Email</FormLabel>
          <FormControl>
            <Input ref="emailInput" v-model="email" type="email" name="email" autocomplete="username" required />
          </FormControl>
          <FormMessage />
        </FormItem>
        <FormItem id="sign-in-password" :error="errors.password || fieldErrors?.password">
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
        </FormItem>
        <Button type="submit" class="w-full" :disabled="loading">
          {{ loading ? 'Signing in…' : 'Sign in' }}
        </Button>
      </form>
    </CardContent>
    <CardFooter class="text-muted-foreground justify-center text-sm">
      <p>
        New to Sponga47 Reviews?
        <RouterLink :to="signUpPath" class="text-primary font-medium underline-offset-4 hover:underline">
          Create an account
        </RouterLink>
      </p>
    </CardFooter>
  </Card>
</template>
