<script setup lang="ts">
import type { Restaurant } from '@reviews/sdk'
import type { FilterDef, Option } from '@/components/ui/filter-bar'
import { PlusIcon, UtensilsIcon } from 'lucide-vue-next'
import { computed, watch } from 'vue'
import PagedList from '@/components/app/PagedList.vue'
import RestaurantCard from '@/components/app/RestaurantCard.vue'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { FilterBar } from '@/components/ui/filter-bar'
import { useListQuery } from '@/composables/useListQuery'
import { usePagedList } from '@/composables/usePagedList'
import { usePageTitle } from '@/composables/usePageTitle'
import { restaurantsApi } from '@/lib/api'
import { formatCount } from '@/lib/format'

usePageTitle('Restaurants')

const limit = 20

const defs: FilterDef[] = [
  { key: 'tags', label: 'Tags', type: 'tags', help: 'Shows restaurants with at least one of the tags.', chip: (v) => `Tag: ${v}` },
]

const sortOptions: Option[] = [
  { value: '-created', label: 'Newest' },
  { value: 'created', label: 'Oldest' },
  { value: 'name', label: 'Name, A to Z' },
  { value: '-name', label: 'Name, Z to A' },
  { value: '-updated', label: 'Recently updated' },
]

const { filters, sort, active } = useListQuery(defs, sortOptions, '-created')

const list = usePagedList<Restaurant>(async (cursor) => {
  const tags = filters.value.tags as string[] | undefined
  const page = await restaurantsApi.listRestaurants({
    limit,
    cursor,
    sort: sort.value,
    tagsHasAny: tags?.length ? tags : undefined,
  })
  return { items: page.items, nextCursor: page.nextCursor }
})

watch([filters, sort], () => void list.reload(), { immediate: true })

const status = computed(() => {
  if (list.loading.value) {
    return 'Loading restaurants'
  }
  const count = formatCount(list.items.value.length, 'restaurant')
  return list.cursor.value ? `${count} shown, more to load` : count
})
</script>

<template>
  <main id="main" class="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-8 sm:px-6">
    <div class="flex flex-wrap items-center justify-between gap-4">
      <h1 class="text-4xl font-extrabold tracking-tight">Restaurants</h1>
      <Button as-child>
        <RouterLink to="/restaurants/new"><PlusIcon aria-hidden="true" />Add restaurant</RouterLink>
      </Button>
    </div>

    <FilterBar
      v-model:filters="filters"
      v-model:sort="sort"
      :defs="defs"
      :sort-options="sortOptions"
      sort-label="Sort by"
      :status="status"
    />

    <PagedList
      :list="list"
      noun="restaurants"
      list-class="lg:grid-cols-2 xl:grid-cols-3"
      skeleton-class="h-40 rounded-xl"
      :skeleton-count="6"
    >
      <template #item="{ item }"><RestaurantCard :restaurant="item" /></template>
      <template #empty>
        <EmptyState
          v-if="active"
          :icon="UtensilsIcon"
          title="No restaurants match these filters"
          description="Remove a filter or clear all filters."
        >
          <template #action>
            <Button variant="outline" @click="filters = {}">Clear filters</Button>
          </template>
        </EmptyState>
        <EmptyState
          v-else
          :icon="UtensilsIcon"
          title="No restaurants yet"
          description="Add the first restaurant, then write a review of it."
        >
          <template #action>
            <Button as-child><RouterLink to="/restaurants/new">Add restaurant</RouterLink></Button>
          </template>
        </EmptyState>
      </template>
    </PagedList>
  </main>
</template>
