import type { Ref } from 'vue'
import { ref } from 'vue'

// MapConfig has the tile server of the maps. The app server gives it in
// /map-config.json. See decision 00018.
export interface MapConfig {
  // tiles is a URL template with {z}, {x}, and {y}.
  tiles: string
  // attribution is the credit of a tile server that is not from
  // OpenStreetMap.
  attribution: string
}

const osmTiles = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png'

let loading: Promise<MapConfig | null> | undefined

// loadMapConfig returns null when the maps are off. The Vite dev server has
// no /map-config.json. It then uses the tiles of OpenStreetMap.
export function loadMapConfig(): Promise<MapConfig | null> {
  loading ??= fetch('/map-config.json')
    .then(async (res) => {
      if (!res.ok) {
        throw new Error(`map config ${res.status}`)
      }
      const config = (await res.json()) as Partial<MapConfig>
      return config.tiles ? { tiles: config.tiles, attribution: config.attribution ?? '' } : null
    })
    .catch(() => (import.meta.env.DEV ? { tiles: osmTiles, attribution: '' } : null))
  return loading
}

// useMapConfig is null until the config loads, and when the maps are off.
export function useMapConfig(): Ref<MapConfig | null> {
  const config = ref<MapConfig | null>(null)
  void loadMapConfig().then((c) => (config.value = c))
  return config
}

// resetMapConfig lets a test load the config again.
export function resetMapConfig() {
  loading = undefined
}
