import { defineAsyncComponent } from 'vue'

export interface MapMarker {
  id: string
  latitude: number
  longitude: number
  // label is the accessible name of the marker.
  label: string
  // score makes the pin a score stamp. null is a stamp without reviews.
  // Without score, the marker is a plain pin. See decision 00008.
  score?: number | null
}

export interface MapPoint {
  latitude: number
  longitude: number
}

// MapView loads Leaflet. Pages without a map do not load it.
export const MapView = defineAsyncComponent(() => import('./MapView.vue'))
