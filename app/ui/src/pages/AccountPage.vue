<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import NewPasswordFields from '@/components/app/NewPasswordFields.vue'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { FormControl, FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { PasswordInput } from '@/components/ui/password-input'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, endSession, fieldErrors, retryText, usersApi } from '@/lib/api'
import { clearAllDrafts } from '@/lib/drafts'
import { clearSession, currentUser, saveSession, saveUser } from '@/lib/session'
import { checkName } from '@/lib/validation'

usePageTitle('Account')

const router = useRouter()

const wrongPassword = 'The password is not correct.'
const manyPasswords = 'Too many wrong passwords. Try again later.'

// refuse handles the failures that all three forms share. It returns true
// when it told the person the reason.
function refuse(status: number | null, title: string): boolean {
  if (status === 401) {
    endSession()
    return true
  }
  if (status === 422) {
    return false
  }
  toast.error(title, { description: retryText(status) })
  return true
}

const name = ref(currentUser.value?.name ?? '')
const nameError = ref('')
const savingName = ref(false)
const nameChanged = computed(() => name.value !== currentUser.value?.name)

// Another tab or a session refresh can change the stored name. A field that
// the person did not change follows it.
watch(
  () => currentUser.value?.name,
  (now, before) => {
    if (name.value === before) {
      name.value = now ?? ''
    }
  },
)

async function saveName() {
  nameError.value = checkName(name.value, 'Enter your name.')
  if (nameError.value) {
    document.getElementById('account-name')?.focus()
    return
  }
  savingName.value = true
  try {
    saveUser(await usersApi.updateUser({ userPatch: { name: name.value } }))
    toast.success('Your name is changed')
  } catch (e) {
    const { status, problem } = await apiFailure(e)
    if (!refuse(status, 'Your name was not changed')) {
      nameError.value = fieldErrors(problem).name ?? 'Enter your name.'
    }
  } finally {
    savingName.value = false
  }
}

const currentPassword = ref('')
const currentPasswordError = ref('')
const newPassword = ref('')
const newPasswordError = ref('')
const savingPassword = ref(false)
const passwordFields = ref<InstanceType<typeof NewPasswordFields> | null>(null)

async function savePassword() {
  currentPasswordError.value = currentPassword.value === '' ? 'Enter your current password.' : ''
  newPasswordError.value = ''
  if (currentPasswordError.value) {
    document.getElementById('account-current-password')?.focus()
    return
  }
  if (!passwordFields.value?.validate()) {
    return
  }
  savingPassword.value = true
  try {
    const passwordChange = { currentPassword: currentPassword.value, password: newPassword.value }
    saveSession(await usersApi.changePassword({ passwordChange, cookie: true }))
    currentPassword.value = ''
    passwordFields.value.clear()
    toast.success('Your password is changed', { description: 'Your other devices are signed out.' })
  } catch (e) {
    const { status, problem } = await apiFailure(e)
    if (status === 429 && problem?.code === 'wrong_password_limit') {
      toast.error('Your password was not changed', { description: manyPasswords })
    } else if (!refuse(status, 'Your password was not changed')) {
      const api = fieldErrors(problem)
      currentPasswordError.value = api.current_password ? wrongPassword : ''
      newPasswordError.value = api.password ? 'Use 8 to 71 characters.' : ''
    }
  } finally {
    savingPassword.value = false
  }
}

const deleteOpen = ref(false)
const deletePassword = ref('')
const deleteError = ref('')
const deleting = ref(false)

async function deleteAccount() {
  deleteError.value = deletePassword.value === '' ? 'Enter your password.' : ''
  if (deleteError.value) {
    document.getElementById('account-delete-password')?.focus()
    return
  }
  deleting.value = true
  try {
    await usersApi.deleteUser({ userDelete: { password: deletePassword.value } })
    deleteOpen.value = false
    clearSession()
    clearAllDrafts()
    void router.push('/')
    toast('Your account is deleted')
  } catch (e) {
    const { status, problem } = await apiFailure(e)
    if (status === 429 && problem?.code === 'wrong_password_limit') {
      deleteError.value = manyPasswords
    } else if (status === 401) {
      deleteOpen.value = false
      endSession()
    } else if (status === 422) {
      deleteError.value = wrongPassword
    } else {
      deleteError.value = `Your account was not deleted. ${retryText(status)}`
    }
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <main id="main" class="mx-auto grid w-full max-w-xl gap-6 px-4 pt-8 pb-12">
    <div>
      <h1 class="font-serif text-[34px] leading-10 font-semibold">Account</h1>
      <p class="text-muted-foreground text-sm">{{ currentUser?.email }}</p>
    </div>

    <Card>
      <CardHeader>
        <CardTitle as="h2">Name</CardTitle>
      </CardHeader>
      <CardContent>
        <form class="grid gap-4" novalidate @submit.prevent="saveName">
          <FormItem id="account-name" :error="nameError" description>
            <FormLabel>Name</FormLabel>
            <FormControl>
              <Input v-model="name" name="name" autocomplete="nickname" maxlength="255" required />
            </FormControl>
            <FormDescription>Other people see this name on your restaurants and reviews.</FormDescription>
            <FormMessage />
          </FormItem>
          <Button type="submit" class="justify-self-start" :disabled="savingName || !nameChanged">Save name</Button>
        </form>
      </CardContent>
    </Card>

    <Card>
      <CardHeader>
        <CardTitle as="h2">Password</CardTitle>
        <CardDescription>A change signs you out on your other devices.</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="grid gap-4" novalidate @submit.prevent="savePassword">
          <!-- Password managers use the email field to find the account. -->
          <input type="email" name="email" autocomplete="username" :value="currentUser?.email" hidden readonly />
          <FormItem id="account-current-password" :error="currentPasswordError">
            <FormLabel>Current password</FormLabel>
            <FormControl>
              <PasswordInput v-model="currentPassword" name="current-password" autocomplete="current-password" required />
            </FormControl>
            <FormMessage />
          </FormItem>
          <NewPasswordFields ref="passwordFields" v-model="newPassword" id-prefix="account-new" :api-error="newPasswordError" />
          <Button type="submit" class="justify-self-start" :disabled="savingPassword">Change password</Button>
        </form>
      </CardContent>
    </Card>

    <Card>
      <CardHeader>
        <CardTitle as="h2">Delete account</CardTitle>
        <CardDescription>
          This deletes your account, your reviews, your photos, and your lists. Your restaurants stay, without your
          name. You cannot undo it.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <AlertDialog v-model:open="deleteOpen">
          <AlertDialogTrigger as-child>
            <Button variant="destructive">Delete account</Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <form class="grid gap-4" novalidate @submit.prevent="deleteAccount">
              <AlertDialogHeader>
                <AlertDialogTitle>Delete your account?</AlertDialogTitle>
                <AlertDialogDescription>
                  Your reviews, photos, and lists are deleted with it. You cannot undo it. Enter your password to
                  confirm.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <FormItem id="account-delete-password" :error="deleteError">
                <FormLabel>Password</FormLabel>
                <FormControl>
                  <PasswordInput v-model="deletePassword" name="password" autocomplete="current-password" required />
                </FormControl>
                <FormMessage />
              </FormItem>
              <AlertDialogFooter>
                <AlertDialogCancel type="button">Cancel</AlertDialogCancel>
                <Button type="submit" variant="destructive" :disabled="deleting">Delete account</Button>
              </AlertDialogFooter>
            </form>
          </AlertDialogContent>
        </AlertDialog>
      </CardContent>
    </Card>
  </main>
</template>
