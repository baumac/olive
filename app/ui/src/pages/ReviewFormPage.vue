<script setup lang="ts">
import type { Restaurant, Review, ReviewCreate, ReviewPatch } from '@reviews/sdk'
import { CircleAlertIcon, XIcon } from 'lucide-vue-next'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import ErrorPage from '@/components/app/ErrorPage.vue'
import ReviewPhotos from '@/components/app/ReviewPhotos.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { FormControl, FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { PhotoUpload } from '@/components/ui/photo-upload'
import { RatingInput } from '@/components/ui/rating-input'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, fieldErrors, restaurantsApi } from '@/lib/api'
import { clearDraft, saveDraft, takeDraft } from '@/lib/drafts'
import { addPhoto, keepPendingPhotos, takePendingPhotos } from '@/lib/photos'
import { type ScoreField, scoreFields, scoresOf } from '@/lib/reviews'
import { currentUser } from '@/lib/session'
import { checkLongText, checkName, maxPhotos } from '@/lib/validation'

const props = defineProps<{ restaurantId: string; reviewId?: string }>()
// busyRetryAfter is the longest Retry-After in seconds of a busy server.
const busyRetryAfter = 10
const router = useRouter()

const editing = computed(() => props.reviewId !== undefined)
usePageTitle(() => (editing.value ? 'Edit review' : 'Write a review'))

interface Fields {
  scores: Record<string, number | null>
  title: string
  feedback: string
}

const restaurant = ref<Restaurant | null>(null)
const fields = ref<Fields>({
  scores: { space: null, service: null, food: null, price: null, cleanliness: null },
  title: '',
  feedback: '',
})
const errors = ref<Record<string, string>>({})
const alert = ref<{ title: string; description: string; link?: string } | null>(null)
const saving = ref(false)
const loading = ref(true)
const failed = ref<{ status: number | null } | null>(null)

