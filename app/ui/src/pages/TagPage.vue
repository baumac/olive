<script setup lang="ts">
import type { RecipeSummary, Restaurant, TagSummary } from '@reviews/sdk'
import { SlidersHorizontalIcon, TagIcon } from 'lucide-vue-next'
import { computed, watch } from 'vue'
import ErrorPage from '@/components/app/ErrorPage.vue'
import KindSwitch from '@/components/app/KindSwitch.vue'
import PagedList from '@/components/app/PagedList.vue'
import RecipeCard from '@/components/app/RecipeCard.vue'
import RestaurantCard from '@/components/app/RestaurantCard.vue'
import { EmptyState } from '@/components/ui/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { useKindQuery } from '@/composables/useKindQuery'
import { useLoad } from '@/composables/useLoad'
import { usePagedList } from '@/composables/usePagedList'
import { usePageTitle } from '@/composables/usePageTitle'
import { recipesApi, restaurantsApi, tagsApi } from '@/lib/api'
import { tagCountText } from '@/lib/tags'

const props = defineProps<{ name: string }>()
const kind = useKindQuery()

const tag = useLoad<TagSummary>(() => tagsApi.getTag({ tagName: props.name }))

// Each kind has its own section. The overall sort of the API also counts the
// reviewers. The UI then cannot merge the two lists. See decision 00011.
const restaurants = usePagedList<Restaurant>(async (cursor) => {
  const page = await restaurantsApi.listRestaurants({ tagsHas: props.name, cursor, sort: '-overall' })
  return { items: page.items, nextCursor: page.nextCursor }
})
const recipes = usePagedList<RecipeSummary>(async (cursor) => {
  const page = await recipesApi.listRecipes({ tagsHas: props.name, cursor, sort: '-overall' })
  return { items: page.items, nextCursor: page.nextCursor }
})

const showRestaurants = computed(() => kind.value !== 'recipe')
const showRecipes = computed(() => kind.value !== 'restaurant')

watch(() => props.name, () => void tag.load(), { immediate: true })

// A list loads when its section shows, one time for each tag. A switch back
// to the kind keeps the loaded pages.
function loadWhenShown(list: { reload: () => Promise<void> }, shown: () => boolean) {
  let loadedFor: string | null = null
  watch(
    [() => props.name, shown],
    ([name, show]) => {
      if (show && loadedFor !== name) {
        loadedFor = name
        void list.reload()
      }
    },
    { immediate: true },
  )
}
loadWhenShown(restaurants, () => showRestaurants.value)
loadWhenShown(recipes, () => showRecipes.value)

usePageTitle(() => (tag.data.value ? `Tag ${tag.data.value.name}` : ''))
const filterQuery = computed(() => `?tags=${encodeURIComponent(props.name)}`)

// The API gives 400 for a name that is not a tag. The link then has a typo.
const failedStatus = computed(() => {
  const status = tag.failed.value?.status ?? null
  return status === 400 ? 404 : status
})
</script>

<template>
  <ErrorPage v-if="tag.failed.value" :status="failedStatus" @retry="tag.load" />
  <main v-else id="main" class="mx-auto flex max-w-2xl flex-col gap-6 px-4 py-6 sm:px-6 md:py-10">
    <div v-if="tag.loading.value" aria-busy="true" aria-label="Loading tag" class="flex flex-col gap-4">
      <Skeleton class="h-10 w-1/2" />
      <Skeleton class="h-6 w-40" />
    </div>
    <template v-else-if="tag.data.value">
      <header class="flex flex-col gap-2">
        <p class="kicker flex items-center gap-1.5"><TagIcon class="size-4" aria-hidden="true" />Tag</p>
        <h1 class="font-serif text-[40px] leading-[44px] font-semibold tracking-[-0.01em] wrap-anywhere">{{ tag.data.value.name }}</h1>
        <p class="text-muted-foreground text-sm">{{ tagCountText(tag.data.value) }}</p>
      </header>
      <KindSwitch :kind="kind" label="Kind of the tagged things" />

      <section v-if="showRestaurants" aria-labelledby="tag-restaurants-heading" class="flex flex-col gap-3">
        <div class="flex flex-wrap items-baseline justify-between gap-2">
          <h2 id="tag-restaurants-heading" class="font-serif text-2xl font-semibold">Restaurants</h2>
          <RouterLink
            v-if="tag.data.value.restaurantCount"
            :to="`/restaurants${filterQuery}`"
            class="text-primary inline-flex min-h-11 items-center gap-1.5 text-sm font-semibold underline-offset-4 hover:underline"
          >
            <SlidersHorizontalIcon class="size-4" aria-hidden="true" />Sort and filter<span class="sr-only"> the restaurants</span>
          </RouterLink>
        </div>
        <PagedList :list="restaurants" noun="restaurants" list-class="gap-2.5" skeleton-class="h-[86px] rounded-xl" :skeleton-count="3" unfiltered>
          <template #item="{ item }">
            <RestaurantCard :restaurant="item" heading="h3" />
          </template>
          <template #empty>
            <EmptyState :icon="TagIcon" title="No restaurants with this tag" :description="`No restaurant has the tag ${tag.data.value.name} yet.`" />
          </template>
        </PagedList>
      </section>

      <section v-if="showRecipes" aria-labelledby="tag-recipes-heading" class="flex flex-col gap-3">
        <div class="flex flex-wrap items-baseline justify-between gap-2">
          <h2 id="tag-recipes-heading" class="font-serif text-2xl font-semibold">Recipes</h2>
          <RouterLink
            v-if="tag.data.value.recipeCount"
            :to="`/recipes${filterQuery}`"
            class="text-primary inline-flex min-h-11 items-center gap-1.5 text-sm font-semibold underline-offset-4 hover:underline"
          >
            <SlidersHorizontalIcon class="size-4" aria-hidden="true" />Sort and filter<span class="sr-only"> the recipes</span>
          </RouterLink>
        </div>
        <PagedList :list="recipes" noun="recipes" list-class="gap-2.5" skeleton-class="h-[86px] rounded-xl" :skeleton-count="3" unfiltered>
          <template #item="{ item }">
            <RecipeCard :recipe="item" heading="h3" />
          </template>
          <template #empty>
            <EmptyState :icon="TagIcon" title="No recipes with this tag" :description="`No recipe has the tag ${tag.data.value.name} yet.`" />
          </template>
        </PagedList>
      </section>
    </template>
  </main>
</template>
