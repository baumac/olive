<script setup lang="ts">
import { CircleAlertIcon, MessageSquareIcon } from 'lucide-vue-next'
import { computed, watch } from 'vue'
import EntryCard from '@/components/app/EntryCard.vue'
import KindSwitch from '@/components/app/KindSwitch.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { LoadMore } from '@/components/ui/load-more'
import { Skeleton } from '@/components/ui/skeleton'
import { type EntryFetch, useEntries } from '@/composables/useEntries'
import { useKindQuery } from '@/composables/useKindQuery'
import { usePageTitle } from '@/composables/usePageTitle'
import { recipesApi, restaurantsApi } from '@/lib/api'
import { groupByWritten } from '@/lib/entries'
import { formatCount } from '@/lib/format'
import { currentUser } from '@/lib/session'

usePageTitle('On the table')

const limit = 20
const kind = useKindQuery()

// The table lists the reviews by the time they were written. A late review
// of an old dinner is then at the top. See decision 00009.
const restaurantReviews: EntryFetch = async (cursor) => {
  const page = await restaurantsApi.listAllRestaurantReviews({ limit, cursor, sort: '-created' })
  return { items: page.items, nextCursor: page.nextCursor }
}
const recipeReviews: EntryFetch = async (cursor) => {
  const page = await recipesApi.listAllRecipeReviews({ limit, cursor, sort: '-created' })
  return { items: page.items, nextCursor: page.nextCursor }
}

const entries = useEntries(
  () => (kind.value === 'restaurant' ? [restaurantReviews] : kind.value === 'recipe' ? [recipeReviews] : [restaurantReviews, recipeReviews]),
  () => 'created',
)
watch(kind, () => void entries.reload(), { immediate: true })

const groups = computed(() => groupByWritten(entries.items.value))
const today = new Intl.DateTimeFormat('en-GB', { weekday: 'long', day: 'numeric', month: 'long' }).format(new Date())

const status = computed(() => {
  if (entries.loading.value) {
    return 'Loading entries'
  }
  const count = formatCount(entries.items.value.length, 'entry', 'entries')
  return entries.hasMore.value ? `${count} shown, more to load` : count
})
</script>

<template>
  <main id="main" class="mx-auto flex max-w-3xl flex-col gap-5 px-4 py-6 sm:px-6 md:py-10">
    <header class="flex flex-col gap-1.5">
      <p class="text-muted-foreground text-sm font-semibold [font-variant-caps:all-small-caps] tracking-[0.08em]">{{ today }}</p>
      <h1 class="font-serif text-[34px] leading-10 font-semibold tracking-[-0.01em] md:text-[46px] md:leading-[50px]">On the table</h1>
      <p class="text-muted-foreground text-[15px] leading-[22px] md:text-[17px]">What everyone ate and cooked lately.</p>
    </header>
    <p v-if="!currentUser" class="text-muted-foreground max-w-2xl">
      These are the latest visits and cooks of a small group of friends and family. Each score goes from 1 to 10, and a higher
      score is always better.
      <RouterLink to="/sign-up" class="text-foreground font-medium underline underline-offset-4">Sign up</RouterLink>
      to add your own.
    </p>
    <KindSwitch :kind="kind" label="Show" :labels="{ all: 'All', restaurant: 'Ate out', recipe: 'Cooked' }" />
    <p role="status" class="sr-only">{{ status }}</p>

    <Alert v-if="entries.failed.value" variant="destructive">
      <CircleAlertIcon aria-hidden="true" />
      <AlertTitle>Entries did not load</AlertTitle>
      <AlertDescription class="flex flex-wrap items-center gap-3">
        Check your connection and try again.
        <Button variant="outline" size="sm" @click="entries.reload">Try again</Button>
      </AlertDescription>
    </Alert>
    <ul v-else-if="entries.loading.value" aria-busy="true" aria-label="Loading entries" class="flex flex-col gap-5">
      <li v-for="n in 3" :key="n"><Skeleton class="h-72 rounded-2xl" /></li>
    </ul>
    <EmptyState
      v-else-if="entries.items.value.length === 0"
      :icon="MessageSquareIcon"
      title="Nothing on the table yet"
      description="The first visit or cook shows here."
    >
      <template v-if="currentUser" #action>
        <Button as-child><RouterLink to="/reviews/new">Add review</RouterLink></Button>
      </template>
    </EmptyState>
    <template v-else>
      <section v-for="g in groups" :key="g.key" class="flex flex-col gap-5" :aria-labelledby="`group-${g.key}`">
        <h2 :id="`group-${g.key}`" class="kicker mt-3">{{ g.label }}</h2>
        <ul class="flex flex-col gap-5">
          <li v-for="e in g.entries" :key="e.id"><EntryCard :review="e" /></li>
        </ul>
      </section>
      <Alert v-if="entries.moreFailed.value" variant="destructive">
        <CircleAlertIcon aria-hidden="true" />
        <AlertTitle>More entries did not load</AlertTitle>
        <AlertDescription>Check your connection and try again.</AlertDescription>
      </Alert>
      <LoadMore :cursor="entries.hasMore.value ? 'more' : null" :loading="entries.loadingMore.value" label="Load more entries" @load="entries.loadMore" />
    </template>
  </main>
</template>
