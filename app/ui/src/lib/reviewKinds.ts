import type {
  PantryItem,
  PantryItemPhoto,
  PantryItemReview,
  PantryItemReviewCreate,
  PantryItemVersion,
  Photo,
  Recipe,
  RecipePhoto,
  RecipeReview,
  RecipeReviewCreate,
  RecipeSummary,
  RecipeVersion,
  Restaurant,
  RestaurantPhoto,
  RestaurantReview,
  RestaurantReviewCreate,
  RestaurantVersion,
} from '@reviews/sdk'
import { listsApi, pantryApi, recipesApi, restaurantsApi, reviewersApi } from './api'
import { type ScoreField, pantryItemScoreFields, recipeScoreFields, scoreFields } from './reviews'

// A review kind is a kind of thing that users review. See decisions 00004
// and 00012.
export type ReviewKind = 'restaurant' | 'recipe' | 'pantryItem'
export type AnyReview = RestaurantReview | RecipeReview | PantryItemReview
export type Parent = Restaurant | Recipe | PantryItem
export type ParentSummary = Restaurant | RecipeSummary | PantryItem
export type ParentPhoto = RestaurantPhoto | RecipePhoto | PantryItemPhoto
export type AnyVersion = RestaurantVersion | RecipeVersion | PantryItemVersion

export interface ReviewFields {
  // date is the day of the visit or the cook, such as 2024-11-04.
  date: string
  scores: Record<string, number>
  title: string
  feedback?: string | null
}

export interface Page<T> {
  items: T[]
  nextCursor: string | null
}

// ListParams are the query parameters of a review list. The score filters
// have the names of the score fields.
export type ListParams = Record<string, unknown> & { limit?: number; cursor?: string; sort?: string }

export interface KindInfo {
  kind: ReviewKind
  noun: string
  plural: string
  // label is the noun at the start of a label, such as "Pantry item".
  label: string
  // path is the path of the list page, such as /restaurants.
  path: string
  // query is the value of kind in the URL of the table, the journal, and
  // the tag page.
  query: string
  // dateLabel names the date of a review. See decision 00005.
  dateLabel: string
  // occasion is the noun of one visit, one cook, or one tasting.
  occasion: string
  scoreFields: ScoreField[]
  getParent(id: string): Promise<Parent>
  getReview(parentId: string, reviewId: string): Promise<AnyReview>
  listReviews(parentId: string, params: ListParams): Promise<Page<AnyReview>>
  listReviewerReviews(reviewerId: string, params: ListParams): Promise<Page<AnyReview>>
  // listAdded lists the parents that a user created.
  listAdded(creatorId: string, params: { limit?: number; cursor?: string; sort?: string }): Promise<Page<ParentSummary>>
  createReview(parentId: string, review: ReviewFields): Promise<AnyReview>
  updateReview(parentId: string, reviewId: string, review: ReviewFields): Promise<AnyReview>
  deleteReview(parentId: string, reviewId: string): Promise<void>
  // addPhoto sends the file with its own content type. The SDK sets image/*
  // for all images. The API needs the real type.
  addPhoto(parentId: string, reviewId: string, file: File): Promise<Photo>
  updatePhoto(parentId: string, reviewId: string, photoId: string, alt: string | null): Promise<Photo>
  deletePhoto(parentId: string, reviewId: string, photoId: string): Promise<void>
  listPhotos(parentId: string, params: { limit?: number; cursor?: string }): Promise<Page<ParentPhoto>>
  addToList(listRef: string, id: string): Promise<void>
  removeFromList(listRef: string, id: string): Promise<void>
  // listVersions lists the history, the newest first. See decision 00004.
  listVersions(parentId: string, params: { limit?: number; cursor?: string }): Promise<Page<AnyVersion>>
}

// The score fields of the SDK have the type any. These builders name each
// field. A missing or a wrong score is then a type error.
function restaurantScores(s: Record<string, number>): Omit<RestaurantReviewCreate, 'date' | 'title'> {
  return { space: s.space, service: s.service, food: s.food, price: s.price }
}

function recipeScores(s: Record<string, number>): Omit<RecipeReviewCreate, 'date' | 'title'> {
  return { taste: s.taste, ease: s.ease, instructions: s.instructions, value: s.value }
}

function pantryItemScores(s: Record<string, number>): Omit<PantryItemReviewCreate, 'date' | 'title'> {
  return { taste: s.taste, aroma: s.aroma, texture: s.texture, value: s.value }
}

function contentType(file: File) {
  return async ({ init }: { init: RequestInit }) => ({ headers: { ...(init.headers as Record<string, string>), 'Content-Type': file.type } })
}

