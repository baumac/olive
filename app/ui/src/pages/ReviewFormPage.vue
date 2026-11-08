<script setup lang="ts">
import type { Restaurant, Review, ReviewCreate, ReviewPatch } from '@reviews/sdk'
import { CircleAlertIcon } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import ErrorPage from '@/components/app/ErrorPage.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { FormControl, FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { RatingInput } from '@/components/ui/rating-input'
import { Skeleton } from '@/components/ui/skeleton'
import { TagInput } from '@/components/ui/tag-input'
import { Textarea } from '@/components/ui/textarea'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, fieldErrors, restaurantsApi } from '@/lib/api'
import { clearDraft, saveDraft, takeDraft } from '@/lib/drafts'
import { type ScoreField, scoreFields, scoresOf, tagList } from '@/lib/reviews'
import { currentUser } from '@/lib/session'
import { checkLongText, checkName } from '@/lib/validation'

const props = defineProps<{ restaurantId: string; reviewId?: string }>()
const router = useRouter()

const editing = computed(() => props.reviewId !== undefined)
usePageTitle(() => (editing.value ? 'Edit review' : 'Write a review'))

interface Fields {
  scores: Record<string, number | null>
  title: string
  feedback: string
  tags: string[]
}

const restaurant = ref<Restaurant | null>(null)
const fields = ref<Fields>({
  scores: { space: null, service: null, food: null, price: null, cleanliness: null },
  title: '',
  feedback: '',
  tags: [],
})
const errors = ref<Record<string, string>>({})
const alert = ref<{ title: string; description: string; link?: string } | null>(null)
const saving = ref(false)
const loading = ref(true)
const failed = ref<{ status: number | null } | null>(null)

const draftKey = computed(() => `review:${props.restaurantId}:${props.reviewId ?? 'new'}`)
const restaurantPath = computed(() => `/restaurants/${props.restaurantId}`)

onMounted(async () => {
  try {
    restaurant.value = await restaurantsApi.getRestaurant({ restaurantId: props.restaurantId })
    if (props.reviewId) {
      const review: Review = await restaurantsApi.getReview({ restaurantId: props.restaurantId, reviewId: props.reviewId })
      if (review.creator.id !== currentUser.value?.id) {
        failed.value = { status: 403 }
        return
      }
      for (const s of scoresOf(review)) {
        fields.value.scores[s.name] = s.value
      }
      fields.value.title = review.title
      fields.value.feedback = review.feedback ?? ''
      fields.value.tags = tagList(review.tags)
    } else if (currentUser.value) {
      // A user has at most one review per restaurant. The API refuses a
      // second one with 409. The page opens the existing review instead.
      const own = await restaurantsApi.listReviews({
        restaurantId: props.restaurantId,
        creatorEq: currentUser.value.id,
        limit: 1,
      })
      if (own.items[0]) {
        void router.replace(`${restaurantPath.value}/reviews/${own.items[0].id}/edit`)
        return
      }
    }
    const draft = takeDraft<Fields>(draftKey.value)
    if (draft) {
      // A draft from before the four scores has null scores. The loaded
      // scores then stay.
      const scores = { ...fields.value.scores }
      for (const s of scoreFields) {
        scores[s.name] = draft.scores?.[s.name] ?? scores[s.name] ?? null
      }
      fields.value = { ...draft, scores }
    }
  } catch (error) {
    failed.value = { status: (await apiFailure(error)).status }
  } finally {
    loading.value = false
  }
})

function validate(): boolean {
  const next: Record<string, string> = {}
  for (const s of scoreFields) {
    if (fields.value.scores[s.name] == null) {
      next[s.name] = `Select a ${s.label.toLowerCase()} score.`
    }
  }
  next.title = checkName(fields.value.title, 'Enter a title with at least one character that is not a space.')
  next.feedback = checkLongText(fields.value.feedback)
  next.tags = errors.value.tags ?? ''
  errors.value = next
  const order = [...scoreFields.map((s) => s.name), 'title', 'feedback', 'tags']
  const first = order.find((k) => next[k])
  if (first) {
    focusField(`review-${first}`)
  }
  return !first
}

// focusField focuses the field, or the selected or first radio of a
// rating group.
function focusField(id: string) {
  const el = document.getElementById(id)
  const target = el?.matches('input, textarea, button')
    ? el
    : el?.querySelector<HTMLElement>('[data-state=checked], button')
  target?.focus()
}

// validate runs first. Each score then has a value.
function scoreValues() {
  const { space, service, food, price, cleanliness } = fields.value.scores as Record<ScoreField['name'], number>
  return { space, service, food, price, cleanliness }
}

