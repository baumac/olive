<script setup lang="ts">
import type { Place } from '@reviews/sdk'
import { LocateFixedIcon, MapPinIcon } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import PlaceResults from '@/components/app/PlaceResults.vue'
import PlaceSearchCredit from '@/components/app/PlaceSearchCredit.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { MapPoint } from '@/components/ui/map'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { usePlaceSearch } from '@/composables/usePlaceSearch'
import type { PositionFailure } from '@/lib/preferredLocation'
import {
  currentPosition,
  defaultRadius,
  formatRadius,
  homePoint,
  preferredLocation,
  radiusOptions,
  setPreferredLocation,
} from '@/lib/preferredLocation'
import { pendingApproval, signedIn } from '@/lib/session'

// Reka UI Select needs a value for each item.
const anyDistance = 'any'

const open = ref(false)
const locating = ref(false)
const failure = ref('')
const canLocate = 'geolocation' in navigator
// The place search of the API needs an approved user.
const canSearch = computed(() => signedIn.value && !pendingApproval.value)
const { text, places, state, retryAfter } = usePlaceSearch(homePoint)

// locateRequest changes with each new location. A position of the device
// that comes later does not replace it.
let locateRequest = 0

function set(point: MapPoint, label: string) {
  locateRequest++
  const radius = preferredLocation.value ? preferredLocation.value.radius : defaultRadius
  setPreferredLocation({ latitude: point.latitude, longitude: point.longitude, label, radius })
  text.value = ''
  failure.value = ''
  open.value = false
}

const failures: Record<PositionFailure, string> = {
  denied: 'The browser did not give the location. Allow the location for this site in the browser settings.',
  insecure: 'The browser gives the location only to a site with HTTPS.',
  unavailable: 'The location of this device is not available. Try again.',
}

async function useCurrent() {
  const request = ++locateRequest
  locating.value = true
  failure.value = ''
  try {
    const point = await currentPosition()
    if (request === locateRequest) {
      set(point, 'Current location')
    }
  } catch (reason) {
    if (request === locateRequest) {
      failure.value = failures[reason as PositionFailure] ?? failures.unavailable
    }
  } finally {
    locating.value = false
  }
}

function select(place: Place) {
  set(place.location, place.name ?? place.location.address)
}

function setRadius(value: string) {
  const l = preferredLocation.value
  if (l) {
    setPreferredLocation({ ...l, radius: value === anyDistance ? null : Number(value) })
  }
}

function clear() {
  locateRequest++
  setPreferredLocation(null)
  open.value = false
}
</script>

<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button variant="outline" :aria-label="preferredLocation ? `Location: ${preferredLocation.label}` : undefined">
        <MapPinIcon aria-hidden="true" />
        <span class="max-w-40 truncate">{{ preferredLocation ? preferredLocation.label : 'Set location' }}</span>
      </Button>
    </PopoverTrigger>
    <PopoverContent align="start" class="grid max-h-(--reka-popover-content-available-height) w-80 gap-4 overflow-y-auto">
      <template v-if="preferredLocation">
        <p class="text-sm">
          <span class="text-muted-foreground block">Your location</span>
          <span class="font-medium">{{ preferredLocation.label }}</span>
        </p>
        <div class="grid gap-2">
          <Label for="location-distance">Distance</Label>
          <Select
            :model-value="preferredLocation.radius === null ? anyDistance : String(preferredLocation.radius)"
            @update:model-value="setRadius($event as string)"
          >
            <SelectTrigger id="location-distance" class="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="r in radiusOptions" :key="r" :value="String(r)">Within {{ formatRadius(r) }}</SelectItem>
              <SelectItem :value="anyDistance">Any distance</SelectItem>
            </SelectContent>
          </Select>
          <p class="text-muted-foreground text-sm">
            The list shows only the restaurants in this distance. The map shows the restaurants of its view.
          </p>
        </div>
      </template>
      <p v-else class="text-muted-foreground text-sm">
        Set your location to see the nearest restaurants first. The maps then open there. Only this browser keeps it.
      </p>
      <Button v-if="canLocate" type="button" variant="secondary" :disabled="locating" @click="useCurrent">
        <LocateFixedIcon aria-hidden="true" />
        {{ locating ? 'Getting the location' : 'Use my current location' }}
      </Button>
      <p v-if="failure" class="text-destructive text-sm" role="alert">{{ failure }}</p>
      <div v-if="canSearch" class="grid gap-2">
        <Label for="location-search">{{ preferredLocation ? 'Change to a place' : 'Or search for a place' }}</Label>
        <Input
          id="location-search"
          v-model="text"
          type="search"
          name="location-search"
          maxlength="255"
          autocomplete="off"
          placeholder="City, area, or address"
        />
        <p v-if="state === 'failed'" class="text-sm" role="status">The place search does not answer. Try again later.</p>
        <PlaceResults :places="places" :state="state" :retry-after="retryAfter" empty="No places found. Change the search." @select="select" />
        <PlaceSearchCredit />
      </div>
      <p v-else-if="!signedIn" class="text-muted-foreground text-sm">
        <RouterLink to="/sign-in" class="text-foreground font-medium underline underline-offset-4">Sign in</RouterLink>
        to search for a place.
      </p>
      <div v-if="preferredLocation">
        <Button type="button" variant="outline" size="sm" @click="clear">Clear location</Button>
      </div>
    </PopoverContent>
  </Popover>
</template>
