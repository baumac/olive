<script setup lang="ts">
import 'leaflet/dist/leaflet.css'
import type { HTMLAttributes } from 'vue'
import L from 'leaflet'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { MapConfig } from '@/lib/mapConfig'
import type { Box } from '@/lib/places'
import { cn } from '@/lib/utils'
import type { MapMarker, MapPoint } from '.'

const props = withDefaults(
  defineProps<{
    config: MapConfig
    // label is the accessible name of the map.
    label: string
    center: MapPoint
    zoom?: number
    markers?: MapMarker[]
    selected?: string
    // pick lets a click on the map and a drag of the marker select a point.
    pick?: boolean
    // wheelZoom is off for a map inside a page that scrolls.
    wheelZoom?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { zoom: 13, markers: () => [], selected: undefined, pick: false, wheelZoom: false, class: undefined },
)

const emit = defineEmits<{
  // move has the box of the view and the view itself.
  move: [box: Box, center: MapPoint, zoom: number]
  select: [id: string]
  pick: [point: MapPoint]
}>()

const el = ref<HTMLElement>()
let map: L.Map | undefined
const layer = L.layerGroup()

// The Content Security Policy of decision 00005 blocks data: URLs. Leaflet
// stops the load of a tile with a data: image. Tiles uses a blob in its
// place. It replaces the three methods of L.TileLayer that use the image.
const emptyGif = Uint8Array.from(atob('R0lGODlhAQABAAD/ACwAAAAAAQABAAACADs='), (c) => c.charCodeAt(0))
let emptyImageURL: string | undefined

function emptyImage(): string {
  emptyImageURL ??= URL.createObjectURL(new Blob([emptyGif], { type: 'image/gif' }))
  return emptyImageURL
}

interface LoadingTile {
  coords: { z: number }
  el: HTMLImageElement
}

type TileInternals = L.TileLayer & { _tiles: Record<string, LoadingTile>; _tileZoom?: number; _map?: L.Map }

interface GridInternals {
  _removeTile(this: TileInternals, key: string): void
  _tileReady(this: TileInternals, coords: unknown, err: unknown, tile?: HTMLImageElement): void
}

const grid = L.GridLayer.prototype as unknown as GridInternals

const Tiles = L.TileLayer.extend({
  _abortLoading(this: TileInternals) {
    for (const [key, tile] of Object.entries(this._tiles)) {
      if (tile.coords.z === this._tileZoom) {
        continue
      }
      tile.el.onload = null
      tile.el.onerror = null
      if (!tile.el.complete) {
        tile.el.src = emptyImage()
        tile.el.remove()
        delete this._tiles[key]
        this.fire('tileabort', { tile: tile.el, coords: tile.coords })
      }
    }
  },
  _removeTile(this: TileInternals, key: string) {
    const tile = this._tiles[key]
    if (!tile) {
      return
    }
    tile.el.setAttribute('src', emptyImage())
    grid._removeTile.call(this, key)
  },
  _tileReady(this: TileInternals, coords: unknown, err: unknown, tile?: HTMLImageElement) {
    if (!this._map || tile?.getAttribute('src') === emptyImage()) {
      return
    }
    grid._tileReady.call(this, coords, err, tile)
  },
}) as unknown as new (url: string, options: L.TileLayerOptions) => L.TileLayer

function escapeHtml(text: string): string {
  return text.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`)
}

const pin =
  '<svg viewBox="0 0 28 36" width="28" height="36" aria-hidden="true"><path d="M14 0C6.3 0 0 6.2 0 13.9 0 24 14 36 14 36s14-12 14-22.1C28 6.2 21.7 0 14 0z" fill="currentColor"/><circle cx="14" cy="14" r="5" fill="#fff"/></svg>'

function icon(selected: boolean) {
  return L.divIcon({
    html: pin,
    className: selected ? 'map-pin map-pin-selected' : 'map-pin',
    iconSize: [28, 36],
    iconAnchor: [14, 36],
  })
}

// drawn has the markers on the map by their ID. draw keeps the marker of
// an ID that stays. A marker with the keyboard focus then keeps it.
const drawn = new Map<string, { marker: L.Marker; selected: boolean; draggable: boolean }>()

function draw() {
  const ids = new Set(props.markers.map((m) => m.id))
  for (const [id, old] of drawn) {
    if (!ids.has(id) || old.draggable !== props.pick) {
      old.marker.remove()
      drawn.delete(id)
    }
  }
  for (const m of props.markers) {
    const selected = m.id === props.selected
    let entry = drawn.get(m.id)
    if (!entry) {
      const marker = L.marker([m.latitude, m.longitude], { icon: icon(selected), draggable: props.pick })
      marker.on('click', () => emit('select', m.id))
      // Leaflet opens only a popup with the keyboard. The marker has the
      // button role, so Enter and Space select it.
      marker.on('keypress', (e: L.LeafletKeyboardEvent) => {
        if (e.originalEvent.key === 'Enter' || e.originalEvent.key === ' ') {
          e.originalEvent.preventDefault()
          emit('select', m.id)
        }
      })
      marker.on('dragend', () => {
        const { lat, lng } = marker.getLatLng().wrap()
        emit('pick', { latitude: lat, longitude: lng })
      })
      marker.addTo(layer)
      entry = { marker, selected, draggable: props.pick }
      drawn.set(m.id, entry)
    } else {
      const at = entry.marker.getLatLng()
      if (at.lat !== m.latitude || at.lng !== m.longitude) {
        entry.marker.setLatLng([m.latitude, m.longitude])
      }
      if (entry.selected !== selected) {
        entry.marker.getElement()?.classList.toggle('map-pin-selected', selected)
        entry.selected = selected
      }
    }
    entry.marker.setZIndexOffset(selected ? 1000 : 0)
    const el = entry.marker.getElement()
    el?.setAttribute('aria-label', m.label)
    el?.setAttribute('title', m.label)
  }
}

onMounted(() => {
  if (!el.value) {
    return
  }
  map = L.map(el.value, {
    attributionControl: false,
    scrollWheelZoom: props.wheelZoom,
    worldCopyJump: true,
    minZoom: 2,
    maxZoom: 19,
  })
  const credit = '<a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noopener noreferrer">© OpenStreetMap contributors</a>'
  L.control
    .attribution({ prefix: false })
    .addAttribution([escapeHtml(props.config.attribution), credit].filter(Boolean).join(' '))
    .addTo(map)
  new Tiles(props.config.tiles, { maxZoom: 19 }).addTo(map)
  layer.addTo(map)
  map.on('moveend', () => {
    if (!map) {
      return
    }
    const b = map.getBounds()
    const c = map.getCenter().wrap()
    emit('move', { west: b.getWest(), south: b.getSouth(), east: b.getEast(), north: b.getNorth() }, { latitude: c.lat, longitude: c.lng }, map.getZoom())
  })
  map.on('click', (e) => {
    if (props.pick) {
      const { lat, lng } = e.latlng.wrap()
      emit('pick', { latitude: lat, longitude: lng })
    }
  })
  map.setView([props.center.latitude, props.center.longitude], props.zoom)
  draw()
})

watch(() => [props.markers, props.selected, props.pick], draw, { deep: true })
watch(
  () => [props.center.latitude, props.center.longitude] as const,
  ([latitude, longitude]) => map?.setView([latitude, longitude], map.getZoom()),
)

onBeforeUnmount(() => {
  map?.remove()
  map = undefined
})
</script>

<template>
  <!-- isolate keeps the z-index values of Leaflet inside the map. -->
  <div
    ref="el"
    role="region"
    :aria-label="label"
    :class="cn('border-border isolate h-64 w-full overflow-hidden rounded-md border', props.class)"
  />
</template>

<style>
/* The tiles are light in both themes. The pins therefore have fixed colors. */
.map-pin {
  color: oklch(0.45 0.12 125);
  filter: drop-shadow(0 1px 1px rgb(0 0 0 / 0.4));
}

.map-pin-selected {
  color: oklch(0.2 0.01 107);
}

.map-pin:focus-visible {
  outline: 3px solid var(--ring);
  outline-offset: 2px;
}
</style>
