<script setup lang="ts">
import type { Reviewer } from '@reviews/sdk'
import { computed, watch } from 'vue'
import ErrorPage from '@/components/app/ErrorPage.vue'
import KindSwitch from '@/components/app/KindSwitch.vue'
import ReviewerAdded from '@/components/app/ReviewerAdded.vue'
import ReviewerLists from '@/components/app/ReviewerLists.vue'
import ReviewList from '@/components/app/ReviewList.vue'
import { InitialsAvatar } from '@/components/ui/avatar'
import { RatingDisplay } from '@/components/ui/rating-display'
import { Skeleton } from '@/components/ui/skeleton'
import { useKindQuery } from '@/composables/useKindQuery'
import { useLoad } from '@/composables/useLoad'
import { usePageTitle } from '@/composables/usePageTitle'
import { reviewersApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { currentUser } from '@/lib/session'

const props = defineProps<{ reviewerId: string }>()

const reviewer = useLoad<Reviewer>(() => reviewersApi.getReviewer({ reviewerId: props.reviewerId }))
watch(() => props.reviewerId, () => void reviewer.load(), { immediate: true })

usePageTitle(() => reviewer.data.value?.name ?? '')

const isSelf = computed(() => currentUser.value?.id === props.reviewerId)
const kind = useKindQuery()

// summary reads after the average, such as "average of 12 restaurant
// reviews". Each kind has its own average. See decision 00004.
const summary = computed(() => {
  const r = reviewer.data.value
  if (!r) {
    return ''
  }
  const parts = [r.restaurantReviewCount ? `average of ${formatCount(r.restaurantReviewCount, 'restaurant review')}` : 'No restaurant reviews yet']
  if (r.restaurantCount) {
    parts.push(`${formatCount(r.restaurantCount, 'restaurant')} added`)
  }
  return parts.join(' · ')
})
const recipeSummary = computed(() => {
  const r = reviewer.data.value
  if (!r) {
    return ''
  }
  const parts = [r.recipeReviewCount ? `average of ${formatCount(r.recipeReviewCount, 'recipe review')}` : 'No recipe reviews yet']
  if (r.recipeCount) {
    parts.push(`${formatCount(r.recipeCount, 'recipe')} added`)
  }
  return parts.join(' · ')
})

// The API gives 400 for a malformed ID. The link then has a typo.
const failedStatus = computed(() => {
  const status = reviewer.failed.value?.status ?? null
  return status === 400 ? 404 : status
})
</script>

<template>
  <ErrorPage v-if="reviewer.failed.value" :status="failedStatus" @retry="reviewer.load" />
  <main v-else id="main" class="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-8 sm:px-6">
    <div v-if="reviewer.loading.value" aria-busy="true" aria-label="Loading reviewer" class="flex flex-col gap-4">
      <Skeleton class="h-10 w-1/2" />
      <Skeleton class="h-6 w-40" />
    </div>
    <template v-else-if="reviewer.data.value">
      <section class="flex items-center gap-4">
        <InitialsAvatar :name="reviewer.data.value.name" size="lg" />
        <div class="flex min-w-0 flex-col gap-1">
          <h1 class="text-4xl font-extrabold tracking-tight break-words">{{ reviewer.data.value.name }}</h1>
          <p class="flex flex-wrap items-baseline gap-x-2">
            <RatingDisplay
              v-if="reviewer.data.value.restaurantAverageOverall !== null"
              :value="reviewer.data.value.restaurantAverageOverall"
              :max="10"
              label="Average restaurant score"
              size="md"
            />
            <span class="text-muted-foreground text-sm">{{ summary }}</span>
          </p>
          <p v-if="reviewer.data.value.recipeReviewCount || reviewer.data.value.recipeCount" class="flex flex-wrap items-baseline gap-x-2">
            <RatingDisplay
              v-if="reviewer.data.value.recipeAverageOverall !== null"
              :value="reviewer.data.value.recipeAverageOverall"
              :max="10"
              label="Average recipe score"
              size="md"
            />
            <span class="text-muted-foreground text-sm">{{ recipeSummary }}</span>
          </p>
        </div>
      </section>
      <section class="flex flex-col gap-4" aria-labelledby="lists-heading">
        <h2 id="lists-heading" class="text-3xl font-semibold tracking-tight">Lists</h2>
        <ReviewerLists :reviewer-id="reviewerId" :own="isSelf" />
      </section>
      <!-- The switch selects the added items and the reviews of one kind. -->
      <KindSwitch :kind="kind" label="Restaurants or recipes" />
      <section v-if="kind === 'restaurant' && reviewer.data.value.restaurantCount" class="flex flex-col gap-6" aria-labelledby="restaurants-heading">
        <h2 id="restaurants-heading" class="text-3xl font-semibold tracking-tight">Restaurants added</h2>
        <ReviewerAdded :creator="reviewerId" kind="restaurant" />
      </section>
      <section v-if="kind === 'recipe' && reviewer.data.value.recipeCount" class="flex flex-col gap-6" aria-labelledby="recipes-heading">
        <h2 id="recipes-heading" class="text-3xl font-semibold tracking-tight">Recipes added</h2>
        <ReviewerAdded :creator="reviewerId" kind="recipe" />
      </section>
      <section class="flex flex-col gap-6" aria-labelledby="reviews-heading">
        <h2 id="reviews-heading" class="text-3xl font-semibold tracking-tight">Reviews</h2>
        <ReviewList :key="kind" :creator="reviewerId" :kind="kind" :reviewer-name="isSelf ? undefined : reviewer.data.value.name" />
      </section>
    </template>
  </main>
</template>
