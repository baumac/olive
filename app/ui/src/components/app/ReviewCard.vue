<script setup lang="ts">
import { computed, ref } from 'vue'
import { RatingDisplay } from '@/components/ui/rating-display'
import { isEdited } from '@/lib/format'
import { type AnyReview, kindOf, kinds, parentNameOf, parentPathOf, reviewPathOf, scoreOf } from '@/lib/reviewKinds'
import ReviewDate from './ReviewDate.vue'
import ReviewerLink from './ReviewerLink.vue'
import ReviewPhotos from './ReviewPhotos.vue'

const props = defineProps<{
  review: AnyReview
  // showRestaurant names the restaurant or the recipe. Lists of many
  // restaurants or recipes need it.
  showRestaurant?: boolean
  // showCreator names the creator with a link to the reviewer page. A list
  // of one reviewer does not need it.
  showCreator?: boolean
}>()

const scoresOpen = ref(false)
const fields = computed(() => kinds[kindOf(props.review)].scoreFields)
const edited = computed(() => isEdited(props.review.created, props.review.updated))
</script>

<template>
  <article class="grid gap-2 py-5">
    <div class="flex items-baseline gap-3">
      <RatingDisplay :value="review.overall" :max="10" label="Overall" size="md" class="w-8 shrink-0" />
      <h3 class="min-w-0 leading-snug font-semibold break-words">
        <RouterLink :to="reviewPathOf(review)" class="rounded-sm hover:underline">
          {{ review.title }}
        </RouterLink>
      </h3>
    </div>
    <div class="grid gap-2 pl-11">
      <p v-if="showRestaurant" class="text-muted-foreground text-sm">
        <RouterLink :to="parentPathOf(review)" class="hover:underline">{{ parentNameOf(review) }}</RouterLink>
      </p>
      <p v-if="review.feedback" class="line-clamp-3 whitespace-pre-line">{{ review.feedback }}</p>
      <ReviewPhotos :review="review" />
      <p class="text-muted-foreground text-sm">
        <template v-if="showCreator"><ReviewerLink :creator="review.creator" /> · </template>
        <ReviewDate :review="review" /><template v-if="edited"> · edited</template> ·
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
        <li v-for="f in fields" :key="f.name">
          <RatingDisplay :value="scoreOf(review, f)" :max="f.max" :label="f.label" show-label />
        </li>
      </ul>
    </div>
  </article>
</template>
