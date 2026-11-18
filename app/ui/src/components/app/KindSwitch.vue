<script setup lang="ts">
import { useRoute } from 'vue-router'
import type { ReviewKind } from '@/lib/reviewKinds'

// KindSwitch selects the restaurant part or the recipe part of a page. The
// kind is in the URL. A switch drops the filters of the other kind.
defineProps<{ kind: ReviewKind; label: string }>()

const route = useRoute()
const options: { kind: ReviewKind; label: string }[] = [
  { kind: 'restaurant', label: 'Restaurants' },
  { kind: 'recipe', label: 'Recipes' },
]
</script>

<template>
  <nav :aria-label="label" class="bg-muted flex w-fit gap-1 rounded-lg p-1">
    <RouterLink
      v-for="o in options"
      :key="o.kind"
      :to="o.kind === 'recipe' ? `${route.path}?kind=recipes` : route.path"
      replace
      :aria-current="kind === o.kind ? 'page' : undefined"
      :class="[
        'rounded-md px-3 py-1.5 text-sm font-medium',
        kind === o.kind ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground',
      ]"
    >
      {{ o.label }}
    </RouterLink>
  </nav>
</template>
