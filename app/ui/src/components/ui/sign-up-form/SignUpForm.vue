<script setup lang="ts">
import { CircleAlertIcon } from 'lucide-vue-next'
import { onMounted, ref, watch } from 'vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { FormControl, FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { PasswordInput } from '@/components/ui/password-input'
import { checkEmail, checkName, checkPassword } from '@/lib/validation'

export interface SignUpError {
  title: string
  description: string
  // signIn shows a link to the sign in page in the alert.
  signIn?: boolean
}

const props = defineProps<{
  loading: boolean
  error: SignUpError | null
  errors: Record<string, string>
  signInPath: string
}>()

const emit = defineEmits<{ submit: [value: { name: string; email: string; password: string }] }>()

type Field = 'name' | 'email' | 'password'
const order: Field[] = ['name', 'email', 'password']

const values = ref<Record<Field, string>>({ name: '', email: '', password: '' })
const messages = ref<Record<string, string>>({})

const rules: Record<Field, () => string> = {
  name: () => checkName(values.value.name, 'Enter your name.'),
  email: () => checkEmail(values.value.email),
  password: () => checkPassword(values.value.password),
}

// The API checks the same rules. Its 422 errors show next to the fields.
watch(
  () => props.errors,
  (errs) => (messages.value = { ...messages.value, ...errs }),
)

function check(field: Field) {
  messages.value = { ...messages.value, [field]: rules[field]() }
}

const fieldIds: Record<Field, string> = {
  name: 'sign-up-name',
  email: 'sign-up-email',
  password: 'sign-up-password',
}

onMounted(() => document.getElementById(fieldIds.name)?.focus())

function submit() {
  order.forEach(check)
  const first = order.find((f) => messages.value[f])
  if (first) {
    document.getElementById(fieldIds[first])?.focus()
    return
  }
  emit('submit', { name: values.value.name, email: values.value.email, password: values.value.password })
}
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle as="h1" class="text-2xl">Create an account</CardTitle>
      <CardDescription>With an account you can add restaurants and write reviews.</CardDescription>
    </CardHeader>
    <CardContent>
      <form class="grid gap-4" novalidate @submit.prevent="submit">
        <Alert v-if="error" variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertTitle>{{ error.title }}</AlertTitle>
          <AlertDescription>
            {{ error.description }}
            <RouterLink v-if="error.signIn" :to="signInPath" class="font-medium underline underline-offset-4">
              Sign in
            </RouterLink>
          </AlertDescription>
        </Alert>
        <FormItem :id="fieldIds.name" :error="messages.name" description>
          <FormLabel>Name</FormLabel>
          <FormControl>
            <Input v-model="values.name" name="name" autocomplete="nickname" maxlength="255" required @blur="check('name')" />
          </FormControl>
          <FormDescription>Other people see this name on your restaurants and reviews.</FormDescription>
          <FormMessage />
        </FormItem>
        <FormItem :id="fieldIds.email" :error="messages.email">
          <FormLabel>Email</FormLabel>
          <FormControl>
            <Input
              v-model="values.email"
              type="email"
              name="email"
              autocomplete="email"
              maxlength="255"
              required
              @blur="check('email')"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
        <FormItem :id="fieldIds.password" :error="messages.password" description>
          <FormLabel>Password</FormLabel>
          <FormControl>
            <PasswordInput
              v-model="values.password"
              name="password"
              autocomplete="new-password"
              required
              @blur="check('password')"
            />
          </FormControl>
          <FormDescription>8 to 71 characters. Show the password to check it.</FormDescription>
          <FormMessage />
        </FormItem>
        <Button type="submit" class="w-full" :disabled="loading">
          {{ loading ? 'Creating account…' : 'Create account' }}
        </Button>
      </form>
    </CardContent>
    <CardFooter class="text-muted-foreground justify-center text-sm">
      <p>
        Already have an account?
        <RouterLink :to="signInPath" class="text-primary font-medium underline-offset-4 hover:underline">
          Sign in
        </RouterLink>
      </p>
    </CardFooter>
  </Card>
</template>
