<script setup lang="ts" generic="T extends { id: string }">
import type { PagedList } from '@/composables/usePagedList'
import { CircleAlertIcon } from 'lucide-vue-next'
import { nextTick } from 'vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { LoadMore } from '@/components/ui/load-more'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

const props = defineProps<{
  list: PagedList<T>
  // noun names the items in plural, such as "reviews".
  noun: string
  listClass?: string
  // itemClass sets the shape of the focus outline of an item.
  itemClass?: string
  skeletonClass: string
  skeletonCount: number
  // unfiltered is for a list without a search or filters.
  unfiltered?: boolean
  // focusItem moves the focus to an item of the items slot after a load.
  focusItem?: (index: number) => void
}>()

// The items slot shows all items in its own list in place of the item slot.
const slots = defineSlots<{
  item?(props: { item: T }): unknown
  items?(props: { items: T[] }): unknown
  empty?(): unknown
}>()

const title = props.noun.charAt(0).toUpperCase() + props.noun.slice(1)

function itemId(item: T) {
  return `${props.noun}-${item.id}`
}

// load moves focus to the first new item, so a keyboard user can go on.
async function load(cursor: string) {
  const start = props.list.items.value.length
  await props.list.loadMore(cursor)
  await nextTick()
  const first = props.list.items.value[start]
  if (!first) {
    return
  }
  if (props.focusItem) {
    props.focusItem(start)
  } else {
    document.getElementById(itemId(first))?.focus()
  }
}
</script>

<template>
  <Alert v-if="list.failed.value" variant="destructive">
    <CircleAlertIcon aria-hidden="true" />
    <AlertTitle>{{ title }} did not load</AlertTitle>
    <AlertDescription v-if="list.failed.value.detail && !unfiltered">
      {{ list.failed.value.detail }} Change the search or the filters.
    </AlertDescription>
    <AlertDescription v-else class="flex flex-wrap items-center gap-3">
      Check your connection and try again.
      <Button variant="outline" size="sm" @click="list.reload">Try again</Button>
    </AlertDescription>
  </Alert>

  <ul v-else-if="list.loading.value" aria-busy="true" :aria-label="`Loading ${noun}`" :class="cn('grid gap-4', listClass)">
    <li v-for="n in skeletonCount" :key="n"><Skeleton :class="skeletonClass" /></li>
  </ul>

  <slot v-else-if="list.items.value.length === 0" name="empty" />

  <template v-else>
    <slot v-if="slots.items" name="items" :items="list.items.value" />
    <ul v-else :class="cn('grid gap-4', listClass)">
      <li v-for="item in list.items.value" :id="itemId(item)" :key="item.id" tabindex="-1" :class="itemClass ?? 'rounded-xl'">
        <slot name="item" :item="item" />
      </li>
    </ul>
    <Alert v-if="list.moreFailed.value" variant="destructive">
      <CircleAlertIcon aria-hidden="true" />
      <AlertTitle>More {{ noun }} did not load</AlertTitle>
      <AlertDescription>Check your connection and try again.</AlertDescription>
    </Alert>
    <LoadMore :cursor="list.cursor.value" :loading="list.loadingMore.value" :label="`Load more ${noun}`" @load="load" />
  </template>
</template>
