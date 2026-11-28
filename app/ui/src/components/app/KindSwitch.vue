<script setup lang="ts">
import { useRoute } from 'vue-router'
import type { KindChoice } from '@/composables/useKindQuery'
import { kinds, reviewKinds } from '@/lib/reviewKinds'

// KindSwitch selects all entries or the entries of one kind. The kind is in
// the URL. A switch drops the filters of the other kinds.
withDefaults(
  defineProps<{ kind: KindChoice; label: string; labels?: Record<KindChoice, string> }>(),
  { labels: () => ({ all: 'All', restaurant: 'Restaurants', recipe: 'Recipes', pantryItem: 'Pantry' }) },
)

const route = useRoute()
const options: KindChoice[] = ['all', ...reviewKinds]

function queryOf(choice: KindChoice): string {
  return choice === 'all' ? '' : `?kind=${kinds[choice].query}`
}
</script>

<template>
  <nav :aria-label="label" class="flex flex-wrap gap-2">
    <RouterLink
      v-for="o in options"
      :key="o"
      :to="`${route.path}${queryOf(o)}`"
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
