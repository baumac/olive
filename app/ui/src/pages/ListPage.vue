<script setup lang="ts">
import type { RecipeSummary, Restaurant, SavedList } from '@reviews/sdk'
import { HeartIcon, ListIcon, MapIcon, PencilIcon, SlidersHorizontalIcon } from 'lucide-vue-next'
import { formatDate, formatFullDate, formatMinutes } from '@/lib/format'
import { tagList } from '@/lib/reviews'
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import ErrorPage from '@/components/app/ErrorPage.vue'
import FavoriteButton from '@/components/app/FavoriteButton.vue'
import PagedList from '@/components/app/PagedList.vue'
import ReviewerLink from '@/components/app/ReviewerLink.vue'
import VisitorNote from '@/components/app/VisitorNote.vue'
import { RatingDisplay } from '@/components/ui/rating-display'
import { Button } from '@/components/ui/button'
import { ConfirmDelete } from '@/components/ui/confirm-delete'
import { EmptyState } from '@/components/ui/empty-state'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { useLoad } from '@/composables/useLoad'
import { usePagedList } from '@/composables/usePagedList'
import { usePageTitle } from '@/composables/usePageTitle'
import { listsApi, recipesApi, restaurantsApi } from '@/lib/api'
import { forgetOwnLists, listCountText, listFailureText } from '@/lib/lists'
import { useMapConfig } from '@/lib/mapConfig'
import { currentUser } from '@/lib/session'

const props = defineProps<{ listId: string }>()
const router = useRouter()

const list = useLoad<SavedList>(() => listsApi.getList({ listId: props.listId }))

// The menu shows each section with the highest overall first. limit is the
// maximum of the API. See decision 00006.
const limit = 100
const restaurants = usePagedList<Restaurant>(async (cursor) => {
  const page = await restaurantsApi.listRestaurants({ listEq: props.listId, limit, cursor, sort: '-overall' })
  return { items: page.items, nextCursor: page.nextCursor }
})
const recipes = usePagedList<RecipeSummary>(async (cursor) => {
  const page = await recipesApi.listRecipes({ listEq: props.listId, limit, cursor, sort: '-overall' })
  return { items: page.items, nextCursor: page.nextCursor }
})

watch(() => props.listId, () => void list.load(), { immediate: true })
watch(
  () => props.listId,
  () => {
    void restaurants.reload()
    void recipes.reload()
  },
  { immediate: true },
)

usePageTitle(() => (list.data.value ? `${list.data.value.name} by ${list.data.value.creator.name}` : ''))

const isOwner = computed(() => !!list.data.value && currentUser.value?.id === list.data.value.creator.id)
const isFavorites = computed(() => list.data.value?.kind === 'favorites')
const mapConfig = useMapConfig()
const listQuery = computed(() => `?list=${encodeURIComponent(props.listId)}`)
const restaurantCount = computed(() => list.data.value?.restaurantCount ?? 0)
const recipeCount = computed(() => list.data.value?.recipeCount ?? 0)

function restaurantMeta(r: Restaurant): string {
  return [r.location.address, ...tagList(r.tags)].join(' · ')
}
function recipeMeta(r: RecipeSummary): string {
  return [...(r.totalTime ? [formatMinutes(r.totalTime)] : []), ...tagList(r.tags)].join(' · ')
}

// The API gives 400 for a malformed ID. The link then has a typo.
const failedStatus = computed(() => {
  const status = list.failed.value?.status ?? null
  return status === 400 ? 404 : status
})

const renaming = ref(false)
const name = ref('')
const renameError = ref('')
const saving = ref(false)
const renameButton = ref<InstanceType<typeof Button> | null>(null)

async function startRename() {
  name.value = list.data.value?.name ?? ''
  renameError.value = ''
  renaming.value = true
  await nextTick()
  document.getElementById('list-name')?.focus()
}

async function stopRename() {
  renaming.value = false
  await nextTick()
  ;(renameButton.value?.$el as HTMLElement | undefined)?.focus()
}

