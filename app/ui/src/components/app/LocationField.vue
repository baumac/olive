<script setup lang="ts">
import type { Location, Place } from '@reviews/sdk'
import { MapPinIcon } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import PlaceResults from '@/components/app/PlaceResults.vue'
import PlaceSearchCredit from '@/components/app/PlaceSearchCredit.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { MapView } from '@/components/ui/map'
import type { MapPoint } from '@/components/ui/map'
import { useMapConfig } from '@/lib/mapConfig'
import { usePlaceSearch } from '@/composables/usePlaceSearch'
import { parseCoordinate, roundCoordinate } from '@/lib/places'
import { homePoint } from '@/lib/preferredLocation'

defineProps<{
  id: string
  invalid?: boolean
  describedBy?: string
  // locked shows only the location. The API refuses a change.
  locked?: boolean
}>()

const location = defineModel<Location | null>({ default: null })
const emit = defineEmits<{ selected: [place: Place] }>()

const mapConfig = useMapConfig()
const byHand = ref(false)
const { text, places, state, retryAfter } = usePlaceSearch(homePoint)
const hand = ref({ address: '', latitude: '', longitude: '' })

watch(state, (next) => {
  if (next === 'failed') {
    enterByHand()
  }
})

function select(place: Place) {
  location.value = place.location
  emit('selected', place)
  text.value = ''
}

function enterByHand() {
  const l = location.value
  hand.value = l
    ? { address: l.address, latitude: String(l.latitude), longitude: String(l.longitude) }
    : { address: '', latitude: '', longitude: '' }
  mapCenter.value = l ?? homePoint()
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
const mapCenter = ref<MapPoint>(homePoint())
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
      <PlaceResults
        :places="places"
        :state="state"
        :retry-after="retryAfter"
        empty="No places found. Change the search, or enter the location by hand."
        @select="select"
      />
      <div class="flex flex-wrap items-center justify-between gap-2">
        <Button type="button" variant="outline" size="sm" @click="enterByHand">Enter the location by hand</Button>
        <PlaceSearchCredit />
      </div>
    </template>
  </div>
</template>