const restaurant: KindInfo = {
  kind: 'restaurant',
  noun: 'restaurant',
  plural: 'restaurants',
  label: 'Restaurant',
  path: '/restaurants',
  query: 'restaurants',
  dateLabel: 'Visited on',
  occasion: 'visit',
  scoreFields,
  getParent: (restaurantId) => restaurantsApi.getRestaurant({ restaurantId }),
  getReview: (restaurantId, reviewId) => restaurantsApi.getRestaurantReview({ restaurantId, reviewId }),
  listReviews: (restaurantId, params) => restaurantsApi.listRestaurantReviews({ restaurantId, ...params }),
  listReviewerReviews: (reviewerId, params) => reviewersApi.listReviewerRestaurantReviews({ reviewerId, ...params }),
  listAdded: (creatorEq, params) => restaurantsApi.listRestaurants({ creatorEq, ...params }),
  createReview: (restaurantId, { date, scores, title, feedback }) =>
    restaurantsApi.createRestaurantReview({
      restaurantId,
      restaurantReviewCreate: { date, ...restaurantScores(scores), title, feedback: feedback ?? undefined },
    }),
  updateReview: (restaurantId, reviewId, { date, scores, title, feedback }) =>
    restaurantsApi.updateRestaurantReview({ restaurantId, reviewId, restaurantReviewPatch: { date, ...restaurantScores(scores), title, feedback } }),
  deleteReview: (restaurantId, reviewId) => restaurantsApi.deleteRestaurantReview({ restaurantId, reviewId }),
  addPhoto: (restaurantId, reviewId, file) => restaurantsApi.addRestaurantReviewPhoto({ restaurantId, reviewId, body: file }, contentType(file)),
  updatePhoto: (restaurantId, reviewId, photoId, alt) =>
    restaurantsApi.updateRestaurantReviewPhoto({ restaurantId, reviewId, photoId, photoPatch: { alt } }),
  deletePhoto: (restaurantId, reviewId, photoId) => restaurantsApi.deleteRestaurantReviewPhoto({ restaurantId, reviewId, photoId }),
  listPhotos: (restaurantId, params) => restaurantsApi.listRestaurantPhotos({ restaurantId, ...params }),
  addToList: (listRef, restaurantId) => listsApi.addListRestaurant({ listRef, restaurantId }),
  removeFromList: (listRef, restaurantId) => listsApi.removeListRestaurant({ listRef, restaurantId }),
  listVersions: (restaurantId, params) => restaurantsApi.listRestaurantVersions({ restaurantId, ...params }),
}

const recipe: KindInfo = {
  kind: 'recipe',
  noun: 'recipe',
  plural: 'recipes',
  label: 'Recipe',
  path: '/recipes',
  query: 'recipes',
  dateLabel: 'Cooked on',
  occasion: 'cook',
  scoreFields: recipeScoreFields,
  getParent: (recipeId) => recipesApi.getRecipe({ recipeId }),
  getReview: (recipeId, reviewId) => recipesApi.getRecipeReview({ recipeId, reviewId }),
  listReviews: (recipeId, params) => recipesApi.listRecipeReviews({ recipeId, ...params }),
  listReviewerReviews: (reviewerId, params) => reviewersApi.listReviewerRecipeReviews({ reviewerId, ...params }),
  listAdded: (creatorEq, params) => recipesApi.listRecipes({ creatorEq, ...params }),
  createReview: (recipeId, { date, scores, title, feedback }) =>
    recipesApi.createRecipeReview({ recipeId, recipeReviewCreate: { date, ...recipeScores(scores), title, feedback: feedback ?? undefined } }),
  updateReview: (recipeId, reviewId, { date, scores, title, feedback }) =>
    recipesApi.updateRecipeReview({ recipeId, reviewId, recipeReviewPatch: { date, ...recipeScores(scores), title, feedback } }),
  deleteReview: (recipeId, reviewId) => recipesApi.deleteRecipeReview({ recipeId, reviewId }),
  addPhoto: (recipeId, reviewId, file) => recipesApi.addRecipeReviewPhoto({ recipeId, reviewId, body: file }, contentType(file)),
  updatePhoto: (recipeId, reviewId, photoId, alt) =>
    recipesApi.updateRecipeReviewPhoto({ recipeId, reviewId, photoId, photoPatch: { alt } }),
  deletePhoto: (recipeId, reviewId, photoId) => recipesApi.deleteRecipeReviewPhoto({ recipeId, reviewId, photoId }),
  listPhotos: (recipeId, params) => recipesApi.listRecipePhotos({ recipeId, ...params }),
  addToList: (listRef, recipeId) => listsApi.addListRecipe({ listRef, recipeId }),
  removeFromList: (listRef, recipeId) => listsApi.removeListRecipe({ listRef, recipeId }),
  listVersions: (recipeId, params) => recipesApi.listRecipeVersions({ recipeId, ...params }),
}

