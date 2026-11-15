import type { Place } from '@reviews/sdk'
import { onBeforeUnmount, ref, watch } from 'vue'
import type { MapPoint } from '@/components/ui/map'
import { apiFailure, placesApi } from '@/lib/api'
import { minPlaceSearch } from '@/lib/places'

export type PlaceSearchState = 'idle' | 'loading' | 'done' | 'limited' | 'failed'

// usePlaceSearch gets the places for the text after a pause in typing. The
// search prefers places near the point of center.
export function usePlaceSearch(center: () => MapPoint) {
  const text = ref('')
  const places = ref<Place[]>([])
  const state = ref<PlaceSearchState>('idle')
  // retryAfter is the wait in seconds after a 429.
  const retryAfter = ref(0)

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
        const { latitude, longitude } = center()
        const list = await placesApi.searchPlaces({ search, latitude, longitude })
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
        }
      }
    }, 400)
  })

  onBeforeUnmount(() => clearTimeout(timer))

  return { text, places, state, retryAfter }
}
