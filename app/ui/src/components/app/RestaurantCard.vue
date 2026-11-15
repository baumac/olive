<script setup lang="ts">
import type { Restaurant } from '@reviews/sdk'
import { MapPinIcon } from 'lucide-vue-next'
import { computed } from 'vue'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { CoverPhoto } from '@/components/ui/cover-photo'
import { RatingDisplay } from '@/components/ui/rating-display'
import type { MapPoint } from '@/components/ui/map'
import { formatCount } from '@/lib/format'
import { distanceMeters, formatDistance } from '@/lib/preferredLocation'
import { tagList } from '@/lib/reviews'
import FavoriteButton from './FavoriteButton.vue'

// heading is the element of the name. A list below an h2 uses h3.
// from is the point for the distance to the restaurant.
const props = withDefaults(defineProps<{ restaurant: Restaurant; heading?: 'h2' | 'h3'; from?: MapPoint | null }>(), {
  heading: 'h2',
  from: null,
})

const distance = computed(() =>
  props.from && props.restaurant.location ? formatDistance(distanceMeters(props.from, props.restaurant.location)) : null,
)

defineEmits<{ favorite: [favorite: boolean] }>()
</script>

<template>
  <Card class="hover:border-ring relative h-full overflow-hidden pt-0">
    <!-- The name of the card link is the name. The cover then has no alt text. -->
    <CoverPhoto :photo="restaurant.cover" size="card" />
    <!-- The heart is above the link layer of the card. -->
    <FavoriteButton
      :restaurant-id="restaurant.id"
      :name="restaurant.name"
      :favorite="restaurant.favorite"
      class="absolute top-2 right-2 z-10"
      @change="$emit('favorite', $event)"
    />
    <CardHeader>
      <CardTitle :as="heading" class="text-base leading-snug">
        <!-- The after element makes the full card the link. -->
        <RouterLink :to="`/restaurants/${restaurant.id}`" class="rounded-sm after:absolute after:inset-0 after:rounded-xl">
          {{ restaurant.name }}
        </RouterLink>
      </CardTitle>
      <CardDescription v-if="restaurant.description" class="line-clamp-2 whitespace-pre-line">
        {{ restaurant.description }}
      </CardDescription>
    </CardHeader>
    <CardContent class="grid gap-3">
      <p class="flex flex-wrap items-baseline gap-x-2">
        <RatingDisplay v-if="restaurant.rating" :value="restaurant.rating.overall" :max="10" label="Overall" size="md" />
        <span class="text-muted-foreground text-sm">
          {{ restaurant.reviewCount ? formatCount(restaurant.reviewCount, 'review') : 'No reviews yet' }}
        </span>
      </p>
      <p v-if="restaurant.location" class="text-muted-foreground flex items-start gap-1.5 text-sm">
        <MapPinIcon class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
        <span class="line-clamp-1">
          <template v-if="distance"><span class="sr-only">Distance: </span>{{ distance }}<span aria-hidden="true"> · </span></template>
          <span class="sr-only">Address: </span>{{ restaurant.location.address }}
        </span>
      </p>
      <div v-if="tagList(restaurant.tags).length" class="flex flex-wrap gap-1.5">
        <Badge v-for="tag in tagList(restaurant.tags)" :key="tag" variant="secondary">{{ tag }}</Badge>
      </div>
    </CardContent>
  </Card>
</template>
