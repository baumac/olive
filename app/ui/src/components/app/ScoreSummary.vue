<script setup lang="ts">
import type { RestaurantRating, Review } from '@reviews/sdk'
import { RatingDisplay } from '@/components/ui/rating-display'
import { scoreFields } from '@/lib/reviews'

// The default slot adds text after the scale, such as the review count.
defineProps<{ scores: Review | RestaurantRating }>()

const max = 10
</script>

<template>
  <div class="grid gap-2">
    <p class="flex flex-wrap items-baseline gap-x-2">
      <RatingDisplay :value="scores.overall" :max="max" label="Overall" size="lg" />
      <span class="text-muted-foreground text-sm">out of {{ max }}<template v-if="$slots.default"> · <slot /></template></span>
    </p>
    <ul class="flex flex-wrap gap-x-4 gap-y-1">
      <li v-for="f in scoreFields" :key="f.name">
        <RatingDisplay :value="scores[f.name]" :max="f.max" :label="f.label" show-label />
      </li>
    </ul>
  </div>
</template>
