<script setup lang="ts">
import { computed } from 'vue'
import { CoverPhoto } from '@/components/ui/cover-photo'
import { RatingDisplay } from '@/components/ui/rating-display'
import { ScoreStamp } from '@/components/ui/score-stamp'
import { parseDay } from '@/lib/format'
import { type AnyReview, parentNameOf, reviewPathOf } from '@/lib/reviewKinds'
import ReviewDate from './ReviewDate.vue'

// EntryTile is one review in the month grid of a journal. See decision 00006.
const props = defineProps<{ review: AnyReview }>()

const photo = computed(() => props.review.photos[0] ?? null)
const day = computed(() => parseDay(props.review.date)?.getDate() ?? '')
</script>

<template>
  <article class="relative flex flex-col gap-1.5">
    <div class="bg-accent relative aspect-square overflow-hidden rounded-xl">
      <CoverPhoto v-if="photo" :photo="photo" size="card" class="aspect-square size-full" />
      <ScoreStamp v-if="photo" :value="review.overall" size="sm" class="absolute right-1.5 bottom-1.5" />
      <RatingDisplay v-else :value="review.overall" :max="10" label="Overall" size="md" class="absolute right-2.5 bottom-2" />
      <span class="bg-card absolute top-1.5 left-1.5 rounded-full px-2 py-0.5 text-xs font-bold tabular-nums" aria-hidden="true">{{ day }}</span>
    </div>
    <h4 class="truncate text-[13px] leading-[17px] font-semibold">
      <RouterLink :to="reviewPathOf(review)" class="rounded-sm after:absolute after:inset-0 after:rounded-xl">
        <!-- Chromium puts a space before a hidden span, so the hidden text repeats the name. -->
        <span class="sr-only">{{ parentNameOf(review) }}: {{ review.title }}, <ReviewDate :review="review" /></span>
        <span aria-hidden="true">{{ parentNameOf(review) }}</span>
      </RouterLink>
    </h4>
  </article>
</template>
