<script setup lang="ts">
import type { Reviewer } from '@reviews/sdk'
import { computed, watch } from 'vue'
import ErrorPage from '@/components/app/ErrorPage.vue'
import ReviewList from '@/components/app/ReviewList.vue'
import { InitialsAvatar } from '@/components/ui/avatar'
import { Skeleton } from '@/components/ui/skeleton'
import { useLoad } from '@/composables/useLoad'
import { usePageTitle } from '@/composables/usePageTitle'
import { reviewersApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { currentUser } from '@/lib/session'

const props = defineProps<{ reviewerId: string }>()

const reviewer = useLoad<Reviewer>(() => reviewersApi.getReviewer({ reviewerId: props.reviewerId }))
watch(() => props.reviewerId, () => void reviewer.load(), { immediate: true })

usePageTitle(() => reviewer.data.value?.name ?? '')

const isSelf = computed(() => currentUser.value?.id === props.reviewerId)

// The API gives 400 for a malformed ID. The link then has a typo.
const failedStatus = computed(() => {
  const status = reviewer.failed.value?.status ?? null
  return status === 400 ? 404 : status
})
</script>

<template>
  <ErrorPage v-if="reviewer.failed.value" :status="failedStatus" @retry="reviewer.load" />
  <main v-else id="main" class="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-8 sm:px-6">
    <div v-if="reviewer.loading.value" aria-busy="true" aria-label="Loading reviewer" class="flex flex-col gap-4">
      <Skeleton class="h-10 w-1/2" />
      <Skeleton class="h-6 w-40" />
    </div>
    <template v-else-if="reviewer.data.value">
      <section class="flex items-center gap-4">
        <InitialsAvatar :name="reviewer.data.value.name" size="lg" />
        <div class="flex min-w-0 flex-col gap-1">
          <h1 class="text-4xl font-extrabold tracking-tight break-words">{{ reviewer.data.value.name }}</h1>
          <p class="text-muted-foreground text-sm">
            {{ reviewer.data.value.reviewCount ? formatCount(reviewer.data.value.reviewCount, 'review') : 'No reviews yet' }}
          </p>
        </div>
      </section>
      <section class="flex flex-col gap-6" aria-labelledby="reviews-heading">
        <h2 id="reviews-heading" class="text-3xl font-semibold tracking-tight">Reviews</h2>
        <ReviewList :creator="reviewerId" :reviewer-name="isSelf ? undefined : reviewer.data.value.name" />
      </section>
    </template>
  </main>
</template>
