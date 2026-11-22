import { ref } from 'vue'
import type { MapPoint } from '@/components/ui/map'
import { defaultCenter } from './places'

// PreferredLocation is the point that a person sets for the restaurant list
// and the maps. Only the browser keeps it. See decision 00008.
export interface PreferredLocation {
  latitude: number
  longitude: number
  label: string
  // radius is in meters. null shows the restaurants at each distance.
  radius: number | null
}

// The API allows a radius of at most 50 km.
export const radiusOptions = [1000, 5000, 10000, 25000, 50000]
export const defaultRadius = 10000

const key = 'reviews-location'

function read(): PreferredLocation | null {
  try {
    const v = JSON.parse(localStorage.getItem(key) ?? 'null') as Partial<PreferredLocation> | null
    if (
      !v ||
      typeof v.latitude !== 'number' ||
      Math.abs(v.latitude) > 90 ||
      typeof v.longitude !== 'number' ||
      Math.abs(v.longitude) > 180 ||
      typeof v.label !== 'string'
    ) {
      return null
    }
    const radius = radiusOptions.includes(v.radius as number) ? (v.radius as number) : null
    return { latitude: v.latitude, longitude: v.longitude, label: v.label, radius }
  } catch {
    return null
  }
}

export const preferredLocation = ref<PreferredLocation | null>(read())

export function setPreferredLocation(next: PreferredLocation | null) {
  preferredLocation.value = next
  try {
    if (next) {
      localStorage.setItem(key, JSON.stringify(next))
    } else {
      localStorage.removeItem(key)
    }
  } catch {
    // Without storage, the location is lost after a reload.
  }
}

// loadPreferredLocation reads the storage again. A test changes the storage
// after the module loads.
export function loadPreferredLocation() {
  preferredLocation.value = read()
}

// homePoint is the center of a map without a restaurant. The place search
// prefers places near it.
export function homePoint(): MapPoint {
  const l = preferredLocation.value
  return l ? { latitude: l.latitude, longitude: l.longitude } : defaultCenter
}

// insecure is a page without HTTPS. Browsers give it no position.
export type PositionFailure = 'denied' | 'insecure' | 'unavailable'

// currentPosition asks the browser for the position of the device. The
// browser asks the person for permission. It rejects with a PositionFailure.
export function currentPosition(): Promise<MapPoint> {
  return new Promise((resolve, reject) => {
    if (!('geolocation' in navigator)) {
      reject('unavailable' satisfies PositionFailure)
      return
    }
    if (window.isSecureContext === false) {
      reject('insecure' satisfies PositionFailure)
      return
    }
    navigator.geolocation.getCurrentPosition(
      (p) =>
        // 3 decimals are about 100 meters. The API then does not get the
        // exact position of the person.
        resolve({ latitude: Math.round(p.coords.latitude * 1e3) / 1e3, longitude: Math.round(p.coords.longitude * 1e3) / 1e3 }),
      (e) => reject((e.code === e.PERMISSION_DENIED ? 'denied' : 'unavailable') satisfies PositionFailure),
      { timeout: 10000, maximumAge: 300000 },
    )
  })
}

// distanceMeters is the great circle distance between two points.
export function distanceMeters(a: MapPoint, b: MapPoint): number {
  const rad = Math.PI / 180
  const dLat = (b.latitude - a.latitude) * rad
  const dLon = (b.longitude - a.longitude) * rad
  const h = Math.sin(dLat / 2) ** 2 + Math.cos(a.latitude * rad) * Math.cos(b.latitude * rad) * Math.sin(dLon / 2) ** 2
  return 2 * 6371000 * Math.asin(Math.min(1, Math.sqrt(h)))
}

// formatDistance does not show less than 100 meters. The position of a
// device has an error of about this size.
export function formatDistance(meters: number): string {
  if (meters < 100) {
    return 'Less than 100 m'
  }
  const rounded = Math.round(meters / 50) * 50
  if (rounded < 1000) {
    return `${rounded} m`
  }
  const km = Math.round(meters / 100) / 10
  return km < 10 ? `${km.toFixed(1)} km` : `${Math.round(meters / 1000)} km`
}

export function formatRadius(meters: number): string {
  return `${meters / 1000} km`
}
