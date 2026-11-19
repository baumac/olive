import { describe, expect, it } from 'vitest'
import {
  Configuration,
  RestaurantsApi,
  ResponseError,
  UsersApi,
  type FetchAPI,
} from '../src/index.js'

interface Sent {
  url: URL
  method: string
  headers: Headers
  body: unknown
}

function fakeFetch(status: number, body: unknown, headers: Record<string, string> = {}) {
  const sent: Sent[] = []
  const fetchApi: FetchAPI = async (input, init) => {
    sent.push({
      url: new URL(String(input), 'http://localhost'),
      method: init?.method ?? 'GET',
      headers: new Headers(init?.headers),
      body: typeof init?.body === 'string' ? JSON.parse(init.body) : undefined,
    })
    const contentType = status >= 400 ? 'application/problem+json' : 'application/json'
    return new Response(status === 204 ? null : JSON.stringify(body), {
      status,
      headers: { 'Content-Type': contentType, ...headers },
    })
  }
  const config = new Configuration({ basePath: '/v1', fetchApi, accessToken: async () => 'stored-token' })
  return { sent, config }
}

const review = {
  id: 'r7d4f0h2j9l5s3v',
  restaurant: 'k3x9m2p7q1w8z4a',
  restaurant_name: "Luigi's Trattoria",
  space: 7,
  service: 9,
  food: 8,
  price: 6,
  cleanliness: 5,
  title: 'Best carbonara in town',
  creator: { id: 'u2g6k0m4o8q1s5w', name: 'Joe' },
  photos: [],
  created: '2024-11-04T01:38:00.431Z',
  updated: '2024-11-05T09:12:00.000Z',
}

const restaurant = {
  id: 'k3x9m2p7q1w8z4a',
  name: "Luigi's Trattoria",
  tags: ['italian', 'pasta'],
  location: { latitude: 41.3766, longitude: 2.1893, address: 'Carrer de Sant Carles 4, 08003 Barcelona', place_id: 'osm:N1' },
  creator: null,
  review_count: 0,
  rating: null,
  cover: null,
  created: '2024-11-04T01:26:55.002Z',
  updated: '2024-11-04T01:26:55.002Z',
}

const location = { latitude: 41.3766, longitude: 2.1893, address: 'Carrer de Sant Carles 4, 08003 Barcelona', placeId: 'osm:N1' }

describe('query parameters', () => {
  it('sends deepObject and exploded array filters', async () => {
    const { sent, config } = fakeFetch(200, { items: [], next_cursor: null })
    await new RestaurantsApi(config).listRestaurantReviews({
      restaurantId: 'k3x9m2p7q1w8z4a',
      food: { gte: 4, lt: 5 },
      creatorIn: ['u5n1c8r3t6y0b2e', 'u2g6k0m4o8q1s5w'],
      created: { gte: '2024-11-01T00:00:00.000Z' },
    })
    const q = sent[0].url.searchParams
    expect(sent[0].url.pathname).toBe('/v1/restaurants/k3x9m2p7q1w8z4a/reviews')
    expect(q.get('food[gte]')).toBe('4')
    expect(q.get('food[lt]')).toBe('5')
    expect(q.has('food[eq]')).toBe(false)
    expect(q.getAll('creator[in]')).toEqual(['u5n1c8r3t6y0b2e', 'u2g6k0m4o8q1s5w'])
    expect(q.get('created[gte]')).toBe('2024-11-01T00:00:00.000Z')
  })

  it('sends the near filter as a deepObject', async () => {
    const { sent, config } = fakeFetch(200, { items: [restaurant], next_cursor: null })
    const page = await new RestaurantsApi(config).listRestaurants({ near: { latitude: 41.3766, longitude: 2.1893, radius: 75 } })
    const q = sent[0].url.searchParams
    expect([q.get('near[latitude]'), q.get('near[longitude]'), q.get('near[radius]')]).toEqual(['41.3766', '2.1893', '75'])
    expect(page.items[0].location).toEqual(location)
  })
})

describe('merge patch bodies', () => {
  it('keeps null and leaves out missing properties', async () => {
    const { sent, config } = fakeFetch(200, restaurant)
    await new RestaurantsApi(config).updateRestaurant({
      restaurantId: 'k3x9m2p7q1w8z4a',
      restaurantPatch: { description: null, tags: null },
    })
    expect(sent[0].method).toBe('PATCH')
    expect(sent[0].headers.get('Content-Type')).toBe('application/merge-patch+json')
    expect(sent[0].body).toEqual({ description: null, tags: null })
  })

  it('sends only the given scores of a review patch', async () => {
    const { sent, config } = fakeFetch(200, review)
    await new RestaurantsApi(config).updateRestaurantReview({
      restaurantId: 'k3x9m2p7q1w8z4a',
      reviewId: 'r7d4f0h2j9l5s3v',
      restaurantReviewPatch: { price: 4, feedback: null },
    })
    expect(sent[0].body).toEqual({ price: 4, feedback: null })
  })
})

