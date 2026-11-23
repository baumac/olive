<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { formatScore } from '@/lib/format'
import { cn } from '@/lib/utils'

// ScoreStamp shows an overall on a photo. See decision 00009.
const props = withDefaults(defineProps<{ value: number; size?: 'sm' | 'md'; class?: HTMLAttributes['class'] }>(), { size: 'md' })
</script>

<template>
  <span
    role="img"
    :aria-label="`Overall: ${formatScore(value)} out of 10`"
    :class="
      cn(
        'bg-saffron text-saffron-foreground score-numerals inline-flex shrink-0 items-center justify-center rounded-full shadow-[0_1px_2px_0_#00000040]',
        size === 'md' ? 'size-14 text-[21px] font-[650]' : 'size-9 text-sm font-bold',
        props.class,
      )
    "
  >
    <span aria-hidden="true">{{ formatScore(value) }}</span>
  </span>
</template>
