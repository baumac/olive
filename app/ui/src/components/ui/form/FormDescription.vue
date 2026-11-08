<script setup lang="ts">
import { useFormItem } from './context'

const { id } = useFormItem()
</script>

<template>
  <p :id="`${id}-description`" class="text-muted-foreground text-sm">
    <slot />
  </p>
</template>
