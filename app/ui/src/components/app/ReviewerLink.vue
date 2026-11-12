<script setup lang="ts">
import type { Creator } from '@reviews/sdk'

defineProps<{ creator: Creator }>()
</script>

<template>
  <RouterLink :to="`/reviewers/${creator.id}`" class="rounded-sm hover:underline">{{ creator.name }}</RouterLink>
</template>
