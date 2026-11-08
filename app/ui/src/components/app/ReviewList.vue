<script setup lang="ts">
import type { Restaurant, Review } from '@reviews/sdk'
import type { FilterDef, Option } from '@/components/ui/filter-bar'
import { MessageSquareIcon } from 'lucide-vue-next'
import { computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { FilterBar } from '@/components/ui/filter-bar'
import { useListQuery } from '@/composables/useListQuery'
import { usePagedList } from '@/composables/usePagedList'
import { restaurantsApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { kindLabels } from '@/lib/reviews'
import PagedList from './PagedList.vue'
import ReviewCard from './ReviewCard.vue'

const props = defineProps<{
  restaurant: Restaurant
  writePath: string
}>()

const limit = 20

function scoreOptions(max: number): Option[] {
  return Array.from({ length: max }, (_, i) => max - i).map((n) => ({ value: String(n), label: `${n} or more` }))
}

const joc = 'Matches only Joc de cartes reviews.'
const defs: FilterDef[] = [
  { key: 'tags', label: 'Tags', type: 'tags', help: 'Shows reviews with at least one of the tags.', chip: (v) => `Tag: ${v}` },
  {
    key: 'kind',
    label: 'Kind',
    type: 'select',
    options: Object.entries(kindLabels).map(([value, label]) => ({ value, label })),
    chip: (v) => `Kind: ${kindLabels[v] ?? v}`,
  },
  { key: 'rating', label: 'Rating', type: 'select', options: scoreOptions(5), help: 'Matches only Basic reviews.', chip: (v) => `Rating: ${v} or more` },
  { key: 'food', label: 'Food', type: 'select', options: scoreOptions(10), help: joc, chip: (v) => `Food: ${v} or more` },
  { key: 'service', label: 'Service', type: 'select', options: scoreOptions(10), help: joc, chip: (v) => `Service: ${v} or more` },
  { key: 'space', label: 'Space', type: 'select', options: scoreOptions(10), help: joc, chip: (v) => `Space: ${v} or more` },
  { key: 'price', label: 'Price', type: 'select', options: scoreOptions(10), help: joc, chip: (v) => `Price: ${v} or more` },
]

const sortOptions: Option[] = [
  { value: '-created', label: 'Newest' },
  { value: 'created', label: 'Oldest' },
  { value: '-rating', label: 'Highest rating' },
  { value: 'rating', label: 'Lowest rating' },
  { value: '-food', label: 'Best food' },
  { value: '-service', label: 'Best service' },
  { value: '-space', label: 'Best space' },
  { value: '-price', label: 'Best value for money' },
  { value: 'title', label: 'Title, A to Z' },
]

const { filters, sort, active } = useListQuery(defs, sortOptions, '-created')

function atLeast(key: string) {
  const v = Number(filters.value[key])
  return Number.isInteger(v) && v > 0 ? { gte: v } : undefined
}

const list = usePagedList<Review>(async (cursor) => {
  const f = filters.value
  const tags = f.tags as string[] | undefined
  const page = await restaurantsApi.listReviews({
    restaurantId: props.restaurant.id,
    limit,
    cursor,
    sort: sort.value,
    tagsHasAny: tags?.length ? tags : undefined,
    kindIn: typeof f.kind === 'string' ? [f.kind] : undefined,
    rating: atLeast('rating'),
    food: atLeast('food'),
    service: atLeast('service'),
    space: atLeast('space'),
    price: atLeast('price'),
  })
  return { items: page.items, nextCursor: page.nextCursor }
})

watch([filters, sort, () => props.restaurant.id], () => void list.reload(), { immediate: true })

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
      <template #item="{ item }"><ReviewCard :review="item" /></template>
      <template #empty>
        <EmptyState
          v-if="active"
          :icon="MessageSquareIcon"
          title="No reviews match these filters"
          description="Remove a filter or clear all filters."
        >
          <template #action>
            <Button variant="outline" @click="filters = {}">Clear filters</Button>
          </template>
        </EmptyState>
        <EmptyState
          v-else
          :icon="MessageSquareIcon"
          title="No reviews yet"
          :description="`Be the first to review ${restaurant.name}.`"
        >
          <template #action>
            <Button as-child><RouterLink :to="writePath">Write a review</RouterLink></Button>
          </template>
        </EmptyState>
      </template>
    </PagedList>
  </div>
</template>
