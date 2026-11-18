<script setup lang="ts">
import type { Recipe, Restaurant } from '@reviews/sdk'
import { watch } from 'vue'
import { usePagedList } from '@/composables/usePagedList'
import { type Parent, type ReviewKind, kinds } from '@/lib/reviewKinds'
import PagedList from './PagedList.vue'
import RecipeCard from './RecipeCard.vue'
import RestaurantCard from './RestaurantCard.vue'

// ReviewerAdded lists the restaurants or the recipes that a user added.
const props = defineProps<{ creator: string; kind: ReviewKind }>()

// The list is below the reviews. A short page keeps it small.
const limit = 6

const list = usePagedList<Parent>(async (cursor) => {
  const page = await kinds[props.kind].listAdded(props.creator, { limit, cursor, sort: '-created' })
  return { items: page.items, nextCursor: page.nextCursor }
})

watch(() => props.creator, () => void list.reload(), { immediate: true })
</script>

<template>
  <PagedList
    :list="list"
    :noun="kinds[kind].plural"
    list-class="lg:grid-cols-2 xl:grid-cols-3"
    skeleton-class="h-36 rounded-xl"
    :skeleton-count="3"
    unfiltered
  >
    <template #item="{ item }">
      <RecipeCard v-if="kind === 'recipe'" :recipe="item as Recipe" heading="h3" />
      <RestaurantCard v-else :restaurant="item as Restaurant" heading="h3" />
    </template>
    <!-- The profile has a count above 0. A delete after that empties the list. -->
    <template #empty>
      <p class="text-muted-foreground text-sm">These {{ kinds[kind].plural }} are not available now.</p>
    </template>
  </PagedList>
</template>