describe('reviews', () => {
  it('reads the scores and the restaurant name', async () => {
    const { config } = fakeFetch(200, { items: [review], next_cursor: null })
    const page = await new RestaurantsApi(config).listRestaurantReviews({ restaurantId: 'k3x9m2p7q1w8z4a' })

    expect(page.items[0]).toMatchObject({ space: 7, service: 9, food: 8, price: 6, cleanliness: 5, restaurantName: "Luigi's Trattoria" })
    expect(page.nextCursor).toBeNull()
  })

  it('keeps timestamps as strings', async () => {
    const { config } = fakeFetch(200, review)
    const got = await new RestaurantsApi(config).getRestaurantReview({
      restaurantId: 'k3x9m2p7q1w8z4a',
      reviewId: 'r7d4f0h2j9l5s3v',
    })
    expect(got.created).toBe('2024-11-04T01:38:00.431Z')
  })
})

describe('restaurant rating', () => {
  it('keeps a null rating and reads the averages', async () => {
    const rated = { ...restaurant, review_count: 2, rating: { overall: 7.6, space: 7.5, service: 8.5, food: 9, price: 5.5, cleanliness: 7.5 } }
    const { config } = fakeFetch(200, { items: [restaurant, rated], next_cursor: null })
    const page = await new RestaurantsApi(config).listRestaurants({})
    expect(page.items[0].rating).toBeNull()
    expect(page.items[1].reviewCount).toBe(2)
    expect(page.items[1].rating?.overall).toBe(7.6)
  })
})

describe('errors and headers', () => {
  it('throws ResponseError with the Problem Details body', async () => {
    const { config } = fakeFetch(404, { type: 'about:blank', title: 'Not Found', status: 404 })
    const error = await new RestaurantsApi(config)
      .getRestaurant({ restaurantId: 'k3x9m2p7q1w8z4a' })
      .catch((e: unknown) => e)
    expect(error).toBeInstanceOf(ResponseError)
    const response = (error as ResponseError).response
    expect(response.status).toBe(404)
    expect(await response.json()).toMatchObject({ title: 'Not Found' })
  })

  it('gives the Location header through the raw response', async () => {
    const { config } = fakeFetch(201, restaurant, { Location: '/v1/restaurants/k3x9m2p7q1w8z4a' })
    const raw = await new RestaurantsApi(config).createRestaurantRaw({
      restaurantCreate: { name: "Luigi's Trattoria", location },
    })
    expect(raw.raw.headers.get('Location')).toBe('/v1/restaurants/k3x9m2p7q1w8z4a')
    expect((await raw.value()).name).toBe("Luigi's Trattoria")
  })
})

describe('auth', () => {
  it('does not send the stored token on sign in', async () => {
    const { sent, config } = fakeFetch(200, {
      token: 't',
      expires: '2024-11-08T08:12:40.517Z',
      user: { id: 'u5n1c8r3t6y0b2e', email: 'ana@example.com', name: 'Ana', created: restaurant.created, updated: restaurant.updated },
    })
    await new UsersApi(config).signIn({ credentials: { email: 'ana@example.com', password: 'correct-horse-battery' } })
    expect(sent[0].headers.has('Authorization')).toBe(false)
  })

  it('sends the token as a bearer token on create', async () => {
    const { sent, config } = fakeFetch(201, restaurant)
    await new RestaurantsApi(config).createRestaurant({ restaurantCreate: { name: "Luigi's Trattoria", location } })
    expect(sent[0].headers.get('Authorization')).toBe('Bearer stored-token')
  })
})

describe('review photos', () => {
  const photo = {
    id: 'p4h7j1l5n9q2s6v',
    alt: null,
    width: 1536,
    height: 2048,
    urls: { full: '/v1/x/full', thumb: '/v1/x/thumb' },
    created: '2024-11-06T19:04:30.811Z',
  }

  it('sends the image with the content type of the call and the token', async () => {
    const { sent, config } = fakeFetch(201, photo)
    const api = new RestaurantsApi(config)
    const body = new Blob(['png bytes'], { type: 'image/png' })
    const got = await api.addRestaurantReviewPhoto(
      { restaurantId: 'k3x9m2p7q1w8z4a', reviewId: 'r7d4f0h2j9l5s3v', body },
      async ({ init }) => ({ headers: { ...(init.headers as Record<string, string>), 'Content-Type': 'image/png' } }),
    )
    expect(got.alt).toBeNull()
    expect(sent[0]!.url.pathname).toBe('/v1/restaurants/k3x9m2p7q1w8z4a/reviews/r7d4f0h2j9l5s3v/photos')
    expect(sent[0]!.headers.get('Content-Type')).toBe('image/png')
    expect(sent[0]!.headers.get('Authorization')).toBe('Bearer stored-token')
  })

  it('reads the photos of a review', async () => {
    const { config } = fakeFetch(200, { ...review, photos: [photo] })
    const got = await new RestaurantsApi(config).getRestaurantReview({ restaurantId: review.restaurant, reviewId: review.id })
    expect(got.photos[0]!.urls.thumb).toBe('/v1/x/thumb')
  })
})
