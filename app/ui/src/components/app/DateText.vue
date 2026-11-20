<script setup lang="ts">
import { computed } from 'vue'
import { formatDate, formatFullDate, isEdited, isoDate } from '@/lib/format'

const props = defineProps<{
  created: string
  updated?: string
}>()

const edited = computed(() => isEdited(props.created, props.updated))
</script>

<template>
  <span>
    <time :datetime="isoDate(created)" :title="formatFullDate(created)">{{ formatDate(created) }}</time>
    <template v-if="edited"> · edited</template>
  </span>
</template>
