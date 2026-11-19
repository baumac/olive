<script setup lang="ts">
import type { RecipeSummary } from '@reviews/sdk'
import { ClockIcon } from 'lucide-vue-next'
import { formatMinutes } from '@/lib/format'
import ParentCard from './ParentCard.vue'

// heading is the element of the name. A list below an h2 uses h3.
withDefaults(defineProps<{ recipe: RecipeSummary; heading?: 'h2' | 'h3' }>(), { heading: 'h2' })

defineEmits<{ favorite: [favorite: boolean] }>()
</script>

<template>
  <ParentCard :parent="recipe" kind="recipe" :heading="heading" @favorite="$emit('favorite', $event)">
    <p v-if="recipe.totalTime" class="text-muted-foreground flex items-center gap-1.5 text-sm">
      <ClockIcon class="size-4 shrink-0" aria-hidden="true" />
      <span><span class="sr-only">Total time: </span>{{ formatMinutes(recipe.totalTime) }}</span>
    </p>
  </ParentCard>
</template>
