<script setup lang="ts">
import type { PantryItem } from '@reviews/sdk'
import { computed } from 'vue'
import { categoryIcon, pantryItemMeta } from '@/lib/pantry'
import ParentCard from './ParentCard.vue'

// heading is the element of the name. A list below an h2 uses h3.
const props = withDefaults(defineProps<{ item: PantryItem; heading?: 'h2' | 'h3' }>(), { heading: 'h2' })

defineEmits<{ favorite: [favorite: boolean] }>()

const icon = computed(() => categoryIcon(props.item.category))
</script>

<template>
  <ParentCard :parent="item" kind="pantryItem" :heading="heading" @favorite="$emit('favorite', $event)">
    <p class="text-muted-foreground flex min-w-0 items-center gap-1 text-[13px] leading-[18px]">
      <component :is="icon" class="size-3.5 shrink-0" aria-hidden="true" />
      <span class="truncate">{{ pantryItemMeta(item) }}</span>
    </p>
  </ParentCard>
</template>
