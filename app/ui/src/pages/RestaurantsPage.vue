<script setup lang="ts">
import type { Restaurant } from '@reviews/sdk'
import { ListIcon, MapIcon, PlusIcon, UtensilsIcon } from 'lucide-vue-next'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PagedList from '@/components/app/PagedList.vue'
import RestaurantCard from '@/components/app/RestaurantCard.vue'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { FilterBar } from '@/components/ui/filter-bar'
import type { MapPoint } from '@/components/ui/map'
import { MapView } from '@/components/ui/map'
import { searchRelevance, useListQuery } from '@/composables/useListQuery'
import { usePagedList } from '@/composables/usePagedList'
import { usePageTitle } from '@/composables/usePageTitle'
import { restaurantsApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import type { MapConfig } from '@/lib/mapConfig'
import { loadMapConfig } from '@/lib/mapConfig'
import type { Box } from '@/lib/places'
import { defaultCenter, mapBox } from '@/lib/places'
import { restaurantFilterDefs, restaurantFilterParams, restaurantSortOptions } from '@/lib/restaurantFilters'

usePageTitle('Restaurants')

const limit = 20
// mapLimit is the maximum limit of the API.
const mapLimit = 100

const route = useRoute()
const router = useRouter()
const mapConfig = ref<MapConfig | null>(null)
const mapConfigLoaded = ref(false)
void loadMapConfig().then((config) => {
  mapConfig.value = config
  mapConfigLoaded.value = true
})

const { filters, sort, active, sortOptions } = useListQuery(restaurantFilterDefs, restaurantSortOptions, '-created', searchRelevance, ['view'])

// The map view is in the URL. A link to the map then opens the map.
const mapView = computed(() => route.query.view === 'map' && mapConfig.value !== null)

function show(view: 'list' | 'map') {
  const query = { ...route.query }
  if (view === 'map') {
    query.view = 'map'
  } else {
    delete query.view
  }
  void router.replace({ query })
}

const list = usePagedList<Restaurant>(async (cursor) => {
  const page = await restaurantsApi.listRestaurants({
    limit,
    cursor,
    sort: sort.value,
    ...restaurantFilterParams(filters.value),
  })
  return { items: page.items, nextCursor: page.nextCursor }
})

// A link to the map view waits for the map config. The list then does not
// load a page that the map view hides.
watch(
  [filters, sort, mapView, mapConfigLoaded],
  () => {
    const waiting = route.query.view === 'map' && !mapConfigLoaded.value
    if (!mapView.value && !waiting) {
      void list.reload()
    }
  },
  { immediate: true },
)

const onMap = ref<Restaurant[]>([])
const mapLoading = ref(false)
const mapFailed = ref(false)
const mapHasMore = ref(false)
const selectedId = ref<string>()
const selected = computed(() => onMap.value.find((r) => r.id === selectedId.value))
const box = ref<Box>()
let mapRequest = 0
let moveTimer: ReturnType<typeof setTimeout> | undefined

async function loadMap() {
  if (!box.value) {
    return
  }
  const request = ++mapRequest
  mapLoading.value = true
  try {
    const page = await restaurantsApi.listRestaurants({
      limit: mapLimit,
      sort: sort.value,
      within: mapBox(box.value),
      ...restaurantFilterParams(filters.value),
    })
    if (request === mapRequest) {
      onMap.value = page.items
      mapHasMore.value = page.nextCursor !== null
      mapFailed.value = false
    }
  } catch {
    if (request === mapRequest) {
      mapFailed.value = true
    }
  } finally {
    if (request === mapRequest) {
      mapLoading.value = false
    }
  }
}

// The delay waits for the end of a pan or a zoom with more than one step.
function moved(next: Box, center: MapPoint, zoom: number) {
  box.value = next
  lastView.center = center
  lastView.zoom = zoom
  clearTimeout(moveTimer)
  moveTimer = setTimeout(() => void loadMap(), 300)
}

watch([filters, sort], () => mapView.value && void loadMap())
onBeforeUnmount(() => clearTimeout(moveTimer))

// startView does not change while the page is open. A new center would move
// the map and start a new load.
const startView = { ...lastView }

const markers = computed(() =>
  onMap.value.flatMap((r) => (r.location ? [{ id: r.id, latitude: r.location.latitude, longitude: r.location.longitude, label: r.name }] : [])),
)

const status = computed(() => {
  if (mapView.value) {
    if (mapLoading.value && onMap.value.length === 0) {
      return 'Loading restaurants'
    }
    const count = `${formatCount(onMap.value.length, 'restaurant')} on the map`
    return mapHasMore.value ? `${count}, zoom in to see more` : count
  }
  if (list.loading.value) {
    return 'Loading restaurants'
  }
  const count = formatCount(list.items.value.length, 'restaurant')
  return list.cursor.value ? `${count} shown, more to load` : count
})
</script>

<script lang="ts">
// lastView is the view of the map before the user left the page. The map
// opens with it again until a reload.
const lastView: { center: { latitude: number; longitude: number }; zoom: number } = { center: defaultCenter, zoom: 13 }
</script>

<template>
  <main id="main" class="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-8 sm:px-6">
    <h1 class="sr-only">Restaurants</h1>

    <FilterBar
      v-model:filters="filters"
      v-model:sort="sort"
      :defs="restaurantFilterDefs"
      :sort-options="sortOptions"
      sort-label="Sort by"
      :status="status"
    >
      <template #end>
        <div v-if="mapConfig" class="flex gap-1" role="group" aria-label="View">
          <Button :variant="mapView ? 'ghost' : 'secondary'" size="icon" aria-label="List" title="List" :aria-pressed="!mapView" @click="show('list')">
            <ListIcon aria-hidden="true" />
          </Button>
          <Button :variant="mapView ? 'secondary' : 'ghost'" size="icon" aria-label="Map" title="Map" :aria-pressed="mapView" @click="show('map')">
            <MapIcon aria-hidden="true" />
          </Button>
        </div>
        <Button as-child>
          <RouterLink to="/restaurants/new"><PlusIcon aria-hidden="true" />Add restaurant</RouterLink>
        </Button>
      </template>
    </FilterBar>

    <div v-if="mapView && mapConfig" class="grid gap-4">
      <MapView
        :config="mapConfig"
        label="Map of the restaurants"
        :center="startView.center"
        :zoom="startView.zoom"
        :markers="markers"
        :selected="selectedId"
        wheel-zoom
        class="h-[60vh] min-h-80"
        @move="moved"
        @select="selectedId = $event"
      />
      <p v-if="mapFailed" class="text-destructive text-sm" role="alert">The restaurants did not load. Move the map to try again.</p>
      <p class="text-muted-foreground text-sm">
        Select a pin to see the restaurant. Restaurants without a location are only in the list.
      </p>
      <div class="max-w-md" aria-live="polite">
        <RestaurantCard v-if="selected" :restaurant="selected" />
      </div>
    </div>

    <PagedList
      v-else
      :list="list"
      noun="restaurants"
      list-class="lg:grid-cols-2 xl:grid-cols-3"
      skeleton-class="h-36 rounded-xl"
      :skeleton-count="6"
    >
      <template #item="{ item }"><RestaurantCard :restaurant="item" /></template>
      <template #empty>
        <EmptyState
          v-if="active"
          :icon="UtensilsIcon"
          title="No restaurants match these filters"
          description="Remove a filter or clear all filters."
        >
          <template #action>
            <Button variant="outline" @click="filters = {}">Clear filters</Button>
          </template>
        </EmptyState>
        <EmptyState
          v-else
          :icon="UtensilsIcon"
          title="No restaurants yet"
          description="Add the first restaurant, then write a review of it."
        >
          <template #action>
            <Button as-child><RouterLink to="/restaurants/new">Add restaurant</RouterLink></Button>
          </template>
        </EmptyState>
      </template>
    </PagedList>
  </main>
</template>
