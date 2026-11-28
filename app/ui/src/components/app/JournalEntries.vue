<script setup lang="ts">
import { CircleAlertIcon, MessageSquareIcon } from 'lucide-vue-next'
import { computed, watch } from 'vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { LoadMore } from '@/components/ui/load-more'
import { Skeleton } from '@/components/ui/skeleton'
import { type EntryFetch, loadMoreAndFocus, useEntries } from '@/composables/useEntries'
import { groupByMonth } from '@/lib/entries'
import { type AnyReview, kinds, reviewKinds } from '@/lib/reviewKinds'
import EntryTile from './EntryTile.vue'

// JournalEntries shows the reviews of all kinds by the month of their date.
// With all kinds, the order is always the newest date first. See decision
// 00006.
const props = defineProps<{ reviewerId: string; reviewerName?: string }>()

const limit = 20
const fetchers = (): EntryFetch[] =>
  reviewKinds.map((k) => async (cursor) => {
    const page = await kinds[k].listReviewerReviews(props.reviewerId, { limit, cursor, sort: '-date' })
    return { items: page.items, nextCursor: page.nextCursor }
  })

const entries = useEntries(fetchers, () => 'date')
watch(() => props.reviewerId, () => void entries.reload(), { immediate: true })

const groups = computed(() => groupByMonth(entries.items.value))
const entryId = (review: AnyReview) => `entries-${review.id}`
</script>

<template>
  <Alert v-if="entries.failed.value" variant="destructive">
    <CircleAlertIcon aria-hidden="true" />
    <AlertTitle>Reviews did not load</AlertTitle>
    <AlertDescription class="flex flex-wrap items-center gap-3">
      Check your connection and try again.
      <Button variant="outline" size="sm" @click="entries.reload">Try again</Button>
    </AlertDescription>
  </Alert>
  <ul v-else-if="entries.loading.value" aria-busy="true" aria-label="Loading reviews" class="grid grid-cols-3 gap-2.5 sm:grid-cols-4 lg:grid-cols-6">
    <li v-for="n in 6" :key="n"><Skeleton class="aspect-square rounded-xl" /></li>
  </ul>
  <EmptyState
    v-else-if="entries.items.value.length === 0"
    :icon="MessageSquareIcon"
    :title="reviewerName ? 'No reviews yet' : 'You have no reviews yet'"
    :description="reviewerName ? `${reviewerName} has not written a review yet.` : 'Add your first visit, cook, or tasting.'"
  >
    <template v-if="!reviewerName" #action>
      <Button as-child><RouterLink to="/reviews/new">Add review</RouterLink></Button>
    </template>
  </EmptyState>
  <div v-else class="flex flex-col gap-6">
    <section v-for="g in groups" :key="g.key" class="flex flex-col gap-2.5" :aria-label="g.label">
      <h3 class="kicker">{{ g.label }}</h3>
      <ul class="grid grid-cols-3 gap-2.5 sm:grid-cols-4 lg:grid-cols-6">
        <li v-for="e in g.entries" :id="entryId(e)" :key="e.id" tabindex="-1" class="rounded-xl"><EntryTile :review="e" /></li>
      </ul>
    </section>
    <Alert v-if="entries.moreFailed.value" variant="destructive">
      <CircleAlertIcon aria-hidden="true" />
      <AlertTitle>More reviews did not load</AlertTitle>
      <AlertDescription>Check your connection and try again.</AlertDescription>
    </Alert>
    <LoadMore :cursor="entries.hasMore.value ? 'more' : null" :loading="entries.loadingMore.value" label="Load more reviews" @load="loadMoreAndFocus(entries, entryId)" />
  </div>
</template>
