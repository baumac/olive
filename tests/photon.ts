import { createHash } from 'node:crypto'
import type { Server } from 'node:http'
import { createServer } from 'node:http'

// startPhoton starts a fake of the Photon geocoder of decision 00017. The
// app of the tests gets its places from it. The tests then need no network.
// A search with the word "nothing" finds no place. Each other search finds a
// restaurant with the name of the search and then an address.
export async function startPhoton(port: number): Promise<{ url: string; stop: () => Promise<void> }> {
  const server: Server = createServer((req, res) => {
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
  return [
    {
      geometry: { coordinates: [longitude, latitude] },
      properties: { ...address, name: q, housenumber: '4', osm_key: 'amenity', osm_value: 'restaurant', osm_type: 'N', osm_id: hash.readUInt32BE(4) + 1 },
    },
    {
      geometry: { coordinates: [longitude + 0.0001, latitude] },
      properties: { ...address, housenumber: '6', osm_key: 'building', osm_value: 'yes', osm_type: 'W', osm_id: hash.readUInt32BE(8) + 1 },
    },
  ]
}
