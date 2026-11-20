<script setup lang="ts">
import { computed } from 'vue'
import { formatDay } from '@/lib/format'
import { type AnyReview, kindOf, kinds } from '@/lib/reviewKinds'

const props = defineProps<{ review: AnyReview }>()

const label = computed(() => kinds[kindOf(props.review)].dateLabel)
</script>

<template>
  <span>{{ label }} <time :datetime="review.date">{{ formatDay(review.date) }}</time></span>
</template>
