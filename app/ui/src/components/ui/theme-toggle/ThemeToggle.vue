<script setup lang="ts">
import { MoonIcon, SunIcon } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { themeOptions, useTheme } from '@/composables/useTheme'

const { choice, label } = useTheme()
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" size="sm" class="text-muted-foreground">
        <SunIcon class="dark:hidden" aria-hidden="true" />
        <MoonIcon class="hidden dark:block" aria-hidden="true" />
        Theme: {{ label.toLowerCase() }}
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end">
      <DropdownMenuRadioGroup v-model="choice">
        <DropdownMenuRadioItem v-for="o in themeOptions" :key="o.value" :value="o.value">{{ o.label }}</DropdownMenuRadioItem>
      </DropdownMenuRadioGroup>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
