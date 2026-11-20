<script setup lang="ts">
import { ArrowLeftIcon, PencilIcon } from 'lucide-vue-next'
import { computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ConfirmDelete } from '@/components/ui/confirm-delete'
import DateText from '@/components/app/DateText.vue'
import ReviewDate from '@/components/app/ReviewDate.vue'
import ErrorPage from '@/components/app/ErrorPage.vue'
import ReviewerLink from '@/components/app/ReviewerLink.vue'
import ReviewPhotos from '@/components/app/ReviewPhotos.vue'
import ScoreSummary from '@/components/app/ScoreSummary.vue'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useLoad } from '@/composables/useLoad'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, failureText } from '@/lib/api'
import { type AnyReview, type Parent, kinds } from '@/lib/reviewKinds'
import { currentUser } from '@/lib/session'

// The route gives a restaurantId or a recipeId.
const props = defineProps<{ restaurantId?: string; recipeId?: string; reviewId: string }>()
const router = useRouter()

const kind = computed(() => (props.recipeId !== undefined ? kinds.recipe : kinds.restaurant))
const parentId = computed(() => (props.recipeId ?? props.restaurantId)!)

const page = useLoad<{ restaurant: Parent; review: AnyReview }>(async () => {
  const [restaurant, review] = await Promise.all([
    kind.value.getParent(parentId.value),
    kind.value.getReview(parentId.value, props.reviewId),
  ])
  return { restaurant, review }
})
watch(() => [parentId.value, props.reviewId], () => void page.load(), { immediate: true })

usePageTitle(() => page.data.value?.review.title ?? '')

const review = computed(() => page.data.value?.review ?? null)
const isCreator = computed(() => !!currentUser.value && review.value?.creator.id === currentUser.value.id)
const restaurantPath = computed(() => `${kind.value.path}/${parentId.value}`)

async function remove() {
  const title = review.value?.title
  try {
    await kind.value.deleteReview(parentId.value, props.reviewId)
    toast.success('Review deleted', { description: title })
    void router.push(restaurantPath.value)
  } catch (error) {
    toast.error('Review not deleted', { description: failureText(await apiFailure(error)) })
  }
}
</script>

<template>
  <ErrorPage v-if="page.failed.value" :status="page.failed.value.status" @retry="page.load" />
  <main v-else id="main" class="mx-auto flex max-w-2xl flex-col gap-6 px-4 py-8 sm:px-6">
    <div v-if="page.loading.value" aria-busy="true" aria-label="Loading review" class="grid gap-4">
      <Skeleton class="h-5 w-40" /><Skeleton class="h-10 w-3/4" /><Skeleton class="h-24" />
    </div>
    <template v-else-if="page.data.value && review">
      <RouterLink :to="restaurantPath" class="text-muted-foreground hover:text-foreground inline-flex w-fit items-center gap-1 text-sm">
        <ArrowLeftIcon class="size-4" aria-hidden="true" />{{ page.data.value.restaurant.name }}
      </RouterLink>
      <div class="flex flex-col gap-3">
        <h1 class="text-4xl font-extrabold tracking-tight break-words">{{ review.title }}</h1>
        <p class="text-muted-foreground text-sm">
          <ReviewerLink :creator="review.creator" /> · <ReviewDate :review="review" /> · Written
          <DateText :created="review.created" :updated="review.updated" />
        </p>
      </div>
      <ScoreSummary :scores="review" :kind="kind.kind" />
      <p v-if="review.feedback" class="leading-7 whitespace-pre-line">{{ review.feedback }}</p>
      <ReviewPhotos :review="review" :editable="isCreator" @change="review.photos = $event" />
      <div v-if="isCreator" class="flex flex-wrap gap-2 border-t pt-6">
        <Button variant="outline" as-child>
          <RouterLink :to="`${restaurantPath}/reviews/${reviewId}/edit`"><PencilIcon aria-hidden="true" />Edit review</RouterLink>
        </Button>
        <ConfirmDelete
          label="Delete review"
          title="Delete this review?"
          :description="`Your review ${review.title} will be deleted permanently.`"
          @confirm="remove"
        />
      </div>
    </template>
  </main>
</template>
