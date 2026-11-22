<script setup lang="ts">
import type { Restaurant } from '@reviews/sdk'
import { MapPinIcon } from 'lucide-vue-next'
import { computed } from 'vue'
import type { MapPoint } from '@/components/ui/map'
import { distanceMeters, formatDistance } from '@/lib/preferredLocation'
import ParentCard from './ParentCard.vue'

// heading is the element of the name. A list below an h2 uses h3.
// from is the point for the distance to the restaurant.
const props = withDefaults(defineProps<{ restaurant: Restaurant; heading?: 'h2' | 'h3'; from?: MapPoint | null }>(), {
  heading: 'h2',
  from: null,
})

const distance = computed(() =>
  props.from ? formatDistance(distanceMeters(props.from, props.restaurant.location)) : null,
)

defineEmits<{ favorite: [favorite: boolean] }>()
</script>

<template>
  <ParentCard :parent="restaurant" kind="restaurant" :heading="heading" @favorite="$emit('favorite', $event)">
    <p class="text-muted-foreground flex items-start gap-1.5 text-sm">
      <MapPinIcon class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
      <span class="line-clamp-1">
        <template v-if="distance"><span class="sr-only">Distance: </span>{{ distance }}<span aria-hidden="true"> · </span></template>
        <span class="sr-only">Address: </span>{{ restaurant.location.address }}
      </span>
    </p>
  </ParentCard>
</template>
