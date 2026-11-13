<script setup lang="ts">
import type { Location, Place } from '@reviews/sdk'
import { MapPinIcon } from 'lucide-vue-next'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { MapView } from '@/components/ui/map'
import type { MapPoint } from '@/components/ui/map'
import { apiFailure, placesApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { useMapConfig } from '@/lib/mapConfig'
import { defaultCenter, minPlaceSearch, parseCoordinate, roundCoordinate } from '@/lib/places'

defineProps<{
  id: string
  invalid?: boolean
  describedBy?: string
  // locked shows only the location. The API refuses a change.
  locked?: boolean
}>()

const location = defineModel<Location | null>({ default: null })
const emit = defineEmits<{ named: [name: string] }>()

const mapConfig = useMapConfig()
const byHand = ref(false)
const text = ref('')
const places = ref<Place[]>([])
const state = ref<'idle' | 'loading' | 'done' | 'limited' | 'failed'>('idle')
// retryAfter is the wait in seconds after a 429.
const retryAfter = ref(0)
const hand = ref({ address: '', latitude: '', longitude: '' })

let timer: ReturnType<typeof setTimeout> | undefined
let request = 0

// The delay keeps the number of requests to the geocoder low.
watch(text, (value) => {
  clearTimeout(timer)
  request++
  const search = value.trim()
  if (search.length < minPlaceSearch) {
    places.value = []
    state.value = 'idle'
    return
  }
  state.value = 'loading'
  const current = request
  timer = setTimeout(async () => {
    try {
      const list = await placesApi.searchPlaces({ search, ...defaultCenter })
      if (current === request) {
        places.value = list.items
        state.value = 'done'
      }
    } catch (error) {
      const failure = await apiFailure(error)
      if (current !== request) {
        return
      }
      places.value = []
      // A 401 opens the sign in page.
      if (failure.status === 401) {
        state.value = 'idle'
      } else if (failure.status === 429) {
        retryAfter.value = failure.retryAfter ?? 60
        state.value = 'limited'
      } else {
        state.value = 'failed'
        enterByHand()
      }
    }
  }, 400)
})

onBeforeUnmount(() => clearTimeout(timer))

function select(place: Place) {
  location.value = place.location
  if (place.name) {
    emit('named', place.name)
  }
  text.value = ''
}

function enterByHand() {
  const l = location.value
  hand.value = l
    ? { address: l.address, latitude: String(l.latitude), longitude: String(l.longitude) }
    : { address: '', latitude: '', longitude: '' }
  mapCenter.value = l ?? defaultCenter
  mapZoom.value = l ? 17 : 13
  byHand.value = true
}

// handPoint is the point of the fields by hand, or null if they are not
// valid.
const handPoint = computed<MapPoint | null>(() => {
  const latitude = parseCoordinate(hand.value.latitude, 90)
  const longitude = parseCoordinate(hand.value.longitude, 180)
  return latitude !== null && longitude !== null ? { latitude, longitude } : null
})

// mapCenter moves only for a typed point after a blur. A click on the map
// then does not move the map under the pointer.
const mapCenter = ref<MapPoint>(defaultCenter)
const mapZoom = ref(13)

function centerOnPoint() {
  if (handPoint.value) {
    mapCenter.value = handPoint.value
  }
}

function pick(point: MapPoint) {
  hand.value = { ...hand.value, latitude: String(roundCoordinate(point.latitude)), longitude: String(roundCoordinate(point.longitude)) }
}

// A changed location by hand has no place ID. The fields must all be valid.
watch(
  hand,
  (h) => {
    if (!byHand.value) {
      return
    }
    const latitude = parseCoordinate(h.latitude, 90)
    const longitude = parseCoordinate(h.longitude, 180)
    const address = h.address.trim()
    const l = location.value
    // Fields that are equal to the selected place keep its place ID.
    if (l && l.address === address && l.latitude === latitude && l.longitude === longitude) {
      return
    }
    location.value = address && latitude !== null && longitude !== null ? { address, latitude, longitude } : null
  },
  { deep: true },
)
</script>

<template>
  <div class="grid gap-3">
    <Input
      v-if="locked"
      :id="id"
      :model-value="location?.address ?? ''"
      name="address"
      readonly
      :aria-describedby="describedBy"
    />
    <p v-else-if="location && !byHand" class="flex items-start gap-2 text-sm">
      <MapPinIcon class="text-muted-foreground mt-0.5 size-4 shrink-0" aria-hidden="true" />
      <span><span class="sr-only">Selected location: </span>{{ location.address }}</span>
    </p>
    <MapView
      v-if="location && mapConfig && (locked || !byHand)"
      :config="mapConfig"
      label="Map of the selected place"
      :center="location"
      :zoom="16"
      :markers="[{ id: 'location', latitude: location.latitude, longitude: location.longitude, label: location.address }]"
      class="h-48"
    />
    <template v-if="!locked && byHand">
      <p v-if="state === 'failed'" class="text-sm" role="status">
        The place search does not answer. Enter the location by hand.
      </p>
      <div class="grid gap-2">
        <Label :id="`${id}-address-label`" :for="id">Address</Label>
        <!-- The form label of the field also targets this input. Its name is only Address. -->
        <Input
          :id="id"
          v-model="hand.address"
          :aria-labelledby="`${id}-address-label`"
          name="address"
          maxlength="255"
          placeholder="Carrer de Sant Carles 4, 08003 Barcelona"
          :aria-invalid="invalid ? 'true' : undefined"
          :aria-describedby="describedBy"
        />
      </div>
      <div class="grid gap-3 sm:grid-cols-2">
        <div class="grid gap-2">
          <Label :for="`${id}-latitude`">Latitude</Label>
          <Input
            :id="`${id}-latitude`"
            v-model="hand.latitude"
            name="latitude"
            inputmode="decimal"
            placeholder="41.3766"
            @blur="centerOnPoint"
          />
        </div>
        <div class="grid gap-2">
          <Label :for="`${id}-longitude`">Longitude</Label>
          <Input
            :id="`${id}-longitude`"
            v-model="hand.longitude"
            name="longitude"
            inputmode="decimal"
            placeholder="2.1893"
            @blur="centerOnPoint"
          />
        </div>
      </div>
      <template v-if="mapConfig">
        <MapView
          :config="mapConfig"
          label="Map to set the point"
          :center="mapCenter"
          :zoom="mapZoom"
          :markers="handPoint ? [{ id: 'point', ...handPoint, label: 'The point of the restaurant' }] : []"
          pick
          @pick="pick"
        />
        <p class="text-muted-foreground text-sm">
          Click the map or drag the pin to set the point. You can also type the two numbers.
        </p>
      </template>
      <p v-else class="text-muted-foreground text-sm">
        A map app shows the two numbers of a place. Latitude is from -90 to 90. Longitude is from -180 to 180.
      </p>
      <div>
        <Button type="button" variant="outline" size="sm" @click="byHand = false">Search for the place</Button>
      </div>
    </template>
    <template v-else-if="!locked">
      <Input
        :id="id"
        v-model="text"
        type="search"
        name="place-search"
        maxlength="255"
        autocomplete="off"
        :placeholder="location ? 'Search for a different place' : 'Search for the name or the address'"
        :aria-invalid="invalid ? 'true' : undefined"
        :aria-describedby="describedBy"
      />
      <p v-if="state === 'limited'" class="text-sm" role="status">
        Too many searches. Try again in {{ formatCount(retryAfter, 'second') }}.
      </p>
      <p class="sr-only" role="status">
        <template v-if="state === 'loading'">Searching</template>
        <template v-else-if="state === 'done'">{{ places.length ? `${formatCount(places.length, 'place')} found` : 'No places found' }}</template>
      </p>
      <ul v-if="places.length" class="divide-border divide-y rounded-md border" aria-label="Places">
        <li v-for="(place, i) in places" :key="i">
          <button
            type="button"
            class="hover:bg-accent focus-visible:ring-ring/50 w-full rounded-md px-3 py-2 text-left text-sm outline-none focus-visible:ring-[3px]"
            @click="select(place)"
          >
            <span v-if="place.name" class="block font-medium">{{ place.name }}</span>
            <span class="text-muted-foreground block">{{ place.location.address }}</span>
          </button>
        </li>
      </ul>
      <p v-else-if="state === 'done'" class="text-muted-foreground text-sm">
        No places found. Change the search, or enter the location by hand.
      </p>
      <div class="flex flex-wrap items-center justify-between gap-2">
        <Button type="button" variant="outline" size="sm" @click="enterByHand">Enter the location by hand</Button>
        <a
          href="https://www.openstreetmap.org/copyright"
          target="_blank"
          rel="noopener noreferrer"
          class="text-muted-foreground text-xs underline underline-offset-4"
        >
          Place search © OpenStreetMap contributors
        </a>
      </div>
    </template>
  </div>
</template>
