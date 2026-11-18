<script setup lang="ts">
import KindSwitch from '@/components/app/KindSwitch.vue'
import ReviewList from '@/components/app/ReviewList.vue'
import { useKindQuery } from '@/composables/useKindQuery'
import { usePageTitle } from '@/composables/usePageTitle'
import { currentUser } from '@/lib/session'

usePageTitle('My reviews')

const kind = useKindQuery()
</script>

<template>
  <main id="main" class="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-8 sm:px-6">
    <h1 class="text-4xl font-extrabold tracking-tight">My reviews</h1>
    <KindSwitch :kind="kind" label="Reviews of" />
    <ReviewList v-if="currentUser" :key="kind" :creator="currentUser.id" :kind="kind" />
  </main>
</template>
