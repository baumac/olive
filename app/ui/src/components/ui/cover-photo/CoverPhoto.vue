<script setup lang="ts">
import type { Photo } from '@reviews/sdk'
import { ImageOffIcon, UtensilsIcon } from 'lucide-vue-next'
import type { HTMLAttributes } from 'vue'
import { computed, ref, watch } from 'vue'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

const props = defineProps<{
  photo: Photo | null
  // card shows the thumb image. hero shows the full image.
  size: 'card' | 'hero'
  // alt is empty when the photo is part of a link or a button with a name.
  alt?: string
  class?: HTMLAttributes['class']
}>()

// thumbSide is the long side of the thumb variant in the API.
const thumbSide = 480

// srcset lets a narrow screen load the thumb in the hero.
const srcset = computed(() => {
  const p = props.photo
  if (props.size !== 'hero' || !p) {
    return undefined
  }
  const thumbWidth = Math.round(p.width * Math.min(1, thumbSide / Math.max(p.width, p.height)))
  return thumbWidth < p.width ? `${p.urls.thumb} ${thumbWidth}w, ${p.urls.full} ${p.width}w` : undefined
})

const state = ref<'loading' | 'loaded' | 'failed'>('loading')
watch(
  () => props.photo?.id,
  () => (state.value = 'loading'),
)
</script>

<template>
  <div
    data-slot="cover-photo"
    :class="cn('bg-muted text-muted-foreground relative overflow-hidden', size === 'card' ? 'aspect-video' : 'aspect-video sm:aspect-[21/9]', props.class)"
  >
    <span v-if="!photo" class="absolute inset-0 flex items-center justify-center">
      <UtensilsIcon class="size-8" aria-hidden="true" />
    </span>
    <template v-else>
      <Skeleton v-if="state === 'loading'" class="absolute inset-0 rounded-none" />
      <span v-if="state === 'failed'" class="absolute inset-0 flex items-center justify-center">
        <ImageOffIcon class="size-8" aria-hidden="true" />
      </span>
      <img
        v-else
        :src="size === 'card' ? photo.urls.thumb : photo.urls.full"
        :srcset="srcset"
        :sizes="srcset ? '(min-width: 72rem) 69rem, 100vw' : undefined"
        :width="photo.width"
        :height="photo.height"
        :alt="alt ?? ''"
        :loading="size === 'card' ? 'lazy' : 'eager'"
        class="relative size-full object-cover"
        @load="state = 'loaded'"
        @error="state = 'failed'"
      >
    </template>
  </div>
</template>
