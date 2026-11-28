<script setup lang="ts">
import type { PantryItem } from '@reviews/sdk'
import { PlusIcon } from 'lucide-vue-next'
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PagedList from '@/components/app/PagedList.vue'
import PantryItemCard from '@/components/app/PantryItemCard.vue'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { FilterBar } from '@/components/ui/filter-bar'
import { useListFilter } from '@/composables/useListFilter'
import { firstQueryValue, searchRelevance, useListQuery } from '@/composables/useListQuery'
import { usePagedList } from '@/composables/usePagedList'
import { usePageTitle } from '@/composables/usePageTitle'
import { pantryApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import {
  categories,
  isCategory,
  oldPantrySortOptions,
  pantryFilterDefs,
  pantryFilterParams,
  pantryIcon,
  pantrySortOptions,
} from '@/lib/pantry'
import { currentUser } from '@/lib/session'

usePageTitle('Pantry')

const limit = 20
const route = useRoute()
const router = useRouter()

const { def: listFilter, favoritesId } = useListFilter('pantry items')
const filterDefs = computed(() => (listFilter.value ? [...pantryFilterDefs, listFilter.value] : pantryFilterDefs))

// The category chips are not in FilterBar. A change of the filters keeps the
// category. See decision 00012.
const { filters, sort, active: filtered, sortOptions } = useListQuery(filterDefs, pantrySortOptions, '-overall', searchRelevance, ['category'], oldPantrySortOptions)
const category = computed(() => {
  const value = firstQueryValue(route.query.category)
  return isCategory(value) ? value : undefined
})
const active = computed(() => filtered.value || category.value !== undefined)
// filterKey changes only with the values. A load of the lists of the user
// gives new defs. filters then gives a new object with the same values.
const filterKey = computed(() => JSON.stringify([filters.value, category.value]))
const listId = computed(() => (typeof filters.value.list === 'string' ? filters.value.list : undefined))

const list = usePagedList<PantryItem>(async (cursor) => {
  const page = await pantryApi.listPantryItems({ limit, cursor, sort: sort.value, ...pantryFilterParams(filters.value, category.value) })
  return { items: page.items, nextCursor: page.nextCursor }
})

watch([filterKey, sort], () => void list.reload(), { immediate: true })

// categoryPath keeps the filters and the sort of the URL.
function categoryPath(value: string | undefined): string {
  const query = { ...route.query }
  delete query.category
  return router.resolve({ path: route.path, query: value ? { ...query, category: value } : query }).fullPath
}

function clearAll() {
  const sortQuery = firstQueryValue(route.query.sort)
  void router.replace({ path: route.path, query: sortQuery ? { sort: sortQuery } : {} })
}

// A card that leaves the favorites leaves a view of the favorites list.
function favoriteChanged(item: PantryItem, favorite: boolean) {
  if (!favorite && listId.value && listId.value === favoritesId.value) {
    list.items.value = list.items.value.filter((p) => p.id !== item.id)
  }
}

const status = computed(() => {
  if (list.loading.value) {
    return 'Loading pantry items'
  }
  const count = formatCount(list.items.value.length, 'pantry item')
  return list.cursor.value ? `${count} shown, more to load` : count
})
</script>

<template>
  <main id="main" class="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-8 sm:px-6">
    <h1 class="sr-only">Pantry</h1>
    <p v-if="!currentUser && !active" class="text-muted-foreground max-w-2xl text-lg">
      Find a wine, a beer, a cheese, or a chocolate by its scores for taste, aroma, texture, and value.
      <RouterLink to="/sign-up" class="text-foreground font-medium underline underline-offset-4">Sign up</RouterLink>
      to add a pantry item or to write a tasting.
    </p>

    <nav aria-label="Category" class="flex flex-wrap gap-2">
      <RouterLink
        :to="categoryPath(undefined)"
        replace
        :aria-current="category === undefined ? 'page' : undefined"
        :class="[
          'inline-flex h-11 items-center rounded-full border px-4 text-sm font-semibold',
          category === undefined ? 'bg-primary border-primary text-primary-foreground' : 'bg-card border-input hover:bg-accent hover:text-accent-foreground',
        ]"
      >
        All
      </RouterLink>
      <RouterLink
        v-for="c in categories"
        :key="c.value"
        :to="categoryPath(c.value)"
        replace
        :aria-current="category === c.value ? 'page' : undefined"
        :class="[
          'inline-flex h-11 items-center gap-1.5 rounded-full border px-4 text-sm font-semibold',
          category === c.value ? 'bg-primary border-primary text-primary-foreground' : 'bg-card border-input hover:bg-accent hover:text-accent-foreground',
        ]"
      >
        <component :is="c.icon" class="size-4" aria-hidden="true" />{{ c.label }}
      </RouterLink>
    </nav>

    <FilterBar
      v-model:filters="filters"
      v-model:sort="sort"
      :defs="filterDefs"
      :sort-options="sortOptions"
      sort-label="Sort by"
      :status="status"
    >
      <template #end>
        <Button as-child>
          <RouterLink to="/pantry/new"><PlusIcon aria-hidden="true" />Add pantry item</RouterLink>
        </Button>
      </template>
    </FilterBar>

    <PagedList :list="list" noun="pantry items" list-class="gap-2.5 lg:grid-cols-2" skeleton-class="h-[86px] rounded-xl" :skeleton-count="6">
      <template #item="{ item }">
        <PantryItemCard :item="item" @favorite="favoriteChanged(item, $event)" />
      </template>
      <template #empty>
        <EmptyState
          v-if="active"
          :icon="pantryIcon"
          title="No pantry items match these filters"
          description="Remove a filter or clear all filters."
        >
          <template #action>
            <Button variant="outline" @click="clearAll">Clear filters</Button>
          </template>
        </EmptyState>
        <EmptyState v-else :icon="pantryIcon" title="The pantry is empty" description="Add the first pantry item, then write a tasting of it.">
          <template #action>
            <Button as-child><RouterLink to="/pantry/new">Add pantry item</RouterLink></Button>
          </template>
        </EmptyState>
      </template>
    </PagedList>
  </main>
</template>
