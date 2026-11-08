<script setup lang="ts">
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { useFormItem } from './context'

const { id, error } = useFormItem()
</script>

<template>
  <Label :for="id" :class="cn(error && 'text-destructive')">
    <slot />
  </Label>
</template>
