import { toast } from 'vue-sonner'
import { pendingApproval } from './session'

// tellPendingApproval returns true for a user who waits for approval. It
// then shows the reason. The API gives 403 to each write of such a user.
export function tellPendingApproval(): boolean {
  if (!pendingApproval.value) {
    return false
  }
  toast.info('Your account waits for approval', {
    description: 'An administrator must approve it before you can add or change data.',
  })
  return true
}
