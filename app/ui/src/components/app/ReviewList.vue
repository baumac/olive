<script setup lang="ts">
import { MessageSquareIcon } from 'lucide-vue-next'
import { computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { FilterBar } from '@/components/ui/filter-bar'
import { searchRelevance, useListQuery } from '@/composables/useListQuery'
import { usePagedList } from '@/composables/usePagedList'
import { formatCount } from '@/lib/format'
import { oldReviewSortOptionsOf, reviewDefaultSort, reviewFilterDefsOf, reviewFilterParamsOf, reviewSortOptionsOf } from '@/lib/reviewFilters'
import { groupByMonth } from '@/lib/entries'
import { type AnyReview, type Parent, type ReviewKind, kinds } from '@/lib/reviewKinds'
import { currentUser } from '@/lib/session'
import EntryTile from './EntryTile.vue'
import PagedList from './PagedList.vue'
import ReviewCard from './ReviewCard.vue'

// Give a parent or a creator. A parent is a restaurant or a recipe of the
// kind. Its reviews show as a timeline of visits or cooks. Without a parent,
// the list has the reviews of the creator of the kind as journal tiles.
const props = withDefaults(
  defineProps<{
    parent?: Parent
    kind?: ReviewKind
    writePath?: string
    creator?: string
    // reviewerName is the name of a creator who is not the signed in user.
    reviewerName?: string
  }>(),
  { parent: undefined, kind: 'restaurant', writePath: undefined, creator: undefined, reviewerName: undefined },
)

const info = computed(() => kinds[props.kind])
const tiles = computed(() => !props.parent)
const filterDefs = computed(() => reviewFilterDefsOf(props.kind))

const limit = 20
// A shorter list shows without the search, the filters, and the sort.
const toolbarMinimum = 10
const defaultSort = reviewDefaultSort

// The kind query key of the reviewer pages stays after a filter change.
const { filters, sort, active, sortOptions } = useListQuery(
  filterDefs,
  () => reviewSortOptionsOf(props.kind),
  defaultSort,
  searchRelevance,
  ['kind'],
  oldReviewSortOptionsOf(props.kind),
)

const list = usePagedList<AnyReview>(async (cursor) => {
  const params = { limit, cursor, sort: sort.value, ...reviewFilterParamsOf(props.kind, filters.value) }
  const page = props.parent
    ? await info.value.listReviews(props.parent.id, params)
    : await info.value.listReviewerReviews(props.creator ?? '', params)
  return { items: page.items, nextCursor: page.nextCursor }
})

watch([filters, sort, () => props.parent?.id, () => props.creator, () => props.kind], () => void list.reload(), { immediate: true })

defineExpose({ reload: list.reload })

// A list of one creator has no count before it loads. Without a filter, the
// loaded page has all reviews or it has a next page.
const total = computed(() => {
  if (props.parent) {
    return props.parent.reviewCount
  }
  return list.cursor.value ? toolbarMinimum : list.items.value.length
})
// A filter or a sort in the URL keeps the toolbar. The user can then remove it.
const showToolbar = computed(() => active.value || sort.value !== defaultSort || total.value >= toolbarMinimum)

// A sort by date groups the tiles by month. Another sort shows one grid.
function tileGroups(items: AnyReview[]) {
  return sort.value.replace(/^-/, '') === 'date' ? groupByMonth(items) : [{ key: 'all', label: '', entries: items }]
}

const status = computed(() => {
  if (list.loading.value) {
    return 'Loading reviews'
  }
  const count = formatCount(list.items.value.length, 'review')
  return list.cursor.value ? `${count} shown, more to load` : count
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <FilterBar
      v-if="showToolbar"
      v-model:filters="filters"
      v-model:sort="sort"
      :defs="filterDefs"
      :sort-options="sortOptions"
      sort-label="Sort by"
      :status="status"
    />
    <p v-else role="status" class="sr-only">{{ status }}</p>

    <PagedList
      :list="list"
      noun="reviews"
      :list-class="tiles ? undefined : 'ml-1.5 gap-6 border-l-2 pl-[22px]'"
      item-class="rounded-sm"
      :skeleton-class="tiles ? 'h-40 rounded-xl' : 'my-1 h-24 rounded-md'"
      :skeleton-count="3"
    >
      <template v-if="tiles" #items="{ items }">
        <div class="flex flex-col gap-6">
          <section v-for="g in tileGroups(items)" :key="g.key" class="flex flex-col gap-2.5" :aria-label="g.label || undefined">
            <h3 v-if="g.label" class="kicker">{{ g.label }}</h3>
            <ul class="grid grid-cols-3 gap-2.5 sm:grid-cols-4 lg:grid-cols-6">
              <li v-for="e in g.entries" :id="`reviews-${e.id}`" :key="e.id" tabindex="-1" class="rounded-xl"><EntryTile :review="e" /></li>
            </ul>
          </section>
        </div>
      </template>
      <template v-else #item="{ item }">
        <ReviewCard :review="item" :show-creator="true" :own="item.creator.id === currentUser?.id" />
      </template>
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
          v-else-if="parent"
          :icon="MessageSquareIcon"
          title="No reviews yet"
          :description="`Be the first to review ${parent.name}.`"
        >
          <template #action>
            <Button as-child><RouterLink :to="writePath ?? `${info.path}/${parent.id}/reviews/new`">Add a {{ info.occasion }}</RouterLink></Button>
          </template>
        </EmptyState>
        <EmptyState
          v-else-if="reviewerName"
          :icon="MessageSquareIcon"
          title="No reviews yet"
          :description="`${reviewerName} has not reviewed a ${info.noun} yet.`"
        />
        <EmptyState
          v-else
          :icon="MessageSquareIcon"
          title="You have no reviews yet"
          :description="`Add your first ${info.occasion}.`"
        >
          <template #action>
            <Button as-child><RouterLink to="/reviews/new">Add review</RouterLink></Button>
          </template>
        </EmptyState>
      </template>
    </PagedList>
  </div>
</template>