// saved is the review in the edit mode. Its photos change at once.
const saved = ref<Review | null>(null)
// queued has the photos of a new review. They upload after the create.
const queued = ref<{ file: File; alt: string; url: string }[]>([])
const uploading = ref(false)
const uploadError = ref('')
const upload = ref<InstanceType<typeof PhotoUpload> | null>(null)
const photoCount = computed(() => (saved.value?.photos.length ?? 0) + queued.value.length)

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
      saved.value = review
      for (const s of scoresOf(review)) {
        fields.value.scores[s.name] = s.value
      }
      fields.value.title = review.title
      fields.value.feedback = review.feedback ?? ''
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
    if (!props.reviewId) {
      for (const p of takePendingPhotos(draftKey.value)) {
        queued.value.push({ ...p, url: URL.createObjectURL(p.file) })
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
      fields.value = { title: draft.title, feedback: draft.feedback, scores }
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
  errors.value = next
  const order = [...scoreFields.map((s) => s.name), 'title', 'feedback']
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

async function addSelected(file: File, alt: string) {
  uploadError.value = ''
  if (!saved.value) {
    queued.value.push({ file, alt, url: URL.createObjectURL(file) })
    upload.value?.reset()
    return
  }
  uploading.value = true
  try {
    const { photo, altSaved } = await addPhoto(props.restaurantId, saved.value.id, file, alt)
    saved.value.photos = [...saved.value.photos, photo]
    upload.value?.reset()
    if (altSaved) {
      toast.success('Photo added')
    } else {
      toast.error('Description not saved', { description: 'The photo is added. Open it and add the description again.' })
    }
  } catch (error) {
    uploadError.value = await uploadFailure(error)
  } finally {
    uploading.value = false
  }
}

async function uploadFailure(error: unknown): Promise<string> {
  const { status, problem, retryAfter } = await apiFailure(error)
  if (status === 409) {
    return `A review can have at most ${maxPhotos} photos.`
  }
  if (status === 413 || status === 415 || status === 422) {
    return problem?.detail ?? 'Select a JPEG, PNG, or WebP image of at most 10 MB.'
  }
  // The rate limit of the user can wait up to one hour.
  if (status === 429 && retryAfter !== null && retryAfter <= busyRetryAfter) {
    return 'Many photos are uploading now. Try again in a few seconds.'
  }
  if (status === 429) {
    return 'You added many photos in a short time. Try again later.'
  }
  return status === 401 ? 'Sign in again, then try again.' : 'Photo not added. Try again in a moment.'
}

function unqueue(index: number) {
  const [photo] = queued.value.splice(index, 1)
  if (photo) {
    URL.revokeObjectURL(photo.url)
  }
}

onBeforeUnmount(() => queued.value.forEach((p) => URL.revokeObjectURL(p.url)))

// uploadQueued adds the photos of a new review. It tries each photo and
// returns false if one failed. The failed photos then are not on the review.
async function uploadQueued(reviewId: string): Promise<boolean> {
  const failures: string[] = []
  let altFailures = 0
  for (const photo of [...queued.value]) {
    try {
      const { altSaved } = await addPhoto(props.restaurantId, reviewId, photo.file, photo.alt)
      altFailures += altSaved ? 0 : 1
    } catch (error) {
      failures.push(await uploadFailure(error))
    }
    unqueue(queued.value.indexOf(photo))
  }
  if (failures.length) {
    const title = failures.length === 1 ? '1 photo not added' : `${failures.length} photos not added`
    toast.error(title, { description: `${[...new Set(failures)].join(' ')} Add the photos again on this page.` })
  }
  if (altFailures) {
    toast.error('Description not saved', { description: 'Open the photo and add the description again.' })
  }
  return failures.length === 0 && altFailures === 0
}

async function submit() {
  alert.value = null
  if (!validate()) {
    return
  }
  saving.value = true
  saveDraft(draftKey.value, fields.value)
  const { title, feedback } = fields.value
  try {
    let result: Review
    if (props.reviewId) {
      const reviewPatch: ReviewPatch = {
        ...scoreValues(),
        title,
        feedback: feedback || null,
      }
      result = await restaurantsApi.updateReview({ restaurantId: props.restaurantId, reviewId: props.reviewId, reviewPatch })
    } else {
      const reviewCreate: ReviewCreate = {
        ...scoreValues(),
        title,
        feedback: feedback || undefined,
      }
      result = await restaurantsApi.createReview({ restaurantId: props.restaurantId, reviewCreate })
    }
    clearDraft(draftKey.value)
    toast.success('Review saved', { description: result.title })
    if (!(await uploadQueued(result.id))) {
      // The edit page shows the saved review. The person can add the photo again.
      void router.push(`${restaurantPath.value}/reviews/${result.id}/edit`)
      return
    }
    void router.push(restaurantPath.value)
  } catch (error) {
    const { status, problem } = await apiFailure(error)
    // A 401 opens the sign in page. The draft then waits for the return.
    if (status !== 401) {
      clearDraft(draftKey.value)
    } else if (queued.value.length) {
      keepPendingPhotos(draftKey.value, queued.value.map(({ file, alt }) => ({ file, alt })))
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
            :description="s.description"
            :lowest="s.lowest"
            :highest="s.highest"
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
      <div class="flex flex-wrap gap-2">
        <Button type="submit" :disabled="saving || uploading">{{ saving ? 'Saving…' : 'Save review' }}</Button>
        <Button variant="outline" as-child><RouterLink :to="restaurantPath">Cancel</RouterLink></Button>
      </div>
    </form>
    <section v-if="!loading" class="grid gap-4 border-t pt-6" aria-labelledby="review-photos-title">
      <div class="grid gap-1">
        <h2 id="review-photos-title" class="text-xl font-semibold">Photos</h2>
        <p class="text-muted-foreground text-sm">
          {{ saved ? `Optional. Up to ${maxPhotos} photos. Each photo is added at once.` : `Optional. Up to ${maxPhotos} photos. They are added when you save the review.` }}
        </p>
      </div>
      <ReviewPhotos v-if="saved" :review="saved" editable @change="saved.photos = $event" />
      <ul v-if="queued.length" class="grid grid-cols-3 gap-2 sm:grid-cols-5" aria-label="Photos to add">
        <li v-for="(photo, i) in queued" :key="photo.url" class="relative">
          <img :src="photo.url" :alt="photo.alt || `Photo ${i + 1} to add`" class="bg-muted aspect-square w-full rounded-md object-cover">
          <Button
            type="button"
            variant="outline"
            size="icon"
            class="absolute top-1 right-1 size-7"
            :aria-label="`Remove photo ${i + 1}`"
            @click="unqueue(i)"
          >
            <XIcon aria-hidden="true" />
          </Button>
        </li>
      </ul>
      <PhotoUpload
        v-if="photoCount < maxPhotos"
        ref="upload"
        :action-label="saved ? 'Upload photo' : 'Add to review'"
        :busy="uploading"
        :error="uploadError"
        @add="addSelected"
      />
      <p v-else class="text-muted-foreground text-sm">This review has {{ maxPhotos }} photos. Delete one to add another.</p>
    </section>
  </main>
</template>
