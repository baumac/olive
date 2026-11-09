<script setup lang="ts">
import type { Restaurant, Review } from '@reviews/sdk'
import type { FilterDef } from '@/components/ui/filter-bar'
import { MessageSquareIcon } from 'lucide-vue-next'
import { computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { FilterBar } from '@/components/ui/filter-bar'
import { searchRelevance, useListQuery } from '@/composables/useListQuery'
import { usePagedList } from '@/composables/usePagedList'
import { restaurantsApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { reviewFilterDefs, reviewFilterParams, reviewSortOptions } from '@/lib/reviewFilters'
import PagedList from './PagedList.vue'
import ReviewCard from './ReviewCard.vue'

// Without a restaurant, the list has the reviews of all restaurants.
const props = withDefaults(
  defineProps<{
    restaurant?: Restaurant
    writePath?: string
    defs?: FilterDef[]
    // mine is true when the creator filter selects the signed in user.
    mine?: boolean
  }>(),
  { restaurant: undefined, writePath: undefined, defs: () => reviewFilterDefs, mine: false },
)

const limit = 20

const { filters, sort, active, sortOptions } = useListQuery(() => props.defs, reviewSortOptions, '-created', searchRelevance)

const list = usePagedList<Review>(async (cursor) => {
  const params = { limit, cursor, sort: sort.value, ...reviewFilterParams(filters.value) }
  const page = props.restaurant
    ? await restaurantsApi.listReviews({ restaurantId: props.restaurant.id, ...params })
    : await restaurantsApi.listAllReviews(params)
  return { items: page.items, nextCursor: page.nextCursor }
})

watch([filters, sort, () => props.restaurant?.id], () => void list.reload(), { immediate: true })

defineExpose({ reload: list.reload })

const status = computed(() => {
  if (list.loading.value) {
    return 'Loading reviews'
  }
  const count = formatCount(list.items.value.length, 'review')
  return list.cursor.value ? `${count} shown, more to load` : count
})
</script>

<template>
  <div class="flex flex-col gap-6">
    <FilterBar
      v-model:filters="filters"
      v-model:sort="sort"
      :defs="defs"
      :sort-options="sortOptions"
      sort-label="Sort by"
      :status="status"
    />

    <PagedList :list="list" noun="reviews" skeleton-class="h-44 rounded-xl" :skeleton-count="3">
      <template #item="{ item }"><ReviewCard :review="item" :show-restaurant="!restaurant" /></template>
      <template #empty>
        <EmptyState
          v-if="mine && Object.keys(filters).length === 1"
          :icon="MessageSquareIcon"
          title="You have no reviews yet"
          description="Find a restaurant and write the first one."
        >
          <template #action>
            <Button as-child><RouterLink to="/restaurants">Go to restaurants</RouterLink></Button>
          </template>
        </EmptyState>
        <EmptyState
          v-else-if="active"
          :icon="MessageSquareIcon"
          title="No reviews match these filters"
          description="Remove a filter or clear all filters."
        >
          <template #action>
            <Button variant="outline" @click="filters = {}">Clear filters</Button>
          </template>
        </EmptyState>
        <EmptyState
          v-else-if="restaurant"
          :icon="MessageSquareIcon"
          title="No reviews yet"
          :description="`Be the first to review ${restaurant.name}.`"
        >
          <template #action>
            <Button as-child><RouterLink :to="writePath ?? `/restaurants/${restaurant.id}/reviews/new`">Write a review</RouterLink></Button>
          </template>
        </EmptyState>
        <EmptyState
          v-else
          :icon="MessageSquareIcon"
          title="No reviews yet"
          description="Find a restaurant and write the first review."
        >
          <template #action>
            <Button as-child><RouterLink to="/restaurants">Go to restaurants</RouterLink></Button>
          </template>
        </EmptyState>
      </template>
    </PagedList>
  </div>
</template>
