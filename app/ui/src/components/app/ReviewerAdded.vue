<script setup lang="ts">
import type { PantryItem, RecipeSummary, Restaurant } from '@reviews/sdk'
import { watch } from 'vue'
import { usePagedList } from '@/composables/usePagedList'
import { type ParentSummary, type ReviewKind, kinds } from '@/lib/reviewKinds'
import PagedList from './PagedList.vue'
import PantryItemCard from './PantryItemCard.vue'
import RecipeCard from './RecipeCard.vue'
import RestaurantCard from './RestaurantCard.vue'

// ReviewerAdded lists the restaurants, the recipes, or the pantry items that
// a user added.
const props = defineProps<{ creator: string; kind: ReviewKind }>()

// The list is above the reviews. A short page keeps the reviews near the top.
const limit = 3

const list = usePagedList<ParentSummary>(async (cursor) => {
  const page = await kinds[props.kind].listAdded(props.creator, { limit, cursor, sort: '-created' })
  return { items: page.items, nextCursor: page.nextCursor }
})

watch(() => props.creator, () => void list.reload(), { immediate: true })
</script>

<template>
  <PagedList
    :list="list"
    :noun="kinds[kind].plural"
    list-class="gap-2.5 lg:grid-cols-2"
    skeleton-class="h-[86px] rounded-xl"
    :skeleton-count="3"
    unfiltered
  >
    <template #item="{ item }">
      <RecipeCard v-if="kind === 'recipe'" :recipe="item as RecipeSummary" heading="h3" />
      <PantryItemCard v-else-if="kind === 'pantryItem'" :item="item as PantryItem" heading="h3" />
      <RestaurantCard v-else :restaurant="item as Restaurant" heading="h3" />
    </template>
    <!-- The profile has a count above 0. A delete after that empties the list. -->
    <template #empty>
      <p class="text-muted-foreground text-sm">These {{ kinds[kind].plural }} are not available now.</p>
    </template>
  </PagedList>
</template>
