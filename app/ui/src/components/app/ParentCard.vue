<script setup lang="ts">
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { CoverPhoto } from '@/components/ui/cover-photo'
import { RatingDisplay } from '@/components/ui/rating-display'
import { formatCount } from '@/lib/format'
import { type Parent, type ReviewKind, kinds } from '@/lib/reviewKinds'
import { tagList } from '@/lib/reviews'
import FavoriteButton from './FavoriteButton.vue'

// ParentCard is the card of a restaurant or a recipe. The default slot has
// the details of the kind, below the rating.
// heading is the element of the name. A list below an h2 uses h3.
withDefaults(defineProps<{ parent: Parent; kind: ReviewKind; heading?: 'h2' | 'h3' }>(), { heading: 'h2' })

defineEmits<{ favorite: [favorite: boolean] }>()
</script>

<template>
  <Card class="hover:border-ring relative h-full overflow-hidden pt-0">
    <!-- The name of the card link is the name. The cover then has no alt text. -->
    <CoverPhoto :photo="parent.cover" size="card" />
    <!-- The heart is above the link layer of the card. -->
    <FavoriteButton
      :restaurant-id="kind === 'restaurant' ? parent.id : undefined"
      :recipe-id="kind === 'recipe' ? parent.id : undefined"
      :name="parent.name"
      :favorite="parent.favorite"
      class="absolute top-2 right-2 z-10"
      @change="$emit('favorite', $event)"
    />
    <CardHeader>
      <CardTitle :as="heading" class="text-base leading-snug">
        <!-- The after element makes the full card the link. -->
        <RouterLink :to="`${kinds[kind].path}/${parent.id}`" class="rounded-sm after:absolute after:inset-0 after:rounded-xl">
          {{ parent.name }}
        </RouterLink>
      </CardTitle>
      <CardDescription v-if="parent.description" class="line-clamp-2 whitespace-pre-line">
        {{ parent.description }}
      </CardDescription>
    </CardHeader>
    <CardContent class="grid gap-3">
      <p class="flex flex-wrap items-baseline gap-x-2">
        <RatingDisplay v-if="parent.rating" :value="parent.rating.overall" :max="10" label="Overall" size="md" />
        <span class="text-muted-foreground text-sm">
          {{ parent.reviewCount ? formatCount(parent.reviewCount, 'review') : 'No reviews yet' }}
        </span>
      </p>
      <slot />
      <div v-if="tagList(parent.tags).length" class="flex flex-wrap gap-1.5">
        <Badge v-for="tag in tagList(parent.tags)" :key="tag" variant="secondary">{{ tag }}</Badge>
      </div>
    </CardContent>
  </Card>
</template>
