<script setup lang="ts">
import { Slot } from 'reka-ui'
import { computed } from 'vue'
import { useFormItem } from './context'

const { id, error, hasDescription } = useFormItem()

const describedBy = computed(() =>
  [hasDescription.value && `${id}-description`, error.value && `${id}-message`].filter(Boolean).join(' ') || undefined,
)
</script>

<template>
  <Slot :id="id" :aria-describedby="describedBy" :aria-invalid="error ? 'true' : undefined">
    <slot />
  </Slot>
</template>
