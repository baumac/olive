import type { Recipe, RecipeReview, SavedList } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import AppHeader from '@/components/app/AppHeader.vue'
import KindSwitch from '@/components/app/KindSwitch.vue'
import PhotoGallery from '@/components/app/PhotoGallery.vue'
import RecipeCard from '@/components/app/RecipeCard.vue'
import RecipeDetails from '@/components/app/RecipeDetails.vue'
import ReviewCard from '@/components/app/ReviewCard.vue'
import SaveToListMenu from '@/components/app/SaveToListMenu.vue'
import { listsApi, recipesApi, restaurantsApi, reviewersApi } from '@/lib/api'
import { formatMinutes, formatScore } from '@/lib/format'
import { listCountText } from '@/lib/lists'
import { recipeFilterParams } from '@/lib/recipeFilters'
import { recipeReviewFilterParams, reviewFilterDefsOf } from '@/lib/reviewFilters'
import { kindOf, parentNameOf, reviewPathOf } from '@/lib/reviewKinds'
import { overallOf, recipeScoreFields } from '@/lib/reviews'
import { clearSession, saveSession } from '@/lib/session'
import { checkIngredients, checkStep, checkWholeNumber, ingredientLines } from '@/lib/validation'
import ListPage from '@/pages/ListPage.vue'
import RecipeFormPage from '@/pages/RecipeFormPage.vue'
import ReviewerPage from '@/pages/ReviewerPage.vue'
import { expectAccessible } from './axe'
import { rejectWith } from './problem'

const recipe: Recipe = {
  id: 'c4r8e2p6i0s3t7a',
  name: 'Shakshuka',
  description: 'Eggs in a sauce',
  tags: new Set(['breakfast']),
  ingredients: ['6 eggs', '1 onion'],
  steps: ['Cook the onion.', 'Add the eggs.\nCover the pan.'],
  source: 'https://recipes.example/shakshuka',
  totalTime: 90,
  servings: 3,
  creator: { id: 'u5n1c8r3t6y0b2e', name: 'Ana' },
  reviewCount: 1,
  rating: { overall: 8.3, taste: 9, ease: 8, instructions: 8, value: 8 },
  cover: null,
  favorite: false,
  ownReviewCount: 0,
  created: '2024-11-04T01:26:55.002Z',
  updated: '2024-11-04T01:26:55.002Z',
}

const review: RecipeReview = {
  id: 'v2k6m0p4r8t1w5y',
  recipe: recipe.id,
  recipeName: recipe.name,
  date: '2024-11-06',
  overall: 8.25,
  taste: 9,
  ease: 8,
  instructions: 8,
  value: 8,
  title: 'A quick breakfast',
  creator: { id: 'u5n1c8r3t6y0b2e', name: 'Ana' },
  photos: [],
  created: '2024-11-06T19:02:11.120Z',
  updated: '2024-11-06T19:02:11.120Z',
}

const ana = { id: 'u5n1c8r3t6y0b2e', email: 'ana@example.com', name: 'Ana', approved: true, created: '2024-11-04T01:26:55.002Z', updated: '2024-11-04T01:26:55.002Z' }

function signIn() {
  saveSession({ expires: '2099-01-01T00:00:00.000Z', user: ana })
}

function attach() {
  const el = document.createElement('div')
  document.body.appendChild(el)
  return el
}

async function routerAt(path: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
  await router.push(path)
  return router
}

afterEach(() => {
  vi.restoreAllMocks()
  clearSession()
})

