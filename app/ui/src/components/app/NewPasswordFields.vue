<script setup lang="ts">
import { ref, watch } from 'vue'
import { FormControl, FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { PasswordInput } from '@/components/ui/password-input'
import { checkPassword } from '@/lib/validation'

const props = defineProps<{
  // idPrefix makes the field IDs unique on the page.
  idPrefix: string
  // apiError is the 422 message of the API for the password.
  apiError?: string
}>()

const password = defineModel<string>({ required: true })
const confirm = ref('')
const messages = ref({ password: '', confirm: '' })

watch(
  () => props.apiError,
  (message) => (messages.value.password = message ?? ''),
)

function checkConfirm() {
  messages.value.confirm = confirm.value === password.value ? '' : 'The passwords do not match.'
}

// A message that goes away on blur moves the submit button. The click of
// the person then misses the button. The message therefore goes away while
// the person types.
watch([password, confirm], () => {
  if (messages.value.password) {
    messages.value.password = checkPassword(password.value)
  }
  if (messages.value.confirm) {
    checkConfirm()
  }
})

// validate shows the errors and moves the focus to the first one. It
// returns true without errors.
function validate(): boolean {
  messages.value.password = checkPassword(password.value)
  checkConfirm()
  const first = messages.value.password ? 'password' : messages.value.confirm ? 'confirm' : null
  if (first) {
    document.getElementById(`${props.idPrefix}-${first}`)?.focus()
  }
  return first === null
}

function clear() {
  password.value = ''
  confirm.value = ''
  messages.value = { password: '', confirm: '' }
}

defineExpose({ validate, clear })
</script>

<template>
  <FormItem :id="`${idPrefix}-password`" :error="messages.password" description>
    <FormLabel>New password</FormLabel>
    <FormControl>
      <PasswordInput
        v-model="password"
        name="new-password"
        autocomplete="new-password"
        required
        @blur="messages.password = checkPassword(password)"
      />
    </FormControl>
    <FormDescription>8 to 71 characters.</FormDescription>
    <FormMessage />
  </FormItem>
  <FormItem :id="`${idPrefix}-confirm`" :error="messages.confirm">
    <FormLabel>Confirm new password</FormLabel>
    <FormControl>
      <PasswordInput v-model="confirm" name="confirm-password" autocomplete="new-password" required @blur="checkConfirm" />
    </FormControl>
    <FormMessage />
  </FormItem>
</template>
