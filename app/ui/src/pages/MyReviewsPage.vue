<script setup lang="ts">
import ReviewList from '@/components/app/ReviewList.vue'
import { usePageTitle } from '@/composables/usePageTitle'
import { currentUser } from '@/lib/session'

usePageTitle('My reviews')
</script>

<template>
  <main id="main" class="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-8 sm:px-6">
    <h1 class="text-4xl font-extrabold tracking-tight">My reviews</h1>
    <ReviewList v-if="currentUser" :creator="currentUser.id" />
  </main>
</template>
