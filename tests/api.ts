import type { AuthToken, Photo, Restaurant, RestaurantCreate, Review, ReviewCreate } from '@reviews/sdk'
import { Configuration, RestaurantsApi, UsersApi } from '@reviews/sdk'
import { randomUUID } from 'node:crypto'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

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

  createReview(user: TestUser, restaurantId: string, reviewCreate: ReviewCreate): Promise<Review> {
    return new RestaurantsApi(this.config(user.auth.token)).createReview({ restaurantId, reviewCreate })
  }

  deleteReview(user: TestUser, review: Review): Promise<void> {
    return new RestaurantsApi(this.config(user.auth.token)).deleteReview({ restaurantId: review.restaurant, reviewId: review.id })
  }

  // addPhoto sends a file of tests/photos with its own content type.
  async addPhoto(user: TestUser, review: Review, name: string, alt?: string): Promise<Photo> {
    const type = name.endsWith('.png') ? 'image/png' : 'image/jpeg'
    const api = new RestaurantsApi(this.config(user.auth.token))
    const ids = { restaurantId: review.restaurant, reviewId: review.id }
    const photo = await api.addReviewPhoto({ ...ids, body: new Blob([new Uint8Array(photoFile(name))], { type }) }, async ({ init }) => ({
      headers: { ...(init.headers as Record<string, string>), 'Content-Type': type },
    }))
    return alt ? api.updateReviewPhoto({ ...ids, photoId: photo.id, photoPatch: { alt } }) : photo
  }
}

// photoFile reads a test image from tests/photos.
export function photoFile(name: string): Buffer {
  return readFileSync(fileURLToPath(new URL(`./photos/${name}`, import.meta.url)))
}

// uniqueTag gives a tag that only this test uses. A tag filter then shows
// only the data of the test.
export function uniqueTag(prefix: string) {
  return `${prefix}-${randomUUID().slice(0, 8)}`
}
