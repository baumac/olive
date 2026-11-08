<script setup lang="ts">
import { useColorMode } from '@vueuse/core'
import { MoonIcon, SunIcon } from 'lucide-vue-next'
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

// The inline script in index.html reads the same storage key.
const mode = useColorMode({ storageKey: 'vueuse-color-scheme', disableTransition: false })

const labels = { light: 'Light', dark: 'Dark', auto: 'System' } as const
const choice = computed({
  get: () => mode.store.value,
  set: (v) => (mode.store.value = v),
})
const label = computed(() => `Theme: ${labels[choice.value as keyof typeof labels]?.toLowerCase() ?? 'system'}`)
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" size="icon" :aria-label="label">
        <SunIcon class="dark:hidden" aria-hidden="true" />
        <MoonIcon class="hidden dark:block" aria-hidden="true" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end">
      <DropdownMenuRadioGroup v-model="choice">
        <DropdownMenuRadioItem value="light">Light</DropdownMenuRadioItem>
        <DropdownMenuRadioItem value="dark">Dark</DropdownMenuRadioItem>
        <DropdownMenuRadioItem value="auto">System</DropdownMenuRadioItem>
      </DropdownMenuRadioGroup>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
