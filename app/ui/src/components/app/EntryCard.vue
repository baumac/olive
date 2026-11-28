<script setup lang="ts">
import { ChefHatIcon, UtensilsIcon } from 'lucide-vue-next'
import { pantryIcon } from '@/lib/pantry'
import { computed } from 'vue'
import { CoverPhoto } from '@/components/ui/cover-photo'
import { RatingDisplay } from '@/components/ui/rating-display'
import { ScoreStamp } from '@/components/ui/score-stamp'
import { type AnyReview, kindOf, parentNameOf, reviewPathOf } from '@/lib/reviewKinds'
import ReviewDate from './ReviewDate.vue'

// EntryCard shows one review as one experience. The photo variant is for a
// review with photos. See decision 00009.
const props = withDefaults(defineProps<{ review: AnyReview; variant?: 'photo' | 'row'; showCreator?: boolean }>(), {
  variant: undefined,
  showCreator: true,
})

const photo = computed(() => props.review.photos[0] ?? null)
const shape = computed(() => props.variant ?? (photo.value ? 'photo' : 'row'))
const icons = { restaurant: UtensilsIcon, recipe: ChefHatIcon, pantryItem: pantryIcon }
const icon = computed(() => icons[kindOf(props.review)])
const subject = computed(() => parentNameOf(props.review))
</script>

<template>
  <article v-if="shape === 'photo'" class="relative flex flex-col gap-3">
    <div class="relative">
      <CoverPhoto :photo="photo" size="hero" class="h-56 rounded-2xl sm:aspect-auto aspect-auto" />
      <ScoreStamp :value="review.overall" class="absolute right-3 bottom-3" />
    </div>
    <div class="flex flex-col gap-1 px-0.5">
      <p class="kicker flex items-center gap-1.5" aria-hidden="true">
        <component :is="icon" class="size-4" />{{ subject }}
      </p>
      <h3 class="font-serif text-[22px] leading-7 font-[550] wrap-anywhere">
        <!-- The after element makes the full card the link. -->
        <RouterLink :to="reviewPathOf(review)" class="rounded-sm after:absolute after:inset-0 after:rounded-2xl">
          <span class="sr-only">{{ subject }}: </span>{{ review.title }}
        </RouterLink>
      </h3>
      <p class="text-muted-foreground text-[13px] leading-[18px]">
        <template v-if="showCreator">{{ review.creator.name }} · </template><ReviewDate :review="review" />
      </p>
    </div>
  </article>
  <article v-else class="bg-card relative flex items-center gap-3.5 rounded-xl border p-3 pr-3.5">
    <div class="bg-accent text-primary flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-lg">
      <CoverPhoto v-if="photo" :photo="photo" size="card" class="aspect-square size-16" />
      <component :is="icon" v-else class="size-6" aria-hidden="true" />
    </div>
    <div class="flex min-w-0 flex-1 flex-col gap-0.5">
      <p class="kicker text-sm" aria-hidden="true">{{ subject }}</p>
      <h3 class="font-serif text-lg leading-[23px] font-semibold wrap-anywhere">
        <RouterLink :to="reviewPathOf(review)" class="rounded-sm after:absolute after:inset-0 after:rounded-xl">
          <span class="sr-only">{{ subject }}: </span>{{ review.title }}
        </RouterLink>
      </h3>
      <p class="text-muted-foreground text-[13px] leading-[18px]">
        <template v-if="showCreator">{{ review.creator.name }} · </template><ReviewDate :review="review" />
      </p>
    </div>
    <RatingDisplay :value="review.overall" :max="10" label="Overall" size="md" />
  </article>
</template>