describe('recipe helpers', () => {
  it('formats a total time', () => {
    expect([5, 60, 90].map(formatMinutes)).toEqual(['5 min', '1 hr', '1 hr 30 min'])
  })

  it('changes the recipe filters to the parameters of the SDK', () => {
    expect(recipeFilterParams({ search: 'eggs', tags: ['breakfast'], overall: '8', total_time: '30', list: 'l7d2f9h4j1k8m3p' })).toEqual({
      search: 'eggs',
      tagsHasAny: ['breakfast'],
      overall: { gte: 8 },
      totalTime: { lte: 30 },
      listEq: 'l7d2f9h4j1k8m3p',
    })
    expect(recipeReviewFilterParams({ taste: '7' })).toMatchObject({ taste: { gte: 7 }, ease: undefined })
    expect(reviewFilterDefsOf('recipe').map((d) => d.key)).toEqual(['search', 'overall', 'taste', 'ease', 'instructions', 'value'])
  })

  it('checks the ingredients, the steps, and the numbers', () => {
    expect(ingredientLines(' 6 eggs \n\n1 onion\r\n')).toEqual(['6 eggs', '1 onion'])
    expect(checkIngredients([])).toBe('Enter at least one ingredient.')
    expect(checkIngredients(['x'.repeat(201)])).toBe('Use at most 200 characters for each ingredient.')
    expect(checkIngredients(['6 eggs'])).toBe('')
    expect(checkStep(' \n')).toBe('Enter the step, or remove it.')
    expect(checkStep('Mix.\nBake.')).toBe('')
    expect(checkWholeNumber('', 1, 100)).toBe('')
    expect(checkWholeNumber('1.5', 1, 100)).toBe('Enter a whole number from 1 to 100.')
    expect(checkWholeNumber('101', 1, 100)).toBe('Enter a whole number from 1 to 100.')
  })

  it('gives the overall of the four recipe scores', () => {
    expect(overallOf({ taste: 9, ease: 8, instructions: 8, value: 8 }, recipeScoreFields)).toBe(8.3)
    expect(overallOf({ taste: 9, ease: 8, instructions: 8, value: null }, recipeScoreFields)).toBeNull()
  })

  it('knows the kind and the paths of a recipe review', () => {
    expect(kindOf(review)).toBe('recipe')
    expect(parentNameOf(review)).toBe('Shakshuka')
    expect(reviewPathOf(review)).toBe('/recipes/c4r8e2p6i0s3t7a/reviews/v2k6m0p4r8t1w5y')
  })

  it('names the items of a list', () => {
    expect(listCountText({ restaurantCount: 2, recipeCount: 0 })).toBe('2 restaurants')
    expect(listCountText({ restaurantCount: 0, recipeCount: 1 })).toBe('1 recipe')
    expect(listCountText({ restaurantCount: 1, recipeCount: 3 })).toBe('1 restaurant, 3 recipes')
  })
})

describe('RecipeCard', () => {
  it('links the recipe and shows its time', async () => {
    const w = mount(RecipeCard, { props: { recipe }, global: { plugins: [await routerAt('/recipes')] }, attachTo: attach() })
    expect(w.get('h2 a').attributes('href')).toBe('/recipes/c4r8e2p6i0s3t7a')
    expect(w.text()).toContain('1 hr 30 min')
    expect(w.text()).toContain('breakfast')
    await expectAccessible(w.element)
  })

  it('adds the recipe to the favorites', async () => {
    signIn()
    const add = vi.spyOn(listsApi, 'addListRecipe').mockResolvedValue()
    const w = mount(RecipeCard, { props: { recipe }, global: { plugins: [await routerAt('/recipes')] }, attachTo: attach() })
    await w.get('button[aria-label="Favorite Shakshuka"]').trigger('click')
    await flushPromises()
    expect(add).toHaveBeenCalledWith({ listRef: 'favorites', recipeId: recipe.id })
    expect(w.emitted('favorite')).toEqual([[true]])
  })
})

describe('RecipeDetails', () => {
  it('lists the ingredients and numbers the steps', async () => {
    const w = mount(RecipeDetails, { props: { recipe }, attachTo: attach() })
    expect(w.findAll('ul li').map((li) => li.text())).toContain('6 eggs')
    expect(w.findAll('ol li').map((li) => li.text())).toEqual(['Cook the onion.', 'Add the eggs.\nCover the pan.'])
    expect(w.text()).toContain('3 servings')
    expect(w.get('a').attributes('href')).toBe(recipe.source)
    await expectAccessible(w.element)
  })
})

describe('ReviewCard of a recipe review', () => {
  it('links the review and shows the recipe scores', async () => {
    const w = mount(ReviewCard, { props: { review, showRestaurant: true }, global: { plugins: [await routerAt('/recipes')] }, attachTo: attach() })
    expect(w.get('h3 a').attributes('href')).toBe('/recipes/c4r8e2p6i0s3t7a/reviews/v2k6m0p4r8t1w5y')
    expect(w.text()).toContain('Shakshuka')
    expect(w.text()).toContain('Cooked on 6 Nov 2024')
    expect(w.findAll('[role=img]').map((s) => s.attributes('aria-label'))).toEqual([`Overall: ${formatScore(review.overall)} out of 10`])
  })
})

