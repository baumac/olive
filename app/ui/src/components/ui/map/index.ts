import { defineAsyncComponent } from 'vue'

export interface MapMarker {
  id: string
  latitude: number
  longitude: number
  // label is the accessible name of the marker.
  label: string
}

export interface MapPoint {
  latitude: number
  longitude: number
}

// MapView loads Leaflet. Pages without a map do not load it.
export const MapView = defineAsyncComponent(() => import('./MapView.vue'))
