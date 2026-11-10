<script setup lang="ts">
import type { Photo } from '@reviews/sdk'
import { ChevronLeftIcon, ChevronRightIcon, PencilIcon } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { ConfirmDelete } from '@/components/ui/confirm-delete'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog'
import { FormControl, FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { photoLabel } from '@/lib/photos'
import { checkAlt } from '@/lib/validation'

const props = defineProps<{
  photos: Photo[]
  // editable gives the creator the edit and delete actions.
  editable?: boolean
  saving?: boolean
  // error is the text of a failed save.
  error?: string
}>()

// index is the open photo. null closes the viewer.
const index = defineModel<number | null>('index', { required: true })

const emit = defineEmits<{
  saveAlt: [photo: Photo, alt: string]
  delete: [photo: Photo]
  close: [index: number]
}>()

const editing = ref(false)
// closed is the photo of the last close. The dialog shows it while it closes.
const closed = ref(0)
const shownIndex = computed(() => Math.min(index.value ?? closed.value, props.photos.length - 1))
const photo = computed(() => props.photos[shownIndex.value] ?? null)
const alt = ref('')
const altError = ref('')

watch(photo, () => {
  editing.value = false
  altError.value = ''
})

function move(step: number) {
  if (index.value === null || props.photos.length < 2) {
    return
  }
  index.value = (index.value + step + props.photos.length) % props.photos.length
}

function onKeydown(event: KeyboardEvent) {
  if (editing.value) {
    return
  }
  if (event.key === 'ArrowRight') {
    move(1)
  } else if (event.key === 'ArrowLeft') {
    move(-1)
  }
}

function setOpen(open: boolean) {
  if (!open && index.value !== null) {
    closed.value = index.value
    index.value = null
  }
}

// onCloseFocus runs after the focus trap ends. The parent can then move the
// focus to the thumbnail.
function onCloseFocus(event: Event) {
  event.preventDefault()
  emit('close', closed.value)
}

function startEdit() {
  alt.value = photo.value?.alt ?? ''
  altError.value = ''
  editing.value = true
}

function save() {
  altError.value = checkAlt(alt.value)
  if (!altError.value && photo.value) {
    emit('saveAlt', photo.value, alt.value)
  }
}

defineExpose({ stopEdit: () => (editing.value = false) })
</script>

<template>
  <Dialog :open="index !== null && photo !== null" @update:open="setOpen">
    <DialogContent v-if="photo" class="sm:max-w-3xl" @keydown="onKeydown" @close-auto-focus="onCloseFocus">
      <DialogTitle>Photo {{ shownIndex + 1 }} of {{ photos.length }}</DialogTitle>
      <DialogDescription :class="!photo.alt && 'sr-only'">{{ photo.alt || 'This photo has no description.' }}</DialogDescription>
      <img
        :src="photo.urls.full"
        :width="photo.width"
        :height="photo.height"
        :alt="photoLabel(photo, shownIndex, photos.length)"
        class="bg-muted mx-auto max-h-[60vh] w-auto rounded-md object-contain"
      >
      <div v-if="photos.length > 1" class="flex justify-between gap-2">
        <Button variant="outline" @click="move(-1)"><ChevronLeftIcon aria-hidden="true" />Previous photo</Button>
        <Button variant="outline" @click="move(1)">Next photo<ChevronRightIcon aria-hidden="true" /></Button>
      </div>
      <form v-if="editable && editing" class="grid gap-3" novalidate @submit.prevent="save">
        <FormItem id="photo-alt" :error="altError || error" description>
          <FormLabel>Description</FormLabel>
          <FormControl>
            <Input v-model="alt" name="alt" maxlength="255" />
          </FormControl>
          <FormDescription>Describe the photo for people who cannot see it.</FormDescription>
          <FormMessage />
        </FormItem>
        <div class="flex flex-wrap gap-2">
          <Button type="submit" :disabled="saving">{{ saving ? 'Saving…' : 'Save description' }}</Button>
          <Button type="button" variant="outline" @click="editing = false">Cancel</Button>
        </div>
      </form>
      <div v-else-if="editable" class="flex flex-wrap gap-2 border-t pt-4">
        <Button variant="outline" @click="startEdit"><PencilIcon aria-hidden="true" />Edit description</Button>
        <ConfirmDelete
          label="Delete photo"
          title="Delete this photo?"
          description="The photo will be deleted permanently."
          :disabled="saving"
          @confirm="$emit('delete', photo)"
        />
      </div>
    </DialogContent>
  </Dialog>
</template>