async function rename() {
  if (saving.value || name.value.trim() === '') {
    return
  }
  saving.value = true
  renameError.value = ''
  try {
    const updated = await listsApi.updateList({ listId: props.listId, savedListPatch: { name: name.value.trim() } })
    forgetOwnLists()
    list.data.value = updated
    await stopRename()
  } catch (e) {
    renameError.value = await listFailureText(e, 'The list was not renamed. Try again in a moment.')
  } finally {
    saving.value = false
  }
}

async function remove() {
  const data = list.data.value
  try {
    await listsApi.deleteList({ listId: props.listId })
    forgetOwnLists()
    toast.success('List deleted', { description: data?.name })
    void router.push(data ? `/reviewers/${data.creator.id}` : '/restaurants')
  } catch (e) {
    toast.error('List not deleted', { description: await listFailureText(e, 'Try again in a moment.') })
  }
}

function drop(restaurantId: string) {
  restaurants.items.value = restaurants.items.value.filter((r) => r.id !== restaurantId)
  if (list.data.value) {
    list.data.value = { ...list.data.value, restaurantCount: Math.max(0, list.data.value.restaurantCount - 1) }
  }
}

function dropRecipe(recipeId: string) {
  recipes.items.value = recipes.items.value.filter((r) => r.id !== recipeId)
  if (list.data.value) {
    list.data.value = { ...list.data.value, recipeCount: Math.max(0, list.data.value.recipeCount - 1) }
  }
}

async function removeRecipe(recipe: RecipeSummary) {
  try {
    await listsApi.removeListRecipe({ listRef: props.listId, recipeId: recipe.id })
    dropRecipe(recipe.id)
  } catch (e) {
    toast.error('Recipe not removed', { description: await listFailureText(e, 'Try again in a moment.') })
  }
}

function recipeFavoriteChanged(recipe: RecipeSummary, favorite: boolean) {
  if (isOwner.value && isFavorites.value && !favorite) {
    dropRecipe(recipe.id)
  }
}

async function removeRestaurant(restaurant: Restaurant) {
  try {
    await listsApi.removeListRestaurant({ listRef: props.listId, restaurantId: restaurant.id })
    drop(restaurant.id)
  } catch (e) {
    toast.error('Restaurant not removed', { description: await listFailureText(e, 'Try again in a moment.') })
  }
}

// The heart of a card changes the favorites list of the signed in user.
function favoriteChanged(restaurant: Restaurant, favorite: boolean) {
  if (isOwner.value && isFavorites.value && !favorite) {
    drop(restaurant.id)
  }
}
</script>

