<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed, provide } from 'vue'
import { cn } from '@/lib/utils'
import { formItemKey } from './context'

const props = defineProps<{
  id: string
  error?: string
  description?: boolean
  class?: HTMLAttributes['class']
}>()

provide(formItemKey, {
  id: props.id,
  error: computed(() => props.error ?? ''),
  hasDescription: computed(() => props.description ?? false),
})
</script>

<template>
  <div data-slot="form-item" :class="cn('grid gap-2', props.class)">
    <slot />
  </div>
</template>
