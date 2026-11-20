import type {
  AuthToken,
  Location,
  Photo,
  Recipe,
  RecipeCreate,
  RecipeReview,
  RecipeReviewCreate,
  Restaurant,
  RestaurantCreate,
  RestaurantReview,
  RestaurantReviewCreate,
  SavedList,
} from '@reviews/sdk'
import { Configuration, ListsApi, RecipesApi, ResponseError, RestaurantsApi, UsersApi } from '@reviews/sdk'
import { randomUUID } from 'node:crypto'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

// WithTagArray changes the tags of an SDK body from a Set to an array.
type WithTagArray<T> = T extends unknown ? Omit<T, 'tags'> & { tags?: string[] } : never

// TestRestaurant is a create body with an optional location.
type TestRestaurant = Omit<WithTagArray<RestaurantCreate>, 'location'> & { location?: Location }

// WithOptionalDate is a review body without a required date. The helpers
// then use today.
type WithOptionalDate<T> = Omit<T, 'date'> & { date?: string }

// today is the date in UTC. The API allows each date up to today in UTC+14.
function today(): string {
  return new Date().toISOString().slice(0, 10)
}

// randomLocation is a point in the South Pacific. Restaurants of different
// tests are then not near each other.
export function randomLocation(): Location {
  const part = () => Math.round(Math.random() * 1e6) / 1e6
  return { latitude: -40 - part() * 10, longitude: -150 + part() * 40, address: 'Test Street 1, Testville' }
}

export interface TestUser {
  name: string
  email: string
  password: string
  // token is the bearer token. The UI keeps it in the session cookie.
  token: string
  auth: AuthToken
}

// TestApi makes test data through the Public API. Each test makes its own
// users. Tests then can run in parallel.
// superuserTokenEnv has the superuser token of the app of global-setup.ts.
export const superuserTokenEnv = 'REVIEWS_E2E_SUPERUSER_TOKEN'

export class TestApi {
  constructor(
    private readonly baseURL: string,
    private readonly superuserToken = process.env[superuserTokenEnv],
  ) {}

  private config(token?: string) {
    return new Configuration({ basePath: `${this.baseURL}/v1`, accessToken: token })
  }

  // signUp makes an approved user. Use signUpPending for a user that waits
  // for approval.
  async signUp(name = 'Ana'): Promise<TestUser> {
    const user = await this.signUpPending(name)
    await this.approve(user.auth.user.id)
    user.auth.user.approved = true
    return user
  }

  // signUpPending confirms the email address as a superuser. It does not
  // read the confirmation email.
  async signUpPending(name = 'Ana'): Promise<TestUser> {
    const email = `${name.toLowerCase()}-${randomUUID().slice(0, 8)}@example.test`
    const password = 'correct-horse-battery'
    const users = new UsersApi(this.config())
    await users.signUp({ userCreate: { name, email, password } })
    await this.patchUser(await this.userId(email), { verified: true })
    const auth = await users.signIn({ credentials: { email, password } })
    if (!auth.token) {
      throw new Error('sign in gave no token')
    }
    return { name, email, password, token: auth.token, auth }
  }

  private superuserHeaders(): Record<string, string> {
    if (!this.superuserToken) {
      throw new Error('The test API has no superuser token')
    }
    return { 'Content-Type': 'application/json', Authorization: this.superuserToken }
  }

  private async patchUser(userId: string, body: Record<string, boolean>): Promise<void> {
    const res = await fetch(`${this.baseURL}/api/collections/users/records/${userId}`, {
      method: 'PATCH',
      headers: this.superuserHeaders(),
      body: JSON.stringify(body),
    })
    if (!res.ok) {
      throw new Error(`The user did not change: ${res.status} ${await res.text()}`)
    }
  }

  private async userId(email: string): Promise<string> {
    const filter = encodeURIComponent(`email="${email}"`)
    const res = await fetch(`${this.baseURL}/api/collections/users/records?filter=${filter}`, {
      headers: this.superuserHeaders(),
    })
    const { items } = (await res.json()) as { items: { id: string }[] }
    if (!res.ok || items.length !== 1) {
      throw new Error(`The user ${email} was not found: ${res.status}`)
    }
    return items[0]!.id
  }

  // approve changes the user as a superuser does it in the dashboard.
  approve(userId: string, approved = true): Promise<void> {
    return this.patchUser(userId, { approved })
  }

  // approveByEmail approves a user that signed up in the UI.
  async approveByEmail(email: string): Promise<void> {
    await this.approve(await this.userId(email))
  }