describe('KindSwitch and AppHeader', () => {
  it('marks the current kind', async () => {
    const router = await routerAt(`/reviewers/${ana.id}?kind=recipes`)
    const w = mount(KindSwitch, { props: { kind: 'recipe', label: 'Reviews of' }, global: { plugins: [router] }, attachTo: attach() })
    const links = w.findAll('a')
    expect(links.map((a) => a.text())).toEqual(['All', 'Restaurants', 'Recipes'])
    expect(links[2]!.attributes('aria-current')).toBe('page')
    expect(links[0]!.attributes('href')).toBe(`/reviewers/${ana.id}`)
    expect(links[1]!.attributes('href')).toBe(`/reviewers/${ana.id}?kind=restaurants`)
    expect(links[2]!.attributes('href')).toBe(`/reviewers/${ana.id}?kind=recipes`)
    await expectAccessible(w.element)
  })

  it('marks the section of the page', async () => {
    const w = mount(AppHeader, { global: { plugins: [await routerAt('/recipes/c4r8e2p6i0s3t7a')] }, attachTo: attach() })
    const sections = w.get('nav[aria-label=Sections]').findAll('a')
    expect(sections.map((a) => [a.text(), a.attributes('aria-current')])).toEqual([
      ['Table', undefined],
      ['Restaurants', undefined],
      ['Recipes', 'page'],
    ])
  })
})

describe('RecipeFormPage', () => {
  async function mountForm() {
    const router = await routerAt('/recipes/new')
    const w = mount(RecipeFormPage, { global: { plugins: [router] }, attachTo: attach() })
    await flushPromises()
    return { w, router }
  }

  it('adds, moves, and removes steps', async () => {
    const { w } = await mountForm()
    await w.get('#recipe-step-0').setValue('Cook.')
    await w.findAll('button').find((b) => b.text().includes('Add step'))!.trigger('click')
    await w.get('#recipe-step-1').setValue('Serve.')
    expect(w.get('button[aria-label="Move step 1 down"]').attributes('disabled')).toBeUndefined()
    expect(w.get('button[aria-label="Remove step 1"]').attributes('disabled')).toBeUndefined()
    await w.get('button[aria-label="Move step 2 up"]').trigger('click')
    expect((w.get('#recipe-step-0').element as HTMLTextAreaElement).value).toBe('Serve.')
    await w.get('button[aria-label="Remove step 1"]').trigger('click')
    expect(w.findAll('textarea[id^=recipe-step-]').map((t) => (t.element as HTMLTextAreaElement).value)).toEqual(['Cook.'])
    expect(w.get('button[aria-label="Remove step 1"]').attributes('disabled')).toBeDefined()
    await expectAccessible(w.element)
  })

  it('checks the fields before a request', async () => {
    const create = vi.spyOn(recipesApi, 'createRecipe')
    const { w } = await mountForm()
    await w.get('form').trigger('submit')
    expect(create).not.toHaveBeenCalled()
    expect(w.text()).toContain('Enter a name with at least one character that is not a space.')
    expect(w.text()).toContain('Enter at least one ingredient.')
    expect(w.text()).toContain('Enter the step, or remove it.')
  })

  it('sends one ingredient for each line', async () => {
    const create = vi.spyOn(recipesApi, 'createRecipe').mockResolvedValue(recipe)
    const { w, router } = await mountForm()
    await w.get('input[name=name]').setValue('Shakshuka')
    await w.get('textarea[name=ingredients]').setValue('6 eggs\n\n 1 onion ')
    await w.get('#recipe-step-0').setValue('Cook.')
    await w.get('input[name=source]').setValue('recipes.example/shakshuka')
    await w.get('input[name=total-time]').setValue('30')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(create).toHaveBeenCalledWith({
      recipeCreate: {
        name: 'Shakshuka',
        description: undefined,
        tags: undefined,
        ingredients: ['6 eggs', '1 onion'],
        steps: ['Cook.'],
        source: 'https://recipes.example/shakshuka',
        totalTime: 30,
        servings: undefined,
      },
    })
    expect(router.currentRoute.value.path).toBe('/recipes/c4r8e2p6i0s3t7a')
  })

  it('links the recipe with the same source', async () => {
    vi.spyOn(recipesApi, 'createRecipe').mockRejectedValue(
      rejectWith(409, { code: 'source_taken', property: 'source', recipe: 'd5f9h3j7l1n4q8s' }),
    )
    const { w } = await mountForm()
    await w.get('input[name=name]').setValue('Copy')
    await w.get('textarea[name=ingredients]').setValue('6 eggs')
    await w.get('#recipe-step-0').setValue('Cook.')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(w.text()).toContain('This recipe is already here')
    expect(w.findAll('a').find((a) => a.text() === 'Open the recipe')!.attributes('href')).toBe('/recipes/d5f9h3j7l1n4q8s')
  })
})

