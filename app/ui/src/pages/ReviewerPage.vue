<script setup lang="ts">
import type { Reviewer } from '@reviews/sdk'
import { computed, watch } from 'vue'
import ErrorPage from '@/components/app/ErrorPage.vue'
import JournalEntries from '@/components/app/JournalEntries.vue'
import JournalLists from '@/components/app/JournalLists.vue'
import KindSwitch from '@/components/app/KindSwitch.vue'
import ReviewerAdded from '@/components/app/ReviewerAdded.vue'
import ReviewList from '@/components/app/ReviewList.vue'
import VisitorNote from '@/components/app/VisitorNote.vue'
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

// counts reads like "12 visits · 4 cooks · 3 lists". See decision 00006.
const counts = computed(() => {
  const r = reviewer.data.value
  if (!r) {
    return ''
  }
  return [formatCount(r.restaurantReviewCount, 'visit'), formatCount(r.recipeReviewCount, 'cook'), formatCount(r.listCount, 'list')].join(' · ')
})

// The API gives 400 for a malformed ID. The link then has a typo.
const failedStatus = computed(() => {
  const status = reviewer.failed.value?.status ?? null
  return status === 400 ? 404 : status
})
</script>

<template>
  <ErrorPage v-if="reviewer.failed.value" :status="failedStatus" @retry="reviewer.load" />
  <main v-else id="main" class="mx-auto flex max-w-4xl flex-col gap-7 px-4 py-6 sm:px-6 md:py-10">
    <div v-if="reviewer.loading.value" aria-busy="true" aria-label="Loading reviewer" class="flex flex-col gap-4">
      <Skeleton class="h-10 w-1/2" />
      <Skeleton class="h-6 w-40" />
    </div>
    <template v-else-if="reviewer.data.value">
      <section class="flex items-center gap-4" aria-label="Reviewer">
        <InitialsAvatar :name="reviewer.data.value.name" size="lg" />
        <div class="flex min-w-0 flex-col gap-1">
          <h1 class="font-serif text-[34px] leading-10 font-semibold wrap-anywhere">{{ reviewer.data.value.name }}</h1>
          <p class="text-muted-foreground text-sm">{{ counts }}</p>
          <p
            v-if="reviewer.data.value.restaurantAverageOverall !== null || reviewer.data.value.recipeAverageOverall !== null"
            class="text-muted-foreground flex flex-wrap items-baseline gap-x-1.5 text-sm"
          >
            Average score
            <template v-if="reviewer.data.value.restaurantAverageOverall !== null">
              <RatingDisplay :value="reviewer.data.value.restaurantAverageOverall" :max="10" label="Average restaurant score" />
              for restaurants<template v-if="reviewer.data.value.recipeAverageOverall !== null">,</template>
            </template>
            <template v-if="reviewer.data.value.recipeAverageOverall !== null">
              <RatingDisplay :value="reviewer.data.value.recipeAverageOverall" :max="10" label="Average recipe score" />
              for recipes
            </template>
          </p>
        </div>
      </section>
      <JournalLists :reviewer-id="reviewerId" :own="isSelf" />
      <section v-if="reviewer.data.value.restaurantCount" class="flex flex-col gap-3" aria-labelledby="restaurants-heading">
        <h2 id="restaurants-heading" class="font-serif text-[21px] leading-[26px] font-semibold">Restaurants added</h2>
        <ReviewerAdded :creator="reviewerId" kind="restaurant" />
      </section>
      <section v-if="reviewer.data.value.recipeCount" class="flex flex-col gap-3" aria-labelledby="recipes-heading">
        <h2 id="recipes-heading" class="font-serif text-[21px] leading-[26px] font-semibold">Recipes added</h2>
        <ReviewerAdded :creator="reviewerId" kind="recipe" />
      </section>
      <section class="flex flex-col gap-4" aria-labelledby="reviews-heading">
        <h2 id="reviews-heading" class="font-serif text-2xl leading-[30px] font-semibold">{{ isSelf ? 'Your journal' : 'Journal' }}</h2>
        <KindSwitch :kind="kind" label="Restaurants or recipes" />
        <JournalEntries v-if="kind === 'all'" :reviewer-id="reviewerId" :reviewer-name="isSelf ? undefined : reviewer.data.value.name" />
        <ReviewList v-else :key="kind" :creator="reviewerId" :kind="kind" :reviewer-name="isSelf ? undefined : reviewer.data.value.name" />
      </section>
      <VisitorNote :name="reviewer.data.value.name" />
    </template>
  </main>
</template>
