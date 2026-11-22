import { createHash } from 'node:crypto'
import type { Server } from 'node:http'
import { createServer } from 'node:http'

// tile is a gray PNG of 1 by 1 pixel.
const tile = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAAAAAA6fptVAAAACklEQVR4nGO4CwAA3wDeKXlwtgAAAABJRU5ErkJggg==', 'base64')

// startPhoton starts a fake of the Photon geocoder of decision 00010. It also
// gives a map tile for each path under /tiles. See decision 00010.
//
// A search with the word "nothing" finds no place. Each other search finds a
// restaurant with the name of the search and then an address. The restaurant
// of a search with the word "details" has the extra tags of decision 00005.
export async function startPhoton(port: number): Promise<{ url: string; stop: () => Promise<void> }> {
  const server: Server = createServer((req, res) => {
    if (req.url?.startsWith('/tiles/')) {
      res.setHeader('Content-Type', 'image/png')
      res.setHeader('Cache-Control', 'max-age=3600')
      res.end(tile)
      return
    }
    const q = new URL(req.url ?? '/', 'http://photon').searchParams.get('q') ?? ''
    res.setHeader('Content-Type', 'application/json')
    res.end(JSON.stringify({ features: /\bnothing\b/i.test(q) ? [] : placesOf(q) }))
  })
  await new Promise<void>((resolve, reject) => server.once('error', reject).listen(port, '127.0.0.1', resolve))
  return {
    url: `http://127.0.0.1:${port}`,
    stop: () => new Promise((resolve) => server.close(() => resolve())),
  }
}

// placesOf gives each search its own place ID and its own point. Tests with
// different searches then make no duplicates. The points are in the
// Southern Ocean, far from the locations that the tests set.
function placesOf(q: string) {
  const hash = createHash('sha256').update(q.toLowerCase()).digest()
  const latitude = -60 - hash.readUInt16BE(0) / 6554
  const longitude = hash.readUInt16BE(2) / 183 - 179
  const address = { street: 'Test Street', postcode: '08003', city: 'Testville', country: 'Testland' }
  const extra = /\bdetails\b/i.test(q)
    ? { extra: { 'contact:website': 'https://details.example', phone: '+34 932 21 50 12;+34 600 00 00 00', opening_hours: 'Tu-Sa 13:00-16:00; Su 13:00-16:00' } }
    : {}
  return [
    {
      geometry: { coordinates: [longitude, latitude] },
      properties: { ...address, name: q, housenumber: '4', osm_key: 'amenity', osm_value: 'restaurant', osm_type: 'N', osm_id: hash.readUInt32BE(4) + 1, ...extra },
    },
    {
      geometry: { coordinates: [longitude + 0.0001, latitude] },
      properties: { ...address, housenumber: '6', osm_key: 'building', osm_value: 'yes', osm_type: 'W', osm_id: hash.readUInt32BE(8) + 1 },
    },
  ]
}
