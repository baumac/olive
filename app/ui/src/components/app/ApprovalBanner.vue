<script setup lang="ts">
import { ClockIcon } from 'lucide-vue-next'
import { ref } from 'vue'
import { toast } from 'vue-sonner'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { refreshSession } from '@/composables/useAuth'
import { pendingApproval } from '@/lib/session'

const checking = ref(false)

// check gets the user again. An approval changes the stored user.
async function check() {
  checking.value = true
  const result = await refreshSession()
  checking.value = false
  if (result === 'failed') {
    toast.error('The check did not work', { description: 'Check your connection and try again.' })
    return
  }
  // The session ended handler tells a person whose session ended.
  if (result !== 'refreshed') {
    return
  }
  if (pendingApproval.value) {
    toast.info('Your account is not approved yet', { description: 'Try again later.' })
  } else {
    toast.success('Your account is approved', { description: 'You can add restaurants, reviews, and lists now.' })
  }
}
</script>

<template>
  <div v-if="pendingApproval" class="mx-auto w-full max-w-6xl px-4 pt-4 sm:px-6">
    <Alert role="status">
      <ClockIcon aria-hidden="true" />
      <AlertTitle>Your account waits for approval</AlertTitle>
      <AlertDescription class="flex flex-wrap items-center gap-3">
        <span>
          An administrator must approve your account before you can add restaurants, write reviews, or save lists. You
          can read all restaurants and reviews now.
        </span>
        <Button type="button" variant="outline" size="sm" :disabled="checking" @click="check">Check again</Button>
      </AlertDescription>
    </Alert>
  </div>
</template>
