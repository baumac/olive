<script setup lang="ts">
import type { Photo, Review } from '@reviews/sdk'
import { ref } from 'vue'
import { toast } from 'vue-sonner'
import { PhotoGrid } from '@/components/ui/photo-grid'
import { PhotoViewer } from '@/components/ui/photo-viewer'
import { apiFailure, failureText, fieldErrors, restaurantsApi } from '@/lib/api'

const props = defineProps<{
  review: Review
  // editable gives the creator the edit and delete actions in the viewer.
  editable?: boolean
}>()

const emit = defineEmits<{ change: [photos: Photo[]] }>()

const open = ref<number | null>(null)
const saving = ref(false)
const error = ref('')
const grid = ref<InstanceType<typeof PhotoGrid> | null>(null)
const viewer = ref<InstanceType<typeof PhotoViewer> | null>(null)

const ids = () => ({ restaurantId: props.review.restaurant, reviewId: props.review.id })

async function saveAlt(photo: Photo, alt: string) {
  saving.value = true
  error.value = ''
  try {
    const saved = await restaurantsApi.updateReviewPhoto({ ...ids(), photoId: photo.id, photoPatch: { alt: alt || null } })
    emit('change', props.review.photos.map((p) => (p.id === saved.id ? saved : p)))
    viewer.value?.stopEdit()
    toast.success('Description saved')
  } catch (e) {
    const failure = await apiFailure(e)
    error.value =
      failure.status === 422
        ? (fieldErrors(failure.problem).alt ?? 'Check the description.')
        : failureText(failure, 'Description not saved. Try again in a moment.')
  } finally {
    saving.value = false
  }
}

async function remove(photo: Photo) {
  saving.value = true
  try {
    await restaurantsApi.deleteReviewPhoto({ ...ids(), photoId: photo.id })
    const rest = props.review.photos.filter((p) => p.id !== photo.id)
    open.value = rest.length ? Math.min(open.value ?? 0, rest.length - 1) : null
    emit('change', rest)
    toast.success('Photo deleted')
  } catch (e) {
    toast.error('Photo not deleted', { description: failureText(await apiFailure(e)) })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <PhotoGrid ref="grid" :photos="review.photos" :label="`Photos of ${review.title}`" @open="open = $event" />
  <PhotoViewer
    ref="viewer"
    v-model:index="open"
    :photos="review.photos"
    :editable="editable"
    :saving="saving"
    :error="error"
    @save-alt="saveAlt"
    @delete="remove"
    @close="grid?.focus($event)"
  />
</template>
