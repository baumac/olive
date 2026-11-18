<script setup lang="ts">
import type { RecipeRating, RestaurantRating } from '@reviews/sdk'
import { computed } from 'vue'
import { RatingDisplay } from '@/components/ui/rating-display'
import { type AnyReview, type ReviewKind, kinds, scoreOf } from '@/lib/reviewKinds'

// The default slot adds text after the scale, such as the review count. A
// rating does not tell its kind, so each caller gives it.
const props = defineProps<{ scores: AnyReview | RestaurantRating | RecipeRating; kind: ReviewKind }>()

const max = 10
const fields = computed(() => kinds[props.kind].scoreFields)
</script>

<template>
  <div class="grid gap-2">
    <p class="flex flex-wrap items-baseline gap-x-2">
      <RatingDisplay :value="scores.overall" :max="max" label="Overall" size="lg" />
      <span class="text-muted-foreground text-sm">out of {{ max }}<template v-if="$slots.default"> · <slot /></template></span>
    </p>
    <ul class="flex flex-wrap gap-x-4 gap-y-1">
      <li v-for="f in fields" :key="f.name">
        <RatingDisplay :value="scoreOf(scores, f)" :max="f.max" :label="f.label" show-label />
      </li>
    </ul>
  </div>
</template>
