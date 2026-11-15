<script setup lang="ts">
import type { Place } from '@reviews/sdk'
import type { PlaceSearchState } from '@/composables/usePlaceSearch'
import { formatCount } from '@/lib/format'

defineProps<{
  places: Place[]
  state: PlaceSearchState
  retryAfter: number
  // empty is the text for a search without places.
  empty: string
}>()

defineEmits<{ select: [place: Place] }>()
</script>

<template>
  <p v-if="state === 'limited'" class="text-sm" role="status">
    Too many searches. Try again in {{ formatCount(retryAfter, 'second') }}.
  </p>
  <p class="sr-only" role="status">
    <template v-if="state === 'loading'">Searching</template>
    <template v-else-if="state === 'done'">{{ places.length ? `${formatCount(places.length, 'place')} found` : 'No places found' }}</template>
  </p>
  <ul v-if="places.length" class="divide-border divide-y rounded-md border" aria-label="Places">
    <li v-for="(place, i) in places" :key="i">
      <button
        type="button"
        class="hover:bg-accent focus-visible:ring-ring/50 w-full rounded-md px-3 py-2 text-left text-sm outline-none focus-visible:ring-[3px]"
        @click="$emit('select', place)"
      >
        <span v-if="place.name" class="block font-medium">{{ place.name }}</span>
        <span class="text-muted-foreground block">{{ place.location.address }}</span>
      </button>
    </li>
  </ul>
  <p v-else-if="state === 'done'" class="text-muted-foreground text-sm">{{ empty }}</p>
</template>
