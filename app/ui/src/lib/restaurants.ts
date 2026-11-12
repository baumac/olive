import type { Restaurant } from '@reviews/sdk'
import { restaurantsApi } from '@/lib/api'

export interface RestaurantAccess {
  canChange: boolean
  // otherReviews counts the reviews of other users. They block the rename
  // and the delete.
  otherReviews: number
}

// restaurantAccess follows the rules of decision 00016. A restaurant without
// a creator belongs to each user who has a review of it.
export function restaurantAccess(
  restaurant: Pick<Restaurant, 'creator' | 'reviewCount'>,
  userId: string | undefined,
  hasOwnReview: boolean,
): RestaurantAccess {
  const canChange = !!userId && (restaurant.creator ? restaurant.creator.id === userId : hasOwnReview)
  return { canChange, otherReviews: restaurant.reviewCount - (hasOwnReview ? 1 : 0) }
}

export async function findOwnReviewId(restaurantId: string, userId: string): Promise<string | null> {
  const page = await restaurantsApi.listReviews({ restaurantId, creatorEq: userId, limit: 1 })
  return page.items[0]?.id ?? null
}
