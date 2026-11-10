<script setup lang="ts">
import type { Photo } from '@reviews/sdk'
import { ImageOffIcon } from 'lucide-vue-next'
import { ref } from 'vue'
import { Skeleton } from '@/components/ui/skeleton'
import { photoLabel } from '@/lib/photos'

const props = defineProps<{
  photos: Photo[]
  // label names the list for screen readers, such as "Photos of the review".
  label: string
}>()

defineEmits<{ open: [index: number] }>()

const state = ref<Record<string, 'loaded' | 'failed'>>({})
const list = ref<HTMLUListElement | null>(null)

// focus gives the focus back to a thumbnail after the viewer closes. It
// reads the buttons in DOM order. A ref array of v-for has no fixed order.
function focus(index: number) {
  list.value?.querySelectorAll<HTMLButtonElement>(':scope > li > button')[Math.min(index, props.photos.length - 1)]?.focus()
}

defineExpose({ focus })
</script>

<template>
  <ul v-if="photos.length" ref="list" class="grid grid-cols-3 gap-2 sm:grid-cols-5" :aria-label="label">
    <li v-for="(photo, i) in photos" :key="photo.id">
      <button
        type="button"
        class="bg-muted focus-visible:outline-ring relative block aspect-square w-full overflow-hidden rounded-md focus-visible:outline-2 focus-visible:outline-offset-2"
        :aria-label="photoLabel(photo, i, photos.length)"
        @click="$emit('open', i)"
      >
        <Skeleton v-if="!state[photo.id]" class="absolute inset-0 rounded-md" />
        <span v-if="state[photo.id] === 'failed'" class="text-muted-foreground absolute inset-0 flex items-center justify-center">
          <ImageOffIcon class="size-6" aria-hidden="true" />
        </span>
        <img
          v-else
          :src="photo.urls.thumb"
          :width="photo.width"
          :height="photo.height"
          alt=""
          loading="lazy"
          class="size-full object-cover"
          @load="state[photo.id] = 'loaded'"
          @error="state[photo.id] = 'failed'"
        >
      </button>
    </li>
  </ul>
</template>
