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

const sizes = { sm: 'size-7 text-[13px]', default: 'size-9 text-base', lg: 'size-[68px] text-3xl' }
// The large avatar of the journal shows only the first letter.
const letters = computed(() => (props.size === 'lg' && props.name ? Array.from(initials(props.name))[0] : initials(props.name)))
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
      :class="cn('bg-accent text-accent-foreground font-serif font-semibold', !name && 'bg-background text-foreground border-input border border-dashed font-sans')"
    >
      {{ letters }}
    </AvatarFallback>
  </Avatar>
</template>
