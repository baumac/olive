<script setup lang="ts">
import type { FilterDef } from '@/components/ui/filter-bar'
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import ReviewList from '@/components/app/ReviewList.vue'
import { usePageTitle } from '@/composables/usePageTitle'
import { reviewFilterDefs } from '@/lib/reviewFilters'
import { currentUser } from '@/lib/session'

const route = useRoute()

// The creator filter offers only the signed in user. "My reviews" in the
// account menu opens this page with it.
const defs = computed<FilterDef[]>(() => {
  const user = currentUser.value
  if (!user) {
    return reviewFilterDefs
  }
  const mine: FilterDef = {
    key: 'creator',
    label: 'Written by',
    type: 'select',
    options: [{ value: user.id, label: 'You' }],
    chip: () => 'Written by: you',
  }
  return [mine, ...reviewFilterDefs]
})

const mine = computed(() => !!currentUser.value && route.query.creator === currentUser.value.id)
const title = computed(() => (mine.value ? 'My reviews' : 'Reviews'))
usePageTitle(title)
</script>

<template>
  <main id="main" class="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-8 sm:px-6">
    <h1 class="text-4xl font-extrabold tracking-tight">{{ title }}</h1>
    <ReviewList :defs="defs" :mine="mine" />
  </main>
</template>
