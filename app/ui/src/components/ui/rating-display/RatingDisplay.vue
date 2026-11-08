<script setup lang="ts">
import { formatScore } from '@/lib/format'
import { cn } from '@/lib/utils'

defineProps<{
  value: number
  max: number
  label: string
}>()
</script>

<template>
  <div role="img" :aria-label="`${label}: ${formatScore(value)} out of ${max}`" class="flex items-center gap-3">
    <span class="text-muted-foreground w-16 shrink-0 text-sm" aria-hidden="true">{{ label }}</span>
    <span class="flex gap-1" aria-hidden="true">
      <span
        v-for="n in max"
        :key="n"
        :class="cn('size-2.5 rounded-full', n <= value ? 'bg-primary' : 'bg-muted ring-input ring-1 ring-inset')"
      />
    </span>
    <span class="text-sm font-medium tabular-nums" aria-hidden="true">{{ formatScore(value) }} / {{ max }}</span>
  </div>
</template>
