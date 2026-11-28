<script setup lang="ts">
import type { PantryItem, RecipeSummary, Restaurant } from '@reviews/sdk'
import { ChefHatIcon, SearchIcon, UtensilsIcon } from 'lucide-vue-next'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { formatMinutes } from '@/lib/format'
import { pantryApi, recipesApi, restaurantsApi } from '@/lib/api'
import { categoryIcon, pantryItemMeta } from '@/lib/pantry'
import { distanceMeters, formatDistance, preferredLocation } from '@/lib/preferredLocation'
import { type ReviewKind, kinds } from '@/lib/reviewKinds'

type Subject = Restaurant | RecipeSummary | PantryItem

// SubjectPicker finds an existing restaurant, recipe, or pantry item for a
// new review. With a preferred location, the nearest restaurants come first.
// See decisions 00005 and 00008.
const props = defineProps<{ kind: ReviewKind }>()
const emit = defineEmits<{ select: [subject: Subject]; addNew: [] }>()

const info = computed(() => kinds[props.kind])
const limit = 8
const text = ref('')
const results = ref<Subject[]>([])
const loading = ref(false)
const failed = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined
let request = 0

async function search() {
  const r = ++request
  const query = text.value.trim()
  const origin = preferredLocation.value
  if (!query && (props.kind !== 'restaurant' || !origin)) {
    results.value = []
    loading.value = false
    failed.value = false
    return
  }
  loading.value = true
  failed.value = false
  try {
    const near = origin ? { latitude: origin.latitude, longitude: origin.longitude } : undefined
    const page =
      props.kind === 'recipe'
        ? await recipesApi.listRecipes({ limit, search: query })
        : props.kind === 'pantryItem'
          ? await pantryApi.listPantryItems({ limit, search: query })
          : await restaurantsApi.listRestaurants({ limit, search: query || undefined, near, sort: query ? undefined : 'distance' })
    if (r === request) {
      results.value = page.items
    }
  } catch {
    if (r === request) {
      failed.value = true
    }
  } finally {
    if (r === request) {
      loading.value = false
    }
  }
}

watch(text, () => {
  clearTimeout(timer)
  timer = setTimeout(() => void search(), 300)
})
watch(() => props.kind, () => {
  text.value = ''
  void search()
}, { immediate: true })
onBeforeUnmount(() => clearTimeout(timer))

function meta(item: Subject): string {
  if ('location' in item) {
    const origin = preferredLocation.value
    const distance = origin ? `${formatDistance(distanceMeters(origin, item.location))} · ` : ''
    return `${distance}${item.location.address}`
  }
  if ('category' in item) {
    return pantryItemMeta(item)
  }
  return item.totalTime ? formatMinutes(item.totalTime) : ''
}

function iconOf(item: Subject) {
  if ('category' in item) {
    return categoryIcon(item.category)
  }
  return props.kind === 'recipe' ? ChefHatIcon : UtensilsIcon
}
</script>

<template>
  <div class="grid gap-2">
    <label for="subject-search" class="text-sm font-semibold">{{ info.label }}</label>
    <div class="border-input bg-card focus-within:outline-ring flex h-11 items-center gap-2 rounded-full border px-3.5 focus-within:outline-2 focus-within:outline-offset-2">
      <SearchIcon class="text-muted-foreground size-4 shrink-0" aria-hidden="true" />
      <input
        id="subject-search"
        v-model="text"
        type="search"
        autocomplete="off"
        :placeholder="kind === 'pantryItem' ? 'Search the pantry' : `Search ${info.plural}`"
        aria-describedby="subject-results-status"
        class="placeholder:text-muted-foreground min-w-0 flex-1 bg-transparent text-base outline-none md:text-[15px]"
      >
    </div>
    <p id="subject-results-status" role="status" class="sr-only">
      {{ loading ? 'Searching' : text.trim() || results.length ? `${results.length} results` : '' }}
    </p>
    <p v-if="failed" class="text-destructive text-sm" role="alert">The search did not work. Try again in a moment.</p>
    <ul v-if="results.length" class="grid gap-2" :aria-label="kind === 'pantryItem' ? 'Pantry items' : kind === 'recipe' ? 'Recipes' : 'Restaurants'">
      <li v-for="item in results" :key="item.id">
        <button
          type="button"
          class="bg-card hover:border-ring flex min-h-14 w-full items-center gap-3 rounded-xl border p-2.5 text-left"
          @click="emit('select', item)"
        >
          <span class="bg-accent text-primary flex size-10 shrink-0 items-center justify-center rounded-lg">
            <component :is="iconOf(item)" class="size-5" aria-hidden="true" />
          </span>
          <span class="flex min-w-0 flex-col">
            <span class="font-serif text-[17px] font-semibold wrap-anywhere">{{ item.name }}</span>
            <span v-if="meta(item)" class="text-muted-foreground truncate text-[13px]">{{ meta(item) }}</span>
          </span>
        </button>
      </li>
    </ul>
    <p class="text-muted-foreground text-sm">
      Not in the app yet?
      <button type="button" class="text-primary font-semibold underline underline-offset-4" @click="emit('addNew')">
        Add a new {{ info.noun }}
      </button>
    </p>
  </div>
</template>
