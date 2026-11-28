<script setup lang="ts">
import type { PantryItem } from '@reviews/sdk'
import { FactoryIcon, HistoryIcon, MapPinIcon, PencilIcon, PlusIcon, SparklesIcon } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import DateText from '@/components/app/DateText.vue'
import ErrorPage from '@/components/app/ErrorPage.vue'
import FavoriteButton from '@/components/app/FavoriteButton.vue'
import PhotoGallery from '@/components/app/PhotoGallery.vue'
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
import { apiFailure, failureText, pantryApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { categoryIcon, categoryLabel } from '@/lib/pantry'
import { ownReviewsText, parentAccess } from '@/lib/parents'
import { tagList } from '@/lib/reviews'
import { tagPath } from '@/lib/tags'
import { currentUser } from '@/lib/session'

const props = defineProps<{ pantryItemId: string }>()
const router = useRouter()

const item = useLoad<PantryItem>(() => pantryApi.getPantryItem({ pantryItemId: props.pantryItemId }))
// ownReviewCount and favorite belong to the signed in user.
watch([() => props.pantryItemId, () => currentUser.value?.id], () => void item.load(), { immediate: true })

usePageTitle(() => item.data.value?.name ?? '')

const gallery = ref<InstanceType<typeof PhotoGallery> | null>(null)
const coverButton = ref<HTMLButtonElement | null>(null)
const coverLabel = computed(() => {
  const alt = item.data.value?.cover?.alt
  return alt ? `Open the cover photo: ${alt}` : 'Open the cover photo'
})

const access = computed(() => (item.data.value ? parentAccess(item.data.value, currentUser.value?.id) : null))

// details has the optional details of the item that it has.
const details = computed(() => {
  const p = item.data.value
  if (!p) {
    return []
  }
  return [
    { label: 'Producer', value: p.producer, icon: FactoryIcon },
    { label: 'Origin', value: p.origin, icon: MapPinIcon },
    { label: 'Style', value: p.style, icon: SparklesIcon },
  ].filter((d) => d.value)
})

const deleteDescription = computed(() => {
  const text = `${item.data.value?.name} will be deleted permanently.`
  const own = currentUser.value ? (item.data.value?.ownReviewCount ?? 0) : 0
  return own ? `${text} ${ownReviewsText(own)}` : text
})

const writePath = computed(() => `/pantry/${props.pantryItemId}/reviews/new`)

async function remove() {
  const name = item.data.value?.name
  try {
    await pantryApi.deletePantryItem({ pantryItemId: props.pantryItemId })
    toast.success('Pantry item deleted', { description: name })
    void router.push('/pantry')
  } catch (error) {
    const failure = await apiFailure(error)
    const description =
      failure.problem?.code === 'other_reviews'
        ? 'A pantry item with tastings of other people cannot be deleted.'
        : failure.problem?.code === 'not_creator'
          ? 'You do not have permission to delete it.'
          : failureText(failure)
    toast.error('Pantry item not deleted', { description })
  }
}
</script>

<template>
  <ErrorPage v-if="item.failed.value" :status="item.failed.value.status" @retry="item.load" />
  <main v-else id="main" class="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-8 sm:px-6">
    <div v-if="item.loading.value" aria-busy="true" aria-label="Loading pantry item" class="flex flex-col gap-4">
      <Skeleton class="h-10 w-2/3" />
      <Skeleton class="h-6 w-full max-w-2xl" />
      <Skeleton class="h-6 w-1/2" />
    </div>
    <template v-else-if="item.data.value">
      <button
        v-if="item.data.value.cover"
        ref="coverButton"
        type="button"
        class="focus-visible:outline-ring block overflow-hidden rounded-xl focus-visible:outline-2 focus-visible:outline-offset-2"
        :aria-label="coverLabel"
        @click="gallery?.openCover()"
      >
        <CoverPhoto :photo="item.data.value.cover" size="hero" />
      </button>
      <section class="flex min-w-0 flex-col gap-4">
        <p class="kicker flex items-center gap-1.5">
          <component :is="categoryIcon(item.data.value.category)" class="size-4" aria-hidden="true" />
          {{ categoryLabel(item.data.value.category) }}
        </p>
        <div class="flex flex-wrap items-start justify-between gap-4">
          <h1 class="font-serif text-[34px] leading-10 font-semibold tracking-[-0.01em] wrap-anywhere md:text-[46px] md:leading-[50px]">{{ item.data.value.name }}</h1>
          <div class="flex flex-wrap gap-2">
            <FavoriteButton
              :pantry-item-id="pantryItemId"
              :name="item.data.value.name"
              :favorite="item.data.value.favorite"
              labeled
              @change="item.data.value.favorite = $event"
            />
            <SaveToListMenu :pantry-item-id="pantryItemId" />
            <template v-if="access?.canEdit">
              <Button variant="outline" as-child>
                <RouterLink :to="`/pantry/${pantryItemId}/edit`"><PencilIcon aria-hidden="true" />Edit pantry item</RouterLink>
              </Button>
              <Button variant="outline" as-child>
                <RouterLink :to="`/pantry/${pantryItemId}/history`"><HistoryIcon aria-hidden="true" />History</RouterLink>
              </Button>
              <ConfirmDelete
                v-if="access.isCreator && !access.hasOtherReviews"
                label="Delete pantry item"
                title="Delete this pantry item?"
                :description="deleteDescription"
                @confirm="remove"
              />
            </template>
          </div>
        </div>
        <dl v-if="details.length" class="text-muted-foreground flex flex-wrap gap-x-6 gap-y-2 text-sm">
          <div v-for="d in details" :key="d.label" class="flex min-w-0 items-center gap-1.5">
            <component :is="d.icon" class="size-4 shrink-0" aria-hidden="true" />
            <dt class="sr-only">{{ d.label }}</dt>
            <dd class="text-foreground wrap-anywhere">{{ d.value }}</dd>
          </div>
        </dl>
        <p v-if="item.data.value.description" class="text-muted-foreground max-w-2xl text-xl whitespace-pre-line">
          {{ item.data.value.description }}
        </p>
        <ul v-if="tagList(item.data.value.tags).length" class="flex flex-wrap gap-1.5" aria-label="Tags">
          <li v-for="tag in tagList(item.data.value.tags)" :key="tag">
            <Badge variant="secondary" as-child>
              <RouterLink :to="tagPath(tag)" class="hover:bg-secondary/80 inline-flex min-h-6 items-center">{{ tag }}</RouterLink>
            </Badge>
          </li>
        </ul>
        <section v-if="item.data.value.rating" aria-label="Average scores">
          <ScoreSummary :scores="item.data.value.rating" kind="pantryItem">
            {{ formatCount(item.data.value.reviewCount, 'tasting') }}
          </ScoreSummary>
        </section>
        <p class="text-muted-foreground text-sm">
          Added by {{ item.data.value.creator?.name ?? 'Deleted user' }} ·
          <DateText :created="item.data.value.created" :updated="item.data.value.updated" />
        </p>
      </section>

      <PhotoGallery ref="gallery" :parent="item.data.value" kind="pantryItem" @close-cover="coverButton?.focus()" />

      <section class="flex flex-col gap-6" aria-labelledby="reviews-heading">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <h2 id="reviews-heading" class="font-serif text-2xl leading-[30px] font-semibold">Every tasting</h2>
          <Button as-child>
            <RouterLink :to="writePath"><PlusIcon aria-hidden="true" />Add a tasting</RouterLink>
          </Button>
        </div>
        <ReviewList :parent="item.data.value" kind="pantryItem" :write-path="writePath" />
      </section>
    </template>
  </main>
</template>
