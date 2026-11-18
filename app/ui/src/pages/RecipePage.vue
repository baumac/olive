<script setup lang="ts">
import type { Recipe } from '@reviews/sdk'
import { PencilIcon } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import DateText from '@/components/app/DateText.vue'
import ErrorPage from '@/components/app/ErrorPage.vue'
import FavoriteButton from '@/components/app/FavoriteButton.vue'
import PhotoGallery from '@/components/app/PhotoGallery.vue'
import RecipeDetails from '@/components/app/RecipeDetails.vue'
import ReviewList from '@/components/app/ReviewList.vue'
import SaveToListMenu from '@/components/app/SaveToListMenu.vue'
import ScoreSummary from '@/components/app/ScoreSummary.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ConfirmDelete } from '@/components/ui/confirm-delete'
import { CoverPhoto } from '@/components/ui/cover-photo'
import { Skeleton } from '@/components/ui/skeleton'
import { useLoad } from '@/composables/useLoad'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, failureText, recipesApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { findOwnReviewId, parentAccess } from '@/lib/parents'
import { tagList } from '@/lib/reviews'
import { currentUser } from '@/lib/session'

const props = defineProps<{ recipeId: string }>()
const router = useRouter()

const recipe = useLoad<Recipe>(() => recipesApi.getRecipe({ recipeId: props.recipeId }))
watch(() => props.recipeId, () => void recipe.load(), { immediate: true })

usePageTitle(() => recipe.data.value?.name ?? '')

// ownReviewId is the review of the signed in user, if any. A user can have
// at most one review per recipe.
const ownReviewId = ref<string | null>(null)
const ownReviewLookup = ref<'loading' | 'done' | 'failed'>('loading')
watch(
  [() => props.recipeId, currentUser],
  async ([recipeId, user]) => {
    ownReviewId.value = null
    ownReviewLookup.value = 'done'
    if (!user) {
      return
    }
    ownReviewLookup.value = 'loading'
    try {
      ownReviewId.value = await findOwnReviewId('recipe', recipeId, user.id)
      ownReviewLookup.value = 'done'
    } catch {
      ownReviewLookup.value = 'failed'
    }
  },
  { immediate: true },
)

const gallery = ref<InstanceType<typeof PhotoGallery> | null>(null)
const coverButton = ref<HTMLButtonElement | null>(null)
const coverLabel = computed(() => {
  const alt = recipe.data.value?.cover?.alt
  return alt ? `Open the cover photo: ${alt}` : 'Open the cover photo'
})

const access = computed(() => {
  if (!recipe.data.value || ownReviewLookup.value === 'loading') {
    return null
  }
  const hasOwnReview = ownReviewLookup.value === 'failed' ? null : !!ownReviewId.value
  return parentAccess(recipe.data.value, currentUser.value?.id, hasOwnReview)
})

const deleteDescription = computed(() => {
  const text = `${recipe.data.value?.name} will be deleted permanently.`
  return ownReviewId.value ? `${text} Your review and its photos will be deleted too.` : text
})

const writePath = computed(() =>
  ownReviewId.value ? `/recipes/${props.recipeId}/reviews/${ownReviewId.value}/edit` : `/recipes/${props.recipeId}/reviews/new`,
)

async function remove() {
  const name = recipe.data.value?.name
  try {
    await recipesApi.deleteRecipe({ recipeId: props.recipeId })
    toast.success('Recipe deleted', { description: name })
    void router.push('/recipes')
  } catch (error) {
    const failure = await apiFailure(error)
    const description =
      failure.problem?.code === 'other_reviews'
        ? 'A recipe with reviews of other people cannot be deleted.'
        : failure.problem?.code === 'not_creator'
          ? 'You do not have permission to delete it.'
          : failureText(failure)
    toast.error('Recipe not deleted', { description })
  }
}
</script>

<template>
  <ErrorPage v-if="recipe.failed.value" :status="recipe.failed.value.status" @retry="recipe.load" />
  <main v-else id="main" class="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-8 sm:px-6">
    <div v-if="recipe.loading.value" aria-busy="true" aria-label="Loading recipe" class="flex flex-col gap-4">
      <Skeleton class="h-10 w-2/3" />
      <Skeleton class="h-6 w-full max-w-2xl" />
      <Skeleton class="h-6 w-1/2" />
    </div>
    <template v-else-if="recipe.data.value">
      <button
        v-if="recipe.data.value.cover"
        ref="coverButton"
        type="button"
        class="focus-visible:outline-ring block overflow-hidden rounded-xl focus-visible:outline-2 focus-visible:outline-offset-2"
        :aria-label="coverLabel"
        @click="gallery?.openCover()"
      >
        <CoverPhoto :photo="recipe.data.value.cover" size="hero" />
      </button>
      <section class="flex min-w-0 flex-col gap-4">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <h1 class="text-4xl font-extrabold tracking-tight break-words">{{ recipe.data.value.name }}</h1>
          <div class="flex flex-wrap gap-2">
            <FavoriteButton
              :recipe-id="recipeId"
              :name="recipe.data.value.name"
              :favorite="recipe.data.value.favorite"
              labeled
              @change="recipe.data.value.favorite = $event"
            />
            <SaveToListMenu :recipe-id="recipeId" />
            <template v-if="access?.canChange">
              <Button variant="outline" as-child>
                <RouterLink :to="`/recipes/${recipeId}/edit`"><PencilIcon aria-hidden="true" />Edit recipe</RouterLink>
              </Button>
              <ConfirmDelete
                v-if="!access.hasOtherReviews"
                label="Delete recipe"
                title="Delete this recipe?"
                :description="deleteDescription"
                @confirm="remove"
              />
            </template>
          </div>
        </div>
        <p v-if="recipe.data.value.description" class="text-muted-foreground max-w-2xl text-xl whitespace-pre-line">
          {{ recipe.data.value.description }}
        </p>
        <ul v-if="tagList(recipe.data.value.tags).length" class="flex flex-wrap gap-1.5" aria-label="Tags">
          <li v-for="tag in tagList(recipe.data.value.tags)" :key="tag"><Badge variant="secondary">{{ tag }}</Badge></li>
        </ul>
        <section v-if="recipe.data.value.rating" aria-label="Average scores">
          <ScoreSummary :scores="recipe.data.value.rating" kind="recipe">
            {{ formatCount(recipe.data.value.reviewCount, 'review') }}
          </ScoreSummary>
        </section>
        <p class="text-muted-foreground text-sm">
          Added by {{ recipe.data.value.creator?.name ?? 'Deleted user' }} ·
          <DateText :created="recipe.data.value.created" :updated="recipe.data.value.updated" />
        </p>
      </section>

      <RecipeDetails :recipe="recipe.data.value" />

      <PhotoGallery ref="gallery" :parent="recipe.data.value" kind="recipe" @close-cover="coverButton?.focus()" />

      <section class="flex flex-col gap-6" aria-labelledby="reviews-heading">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <h2 id="reviews-heading" class="text-3xl font-semibold tracking-tight">Reviews</h2>
          <Button as-child>
            <RouterLink :to="writePath">{{ ownReviewId ? 'Edit your review' : 'Write a review' }}</RouterLink>
          </Button>
        </div>
        <ReviewList :parent="recipe.data.value" kind="recipe" :write-path="writePath" />
      </section>
    </template>
  </main>
</template>
