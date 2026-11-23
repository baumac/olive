<script setup lang="ts">
import { formatScore } from '@/lib/format'
import { cn } from '@/lib/utils'

const sizes = {
  sm: 'text-lg',
  md: 'text-2xl leading-7',
  lg: 'text-6xl leading-none',
}

withDefaults(
  defineProps<{
    value: number
    max: number
    label: string
    // showLabel shows the label before the number. The accessible name
    // always has the label.
    showLabel?: boolean
    size?: 'sm' | 'md' | 'lg'
  }>(),
  { showLabel: false, size: 'sm' },
)
</script>

<template>
  <span role="img" :aria-label="`${label}: ${formatScore(value)} out of ${max}`" class="inline-flex items-baseline gap-1.5">
    <span v-if="showLabel" class="text-muted-foreground text-sm" aria-hidden="true">{{ label }}</span>
    <span
      :class="cn('text-foreground score-numerals font-semibold', sizes[size])"
      aria-hidden="true"
    >
      {{ formatScore(value) }}
    </span>
  </span>
</template>
