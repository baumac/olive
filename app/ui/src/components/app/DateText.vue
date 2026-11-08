<script setup lang="ts">
import { computed } from 'vue'
import { formatDate, formatFullDate, isoDate, parseTimestamp } from '@/lib/format'

const props = defineProps<{
  created: string
  updated?: string
}>()

const edited = computed(
  () => props.updated !== undefined && parseTimestamp(props.updated) > parseTimestamp(props.created),
)
</script>

<template>
  <span>
    <time :datetime="isoDate(created)" :title="formatFullDate(created)">{{ formatDate(created) }}</time>
    <template v-if="edited"> · edited</template>
  </span>
</template>
