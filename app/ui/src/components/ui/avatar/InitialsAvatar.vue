<script setup lang="ts">
import { computed } from 'vue'
import { initials } from '@/lib/format'
import { cn } from '@/lib/utils'
import Avatar from './Avatar.vue'
import AvatarFallback from './AvatarFallback.vue'

const props = withDefaults(
  defineProps<{
    name: string | null
    size?: 'sm' | 'default' | 'lg'
    // decorative hides the avatar from screen readers. Use it when the name
    // shows beside the avatar.
    decorative?: boolean
  }>(),
  { size: 'default', decorative: true },
)

const sizes = { sm: 'size-6 text-[10px]', default: 'size-8 text-xs', lg: 'size-10 text-sm' }
const label = computed(() => props.name ?? 'Deleted user')
</script>

<template>
  <Avatar
    :class="cn(sizes[size])"
    :aria-hidden="decorative ? 'true' : undefined"
    :role="decorative ? undefined : 'img'"
    :aria-label="decorative ? undefined : label"
  >
    <AvatarFallback
      :class="cn('text-foreground font-medium', !name && 'bg-background border-input border border-dashed')"
    >
      {{ initials(name) }}
    </AvatarFallback>
  </Avatar>
</template>
