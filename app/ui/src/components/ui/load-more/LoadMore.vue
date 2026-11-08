<script setup lang="ts">
import { Loader2Icon } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'

defineProps<{
  cursor: string | null
  loading: boolean
  label: string
}>()

defineEmits<{ load: [cursor: string] }>()
</script>

<template>
  <div v-if="cursor" class="flex justify-center">
    <Button variant="outline" :disabled="loading" class="grid" @click="$emit('load', cursor)">
      <!-- Both layers take one grid cell. The width then stays the same. -->
      <span :class="['col-start-1 row-start-1', loading && 'invisible']">{{ label }}</span>
      <span v-if="loading" class="col-start-1 row-start-1 flex items-center justify-center gap-2">
        <Loader2Icon class="animate-spin" aria-hidden="true" />
        Loading
      </span>
    </Button>
  </div>
</template>
