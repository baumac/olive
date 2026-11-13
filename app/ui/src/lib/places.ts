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
