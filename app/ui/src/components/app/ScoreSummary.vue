<script setup lang="ts">
import type { RecipeRating, RestaurantRating } from '@reviews/sdk'
import { computed } from 'vue'
import { RatingDisplay } from '@/components/ui/rating-display'
import { type AnyReview, type ReviewKind, kinds, scoreOf } from '@/lib/reviewKinds'

// The default slot adds a line under "out of 10", such as the review count.
// A rating does not tell its kind. Each caller gives it.
const props = defineProps<{ scores: AnyReview | RestaurantRating | RecipeRating; kind: ReviewKind }>()

const max = 10
const fields = computed(() => kinds[props.kind].scoreFields)
</script>

<template>
  <div class="bg-card flex w-full max-w-md items-center gap-5 rounded-2xl border px-5 py-[18px]">
    <p class="flex shrink-0 flex-col items-center gap-0.5">
      <RatingDisplay :value="scores.overall" :max="max" label="Overall" size="lg" />
      <span class="text-muted-foreground text-[13px] leading-[18px]">out of {{ max }}</span>
      <span v-if="$slots.default" class="text-muted-foreground text-[13px] leading-[18px]"><slot /></span>
    </p>
    <dl class="flex min-w-0 flex-1 flex-col gap-1.5 text-[15px]">
      <div v-for="f in fields" :key="f.name" class="flex items-baseline">
        <dt>{{ f.label }}</dt>
        <span class="mx-2 min-w-3 flex-1 border-b-2 border-dotted" aria-hidden="true" />
        <dd><RatingDisplay :value="scoreOf(scores, f)" :max="f.max" :label="f.label" /></dd>
      </div>
    </dl>
  </div>
</template>