  // emailLink returns the path of the link with linkPath in the last email
  // to address. The test app writes each email to its log. PocketBase saves
  // the log after some seconds.
  async emailLink(address: string, linkPath: string): Promise<string> {
    const filter = encodeURIComponent(`data.to="${address}" && data.text~"${linkPath}#token="`)
    for (let i = 0; i < 40; i++) {
      const res = await fetch(`${this.baseURL}/api/logs?filter=${filter}&sort=-created&perPage=1`, {
        headers: this.superuserHeaders(),
      })
      if (!res.ok) {
        throw new Error(`The log was not read: ${res.status} ${await res.text()}`)
      }
      const { items } = (await res.json()) as { items: { data: { text: string } }[] }
      const link = items[0]?.data.text.split('\n').find((line) => line.includes(`${linkPath}#token=`))
      if (link) {
        const url = new URL(link)
        return url.pathname + url.hash
      }
      await new Promise((resolve) => setTimeout(resolve, 500))
    }
    throw new Error(`No email with a ${linkPath} link to ${address}`)
  }

  // tokenStatus gives the status of getCurrentUser. 200 means that the token
  // is valid.
  async tokenStatus(token: string): Promise<number> {
    try {
      return (await new UsersApi(this.config(token)).getCurrentUserRaw()).raw.status
    } catch (error) {
      if (error instanceof ResponseError) {
        return error.response.status
      }
      throw error
    }
  }

  // createRestaurant gives a restaurant without a location a random one.
  createRestaurant(user: TestUser, restaurant: TestRestaurant): Promise<Restaurant> {
    const { tags, location, ...rest } = restaurant
    return new RestaurantsApi(this.config(user.token)).createRestaurant({
      restaurantCreate: { ...rest, tags: tags ? new Set(tags) : undefined, location: location ?? randomLocation() },
    })
  }

  // createRecipe gives a recipe without ingredients or steps one of each.
  createRecipe(user: TestUser, recipe: Partial<WithTagArray<RecipeCreate>> & { name: string }): Promise<Recipe> {
    const { tags, ingredients, steps, ...rest } = recipe
    return new RecipesApi(this.config(user.token)).createRecipe({
      recipeCreate: { ...rest, tags: tags ? new Set(tags) : undefined, ingredients: ingredients ?? ['1 egg'], steps: steps ?? ['Cook it.'] },
    })
  }

  createRecipeReview(user: TestUser, recipeId: string, review: WithOptionalDate<RecipeReviewCreate>): Promise<RecipeReview> {
    return new RecipesApi(this.config(user.token)).createRecipeReview({ recipeId, recipeReviewCreate: { date: today(), ...review } })
  }

  addRecipeToList(user: TestUser, listRef: string, recipeId: string): Promise<void> {
    return new ListsApi(this.config(user.token)).addListRecipe({ listRef, recipeId })
  }

  // addRecipePhoto sends a file of tests/photos with its own content type.
  async addRecipePhoto(user: TestUser, review: RecipeReview, name: string): Promise<Photo> {
    const type = name.endsWith('.png') ? 'image/png' : 'image/jpeg'
    return new RecipesApi(this.config(user.token)).addRecipeReviewPhoto(
      { recipeId: review.recipe, reviewId: review.id, body: new Blob([new Uint8Array(photoFile(name))], { type }) },
      async ({ init }) => ({ headers: { ...(init.headers as Record<string, string>), 'Content-Type': type } }),
    )
  }

  createList(user: TestUser, name: string): Promise<SavedList> {
    return new ListsApi(this.config(user.token)).createList({ savedListCreate: { name } })
  }

  addToList(user: TestUser, listRef: string, restaurantId: string): Promise<void> {
    return new ListsApi(this.config(user.token)).addListRestaurant({ listRef, restaurantId })
  }

  createReview(user: TestUser, restaurantId: string, review: WithOptionalDate<RestaurantReviewCreate>): Promise<RestaurantReview> {
    return new RestaurantsApi(this.config(user.token)).createRestaurantReview({
      restaurantId,
      restaurantReviewCreate: { date: today(), ...review },
    })
  }

  deleteReview(user: TestUser, review: RestaurantReview): Promise<void> {
    return new RestaurantsApi(this.config(user.token)).deleteRestaurantReview({ restaurantId: review.restaurant, reviewId: review.id })
  }

  // addPhoto sends a file of tests/photos with its own content type.
  async addPhoto(user: TestUser, review: RestaurantReview, name: string, alt?: string): Promise<Photo> {
    const type = name.endsWith('.png') ? 'image/png' : 'image/jpeg'
    const api = new RestaurantsApi(this.config(user.token))
    const ids = { restaurantId: review.restaurant, reviewId: review.id }
    const photo = await api.addRestaurantReviewPhoto({ ...ids, body: new Blob([new Uint8Array(photoFile(name))], { type }) }, async ({ init }) => ({
      headers: { ...(init.headers as Record<string, string>), 'Content-Type': type },
    }))
    return alt ? api.updateRestaurantReviewPhoto({ ...ids, photoId: photo.id, photoPatch: { alt } }) : photo
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
