<script setup lang="ts">
import type { Review } from '@reviews/sdk'
import { ref } from 'vue'
import { RatingDisplay } from '@/components/ui/rating-display'
import { scoreFields } from '@/lib/reviews'
import DateText from './DateText.vue'
import ReviewerLink from './ReviewerLink.vue'
import ReviewPhotos from './ReviewPhotos.vue'

defineProps<{
  review: Review
  // showRestaurant names the restaurant. Lists of many restaurants need it.
  showRestaurant?: boolean
  // showCreator names the creator with a link to the reviewer page. A list
  // of one reviewer does not need it.
  showCreator?: boolean
}>()

const scoresOpen = ref(false)
</script>

<template>
  <article class="grid gap-2 py-5">
    <div class="flex items-baseline gap-3">
      <RatingDisplay :value="review.overall" :max="10" label="Overall" size="md" class="w-8 shrink-0" />
      <h3 class="min-w-0 leading-snug font-semibold break-words">
        <RouterLink :to="`/restaurants/${review.restaurant}/reviews/${review.id}`" class="rounded-sm hover:underline">
          {{ review.title }}
        </RouterLink>
      </h3>
    </div>
    <div class="grid gap-2 pl-11">
      <p v-if="showRestaurant" class="text-muted-foreground text-sm">
        <RouterLink :to="`/restaurants/${review.restaurant}`" class="hover:underline">{{ review.restaurantName }}</RouterLink>
      </p>
      <p v-if="review.feedback" class="line-clamp-3 whitespace-pre-line">{{ review.feedback }}</p>
      <ReviewPhotos :review="review" />
      <p class="text-muted-foreground text-sm">
        <template v-if="showCreator"><ReviewerLink :creator="review.creator" /> · </template>
        <DateText :created="review.created" :updated="review.updated" /> ·
        <button
          type="button"
          class="rounded-sm underline underline-offset-4"
          :aria-expanded="scoresOpen"
          @click="scoresOpen = !scoresOpen"
        >
          {{ scoresOpen ? 'Hide scores' : 'Show scores' }}<span class="sr-only"> of {{ review.title }}</span>
        </button>
      </p>
      <ul v-if="scoresOpen" class="flex flex-wrap gap-x-4 gap-y-1" :aria-label="`Scores of ${review.title}`">
        <li v-for="f in scoreFields" :key="f.name">
          <RatingDisplay :value="review[f.name]" :max="f.max" :label="f.label" show-label />
        </li>
      </ul>
    </div>
  </article>
</template>
