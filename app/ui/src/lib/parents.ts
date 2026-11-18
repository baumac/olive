import type { Parent, ReviewKind } from './reviewKinds'
import { kinds } from './reviewKinds'

export interface ParentAccess {
  canChange: boolean
  // hasOtherReviews is true if another user has a review. These reviews
  // block the rename and the delete.
  hasOtherReviews: boolean
}

// parentAccess follows the rules of decision 00016 for a restaurant or a
// recipe. A parent without a creator belongs to each user who has a review of
// it. A hasOwnReview of null means that the lookup failed. The API then
// decides.
export function parentAccess(
  parent: Pick<Parent, 'creator' | 'reviewCount'>,
  userId: string | undefined,
  hasOwnReview: boolean | null,
): ParentAccess {
  const canChange = !!userId && (parent.creator ? parent.creator.id === userId : hasOwnReview === true)
  return { canChange, hasOtherReviews: parent.reviewCount - (hasOwnReview === false ? 0 : 1) > 0 }
}

export async function findOwnReviewId(kind: ReviewKind, parentId: string, userId: string): Promise<string | null> {
  const page = await kinds[kind].listReviews(parentId, { creatorEq: userId, limit: 1 })
  return page.items[0]?.id ?? null
}