const pantryItem: KindInfo = {
  kind: 'pantryItem',
  noun: 'pantry item',
  plural: 'pantry items',
  label: 'Pantry item',
  path: '/pantry',
  query: 'pantry',
  dateLabel: 'Tasted on',
  occasion: 'tasting',
  scoreFields: pantryItemScoreFields,
  getParent: (pantryItemId) => pantryApi.getPantryItem({ pantryItemId }),
  getReview: (pantryItemId, reviewId) => pantryApi.getPantryItemReview({ pantryItemId, reviewId }),
  listReviews: (pantryItemId, params) => pantryApi.listPantryItemReviews({ pantryItemId, ...params }),
  listReviewerReviews: (reviewerId, params) => reviewersApi.listReviewerPantryItemReviews({ reviewerId, ...params }),
  listAdded: (creatorEq, params) => pantryApi.listPantryItems({ creatorEq, ...params }),
  createReview: (pantryItemId, { date, scores, title, feedback }) =>
    pantryApi.createPantryItemReview({
      pantryItemId,
      pantryItemReviewCreate: { date, ...pantryItemScores(scores), title, feedback: feedback ?? undefined },
    }),
  updateReview: (pantryItemId, reviewId, { date, scores, title, feedback }) =>
    pantryApi.updatePantryItemReview({ pantryItemId, reviewId, pantryItemReviewPatch: { date, ...pantryItemScores(scores), title, feedback } }),
  deleteReview: (pantryItemId, reviewId) => pantryApi.deletePantryItemReview({ pantryItemId, reviewId }),
  addPhoto: (pantryItemId, reviewId, file) => pantryApi.addPantryItemReviewPhoto({ pantryItemId, reviewId, body: file }, contentType(file)),
  updatePhoto: (pantryItemId, reviewId, photoId, alt) =>
    pantryApi.updatePantryItemReviewPhoto({ pantryItemId, reviewId, photoId, photoPatch: { alt } }),
  deletePhoto: (pantryItemId, reviewId, photoId) => pantryApi.deletePantryItemReviewPhoto({ pantryItemId, reviewId, photoId }),
  listPhotos: (pantryItemId, params) => pantryApi.listPantryItemPhotos({ pantryItemId, ...params }),
  addToList: (listRef, pantryItemId) => listsApi.addListPantryItem({ listRef, pantryItemId }),
  removeFromList: (listRef, pantryItemId) => listsApi.removeListPantryItem({ listRef, pantryItemId }),
  listVersions: (pantryItemId, params) => pantryApi.listPantryItemVersions({ pantryItemId, ...params }),
}

export const kinds: Record<ReviewKind, KindInfo> = { restaurant, recipe, pantryItem }

// reviewKinds has the kinds in the order of the navigation.
export const reviewKinds: ReviewKind[] = ['restaurant', 'recipe', 'pantryItem']

// kindOfQuery reads the kind of a kind parameter in the URL, such as
// kind=pantry. Other values give undefined.
export function kindOfQuery(value: unknown): ReviewKind | undefined {
  return reviewKinds.find((k) => kinds[k].query === value)
}

// subjectKindOf gives the kind of the ID prop of a component. A component
// gets one of the three.
export function subjectKindOf(ids: { restaurantId?: string; recipeId?: string; pantryItemId?: string }): ReviewKind {
  if (ids.pantryItemId !== undefined) {
    return 'pantryItem'
  }
  return ids.recipeId !== undefined ? 'recipe' : 'restaurant'
}

export function kindOf(review: AnyReview): ReviewKind {
  if ('pantryItem' in review) {
    return 'pantryItem'
  }
  return 'recipe' in review ? 'recipe' : 'restaurant'
}

export function parentIdOf(review: AnyReview): string {
  if ('pantryItem' in review) {
    return review.pantryItem
  }
  return 'recipe' in review ? review.recipe : review.restaurant
}

export function parentNameOf(review: AnyReview): string {
  if ('pantryItem' in review) {
    return review.pantryItemName
  }
  return 'recipe' in review ? review.recipeName : review.restaurantName
}

export function parentPathOf(review: AnyReview): string {
  return `${kinds[kindOf(review)].path}/${parentIdOf(review)}`
}

export function reviewPathOf(review: AnyReview): string {
  return `${parentPathOf(review)}/reviews/${review.id}`
}

// scoreOf reads a score of a review by the name of its field.
export function scoreOf(scores: object, field: ScoreField): number {
  return (scores as Record<string, number>)[field.name] ?? 0
}
