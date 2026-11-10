<script setup lang="ts">
import type { Restaurant } from '@reviews/sdk'
import { PencilIcon } from 'lucide-vue-next'
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
import { ratingScores, tagList } from '@/lib/reviews'
import { currentUser } from '@/lib/session'

const props = defineProps<{ restaurantId: string }>()
const router = useRouter()

const restaurant = useLoad<Restaurant>(() => restaurantsApi.getRestaurant({ restaurantId: props.restaurantId }))
watch(() => props.restaurantId, () => void restaurant.load(), { immediate: true })

usePageTitle(() => restaurant.data.value?.name ?? '')

const isCreator = computed(
  () => !!currentUser.value && restaurant.data.value?.creator?.id === currentUser.value.id,
)

// ownReviewId is the review of the signed in user, if any. A user can have
// at most one review per restaurant.
const ownReviewId = ref<string | null>(null)
watch(
  [() => props.restaurantId, currentUser],
  async ([restaurantId, user]) => {
    ownReviewId.value = null
    if (!user) {
      return
    }
    try {
      const page = await restaurantsApi.listReviews({ restaurantId, creatorEq: user.id, limit: 1 })
      ownReviewId.value = page.items[0]?.id ?? null
    } catch {
      ownReviewId.value = null
    }
  },
  { immediate: true },
)

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
        ? 'A restaurant with reviews cannot be deleted.'
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
          <div v-if="isCreator" class="flex flex-wrap gap-2">
            <Button variant="outline" as-child>
              <RouterLink :to="`/restaurants/${restaurantId}/edit`"><PencilIcon aria-hidden="true" />Edit restaurant</RouterLink>
            </Button>
            <ConfirmDelete
              label="Delete restaurant"
              title="Delete this restaurant?"
              :description="`${restaurant.data.value.name} will be deleted permanently. A restaurant with reviews cannot be deleted.`"
              @confirm="remove"
            />
          </div>
        </div>
        <p v-if="restaurant.data.value.description" class="text-muted-foreground max-w-2xl text-xl whitespace-pre-line">
          {{ restaurant.data.value.description }}
        </p>
        <ul v-if="tagList(restaurant.data.value.tags).length" class="flex flex-wrap gap-1.5" aria-label="Tags">
          <li v-for="tag in tagList(restaurant.data.value.tags)" :key="tag"><Badge variant="secondary">{{ tag }}</Badge></li>
        </ul>
        <section v-if="restaurant.data.value.rating" class="grid gap-2" aria-labelledby="rating-heading">
          <h2 id="rating-heading" class="text-sm font-medium">
            Average of {{ formatCount(restaurant.data.value.reviewCount, 'review') }}
          </h2>
          <RatingDisplay
            v-for="s in ratingScores(restaurant.data.value.rating)"
            :key="s.name"
            :value="s.value"
            :max="s.max"
            :label="s.label"
          />
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
