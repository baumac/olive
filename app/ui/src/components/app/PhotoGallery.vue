<script setup lang="ts">
import { computed, ref } from 'vue'
import { PhotoGrid } from '@/components/ui/photo-grid'
import { PhotoViewer } from '@/components/ui/photo-viewer'
import { usePagedList } from '@/composables/usePagedList'
import { type Parent, type ParentPhoto, type ReviewKind, kinds } from '@/lib/reviewKinds'
import PagedList from './PagedList.vue'

// The gallery has the photos of all reviews of a restaurant or a recipe.
const props = withDefaults(defineProps<{ parent: Parent; kind?: ReviewKind }>(), { kind: 'restaurant' })

// closeCover tells the parent to focus the cover after the viewer closes.
const emit = defineEmits<{ closeCover: [] }>()

// 15 photos fill whole rows of 3 and of 5 columns.
const limit = 15

const list = usePagedList<ParentPhoto>(async (cursor) => {
  const page = await kinds[props.kind].listPhotos(props.parent.id, { limit, cursor })
  return { items: page.items, nextCursor: page.nextCursor }
})

// The page makes a new gallery for each restaurant or recipe that it loads.
if (props.parent.cover) {
  void list.reload()
}

const open = ref<number | null>(null)
const fromCover = ref(false)
// coverOnly shows only the cover in the viewer. The gallery can start with a
// different photo when a review changed after the parent loaded.
const coverOnly = ref(false)
const grid = ref<InstanceType<typeof PhotoGrid> | null>(null)

const photos = computed<ParentPhoto[]>(() => (coverOnly.value && props.parent.cover ? [props.parent.cover] : list.items.value))

function openGrid(index: number) {
  fromCover.value = false
  coverOnly.value = false
  open.value = index
}

function openCover() {
  const index = list.items.value.findIndex((p) => p.id === props.parent.cover?.id)
  fromCover.value = true
  coverOnly.value = index < 0
  open.value = Math.max(index, 0)
}

function onClose(index: number) {
  if (fromCover.value) {
    emit('closeCover')
  } else {
    grid.value?.focus(index)
  }
}

defineExpose({ openCover })
</script>

<template>
  <section v-if="parent.cover" class="flex flex-col gap-4" aria-labelledby="photos-heading">
    <h2 id="photos-heading" class="font-serif text-2xl leading-[30px] font-semibold">Photos</h2>
    <PagedList
      :list="list"
      noun="photos"
      list-class="grid-cols-3 gap-2 sm:grid-cols-5"
      skeleton-class="aspect-square w-full rounded-md"
      :skeleton-count="5"
      :focus-item="(i) => grid?.focus(i)"
    >
      <template #items="{ items }">
        <PhotoGrid ref="grid" :photos="items" :label="`Photos of ${parent.name}`" @open="openGrid" />
      </template>
    </PagedList>
    <PhotoViewer v-model:index="open" :photos="photos" @close="onClose">
      <template #caption="{ index }">
        <p v-if="photos[index]" class="text-muted-foreground text-sm">
          From the
          <RouterLink
            :to="`${kinds[kind].path}/${parent.id}/reviews/${photos[index].review}`"
            class="text-foreground rounded-sm font-medium underline underline-offset-4"
          >review by {{ photos[index].creator.name }}</RouterLink>
        </p>
      </template>
    </PhotoViewer>
  </section>
</template>