async function submit() {
  alert.value = null
  if (!validate()) {
    return
  }
  saving.value = true
  saveDraft(draftKey.value, fields.value)
  const { title, feedback, tags } = fields.value
  try {
    let saved: Review
    if (props.reviewId) {
      const reviewPatch: ReviewPatch = {
        ...scoreValues(),
        title,
        feedback: feedback || null,
        tags: tags.length ? new Set(tags) : null,
      }
      saved = await restaurantsApi.updateReview({ restaurantId: props.restaurantId, reviewId: props.reviewId, reviewPatch })
    } else {
      const reviewCreate: ReviewCreate = {
        ...scoreValues(),
        title,
        feedback: feedback || undefined,
        tags: tags.length ? new Set(tags) : undefined,
      }
      saved = await restaurantsApi.createReview({ restaurantId: props.restaurantId, reviewCreate })
    }
    clearDraft(draftKey.value)
    toast.success('Review saved', { description: saved.title })
    void router.push(restaurantPath.value)
  } catch (error) {
    const { status, problem } = await apiFailure(error)
    // A 401 opens the sign in page. The draft then waits for the return.
    if (status !== 401) {
      clearDraft(draftKey.value)
    }
    if (status === 422) {
      errors.value = fieldErrors(problem)
    } else if (status === 409) {
      alert.value = {
        title: 'You already reviewed this restaurant',
        description: 'Each person can write one review per restaurant. Edit your review instead.',
        link: restaurantPath.value,
      }
    } else if (status === 403) {
      alert.value = { title: 'You cannot change this review', description: 'Only the person who wrote it can change it.' }
    } else if (status === 404) {
      alert.value = { title: 'This review no longer exists', description: 'It or its restaurant was deleted.' }
    } else if (status !== 401) {
      alert.value = { title: 'Review not saved', description: 'Check your connection and try again in a moment.' }
    }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <ErrorPage v-if="failed" :status="failed.status" @retry="router.go(0)" />
  <main v-else id="main" class="mx-auto flex max-w-2xl flex-col gap-6 px-4 py-8 sm:px-6">
    <div class="flex flex-col gap-1">
      <h1 class="text-4xl font-extrabold tracking-tight">{{ editing ? 'Edit review' : 'Write a review' }}</h1>
      <p v-if="restaurant" class="text-muted-foreground text-xl">
        <RouterLink :to="restaurantPath" class="hover:underline">{{ restaurant.name }}</RouterLink>
      </p>
    </div>
    <div v-if="loading" aria-busy="true" aria-label="Loading review" class="grid gap-4">
      <Skeleton class="h-9 w-48" /><Skeleton class="h-16" /><Skeleton class="h-9" /><Skeleton class="h-24" />
    </div>
    <form v-else class="grid gap-6" novalidate @submit.prevent="submit">
      <Alert v-if="alert" variant="destructive">
        <CircleAlertIcon aria-hidden="true" />
        <AlertTitle>{{ alert.title }}</AlertTitle>
        <AlertDescription>
          {{ alert.description }}
          <RouterLink v-if="alert.link" :to="alert.link" class="font-medium underline underline-offset-4">
            Go to the restaurant
          </RouterLink>
        </AlertDescription>
      </Alert>

      <div class="grid gap-4">
        <div v-for="s in scoreFields" :key="s.name" class="grid gap-2">
          <span :class="['text-sm font-medium', errors[s.name] && 'text-destructive']" aria-hidden="true">{{ s.label }}</span>
          <RatingInput
            :id="`review-${s.name}`"
            v-model="fields.scores[s.name]"
            :max="s.max"
            :label="s.label"
            :invalid="!!errors[s.name]"
            :described-by="errors[s.name] ? `review-${s.name}-message` : undefined"
          />
          <p v-if="errors[s.name]" :id="`review-${s.name}-message`" class="text-destructive text-sm">
            {{ errors[s.name] }}
          </p>
        </div>
      </div>

      <FormItem id="review-title" :error="errors.title">
        <FormLabel>Title</FormLabel>
        <FormControl>
          <Input v-model="fields.title" name="title" maxlength="255" required placeholder="Best carbonara in town" />
        </FormControl>
        <FormMessage />
      </FormItem>
      <FormItem id="review-feedback" :error="errors.feedback" description>
        <FormLabel>Feedback</FormLabel>
        <FormControl>
          <Textarea v-model="fields.feedback" name="feedback" maxlength="10000" rows="6" />
        </FormControl>
        <FormDescription>Optional. What did you like, and what not?</FormDescription>
        <FormMessage />
      </FormItem>
      <FormItem id="review-tags" :error="errors.tags" description>
        <FormLabel>Tags</FormLabel>
        <TagInput
          id="review-tags"
          v-model="fields.tags"
          :invalid="!!errors.tags"
          :described-by="`review-tags-description${errors.tags ? ' review-tags-message' : ''}`"
          @error="errors = { ...errors, tags: $event }"
        />
        <FormDescription>Optional. Press Enter or comma after each tag, for example date-night.</FormDescription>
        <FormMessage />
      </FormItem>
      <div class="flex flex-wrap gap-2">
        <Button type="submit" :disabled="saving">{{ saving ? 'Saving…' : 'Save review' }}</Button>
        <Button variant="outline" as-child><RouterLink :to="restaurantPath">Cancel</RouterLink></Button>
      </div>
    </form>
  </main>
</template>
