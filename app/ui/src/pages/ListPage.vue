<script setup lang="ts">
import type { Restaurant, SavedList } from '@reviews/sdk'
import { HeartIcon, ListIcon, MapIcon, PencilIcon, SlidersHorizontalIcon } from 'lucide-vue-next'
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import ErrorPage from '@/components/app/ErrorPage.vue'
import PagedList from '@/components/app/PagedList.vue'
import RestaurantCard from '@/components/app/RestaurantCard.vue'
import ReviewerLink from '@/components/app/ReviewerLink.vue'
import { Button } from '@/components/ui/button'
import { ConfirmDelete } from '@/components/ui/confirm-delete'
import { EmptyState } from '@/components/ui/empty-state'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { useLoad } from '@/composables/useLoad'
import { usePagedList } from '@/composables/usePagedList'
import { usePageTitle } from '@/composables/usePageTitle'
import { listsApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { forgetOwnLists, listFailureText } from '@/lib/lists'
import { useMapConfig } from '@/lib/mapConfig'
import { currentUser } from '@/lib/session'

const props = defineProps<{ listId: string }>()
const router = useRouter()

const list = useLoad<SavedList>(() => listsApi.getList({ listId: props.listId }))

const limit = 20
const restaurants = usePagedList<Restaurant>(async (cursor) => {
  const page = await listsApi.listListRestaurants({ listId: props.listId, limit, cursor })
  return { items: page.items, nextCursor: page.nextCursor }
})

watch(
  () => props.listId,
  () => {
    void list.load()
    void restaurants.reload()
  },
  { immediate: true },
)

usePageTitle(() => (list.data.value ? `${list.data.value.name} by ${list.data.value.creator.name}` : ''))

const isOwner = computed(() => !!list.data.value && currentUser.value?.id === list.data.value.creator.id)
const isFavorites = computed(() => list.data.value?.kind === 'favorites')
const mapConfig = useMapConfig()
const filtered = computed(() => `/restaurants?list=${encodeURIComponent(props.listId)}`)

// The API gives 400 for a malformed ID. The link then has a typo.
const failedStatus = computed(() => {
  const status = list.failed.value?.status ?? null
  return status === 400 ? 404 : status
})

const renaming = ref(false)
const name = ref('')
const renameError = ref('')
const saving = ref(false)
const renameButton = ref<InstanceType<typeof Button> | null>(null)

async function startRename() {
  name.value = list.data.value?.name ?? ''
  renameError.value = ''
  renaming.value = true
  await nextTick()
  document.getElementById('list-name')?.focus()
}

async function stopRename() {
  renaming.value = false
  await nextTick()
  ;(renameButton.value?.$el as HTMLElement | undefined)?.focus()
}

async function rename() {
  if (saving.value || name.value.trim() === '') {
    return
  }
  saving.value = true
  renameError.value = ''
  try {
    const updated = await listsApi.updateList({ listId: props.listId, savedListPatch: { name: name.value.trim() } })
    forgetOwnLists()
    list.data.value = updated
    await stopRename()
  } catch (e) {
    renameError.value = await listFailureText(e, 'The list was not renamed. Try again in a moment.')
  } finally {
    saving.value = false
  }
}

async function remove() {
  const data = list.data.value
  try {
    await listsApi.deleteList({ listId: props.listId })
    forgetOwnLists()
    toast.success('List deleted', { description: data?.name })
    void router.push(data ? `/reviewers/${data.creator.id}` : '/restaurants')
  } catch (e) {
    toast.error('List not deleted', { description: await listFailureText(e, 'Try again in a moment.') })
  }
}

function drop(restaurantId: string) {
  restaurants.items.value = restaurants.items.value.filter((r) => r.id !== restaurantId)
  if (list.data.value) {
    list.data.value = { ...list.data.value, itemCount: Math.max(0, list.data.value.itemCount - 1) }
  }
}

async function removeRestaurant(restaurant: Restaurant) {
  try {
    await listsApi.removeListRestaurant({ listRef: props.listId, restaurantId: restaurant.id })
    drop(restaurant.id)
  } catch (e) {
    toast.error('Restaurant not removed', { description: await listFailureText(e, 'Try again in a moment.') })
  }
}

// The heart of a card changes the favorites list of the signed in user.
function favoriteChanged(restaurant: Restaurant, favorite: boolean) {
  if (isOwner.value && isFavorites.value && !favorite) {
    drop(restaurant.id)
  }
}
</script>

<template>
  <ErrorPage v-if="list.failed.value" :status="failedStatus" @retry="list.load" />
  <main v-else id="main" class="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-8 sm:px-6">
    <div v-if="list.loading.value" aria-busy="true" aria-label="Loading list" class="flex flex-col gap-4">
      <Skeleton class="h-10 w-1/2" />
      <Skeleton class="h-6 w-40" />
    </div>
    <template v-else-if="list.data.value">
      <section class="flex flex-col gap-3">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <h1 class="flex min-w-0 items-center gap-3 text-4xl font-extrabold tracking-tight break-words">
            <component :is="isFavorites ? HeartIcon : ListIcon" class="text-muted-foreground size-8 shrink-0" aria-hidden="true" />
            {{ list.data.value.name }}
          </h1>
          <div v-if="isOwner && !isFavorites" class="flex flex-wrap gap-2">
            <Button ref="renameButton" type="button" variant="outline" :aria-expanded="renaming" @click="renaming ? stopRename() : startRename()">
              <PencilIcon aria-hidden="true" />Rename list
            </Button>
            <ConfirmDelete
              label="Delete list"
              title="Delete this list?"
              :description="`${list.data.value.name} will be deleted permanently. Its restaurants stay.`"
              @confirm="remove"
            />
          </div>
        </div>
        <form v-if="renaming" class="flex max-w-md flex-wrap gap-2" @submit.prevent="rename">
          <label for="list-name" class="sr-only">Name of the list</label>
          <Input id="list-name" v-model="name" class="min-w-40 flex-1" maxlength="255" autocomplete="off" :aria-invalid="!!renameError" />
          <Button type="submit" :disabled="saving">Save</Button>
          <Button type="button" variant="ghost" @click="stopRename">Cancel</Button>
          <p v-if="renameError" class="text-destructive w-full text-sm" role="alert">{{ renameError }}</p>
        </form>
        <p class="text-muted-foreground text-sm">
          A list by <ReviewerLink :creator="list.data.value.creator" class="text-foreground font-medium underline underline-offset-4" /> ·
          {{ list.data.value.itemCount ? formatCount(list.data.value.itemCount, 'restaurant') : 'No restaurants yet' }}
        </p>
        <div v-if="list.data.value.itemCount" class="flex flex-wrap gap-2">
          <Button as-child variant="outline" size="sm">
            <RouterLink :to="filtered"><SlidersHorizontalIcon aria-hidden="true" />Sort and filter</RouterLink>
          </Button>
          <Button v-if="mapConfig" as-child variant="outline" size="sm">
            <RouterLink :to="`${filtered}&view=map`"><MapIcon aria-hidden="true" />Show on map</RouterLink>
          </Button>
        </div>
      </section>

      <section aria-labelledby="list-restaurants-heading" class="flex flex-col gap-6">
        <h2 id="list-restaurants-heading" class="sr-only">Restaurants</h2>
        <PagedList
          :list="restaurants"
          noun="restaurants"
          list-class="lg:grid-cols-2 xl:grid-cols-3"
          skeleton-class="h-36 rounded-xl"
          :skeleton-count="3"
          unfiltered
        >
          <template #item="{ item }">
            <div class="flex h-full flex-col gap-2">
              <RestaurantCard :restaurant="item" heading="h3" class="flex-1" @favorite="favoriteChanged(item, $event)" />
              <Button
                v-if="isOwner && !isFavorites"
                type="button"
                variant="outline"
                size="sm"
                :aria-label="`Remove ${item.name} from the list`"
                @click="removeRestaurant(item)"
              >
                Remove
              </Button>
            </div>
          </template>
          <template #empty>
            <EmptyState
              :icon="isFavorites ? HeartIcon : ListIcon"
              title="No restaurants yet"
              :description="
                isOwner
                  ? isFavorites
                    ? 'Select the heart of a restaurant to add it here.'
                    : 'Open a restaurant and select Save to list to add it here.'
                  : `${list.data.value.creator.name} has not added a restaurant yet.`
              "
            >
              <template v-if="isOwner" #action>
                <Button as-child><RouterLink to="/restaurants">Go to restaurants</RouterLink></Button>
              </template>
            </EmptyState>
          </template>
        </PagedList>
      </section>
    </template>
  </main>
</template>
