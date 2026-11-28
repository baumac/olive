<script setup lang="ts">
import type { PantryItem, RecipeSummary, Restaurant, SavedList } from '@reviews/sdk'
import { ChevronRightIcon } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { CoverPhoto } from '@/components/ui/cover-photo'
import { Input } from '@/components/ui/input'
import { RatingDisplay } from '@/components/ui/rating-display'
import { Skeleton } from '@/components/ui/skeleton'
import { useLoad } from '@/composables/useLoad'
import { listsApi, pantryApi, recipesApi, restaurantsApi } from '@/lib/api'
import { forgetOwnLists, itemCountOf, listCountText, listFailureText } from '@/lib/lists'
import { pendingApproval } from '@/lib/session'

// JournalLists shows the favorites as cards and the other lists as rows.
// own is true on the journal of the signed in user. See decision 00006.
const props = defineProps<{ reviewerId: string; own: boolean }>()

// favoritesShown is the number of favorite cards in the row.
const favoritesShown = 6

const lists = useLoad<SavedList[]>(async () => (await listsApi.listReviewerLists({ reviewerId: props.reviewerId })).items)
watch(() => props.reviewerId, () => void lists.load(), { immediate: true })

const favorites = computed(() => lists.data.value?.find((l) => l.kind === 'favorites') ?? null)
const others = computed(() => lists.data.value?.filter((l) => l.kind !== 'favorites') ?? [])
const favoriteCount = computed(() => (favorites.value ? itemCountOf(favorites.value) : 0))

interface FavoriteItem {
  id: string
  path: string
  name: string
  cover: Restaurant['cover']
  overall: number | null
}
const favoriteItems = useLoad<FavoriteItem[]>(async () => {
  const list = favorites.value
  if (!list || favoriteCount.value === 0) {
    return []
  }
  const [restaurants, recipes, pantry] = await Promise.all([
    list.restaurantCount ? restaurantsApi.listRestaurants({ listEq: list.id, limit: favoritesShown, sort: '-overall' }) : null,
    list.recipeCount ? recipesApi.listRecipes({ listEq: list.id, limit: favoritesShown, sort: '-overall' }) : null,
    list.pantryItemCount ? pantryApi.listPantryItems({ listEq: list.id, limit: favoritesShown, sort: '-overall' }) : null,
  ])
  const items: FavoriteItem[] = [
    ...(restaurants?.items ?? []).map((r: Restaurant) => ({ id: r.id, path: `/restaurants/${r.id}`, name: r.name, cover: r.cover, overall: r.rating?.overall ?? null })),
    ...(recipes?.items ?? []).map((r: RecipeSummary) => ({ id: r.id, path: `/recipes/${r.id}`, name: r.name, cover: r.cover, overall: r.rating?.overall ?? null })),
    ...(pantry?.items ?? []).map((p: PantryItem) => ({ id: p.id, path: `/pantry/${p.id}`, name: p.name, cover: p.cover, overall: p.rating?.overall ?? null })),
  ]
  return items.sort((a, b) => (b.overall ?? -1) - (a.overall ?? -1)).slice(0, favoritesShown)
})
watch(favorites, () => void favoriteItems.load())

const name = ref('')
const creating = ref(false)
const error = ref('')

async function create() {
  if (creating.value || name.value.trim() === '') {
    return
  }
  creating.value = true
  error.value = ''
  try {
    const list = await listsApi.createList({ savedListCreate: { name: name.value.trim() } })
    forgetOwnLists()
    const [first, ...rest] = lists.data.value ?? []
    lists.data.value = first ? [first, list, ...rest] : [list]
    name.value = ''
  } catch (e) {
    error.value = await listFailureText(e, 'The list was not created. Try again in a moment.')
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <div v-if="lists.loading.value" aria-busy="true" aria-label="Loading lists" class="flex gap-3">
    <Skeleton class="h-36 w-36 rounded-xl" />
    <Skeleton class="h-36 w-36 rounded-xl" />
  </div>
  <p v-else-if="lists.failed.value" class="text-destructive text-sm" role="alert">
    The lists did not load.
    <button type="button" class="text-foreground font-medium underline underline-offset-4" @click="lists.load">Try again</button>
  </p>
  <template v-else>
    <section v-if="favorites && favoriteCount" class="flex flex-col gap-3" aria-labelledby="favorites-heading">
      <div class="flex items-center justify-between gap-3">
        <h2 id="favorites-heading" class="font-serif text-[21px] leading-[26px] font-semibold">Favorites</h2>
        <RouterLink :to="`/lists/${favorites.id}`" class="text-primary inline-flex min-h-11 items-center text-sm font-semibold underline-offset-4 hover:underline">
          See all {{ favoriteCount }}<span class="sr-only"> favorites</span>
        </RouterLink>
      </div>
      <ul class="-mr-4 flex gap-3 overflow-x-auto pr-4 pb-1">
        <li v-for="f in favoriteItems.data.value ?? []" :key="f.id" class="w-[148px] shrink-0">
          <RouterLink :to="f.path" class="flex flex-col gap-1.5 rounded-xl">
            <CoverPhoto :photo="f.cover" size="card" class="aspect-auto h-[110px] rounded-xl" />
            <span class="flex items-baseline justify-between gap-2">
              <span class="truncate text-sm font-semibold">{{ f.name }}</span>
              <RatingDisplay v-if="f.overall !== null" :value="f.overall" :max="10" label="Overall" />
            </span>
          </RouterLink>
        </li>
      </ul>
    </section>
    <section class="flex flex-col gap-2.5" aria-labelledby="lists-heading">
      <h2 id="lists-heading" class="font-serif text-[21px] leading-[26px] font-semibold">Lists</h2>
      <ul class="flex flex-col gap-2">
        <li v-for="list in [...(favorites ? [favorites] : []), ...others]" :key="list.id">
          <RouterLink
            :to="`/lists/${list.id}`"
            class="bg-card hover:border-ring flex min-h-[60px] items-center gap-3 rounded-xl border pr-3 pl-4"
          >
            <span class="flex min-w-0 flex-1 flex-col gap-0.5">
              <span class="font-serif text-[17px] font-semibold wrap-anywhere">{{ list.name }}</span>
              <span class="text-muted-foreground text-[13px]">{{ listCountText(list) }}</span>
            </span>
            <ChevronRightIcon class="size-5 shrink-0" aria-hidden="true" />
          </RouterLink>
        </li>
      </ul>
      <form v-if="own && !pendingApproval" class="flex max-w-md flex-wrap gap-2" @submit.prevent="create">
        <label for="new-list-name" class="sr-only">Name of a new list</label>
        <Input id="new-list-name" v-model="name" class="min-w-40 flex-1" placeholder="Name of a new list" maxlength="255" autocomplete="off" />
        <Button type="submit" variant="outline" :disabled="creating">New list</Button>
        <p v-if="error" class="text-destructive w-full text-sm" role="alert">{{ error }}</p>
      </form>
    </section>
  </template>
</template>
