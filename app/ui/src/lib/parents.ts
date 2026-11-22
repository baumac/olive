import type { Parent } from './reviewKinds'
import { formatCount } from './format'

export interface ParentAccess {
  // canEdit is true for each signed in user. Each approved user can change
  // each restaurant and recipe. See decision 00004.
  canEdit: boolean
  // isCreator is true if the user can delete the parent.
  isCreator: boolean
  // hasOtherReviews is true if another user has a review. These reviews
  // block the delete.
  hasOtherReviews: boolean
}

// parentAccess follows the rules of decision 00004 for a
// restaurant or a recipe. A parent without a creator belongs to each user who
// has a review of it. ownReviewCount is the count of the user of the token.
// See decision 00005.
export function parentAccess(parent: Pick<Parent, 'creator' | 'reviewCount' | 'ownReviewCount'>, userId: string | undefined): ParentAccess {
  const own = userId ? parent.ownReviewCount : 0
  const isCreator = !!userId && (parent.creator ? parent.creator.id === userId : own > 0)
  return { canEdit: !!userId, isCreator, hasOtherReviews: parent.reviewCount > own }
}

function same(a: unknown, b: unknown): boolean {
  if (a instanceof Set && b instanceof Set) {
    return a.size === b.size && [...a].every((v) => b.has(v))
  }
  if (Array.isArray(a) && Array.isArray(b)) {
    return a.length === b.length && a.every((v, i) => same(v, b[i]))
  }
  return a === b
}

// changedPatch keeps only the properties that differ from the loaded record.
// A form that is older than a change of another user then does not undo it.
export function changedPatch<T extends object>(next: T, loaded: T): Partial<T> {
  return Object.fromEntries(Object.entries(next).filter(([k, v]) => !same(v, loaded[k as keyof T]))) as Partial<T>
}

// ownReviewsText tells that a delete of the parent also deletes the reviews of
// the user.
export function ownReviewsText(count: number): string {
  return count === 1
    ? 'Your review and its photos will be deleted too.'
    : `Your ${formatCount(count, 'review')} and their photos will be deleted too.`
}
