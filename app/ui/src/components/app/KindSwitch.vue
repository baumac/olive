<script setup lang="ts">
import { useRoute } from 'vue-router'
import type { KindChoice } from '@/composables/useKindQuery'

// KindSwitch selects all entries or the entries of one kind. The kind is in
// the URL. A switch drops the filters of the other kind.
withDefaults(
  defineProps<{ kind: KindChoice; label: string; labels?: Record<KindChoice, string> }>(),
  { labels: () => ({ all: 'All', restaurant: 'Restaurants', recipe: 'Recipes' }) },
)

const route = useRoute()
const options: KindChoice[] = ['all', 'restaurant', 'recipe']
const queries: Record<KindChoice, string> = { all: '', restaurant: '?kind=restaurants', recipe: '?kind=recipes' }
</script>

<template>
  <nav :aria-label="label" class="flex flex-wrap gap-2">
    <RouterLink
      v-for="o in options"
      :key="o"
      :to="`${route.path}${queries[o]}`"
      replace
      :aria-current="kind === o ? 'page' : undefined"
      :class="[
        'inline-flex h-11 items-center rounded-full border px-4 text-sm font-semibold',
        kind === o ? 'bg-primary border-primary text-primary-foreground' : 'bg-card border-input hover:bg-accent hover:text-accent-foreground',
      ]"
    >
      {{ labels[o] }}
    </RouterLink>
  </nav>
</template>
