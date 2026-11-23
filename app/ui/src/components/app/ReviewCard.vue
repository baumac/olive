<script setup lang="ts">
import { computed } from 'vue'
import { RatingDisplay } from '@/components/ui/rating-display'
import { isEdited } from '@/lib/format'
import { type AnyReview, parentNameOf, parentPathOf, reviewPathOf } from '@/lib/reviewKinds'
import ReviewDate from './ReviewDate.vue'
import ReviewerLink from './ReviewerLink.vue'

// ReviewCard is one visit or one cook in the timeline of a subject. See
// decision 00005.
const props = defineProps<{
  review: AnyReview
  // showRestaurant names the restaurant or the recipe. Lists of many
  // restaurants or recipes need it.
  showRestaurant?: boolean
  // showCreator names the creator with a link to the journal.
  showCreator?: boolean
  // own marks a review of the signed in user with "You".
  own?: boolean
}>()

const edited = computed(() => isEdited(props.review.created, props.review.updated))
</script>

<template>
  <article class="relative grid gap-1">
    <span class="bg-background border-primary absolute top-1 -left-[29px] size-3 rounded-full border-2" aria-hidden="true" />
    <p class="text-muted-foreground text-sm font-semibold tracking-[0.08em] [font-variant-caps:all-small-caps]">
      <ReviewDate :review="review" /><template v-if="edited"> · edited</template>
    </p>
    <div class="flex items-baseline gap-3">
      <h3 class="min-w-0 flex-1 font-serif text-[19px] leading-6 font-semibold wrap-anywhere">
        <RouterLink :to="reviewPathOf(review)" class="rounded-sm hover:underline">{{ review.title }}</RouterLink>
      </h3>
      <RatingDisplay :value="review.overall" :max="10" label="Overall" size="md" class="shrink-0" />
    </div>
    <p v-if="showRestaurant" class="text-muted-foreground text-sm">
      <RouterLink :to="parentPathOf(review)" class="hover:underline">{{ parentNameOf(review) }}</RouterLink>
    </p>
    <p v-if="review.feedback" class="text-muted-foreground line-clamp-2 text-sm leading-[21px] whitespace-pre-line">{{ review.feedback }}</p>
    <p v-if="own" class="mt-0.5"><span class="bg-accent text-accent-foreground rounded-full px-2 py-0.5 text-xs font-semibold">You</span></p>
    <p v-else-if="showCreator" class="text-sm"><ReviewerLink :creator="review.creator" /></p>
  </article>
</template>
