<script setup lang="ts">
import type { PantryItem, RecipeSummary, Restaurant } from '@reviews/sdk'
import { ChefHatIcon, PencilIcon, UtensilsIcon } from 'lucide-vue-next'
import { computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ConfirmDelete } from '@/components/ui/confirm-delete'
import DateText from '@/components/app/DateText.vue'
import EntryCard from '@/components/app/EntryCard.vue'
import PantryItemCard from '@/components/app/PantryItemCard.vue'
import RecipeCard from '@/components/app/RecipeCard.vue'
import RestaurantCard from '@/components/app/RestaurantCard.vue'
import ReviewDate from '@/components/app/ReviewDate.vue'
import ErrorPage from '@/components/app/ErrorPage.vue'
import ReviewerLink from '@/components/app/ReviewerLink.vue'
import ReviewPhotos from '@/components/app/ReviewPhotos.vue'
import ScoreSummary from '@/components/app/ScoreSummary.vue'
import VisitorNote from '@/components/app/VisitorNote.vue'
import { CoverPhoto } from '@/components/ui/cover-photo'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useLoad } from '@/composables/useLoad'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, failureText } from '@/lib/api'
import { pantryIcon } from '@/lib/pantry'
import { type AnyReview, type Parent, type ReviewKind, kinds, subjectKindOf } from '@/lib/reviewKinds'
import { currentUser } from '@/lib/session'

// The route gives a restaurantId, a recipeId, or a pantryItemId.
const props = defineProps<{ restaurantId?: string; recipeId?: string; pantryItemId?: string; reviewId: string }>()
const router = useRouter()

const kind = computed(() => kinds[subjectKindOf(props)])
const parentId = computed(() => (props.pantryItemId ?? props.recipeId ?? props.restaurantId)!)
const icons: Record<ReviewKind, typeof UtensilsIcon> = { restaurant: UtensilsIcon, recipe: ChefHatIcon, pantryItem: pantryIcon }

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
const firstPhoto = computed(() => review.value?.photos[0] ?? null)

// more has other reviews of the same reviewer and kind. See decision 00005.
const more = useLoad<AnyReview[]>(async () => {
  const r = review.value
  if (!r) {
    return []
  }
  const page = await kind.value.listReviewerReviews(r.creator.id, { limit: 3, sort: '-date' })
  return page.items.filter((i) => i.id !== r.id).slice(0, 2)
})
watch(() => review.value?.id, (id) => id && void more.load())

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
      <CoverPhoto v-if="firstPhoto" :photo="firstPhoto" size="hero" :alt="firstPhoto.alt ?? ''" class="rounded-2xl" />
      <div class="flex flex-col gap-2">
        <RouterLink :to="restaurantPath" class="kicker flex w-fit items-center gap-1.5 rounded-sm hover:underline">
          <component :is="icons[kind.kind]" class="size-4" aria-hidden="true" />{{ page.data.value.restaurant.name }}
        </RouterLink>
        <h1 class="font-serif text-[34px] leading-10 font-semibold tracking-[-0.01em] wrap-anywhere">{{ review.title }}</h1>
        <p class="text-muted-foreground text-sm">
          <ReviewerLink :creator="review.creator" /> · <ReviewDate :review="review" /> · Written
          <DateText :created="review.created" :updated="review.updated" />
        </p>
      </div>
      <section aria-label="Scores"><ScoreSummary :scores="review" :kind="kind.kind" /></section>
      <p v-if="review.feedback" class="text-[17px] leading-7 whitespace-pre-line">{{ review.feedback }}</p>
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
      <section class="flex flex-col gap-3" aria-labelledby="subject-heading">
        <h2 id="subject-heading" class="sr-only">The {{ kind.noun }}</h2>
        <RecipeCard v-if="kind.kind === 'recipe'" :recipe="page.data.value.restaurant as RecipeSummary" heading="h3" />
        <PantryItemCard v-else-if="kind.kind === 'pantryItem'" :item="page.data.value.restaurant as PantryItem" heading="h3" />
        <RestaurantCard v-else :restaurant="page.data.value.restaurant as Restaurant" heading="h3" />
      </section>
      <section v-if="more.data.value?.length" class="flex flex-col gap-3" aria-labelledby="more-heading">
        <h2 id="more-heading" class="font-serif text-[21px] leading-[26px] font-semibold">More from {{ review.creator.name }}</h2>
        <ul class="flex flex-col gap-3">
          <li v-for="e in more.data.value" :key="e.id"><EntryCard :review="e" variant="row" :show-creator="false" /></li>
        </ul>
      </section>
      <VisitorNote :name="review.creator.name" />
    </template>
  </main>
</template>