<template>
  <ErrorPage v-if="list.failed.value" :status="failedStatus" @retry="list.load" />
  <main v-else id="main" class="mx-auto flex max-w-2xl flex-col gap-5 px-4 py-6 sm:px-6 md:py-10">
    <div v-if="list.loading.value" aria-busy="true" aria-label="Loading list" class="flex flex-col gap-4">
      <Skeleton class="h-10 w-1/2" />
      <Skeleton class="h-6 w-40" />
    </div>
    <template v-else-if="list.data.value">
      <nav v-if="mapConfig && restaurantCount" aria-label="View" class="bg-card border-input flex w-fit self-center rounded-full border p-[3px]">
        <span aria-current="page" class="bg-primary text-primary-foreground inline-flex h-9 items-center gap-1.5 rounded-full px-4 text-sm font-semibold">
          <ListIcon class="size-4" aria-hidden="true" />Menu
        </span>
        <RouterLink :to="`/restaurants${listQuery}&view=map`" class="inline-flex h-9 items-center gap-1.5 rounded-full px-4 text-sm font-semibold">
          <MapIcon class="size-4" aria-hidden="true" />Map
        </RouterLink>
      </nav>
      <article class="bg-card rounded-xl border p-2">
        <div class="flex flex-col gap-6 rounded-md border px-5 pt-8 pb-7">
          <header class="flex flex-col items-center gap-2 text-center">
            <p class="text-muted-foreground text-sm font-semibold tracking-[0.08em] [font-variant-caps:all-small-caps]">
              A list by <ReviewerLink :creator="list.data.value.creator" class="text-foreground underline underline-offset-4" />
            </p>
            <h1 class="flex min-w-0 items-center gap-3 font-serif text-[44px] leading-[46px] font-semibold tracking-[-0.01em] wrap-anywhere">
              <HeartIcon v-if="isFavorites" class="text-primary size-8 shrink-0" aria-hidden="true" />
              {{ list.data.value.name }}
            </h1>
            <span class="flex items-center gap-2.5" aria-hidden="true">
              <span class="border-input w-12 border-t" /><span class="bg-primary size-2 rounded-full" /><span class="border-input w-12 border-t" />
            </span>
            <p class="text-muted-foreground text-[13px]">
              {{ restaurantCount || recipeCount ? listCountText(list.data.value) : 'No restaurants or recipes yet' }} · Updated
              <time :datetime="list.data.value.updated" :title="formatFullDate(list.data.value.updated)">{{ formatDate(list.data.value.updated) }}</time>
            </p>
            <div v-if="isOwner && !isFavorites" class="flex flex-wrap justify-center gap-2">
              <Button ref="renameButton" type="button" variant="outline" size="sm" :aria-expanded="renaming" @click="renaming ? stopRename() : startRename()">
                <PencilIcon aria-hidden="true" />Rename list
              </Button>
              <ConfirmDelete
                label="Delete list"
                title="Delete this list?"
                :description="`${list.data.value.name} will be deleted permanently. Its restaurants and recipes stay.`"
                @confirm="remove"
              />
            </div>
            <form v-if="renaming" class="flex w-full max-w-md flex-wrap justify-center gap-2" @submit.prevent="rename">
              <label for="list-name" class="sr-only">Name of the list</label>
              <Input id="list-name" v-model="name" class="min-w-40 flex-1" maxlength="255" autocomplete="off" :aria-invalid="!!renameError" />
              <Button type="submit" :disabled="saving">Save</Button>
              <Button type="button" variant="ghost" @click="stopRename">Cancel</Button>
              <p v-if="renameError" class="text-destructive w-full text-sm" role="alert">{{ renameError }}</p>
            </form>
          </header>

          <section v-if="restaurantCount || !recipeCount" aria-labelledby="list-restaurants-heading" class="flex flex-col gap-4">
            <h2 id="list-restaurants-heading" class="flex items-center gap-3">
              <span class="flex-1 border-t" aria-hidden="true" /><span class="kicker">Restaurants</span><span class="flex-1 border-t" aria-hidden="true" />
            </h2>
            <RouterLink v-if="restaurantCount" :to="`/restaurants${listQuery}`" class="text-primary inline-flex w-fit items-center gap-1.5 self-center text-sm font-semibold underline-offset-4 hover:underline">
              <SlidersHorizontalIcon class="size-4" aria-hidden="true" />Sort and filter<span class="sr-only"> the restaurants</span>
            </RouterLink>
            <PagedList :list="restaurants" noun="restaurants" list-class="gap-4" skeleton-class="h-12 rounded-md" :skeleton-count="3" unfiltered>
              <template #item="{ item }">
                <div class="relative flex items-center gap-2">
                  <div class="flex min-w-0 flex-1 flex-col gap-0.5">
                    <div class="flex items-baseline">
                      <h3 class="font-serif text-[19px] leading-6 font-semibold wrap-anywhere">
                        <RouterLink :to="`/restaurants/${item.id}`" class="rounded-sm after:absolute after:inset-0">{{ item.name }}</RouterLink>
                      </h3>
                      <span class="mx-2 min-w-4 flex-1 border-b-2 border-dotted" aria-hidden="true" />
                      <RatingDisplay v-if="item.rating" :value="item.rating.overall" :max="10" label="Overall" />
                    </div>
                    <p class="text-muted-foreground text-[13px] leading-[18px]">{{ restaurantMeta(item) }}</p>
                  </div>
                  <FavoriteButton :restaurant-id="item.id" :name="item.name" :favorite="item.favorite" class="relative z-10 size-9 shrink-0" @change="favoriteChanged(item, $event)" />
                  <Button
                    v-if="isOwner && !isFavorites"
                    type="button"
                    variant="outline"
                    size="sm"
                    class="relative z-10"
                    :aria-label="`Remove ${item.name} from the list`"
                    @click="removeRestaurant(item)"
                  >
                    Remove
                  </Button>
                </div>
              </template>
              <template #empty>
                <EmptyState
                  :icon="isFavorites ? HeartIcon : ListIcon"
                  title="No restaurants yet"
                  :description="
                    isOwner
                      ? isFavorites
                        ? 'Select the heart of a restaurant to add it here.'
                        : 'Open a restaurant and select Save to list to add it here.'
                      : `${list.data.value.creator.name} has not added a restaurant yet.`
                  "
                >
                  <template v-if="isOwner" #action>
                    <Button as-child><RouterLink to="/restaurants">Go to restaurants</RouterLink></Button>
                  </template>
                </EmptyState>
              </template>
            </PagedList>
          </section>

          <section v-if="recipeCount" aria-labelledby="list-recipes-heading" class="flex flex-col gap-4">
            <h2 id="list-recipes-heading" class="flex items-center gap-3">
              <span class="flex-1 border-t" aria-hidden="true" /><span class="kicker">Recipes</span><span class="flex-1 border-t" aria-hidden="true" />
            </h2>
            <RouterLink :to="`/recipes${listQuery}`" class="text-primary inline-flex w-fit items-center gap-1.5 self-center text-sm font-semibold underline-offset-4 hover:underline">
              <SlidersHorizontalIcon class="size-4" aria-hidden="true" />Sort and filter<span class="sr-only"> the recipes</span>
            </RouterLink>
            <PagedList :list="recipes" noun="recipes" list-class="gap-4" skeleton-class="h-12 rounded-md" :skeleton-count="3" unfiltered>
              <template #item="{ item }">
                <div class="relative flex items-center gap-2">
                  <div class="flex min-w-0 flex-1 flex-col gap-0.5">
                    <div class="flex items-baseline">
                      <h3 class="font-serif text-[19px] leading-6 font-semibold wrap-anywhere">
                        <RouterLink :to="`/recipes/${item.id}`" class="rounded-sm after:absolute after:inset-0">{{ item.name }}</RouterLink>
                      </h3>
                      <span class="mx-2 min-w-4 flex-1 border-b-2 border-dotted" aria-hidden="true" />
                      <RatingDisplay v-if="item.rating" :value="item.rating.overall" :max="10" label="Overall" />
                    </div>
                    <p class="text-muted-foreground text-[13px] leading-[18px]">{{ recipeMeta(item) }}</p>
                  </div>
                  <FavoriteButton :recipe-id="item.id" :name="item.name" :favorite="item.favorite" class="relative z-10 size-9 shrink-0" @change="recipeFavoriteChanged(item, $event)" />
                  <Button
                    v-if="isOwner && !isFavorites"
                    type="button"
                    variant="outline"
                    size="sm"
                    class="relative z-10"
                    :aria-label="`Remove ${item.name} from the list`"
                    @click="removeRecipe(item)"
                  >
                    Remove
                  </Button>
                </div>
              </template>
              <template #empty>
                <EmptyState
                  :icon="isFavorites ? HeartIcon : ListIcon"
                  title="No recipes yet"
                  :description="
                    isOwner
                      ? isFavorites
                        ? 'Select the heart of a recipe to add it here.'
                        : 'Open a recipe and select Save to list to add it here.'
                      : `${list.data.value.creator.name} has not added a recipe yet.`
                  "
                />
              </template>
            </PagedList>
          </section>
          <p class="text-muted-foreground text-center font-serif text-sm italic">Each score is out of 10. A higher score is better.</p>
        </div>
      </article>
      <VisitorNote :name="list.data.value.creator.name" />
    </template>
  </main>
</template>
