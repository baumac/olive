<script setup lang="ts">
import type { Restaurant } from '@reviews/sdk'
import { MapPinIcon, PencilIcon } from 'lucide-vue-next'
import { useMediaQuery } from '@vueuse/core'
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ConfirmDelete } from '@/components/ui/confirm-delete'
import DateText from '@/components/app/DateText.vue'
import ErrorPage from '@/components/app/ErrorPage.vue'
import FavoriteButton from '@/components/app/FavoriteButton.vue'
import RestaurantDetails from '@/components/app/RestaurantDetails.vue'
import PhotoGallery from '@/components/app/PhotoGallery.vue'
import ReviewList from '@/components/app/ReviewList.vue'
import SaveToListMenu from '@/components/app/SaveToListMenu.vue'
import ScoreSummary from '@/components/app/ScoreSummary.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { CoverPhoto } from '@/components/ui/cover-photo'
import { MapView } from '@/components/ui/map'
import { Skeleton } from '@/components/ui/skeleton'
import { useLoad } from '@/composables/useLoad'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, failureText, restaurantsApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { useMapConfig } from '@/lib/mapConfig'
import { directionsLink, osmLink } from '@/lib/places'
import { ownReviewsText, parentAccess } from '@/lib/parents'
import { tagList } from '@/lib/reviews'
import { currentUser } from '@/lib/session'

const props = defineProps<{ restaurantId: string }>()
const router = useRouter()

const restaurant = useLoad<Restaurant>(() => restaurantsApi.getRestaurant({ restaurantId: props.restaurantId }))
// ownReviewCount and favorite belong to the signed in user.
watch([() => props.restaurantId, () => currentUser.value?.id], () => void restaurant.load(), { immediate: true })

usePageTitle(() => restaurant.data.value?.name ?? '')


const mapConfig = useMapConfig()
// The wide layout shows the map next to the summary. The narrow layout
// shows it only after the person opens it.
const wide = useMediaQuery('(min-width: 1024px)')
const mapOpen = ref(false)
const marker = computed(() => {
  const r = restaurant.data.value
  return r?.location ? { id: r.id, latitude: r.location.latitude, longitude: r.location.longitude, label: r.name } : null
})

const gallery = ref<InstanceType<typeof PhotoGallery> | null>(null)
const coverButton = ref<HTMLButtonElement | null>(null)
const coverLabel = computed(() => {
  const alt = restaurant.data.value?.cover?.alt
  return alt ? `Open the cover photo: ${alt}` : 'Open the cover photo'
})

const directions = computed(() => {
  const l = restaurant.data.value?.location
  return l ? directionsLink(l) : null
})

const access = computed(() => (restaurant.data.value ? parentAccess(restaurant.data.value, currentUser.value?.id) : null))

const deleteDescription = computed(() => {
  const text = `${restaurant.data.value?.name} will be deleted permanently.`
  const own = currentUser.value ? (restaurant.data.value?.ownReviewCount ?? 0) : 0
  return own ? `${text} ${ownReviewsText(own)}` : text
})

const writePath = computed(() => `/restaurants/${props.restaurantId}/reviews/new`)

