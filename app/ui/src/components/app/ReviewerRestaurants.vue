<script setup lang="ts">
import type { Restaurant } from '@reviews/sdk'
import { watch } from 'vue'
import { usePagedList } from '@/composables/usePagedList'
import { restaurantsApi } from '@/lib/api'
import PagedList from './PagedList.vue'
import RestaurantCard from './RestaurantCard.vue'

const props = defineProps<{ creator: string }>()

// The list is below the reviews. A short page keeps it small.
const limit = 6

const list = usePagedList<Restaurant>(async (cursor) => {
  const page = await restaurantsApi.listRestaurants({ limit, cursor, sort: '-created', creatorEq: props.creator })
  return { items: page.items, nextCursor: page.nextCursor }
})

watch(() => props.creator, () => void list.reload(), { immediate: true })
</script>

<template>
  <PagedList
    :list="list"
    noun="restaurants"
    list-class="lg:grid-cols-2 xl:grid-cols-3"
    skeleton-class="h-36 rounded-xl"
    :skeleton-count="3"
    unfiltered
  >
    <template #item="{ item }"><RestaurantCard :restaurant="item" heading="h3" /></template>
    <!-- The profile has a count above 0. A delete after that empties the list. -->
    <template #empty><p class="text-muted-foreground text-sm">These restaurants are not available now.</p></template>
  </PagedList>
</template>
