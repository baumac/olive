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
    // cluster joins the pins that are near on the screen into one pin with
    // a count. A click on it zooms to its pins.
    cluster?: boolean
    // clusterLabel gives the accessible name of a pin with a count.
    clusterLabel?: (count: number) => string
    class?: HTMLAttributes['class']
  }>(),
  { zoom: 13, markers: () => [], selected: undefined, pick: false, wheelZoom: false, cluster: false, clusterLabel: (count: number) => `${count} places`, class: undefined },
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

// Leaflet stops the load of a tile with a data: image. The Content Security
// Policy blocks data: URLs. Tiles replaces the three methods that use it.
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

function icon(selected: boolean, shift: number) {
  return L.divIcon({
    html: pin,
    className: selected ? 'map-pin map-pin-selected' : 'map-pin',
    iconSize: [28, 36],
    iconAnchor: [14 - shift, 36],
  })
}

function clusterIcon(count: number) {
  return L.divIcon({ html: String(count), className: 'map-cluster', iconSize: [36, 36], iconAnchor: [18, 18] })
}

// clusterRadius is the distance in pixels below which two pins join.
const clusterRadius = 44
// spread is the distance in pixels between pins that the last zoom level
// cannot separate.
const spread = 22

// Pin is one marker on the map. A cluster has more than one member.
interface Pin {
  key: string
  latitude: number
  longitude: number
  label: string
  shift: number
  members: MapMarker[]
}

function single(m: MapMarker, shift = 0): Pin {
  return { key: m.id, latitude: m.latitude, longitude: m.longitude, label: m.label, shift, members: [m] }
}

function mean(values: number[]): number {
  return values.reduce((a, b) => a + b, 0) / values.length
}

function pins(): Pin[] {
  if (!map || !props.cluster) {
    return props.markers.map((m) => single(m))
  }
  const zoom = map.getZoom()
  const last = zoom >= map.getMaxZoom()
  const groups: { x: number; y: number; alone: boolean; members: MapMarker[] }[] = []
  for (const m of props.markers) {
    const { x, y } = map.project([m.latitude, m.longitude], zoom)
    // The selected pin stays out of a cluster.
    const alone = !last && m.id === props.selected
    const near = alone ? undefined : groups.find((g) => !g.alone && Math.hypot(g.x - x, g.y - y) <= clusterRadius)
    if (near) {
      near.members.push(m)
    } else {
      groups.push({ x, y, alone, members: [m] })
    }
  }
  return groups.flatMap(({ members }) => {
    if (members.length === 1 || last) {
      return members.map((m, i) => single(m, (i - (members.length - 1) / 2) * spread))
    }
    return [
      {
        key: `cluster:${members.map((m) => m.id).join(',')}`,
        latitude: mean(members.map((m) => m.latitude)),
        longitude: mean(members.map((m) => m.longitude)),
        label: props.clusterLabel(members.length),
        shift: 0,
        members,
      },
    ]
  })
}

function zoomTo(members: MapMarker[]) {
  if (!map) {
    return
  }
  // The zoom removes the cluster. The keyboard focus then goes to the map.
  const focused = map.getContainer().contains(document.activeElement)
  map.fitBounds(L.latLngBounds(members.map((m) => [m.latitude, m.longitude])), { padding: [48, 48], maxZoom: map.getMaxZoom() })
  if (focused) {
    map.getContainer().focus()
  }
}

// drawn has the markers on the map by their key. draw keeps the marker of
// a key that stays. A marker with the keyboard focus then keeps it.
const drawn = new Map<string, { marker: L.Marker; selected: boolean; draggable: boolean; shift: number }>()

function draw() {
  const next = pins()
  const shifts = new Map(next.map((p) => [p.key, p.shift]))
  for (const [key, old] of drawn) {
    if (shifts.get(key) !== old.shift || old.draggable !== props.pick) {
      old.marker.remove()
      drawn.delete(key)
    }
  }
  for (const p of next) {
    const cluster = p.members.length > 1
    const selected = p.key === props.selected
    let entry = drawn.get(p.key)
    if (!entry) {
      const marker = L.marker([p.latitude, p.longitude], {
        icon: cluster ? clusterIcon(p.members.length) : icon(selected, p.shift),
        draggable: props.pick && !cluster,
      })
      const activate = () => (cluster ? zoomTo(p.members) : emit('select', p.key))
      marker.on('click', activate)
      // Leaflet opens only a popup with the keyboard. Enter and Space select
      // the marker, like a button.
      marker.on('keypress', (e: L.LeafletKeyboardEvent) => {
        if (e.originalEvent.key === 'Enter' || e.originalEvent.key === ' ') {
          e.originalEvent.preventDefault()
          activate()
        }
      })
      marker.on('dragend', () => {
        const { lat, lng } = marker.getLatLng().wrap()
        emit('pick', { latitude: lat, longitude: lng })
      })
      marker.addTo(layer)
      entry = { marker, selected, draggable: props.pick, shift: p.shift }
      drawn.set(p.key, entry)
    } else {
      const at = entry.marker.getLatLng()
      if (at.lat !== p.latitude || at.lng !== p.longitude) {
        entry.marker.setLatLng([p.latitude, p.longitude])
      }
      if (entry.selected !== selected) {
        entry.marker.getElement()?.classList.toggle('map-pin-selected', selected)
        entry.selected = selected
      }
    }
    entry.marker.setZIndexOffset(selected ? 1000 : 0)
    const el = entry.marker.getElement()
    el?.setAttribute('aria-label', p.label)
    el?.setAttribute('title', p.label)
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
  map.on('zoomend', draw)
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
// The key is a string. A new center object with the same point then does
// not move the map. A move of the map would start a new load of the page.
watch(
  () => `${props.center.latitude},${props.center.longitude}`,
  () => map?.setView([props.center.latitude, props.center.longitude], map.getZoom()),
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

.map-cluster {
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid #fff;
  border-radius: 9999px;
  background: oklch(0.45 0.12 125);
  color: #fff;
  font-size: 0.875rem;
  font-weight: 600;
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.4);
}

.map-pin:focus-visible,
.map-cluster:focus-visible {
  outline: 3px solid var(--ring);
  outline-offset: 2px;
}
</style>