describe('recipe parts of the list and reviewer pages', () => {
  const list: SavedList = {
    id: 'l7d2f9h4j1k8m3p',
    name: 'Brunch',
    kind: 'custom',
    creator: { id: ana.id, name: 'Ana' },
    restaurantCount: 0,
    recipeCount: 1,
    created: '2024-11-04T01:26:55.002Z',
    updated: '2024-11-04T01:26:55.002Z',
  }

  it('shows the recipes of a list and removes one', async () => {
    signIn()
    vi.spyOn(listsApi, 'getList').mockResolvedValue(list)
    vi.spyOn(restaurantsApi, 'listRestaurants').mockResolvedValue({ items: [], nextCursor: null })
    const recipes = vi.spyOn(recipesApi, 'listRecipes').mockResolvedValue({ items: [recipe], nextCursor: null })
    const remove = vi.spyOn(listsApi, 'removeListRecipe').mockResolvedValue()
    const w = mount(ListPage, { props: { listId: list.id }, global: { plugins: [await routerAt(`/lists/${list.id}`)] }, attachTo: attach() })
    await flushPromises()
    expect(recipes.mock.calls[0]![0]).toMatchObject({ listEq: list.id, sort: '-overall' })
    expect(w.text()).toContain('A list by Ana')
    expect(w.text()).toContain('1 recipe')
    expect(w.get('h3 a').attributes('href')).toBe(`/recipes/${recipe.id}`)
    expect(w.findAll('a').find((a) => a.text().includes('Sort and filter'))!.attributes('href')).toBe(`/recipes?list=${list.id}`)
    await w.get(`button[aria-label="Remove ${recipe.name} from the list"]`).trigger('click')
    await flushPromises()
    expect(remove).toHaveBeenCalledWith({ listRef: list.id, recipeId: recipe.id })
    expect(w.text()).toContain('No restaurants or recipes yet')
  })

  it('shows the recipe average and the recipes of a reviewer', async () => {
    vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [] })
    vi.spyOn(reviewersApi, 'getReviewer').mockResolvedValue({
      id: ana.id,
      name: 'Ana',
      restaurantReviewCount: 0,
      restaurantAverageOverall: null,
      restaurantCount: 0,
      recipeReviewCount: 2,
      recipeAverageOverall: 7.5,
      recipeCount: 1,
      listCount: 1,
    })
    vi.spyOn(reviewersApi, 'listReviewerRestaurantReviews').mockResolvedValue({ items: [], nextCursor: null })
    vi.spyOn(reviewersApi, 'listReviewerRecipeReviews').mockResolvedValue({ items: [], nextCursor: null })
    const added = vi.spyOn(recipesApi, 'listRecipes').mockResolvedValue({ items: [recipe], nextCursor: null })
    const w = mount(ReviewerPage, { props: { reviewerId: ana.id }, global: { plugins: [await routerAt(`/reviewers/${ana.id}?kind=recipes`)] }, attachTo: attach() })
    await flushPromises()
    expect(w.find('[aria-label="Average recipe score: 7.5 out of 10"]').exists()).toBe(true)
    expect(w.text()).toContain('0 visits · 2 cooks · 1 list')
    expect(w.findAll('h2').map((h) => h.text())).toContain('Recipes added')
    expect(added.mock.calls[0]![0]).toMatchObject({ creatorEq: ana.id, sort: '-created' })
    expect(w.get('h3 a').attributes('href')).toBe(`/recipes/${recipe.id}`)
  })

  it('saves a recipe to a list with the menu', async () => {
    signIn()
    const lists = vi.spyOn(listsApi, 'listReviewerLists').mockResolvedValue({ items: [{ ...list, hasRecipe: true }] })
    const remove = vi.spyOn(listsApi, 'removeListRecipe').mockResolvedValue()
    const w = mount(SaveToListMenu, { props: { recipeId: recipe.id }, global: { plugins: [await routerAt(`/recipes/${recipe.id}`)] }, attachTo: attach() })
    await w.get('button').trigger('click')
    await flushPromises()
    expect(lists).toHaveBeenCalledWith({ reviewerId: ana.id, recipeEq: recipe.id })
    const box = document.body.querySelector<HTMLInputElement>('input[type=checkbox]')!
    expect(box.checked).toBe(true)
    box.checked = false
    box.dispatchEvent(new Event('change'))
    await flushPromises()
    expect(remove).toHaveBeenCalledWith({ listRef: list.id, recipeId: recipe.id })
    w.unmount()
  })

  it('links a gallery photo to its recipe review', async () => {
    const photo = {
      id: 'p8b3d7f1h5k9m2q',
      alt: null,
      width: 10,
      height: 10,
      urls: { full: '/full', thumb: '/thumb' },
      created: '2024-11-06T19:04:30.811Z',
      review: review.id,
      creator: { id: ana.id, name: 'Ana' },
    }
    const photos = vi.spyOn(recipesApi, 'listRecipePhotos').mockResolvedValue({ items: [photo], nextCursor: null })
    const w = mount(PhotoGallery, { props: { parent: { ...recipe, cover: photo }, kind: 'recipe' }, global: { plugins: [await routerAt('/')] }, attachTo: attach() })
    await flushPromises()
    expect(photos.mock.calls[0]![0]).toMatchObject({ recipeId: recipe.id })
    w.vm.openCover()
    await flushPromises()
    const link = Array.from(document.body.querySelectorAll('a')).find((a) => a.textContent?.includes('review by Ana'))
    expect(link?.getAttribute('href')).toBe(`/recipes/${recipe.id}/reviews/${review.id}`)
    w.unmount()
  })

  it('lets a user who is not the creator change each field of a reviewed recipe', async () => {
    signIn()
    vi.spyOn(recipesApi, 'getRecipe').mockResolvedValue({ ...recipe, creator: { id: 'u2g6k0m4o8q1s5w', name: 'Joe' }, reviewCount: 1 })
    vi.spyOn(recipesApi, 'listRecipeReviews').mockResolvedValue({ items: [], nextCursor: null })
    const update = vi.spyOn(recipesApi, 'updateRecipe').mockResolvedValue(recipe)
    const w = mount(RecipeFormPage, { props: { recipeId: recipe.id }, global: { plugins: [await routerAt(`/recipes/${recipe.id}/edit`)] }, attachTo: attach() })
    await flushPromises()
    for (const field of ['input[name=name]', 'textarea[name=ingredients]', '#recipe-step-0', 'input[name=source]']) {
      expect(w.get(field).attributes('readonly')).toBeUndefined()
    }
    expect(w.text()).not.toContain('You cannot delete it')
    await w.get('input[name=name]').setValue('Green shakshuka')
    await w.get('#recipe-step-0').setValue('Cook the onion slowly.')
    await w.get('input[name=source]').setValue('https://recipes.example/green')
    await w.get('input[name=servings]').setValue('6')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(update.mock.calls[0]![0].recipePatch).toEqual({
      name: 'Green shakshuka',
      steps: ['Cook the onion slowly.', recipe.steps[1]],
      source: 'https://recipes.example/green',
      servings: 6,
    })
    w.unmount()
  })

  it('tells the creator of a reviewed recipe that the delete is blocked', async () => {
    signIn()
    vi.spyOn(recipesApi, 'getRecipe').mockResolvedValue({ ...recipe, reviewCount: 1 })
    vi.spyOn(recipesApi, 'listRecipeReviews').mockResolvedValue({ items: [], nextCursor: null })
    const w = mount(RecipeFormPage, { props: { recipeId: recipe.id }, global: { plugins: [await routerAt(`/recipes/${recipe.id}/edit`)] }, attachTo: attach() })
    await flushPromises()
    expect(w.text()).toContain('Other people have reviewed this recipe. You cannot delete it.')
    expect(w.get('textarea[name=ingredients]').attributes('readonly')).toBeUndefined()
    w.unmount()
  })
})
