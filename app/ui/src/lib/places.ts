import type { Location } from '@reviews/sdk'

// The place search prefers places near this point. It is the center of
// Barcelona.
export const defaultCenter = { latitude: 41.3874, longitude: 2.1686 }

// nearbyRadius is the distance in meters of the "Is it one of these?" check.
export const nearbyRadius = 75

export const minPlaceSearch = 3

export function sameLocation(a: Location | null, b: Location | null): boolean {
  if (!a || !b) {
    return a === b
  }
  return a.latitude === b.latitude && a.longitude === b.longitude && a.address === b.address && (a.placeId ?? '') === (b.placeId ?? '')
}

// parseCoordinate reads a decimal number with a point or a comma. It returns
// null for a value outside of -max to max.
export function parseCoordinate(text: string, max: number): number | null {
  const t = text.trim().replace(',', '.')
  if (!/^-?\d+(\.\d+)?$/.test(t)) {
    return null
  }
  const n = Number(t)
  return Math.abs(n) <= max ? n : null
}

// Box is the box of coordinates of the within filter of the API.
export interface Box {
  west: number
  south: number
  east: number
  north: number
}

function wrapLongitude(v: number): number {
  // The arithmetic below adds a small error. A value in the range stays.
  if (v >= -180 && v <= 180) {
    return v
  }
  const w = ((((v + 180) % 360) + 360) % 360) - 180
  // 180 stays 180. The east side of the world then is not -180.
  return w === -180 && v > 0 ? 180 : w
}

// mapBox changes the view of a map to a box for the API. A map can show
// longitudes outside of -180 to 180, and the world more than one time.
export function mapBox(view: Box): Box {
  const south = Math.max(-90, Math.min(90, view.south))
  const north = Math.max(-90, Math.min(90, view.north))
  if (view.east - view.west >= 360) {
    return { west: -180, south, east: 180, north }
  }
  return { west: wrapLongitude(view.west), south, east: wrapLongitude(view.east), north }
}

// roundCoordinate keeps 6 decimals. This is about 0.1 meters.
export function roundCoordinate(v: number): number {
  return Math.round(v * 1e6) / 1e6
}

// directionsLink has the point as the destination of a route. A phone opens
// its map app. Other devices open OpenStreetMap.
export function directionsLink(l: { latitude: number; longitude: number }, userAgent: string = navigator.userAgent): string {
  const point = `${l.latitude},${l.longitude}`
  if (/Android/.test(userAgent)) {
    return `geo:${point}?q=${point}`
  }
  // An iPad can send the user agent of a Mac.
  if (/iPhone|iPad|iPod|Macintosh/.test(userAgent)) {
    return `https://maps.apple.com/?daddr=${point}`
  }
  return `https://www.openstreetmap.org/directions?to=${encodeURIComponent(point)}`
}

// osmLink opens OpenStreetMap with a marker at the point.
export function osmLink(l: { latitude: number; longitude: number }): string {
  return `https://www.openstreetmap.org/?mlat=${l.latitude}&mlon=${l.longitude}#map=18/${l.latitude}/${l.longitude}`
}
