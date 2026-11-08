<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  icon: Component
  title: string
  description: string
}>()
</script>

<template>
  <div class="border-input flex flex-col items-center gap-3 rounded-xl border border-dashed px-6 py-10 text-center">
    <span class="bg-muted text-muted-foreground flex size-10 items-center justify-center rounded-md">
      <component :is="icon" class="size-5" aria-hidden="true" />
    </span>
    <p class="text-lg font-semibold">{{ title }}</p>
    <p class="text-muted-foreground max-w-sm text-sm">{{ description }}</p>
    <div v-if="$slots.action" class="mt-1">
      <slot name="action" />
    </div>
  </div>
</template>
