<script setup lang="ts">
import { kindLabels } from '@/lib/reviews'

defineProps<{ kind: string }>()
</script>

<template>
  <span :lang="kind === 'joc_de_cartes' ? 'ca' : undefined">{{ kindLabels[kind] ?? kind }}</span>
</template>
