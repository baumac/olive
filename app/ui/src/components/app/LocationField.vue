<script setup lang="ts">
import type { Location, Place } from '@reviews/sdk'
import { MapPinIcon } from 'lucide-vue-next'
import { onBeforeUnmount, ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { apiFailure, placesApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { defaultCenter, minPlaceSearch, parseCoordinate } from '@/lib/places'

defineProps<{
  id: string
  invalid?: boolean
  describedBy?: string
  // locked shows only the location. The API refuses a change.
  locked?: boolean
}>()

const location = defineModel<Location | null>({ default: null })
const emit = defineEmits<{ named: [name: string] }>()

const byHand = ref(false)
const text = ref('')
const places = ref<Place[]>([])
const state = ref<'idle' | 'loading' | 'done' | 'failed'>('idle')
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
      const { status } = await apiFailure(error)
      if (current === request && status !== 401) {
        places.value = []
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
  byHand.value = true
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
    <p v-if="location && (locked || !byHand)" class="flex items-start gap-2 text-sm">
      <MapPinIcon class="text-muted-foreground mt-0.5 size-4 shrink-0" aria-hidden="true" />
      <span><span class="sr-only">Selected location: </span>{{ location.address }}</span>
    </p>
    <template v-if="!locked && byHand">
      <p v-if="state === 'failed'" class="text-sm" role="status">
        The place search does not answer. Enter the location by hand.
      </p>
      <div class="grid gap-2">
        <Label :for="id">Address</Label>
        <Input
          :id="id"
          v-model="hand.address"
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
          <Input :id="`${id}-latitude`" v-model="hand.latitude" name="latitude" inputmode="decimal" placeholder="41.3766" />
        </div>
        <div class="grid gap-2">
          <Label :for="`${id}-longitude`">Longitude</Label>
          <Input :id="`${id}-longitude`" v-model="hand.longitude" name="longitude" inputmode="decimal" placeholder="2.1893" />
        </div>
      </div>
      <p class="text-muted-foreground text-sm">
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
