<script setup lang="ts">
import { useFormItem } from './context'

const { id, error } = useFormItem()
</script>

<template>
  <p v-if="error" :id="`${id}-message`" class="text-destructive text-sm">
    {{ error }}
  </p>
</template>
