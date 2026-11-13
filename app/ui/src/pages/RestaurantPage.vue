<script setup lang="ts">
import type { Restaurant } from '@reviews/sdk'
import { MapPinIcon, PencilIcon } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ConfirmDelete } from '@/components/ui/confirm-delete'
import DateText from '@/components/app/DateText.vue'
import ErrorPage from '@/components/app/ErrorPage.vue'
import ReviewList from '@/components/app/ReviewList.vue'
import { InitialsAvatar } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { RatingDisplay } from '@/components/ui/rating-display'
import { Skeleton } from '@/components/ui/skeleton'
import { useLoad } from '@/composables/useLoad'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, restaurantsApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { findOwnReviewId, restaurantAccess } from '@/lib/restaurants'
import { scoreLegend, scoresOf, tagList } from '@/lib/reviews'
import { currentUser } from '@/lib/session'

const props = defineProps<{ restaurantId: string }>()
const router = useRouter()

const restaurant = useLoad<Restaurant>(() => restaurantsApi.getRestaurant({ restaurantId: props.restaurantId }))
watch(() => props.restaurantId, () => void restaurant.load(), { immediate: true })

usePageTitle(() => restaurant.data.value?.name ?? '')

// ownReviewId is the review of the signed in user, if any. A user can have
// at most one review per restaurant.
const ownReviewId = ref<string | null>(null)
const ownReviewLookup = ref<'loading' | 'done' | 'failed'>('loading')
watch(
  [() => props.restaurantId, currentUser],
  async ([restaurantId, user]) => {
    ownReviewId.value = null
    ownReviewLookup.value = 'done'
    if (!user) {
      return
    }
    ownReviewLookup.value = 'loading'
    try {
      ownReviewId.value = await findOwnReviewId(restaurantId, user.id)
      ownReviewLookup.value = 'done'
    } catch {
      ownReviewLookup.value = 'failed'
    }
  },
  { immediate: true },
)

const access = computed(() => {
  if (!restaurant.data.value || ownReviewLookup.value === 'loading') {
    return null
  }
  const hasOwnReview = ownReviewLookup.value === 'failed' ? null : !!ownReviewId.value
  return restaurantAccess(restaurant.data.value, currentUser.value?.id, hasOwnReview)
})

const deleteDescription = computed(() => {
  const text = `${restaurant.data.value?.name} will be deleted permanently.`
  return ownReviewId.value ? `${text} Your review and its photos will be deleted too.` : text
})

const writePath = computed(() =>
  ownReviewId.value
    ? `/restaurants/${props.restaurantId}/reviews/${ownReviewId.value}/edit`
    : `/restaurants/${props.restaurantId}/reviews/new`,
)

async function remove() {
  const name = restaurant.data.value?.name
  try {
    await restaurantsApi.deleteRestaurant({ restaurantId: props.restaurantId })
    toast.success('Restaurant deleted', { description: name })
    void router.push('/restaurants')
  } catch (error) {
    const { status } = await apiFailure(error)
    const description =
      status === 409
        ? 'A restaurant with reviews of other people cannot be deleted.'
        : status === 403
          ? 'You do not have permission to delete it.'
        : status === 401
          ? 'Sign in again, then try again.'
          : 'Try again in a moment.'
    toast.error('Restaurant not deleted', { description })
  }
}
</script>

<template>
  <ErrorPage v-if="restaurant.failed.value" :status="restaurant.failed.value.status" @retry="restaurant.load" />
  <main v-else id="main" class="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-8 sm:px-6">
    <div v-if="restaurant.loading.value" aria-busy="true" aria-label="Loading restaurant" class="flex flex-col gap-4">
      <Skeleton class="h-10 w-2/3" />
      <Skeleton class="h-6 w-full max-w-2xl" />
      <Skeleton class="h-6 w-1/2" />
    </div>
    <template v-else-if="restaurant.data.value">
      <section class="flex flex-col gap-4">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <h1 class="text-4xl font-extrabold tracking-tight break-words">{{ restaurant.data.value.name }}</h1>
          <div v-if="access?.canChange" class="flex flex-col gap-2 sm:items-end">
            <div class="flex flex-wrap gap-2">
              <Button variant="outline" as-child>
                <RouterLink :to="`/restaurants/${restaurantId}/edit`"><PencilIcon aria-hidden="true" />Edit restaurant</RouterLink>
              </Button>
              <ConfirmDelete
                v-if="!access.hasOtherReviews"
                label="Delete restaurant"
                title="Delete this restaurant?"
                :description="deleteDescription"
                @confirm="remove"
              />
            </div>
            <p v-if="access.hasOtherReviews" class="text-muted-foreground text-sm">
              Other people have reviewed this restaurant. You cannot delete it.
            </p>
          </div>
        </div>
        <p v-if="restaurant.data.value.description" class="text-muted-foreground max-w-2xl text-xl whitespace-pre-line">
          {{ restaurant.data.value.description }}
        </p>
        <p v-if="restaurant.data.value.location" class="flex items-start gap-2">
          <MapPinIcon class="text-muted-foreground mt-1 size-4 shrink-0" aria-hidden="true" />
          <span><span class="sr-only">Address: </span>{{ restaurant.data.value.location.address }}</span>
        </p>
        <p v-else-if="access?.canChange" class="text-muted-foreground flex items-start gap-2">
          <MapPinIcon class="mt-1 size-4 shrink-0" aria-hidden="true" />
          <span>
            This restaurant has no location.
            <RouterLink :to="`/restaurants/${restaurantId}/edit`" class="text-foreground font-medium underline underline-offset-4">
              Add the location
            </RouterLink>
          </span>
        </p>
        <ul v-if="tagList(restaurant.data.value.tags).length" class="flex flex-wrap gap-1.5" aria-label="Tags">
          <li v-for="tag in tagList(restaurant.data.value.tags)" :key="tag"><Badge variant="secondary">{{ tag }}</Badge></li>
        </ul>
        <section v-if="restaurant.data.value.rating" class="grid gap-2" aria-labelledby="rating-heading" aria-describedby="rating-legend">
          <h2 id="rating-heading" class="text-sm font-medium">
            Average of {{ formatCount(restaurant.data.value.reviewCount, 'review') }}
          </h2>
          <RatingDisplay
            v-for="s in scoresOf(restaurant.data.value.rating)"
            :key="s.name"
            :value="s.value"
            :max="s.max"
            :label="s.label"
          />
          <p id="rating-legend" class="text-muted-foreground text-sm">{{ scoreLegend }}</p>
        </section>
        <p class="text-muted-foreground flex items-center gap-2 text-sm">
          <InitialsAvatar :name="restaurant.data.value.creator?.name ?? null" size="sm" />
          <span>
            Added by {{ restaurant.data.value.creator?.name ?? 'Deleted user' }} ·
            <DateText :created="restaurant.data.value.created" :updated="restaurant.data.value.updated" />
          </span>
        </p>
      </section>

      <section class="flex flex-col gap-6" aria-labelledby="reviews-heading">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <h2 id="reviews-heading" class="text-3xl font-semibold tracking-tight">Reviews</h2>
          <Button as-child>
            <RouterLink :to="writePath">{{ ownReviewId ? 'Edit your review' : 'Write a review' }}</RouterLink>
          </Button>
        </div>
        <ReviewList :restaurant="restaurant.data.value" :write-path="writePath" />
      </section>
    </template>
  </main>
</template>