async function remove() {
  const name = restaurant.data.value?.name
  try {
    await restaurantsApi.deleteRestaurant({ restaurantId: props.restaurantId })
    toast.success('Restaurant deleted', { description: name })
    void router.push('/restaurants')
  } catch (error) {
    const failure = await apiFailure(error)
    const description =
      failure.problem?.code === 'other_reviews'
        ? 'A restaurant with reviews of other people cannot be deleted.'
        : failure.problem?.code === 'not_creator'
          ? 'You do not have permission to delete it.'
          : failureText(failure)
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
      <button
        v-if="restaurant.data.value.cover"
        ref="coverButton"
        type="button"
        class="focus-visible:outline-ring block overflow-hidden rounded-xl focus-visible:outline-2 focus-visible:outline-offset-2"
        :aria-label="coverLabel"
        @click="gallery?.openCover()"
      >
        <CoverPhoto :photo="restaurant.data.value.cover" size="hero" />
      </button>
      <section class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_24rem] lg:items-start">
        <div class="flex min-w-0 flex-col gap-4">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <h1 class="text-4xl font-extrabold tracking-tight break-words">{{ restaurant.data.value.name }}</h1>
            <div class="flex flex-wrap gap-2">
              <FavoriteButton
                :restaurant-id="restaurantId"
                :name="restaurant.data.value.name"
                :favorite="restaurant.data.value.favorite"
                labeled
                @change="restaurant.data.value.favorite = $event"
              />
              <SaveToListMenu :restaurant-id="restaurantId" />
              <template v-if="access?.canChange">
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
              </template>
            </div>
          </div>
          <p v-if="restaurant.data.value.description" class="text-muted-foreground max-w-2xl text-xl whitespace-pre-line">
            {{ restaurant.data.value.description }}
          </p>
          <p v-if="restaurant.data.value.location" class="flex flex-wrap items-baseline gap-x-4 gap-y-1">
            <a
              :href="osmLink(restaurant.data.value.location)"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-flex items-start gap-2 underline underline-offset-4"
            >
              <MapPinIcon class="text-muted-foreground mt-1 size-4 shrink-0" aria-hidden="true" />
              <span>
                <span class="sr-only">Address: </span>{{ restaurant.data.value.location.address
                }}<span class="sr-only">, open in OpenStreetMap</span>
              </span>
            </a>
            <!-- A geo: link opens an app. It needs no new tab. -->
            <a
              v-if="directions"
              :href="directions"
              :target="directions.startsWith('https:') ? '_blank' : undefined"
              rel="noopener noreferrer"
              class="text-muted-foreground rounded-sm text-sm font-medium underline underline-offset-4"
            >
              Directions
            </a>
            <button
              v-if="mapConfig && !wide"
              type="button"
              class="text-muted-foreground rounded-sm text-sm font-medium underline underline-offset-4"
              :aria-expanded="mapOpen"
              @click="mapOpen = !mapOpen"
            >
              {{ mapOpen ? 'Hide map' : 'Show map' }}
            </button>
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
          <MapView
            v-if="marker && mapConfig && !wide && mapOpen"
            :config="mapConfig"
            :label="`Map of ${restaurant.data.value.name}`"
            :center="marker"
            :zoom="16"
            :markers="[marker]"
            class="h-56"
          />
          <RestaurantDetails :restaurant="restaurant.data.value" />
          <ul v-if="tagList(restaurant.data.value.tags).length" class="flex flex-wrap gap-1.5" aria-label="Tags">
            <li v-for="tag in tagList(restaurant.data.value.tags)" :key="tag"><Badge variant="secondary">{{ tag }}</Badge></li>
          </ul>
          <section v-if="restaurant.data.value.rating" aria-label="Average scores">
            <ScoreSummary :scores="restaurant.data.value.rating" kind="restaurant">
              {{ formatCount(restaurant.data.value.reviewCount, 'review') }}
            </ScoreSummary>
          </section>
          <p class="text-muted-foreground text-sm">
            Added by {{ restaurant.data.value.creator?.name ?? 'Deleted user' }} ·
            <DateText :created="restaurant.data.value.created" :updated="restaurant.data.value.updated" />
          </p>
        </div>
        <MapView
          v-if="marker && mapConfig && wide"
          :config="mapConfig"
          :label="`Map of ${restaurant.data.value.name}`"
          :center="marker"
          :zoom="16"
          :markers="[marker]"
          class="h-72"
        />
      </section>

      <PhotoGallery ref="gallery" :parent="restaurant.data.value" @close-cover="coverButton?.focus()" />

      <section class="flex flex-col gap-6" aria-labelledby="reviews-heading">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <h2 id="reviews-heading" class="text-3xl font-semibold tracking-tight">Reviews</h2>
          <Button as-child>
            <RouterLink :to="writePath">Write a review</RouterLink>
          </Button>
        </div>
        <ReviewList :parent="restaurant.data.value" :write-path="writePath" />
      </section>
    </template>
  </main>
</template>
