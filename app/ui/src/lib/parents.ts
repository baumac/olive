import type { Parent } from './reviewKinds'
import { formatCount } from './format'

export interface ParentAccess {
  canChange: boolean
  // hasOtherReviews is true if another user has a review. These reviews
  // block the rename and the delete.
  hasOtherReviews: boolean
}

// parentAccess follows the rules of decision 00016 for a restaurant or a
// recipe. A parent without a creator belongs to each user who has a review of
// it. ownReviewCount is the count of the user of the token. See decision
// 00050.
export function parentAccess(parent: Pick<Parent, 'creator' | 'reviewCount' | 'ownReviewCount'>, userId: string | undefined): ParentAccess {
  const own = userId ? parent.ownReviewCount : 0
  const canChange = !!userId && (parent.creator ? parent.creator.id === userId : own > 0)
  return { canChange, hasOtherReviews: parent.reviewCount > own }
}

// ownReviewsText tells that a delete of the parent also deletes the reviews of
// the user.
export function ownReviewsText(count: number): string {
  return count === 1
    ? 'Your review and its photos will be deleted too.'
    : `Your ${formatCount(count, 'review')} and their photos will be deleted too.`
}
