import type { Restaurant } from '@reviews/sdk'
import { restaurantsApi } from '@/lib/api'

export interface RestaurantAccess {
  canChange: boolean
  // hasOtherReviews is true if another user has a review. These reviews
  // block the rename and the delete.
  hasOtherReviews: boolean
}

// restaurantAccess follows the rules of decision 00016. A restaurant without
// a creator belongs to each user who has a review of it. A hasOwnReview of
// null means that the lookup failed. The API then decides.
export function restaurantAccess(
  restaurant: Pick<Restaurant, 'creator' | 'reviewCount'>,
  userId: string | undefined,
  hasOwnReview: boolean | null,
): RestaurantAccess {
  const canChange = !!userId && (restaurant.creator ? restaurant.creator.id === userId : hasOwnReview === true)
  return { canChange, hasOtherReviews: restaurant.reviewCount - (hasOwnReview === false ? 0 : 1) > 0 }
}

export async function findOwnReviewId(restaurantId: string, userId: string): Promise<string | null> {
  const page = await restaurantsApi.listReviews({ restaurantId, creatorEq: userId, limit: 1 })
  return page.items[0]?.id ?? null
}
