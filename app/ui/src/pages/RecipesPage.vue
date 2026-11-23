<script setup lang="ts">
import type { RecipeSummary } from '@reviews/sdk'
import { ChefHatIcon, PlusIcon } from 'lucide-vue-next'
import { computed, watch } from 'vue'
import PagedList from '@/components/app/PagedList.vue'
import RecipeCard from '@/components/app/RecipeCard.vue'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { FilterBar } from '@/components/ui/filter-bar'
import { useListFilter } from '@/composables/useListFilter'
import { searchRelevance, useListQuery } from '@/composables/useListQuery'
import { usePagedList } from '@/composables/usePagedList'
import { usePageTitle } from '@/composables/usePageTitle'
import { recipesApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { oldRecipeSortOptions, recipeFilterDefs, recipeFilterParams, recipeSortOptions } from '@/lib/recipeFilters'
import { currentUser } from '@/lib/session'

usePageTitle('Recipes')

const limit = 20

const { def: listFilter, favoritesId } = useListFilter('recipes')
const filterDefs = computed(() => (listFilter.value ? [...recipeFilterDefs, listFilter.value] : recipeFilterDefs))

const { filters, sort, active, sortOptions } = useListQuery(filterDefs, recipeSortOptions, '-overall', searchRelevance, [], oldRecipeSortOptions)
// filterKey changes only with the values. A load of the lists of the user
// gives new defs. filters then gives a new object with the same values.
const filterKey = computed(() => JSON.stringify(filters.value))
const listId = computed(() => (typeof filters.value.list === 'string' ? filters.value.list : undefined))

const list = usePagedList<RecipeSummary>(async (cursor) => {
  const page = await recipesApi.listRecipes({ limit, cursor, sort: sort.value, ...recipeFilterParams(filters.value) })
  return { items: page.items, nextCursor: page.nextCursor }
})

watch([filterKey, sort], () => void list.reload(), { immediate: true })

// A card that leaves the favorites leaves a view of the favorites list.
function favoriteChanged(recipe: RecipeSummary, favorite: boolean) {
  if (!favorite && listId.value && listId.value === favoritesId.value) {
    list.items.value = list.items.value.filter((r) => r.id !== recipe.id)
  }
}

const status = computed(() => {
  if (list.loading.value) {
    return 'Loading recipes'
  }
  const count = formatCount(list.items.value.length, 'recipe')
  return list.cursor.value ? `${count} shown, more to load` : count
})
</script>

<template>
  <main id="main" class="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-8 sm:px-6">
    <h1 class="sr-only">Recipes</h1>
    <p v-if="!currentUser && !active" class="text-muted-foreground max-w-2xl text-lg">
      Find a recipe by its scores for taste, ease, instructions, and value.
      <RouterLink to="/sign-up" class="text-foreground font-medium underline underline-offset-4">Sign up</RouterLink>
      to add a recipe or to write a review.
    </p>

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
          <RouterLink to="/recipes/new"><PlusIcon aria-hidden="true" />Add recipe</RouterLink>
        </Button>
      </template>
    </FilterBar>

    <PagedList :list="list" noun="recipes" list-class="gap-2.5 lg:grid-cols-2" skeleton-class="h-[86px] rounded-xl" :skeleton-count="6">
      <template #item="{ item }">
        <RecipeCard :recipe="item" @favorite="favoriteChanged(item, $event)" />
      </template>
      <template #empty>
        <EmptyState
          v-if="active"
          :icon="ChefHatIcon"
          title="No recipes match these filters"
          description="Remove a filter or clear all filters."
        >
          <template #action>
            <Button variant="outline" @click="filters = {}">Clear filters</Button>
          </template>
        </EmptyState>
        <EmptyState v-else :icon="ChefHatIcon" title="No recipes yet" description="Add the first recipe, then write a review of it.">
          <template #action>
            <Button as-child><RouterLink to="/recipes/new">Add recipe</RouterLink></Button>
          </template>
        </EmptyState>
      </template>
    </PagedList>
  </main>
</template>
