<script setup lang="ts">
import { CoverPhoto } from '@/components/ui/cover-photo'
import { RatingDisplay } from '@/components/ui/rating-display'
import { formatCount } from '@/lib/format'
import { type ParentSummary, type ReviewKind, kinds } from '@/lib/reviewKinds'
import { tagList } from '@/lib/reviews'
import FavoriteButton from './FavoriteButton.vue'

// ParentCard is the row of a restaurant or a recipe in a list. The default
// slot has the details of the kind, such as the distance. See decision 00009.
// heading is the element of the name. A list below an h2 uses h3.
withDefaults(defineProps<{ parent: ParentSummary; kind: ReviewKind; heading?: 'h2' | 'h3' }>(), { heading: 'h2' })

defineEmits<{ favorite: [favorite: boolean] }>()
</script>

<template>
  <div class="bg-card hover:border-ring relative flex h-full items-center gap-3 rounded-xl border py-2.5 pr-1.5 pl-2.5">
    <!-- The name of the card link is the name. The cover then has no alt text. -->
    <CoverPhoto :photo="parent.cover" size="card" class="aspect-square size-16 shrink-0 rounded-lg" />
    <div class="flex min-w-0 flex-1 flex-col gap-0.5">
      <component :is="heading" class="font-serif text-lg leading-[23px] font-semibold wrap-anywhere">
        <!-- The after element makes the full row the link. -->
        <RouterLink :to="`${kinds[kind].path}/${parent.id}`" class="rounded-sm after:absolute after:inset-0 after:rounded-xl">
          {{ parent.name }}
        </RouterLink>
      </component>
      <slot />
      <p class="text-muted-foreground text-[13px] leading-[18px]">
        {{ parent.reviewCount ? formatCount(parent.reviewCount, 'review') : 'No reviews yet' }}
        <template v-if="tagList(parent.tags).length">
          <span aria-hidden="true"> · </span><span class="sr-only">, tags: </span>{{ tagList(parent.tags).join(' · ') }}
        </template>
      </p>
    </div>
    <RatingDisplay v-if="parent.rating" :value="parent.rating.overall" :max="10" label="Overall" size="md" class="shrink-0" />
    <!-- The heart is above the link layer of the row. -->
    <FavoriteButton
      :restaurant-id="kind === 'restaurant' ? parent.id : undefined"
      :recipe-id="kind === 'recipe' ? parent.id : undefined"
      :name="parent.name"
      :favorite="parent.favorite"
      class="relative z-10 shrink-0"
      @change="$emit('favorite', $event)"
    />
  </div>
</template>
