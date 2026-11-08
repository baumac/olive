import type { AuthToken, Restaurant, RestaurantCreate, Review, ReviewCreate } from '@reviews/sdk'
import { Configuration, RestaurantsApi, UsersApi } from '@reviews/sdk'
import { randomUUID } from 'node:crypto'

// WithTagArray changes the tags of an SDK body from a Set to an array.
type WithTagArray<T> = T extends unknown ? Omit<T, 'tags'> & { tags?: string[] } : never

export interface TestUser {
  name: string
  email: string
  password: string
  auth: AuthToken
}

// TestApi makes test data through the Public API. Each test makes its own
// users. Tests then can run in parallel.
export class TestApi {
  constructor(private readonly baseURL: string) {}

  private config(token?: string) {
    return new Configuration({ basePath: `${this.baseURL}/v1`, accessToken: token })
  }

  async signUp(name = 'Ana'): Promise<TestUser> {
    const email = `${name.toLowerCase()}-${randomUUID().slice(0, 8)}@example.test`
    const password = 'correct-horse-battery'
    const auth = await new UsersApi(this.config()).signUp({ userCreate: { name, email, password } })
    return { name, email, password, auth }
  }

  createRestaurant(user: TestUser, restaurant: WithTagArray<RestaurantCreate>): Promise<Restaurant> {
    const { tags, ...rest } = restaurant
    return new RestaurantsApi(this.config(user.auth.token)).createRestaurant({
      restaurantCreate: { ...rest, tags: tags ? new Set(tags) : undefined },
    })
  }

  // createReview takes tags as an array, like createRestaurant.
  createReview(user: TestUser, restaurantId: string, review: WithTagArray<ReviewCreate>): Promise<Review> {
    const { tags, ...rest } = review
    const reviewCreate = { ...rest, tags: tags ? new Set(tags) : undefined } as ReviewCreate
    return new RestaurantsApi(this.config(user.auth.token)).createReview({ restaurantId, reviewCreate })
  }
}

// uniqueTag gives a tag that only this test uses. A tag filter then shows
// only the data of the test.
export function uniqueTag(prefix: string) {
  return `${prefix}-${randomUUID().slice(0, 8)}`
}
