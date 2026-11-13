---
description: Show the restaurants on maps in the UI, with Leaflet and the tiles of OpenStreetMap.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00017 gave each restaurant a point and an address. The UI showed only the address.

- A person could not see where a restaurant is without a different app.
- A person could not look for restaurants in one part of a city.
- A user who typed a location by hand had to find the latitude and the longitude in a different app.

Expected outcome:
- the restaurant page shows the restaurant on a map
- the restaurants page has a map view with the restaurants of the current filters
- a user sets the point of a location with a click on a map

# Scope
[scope]: #scope

In scope:
- the `within` filter of `listRestaurants`
- the tile server settings and the Content Security Policy of the UI route
- the `MapView` component and the three screens that use it
- the tests

Out of scope:
- groups of pins that are near each other. The map shows at most 100 pins.
- a sort by distance and a "near me" button. The browser does not ask for the location of the user.
- directions to a restaurant. The page links to OpenStreetMap.
- a dark map. The tiles are light in both themes.
- our own tile server, and a proxy for the tiles. See the guide-level explanation.
- a map in the link previews of decision 00013

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The three maps

Restaurant page:
- a map with one pin at the location, under the address
- the link "Open in OpenStreetMap" opens the point in a new tab

Restaurants page:
- the buttons "List" and "Map" select the view. The map view has `view=map` in the URL, so a link can open it.
- the map opens on the center of Barcelona. It shows the restaurants in its view that match the search and the filters.
- a move or a zoom of the map loads the restaurants of the new view
- a click on a pin shows the card of the restaurant under the map. Enter and Space on a pin do the same.
- the map shows at most 100 restaurants. The status then says "zoom in to see more".
- a restaurant without a location is only in the list. A text under the map says this.
- the map opens with its last view when the user comes back to the page, until a reload

Restaurant form:
- a selected place shows on a small map
- "Enter the location by hand" has a map. A click on it sets the latitude and the longitude. The user can also drag the pin.
- the two number fields stay. They are the way for the keyboard, and the pin follows them.

The list view is the text alternative of the map view. Each map has a name for screen readers.

## The tiles

The browser loads the map tiles from `https://tile.openstreetmap.org`. This is the tile server of the OpenStreetMap Foundation. It is free and has no key.
Its usage policy has these rules:
- the map shows "© OpenStreetMap contributors" with a link, always visible. `MapView` shows it in the corner of each map.
- the tile request has a `Referer` header. The app sets no `Referrer-Policy`, so browsers send the origin. A `no-referrer` policy on the UI route would stop the tiles.
- no download of tiles in advance or for offline use. The map loads only the tiles of its view.
- the server has no guarantee. Heavy use can be blocked without notice.

The app is not commercial and has few users, so this is a good start.

Settings:
- `REVIEWS_MAP_TILE_URL` sets a different tile server. It is a URL template with `{z}`, `{x}`, and `{y}`, and optionally `{r}`.
- `REVIEWS_MAP_TILE_URL=off` turns all maps off. The UI then shows the addresses and the number fields of decision 00017.
- `REVIEWS_MAP_ATTRIBUTION` is the credit of a tile server that is not from OpenStreetMap. The map shows it before the OpenStreetMap credit.

The browser sends the IP address of the user and the origin of the site to the tile server. The user accepted this for the start.
The app does not put a proxy in front of the tiles. The usage policy does not recommend one, and the app would carry all tile traffic.
A key in the tile URL of a paid server is visible to all users. Such servers limit a key to one site.

## The within filter

```
GET /v1/restaurants?within[west]=2.1&within[south]=41.3&within[east]=2.2&within[north]=41.4
```

- all four sides are required. `south` more than `north` gets `400 Bad Request`.
- `west` more than `east` is a box across the 180th meridian
- a restaurant without a location does not match
- the filter works with the search, the other filters, the sorts, and the cursors

# Implementation Details
[implementation-details]: #implementation-details

## API

- the `within` parameter is a `deepObject` with the new schema `Box`, like `near` of decision 00017
- `within` in `app/internal/publicapi/list.go` compares the latitude and the longitude with the sides. `idx_restaurants_latitude` is for the latitude.

## App server

- `webui.Options` has `TileURL` and `TileAttribution`. `uiOptions` in `app/main.go` reads the variables.
- `webui.TileOrigin` checks the template and gives its origin. A URL without `{z}`, `{x}`, and `{y}`, or with other parts in braces such as `{s}`, stops the start of the app.
- `img-src` of the Content Security Policy of decision 00005 gets this origin. Without a tile URL it stays `'self' blob:`.
- a host with a space, a semicolon, or a quote is refused. It could add a source or a directive to the policy.
- the UI route serves `/map-config.json` with `tiles` and `attribution`. The UI build is the same for all settings, so the UI gets them at run time.
- this file is not part of the Public API. Only the UI of this app needs it.

## UI

- the map library is Leaflet 1.9.4. It needs no key, has no other packages, and draws image tiles without WebGL.
- `MapView` in `app/ui/src/components/ui/map` is the only file that uses Leaflet. It is an async component, so Leaflet is in its own chunk. Pages without a map do not load it.
- `loadMapConfig` in `app/ui/src/lib/mapConfig.ts` gets `/map-config.json` one time. Without it, the UI shows no map. The Vite dev server has no such file. The UI then uses the tiles of OpenStreetMap.
- Leaflet sets its styles through the style object of each element. `style-src` therefore does not change.
- Leaflet stops the load of a tile with a `data:` image. The policy blocks `data:` URLs. `MapView` replaces the three methods of `L.TileLayer` that use this image, and uses a `blob:` image in its place. `img-src` already allows `blob:`. These methods are internals of Leaflet, so a Leaflet update needs a check. The end-to-end tests fail on a violation.
- the pins are `divIcon` markers with an inline SVG. The default icon of Leaflet needs image paths that the Vite build does not give.
- the pins have fixed colors, because the tiles are light in both themes
- Leaflet opens only popups with the keyboard. `MapView` selects a pin on Enter and Space.
- `MapView` keeps the marker of a restaurant that stays in the view. A pin with the keyboard focus then keeps the focus when the map loads again.
- the map element has `isolation: isolate`. The z-index values of Leaflet then stay under the header and the dialogs.
- the mouse wheel zooms only the map of the restaurants page. The other maps are in pages that scroll.
- `mapBox` in `app/ui/src/lib/places.ts` changes the view of the map to the `within` box. A map can show longitudes outside of -180 to 180, and the world more than one time.
- the restaurants page waits 0.3 seconds after a move before it loads. It ignores the answers of older requests.
- `useListQuery` gets a list of query keys that a filter change keeps. The restaurants page keeps `view`.
- a click on the map in the form rounds the coordinates to 6 decimals, about 0.1 meters

## Tests

- `location_test.go` tests the `within` filter with a box across the 180th meridian
- `webui_test.go` tests `TileOrigin`, the policy with the tile origin, and `/map-config.json`
- `app/ui/test/location.test.ts` tests `mapBox`, the config loader, and the other helpers. jsdom has no layout, so it does not test `MapView`.
- the fake server of `tests/photon.ts` gives one gray tile for each path under `/tiles`. The end-to-end tests need no network.
- `tests/e2e/maps.spec.ts` tests the three maps: the pin and the credit, the map view with filters and the keyboard, and the click that sets the point
- the `noCspViolations` fixture of decision 00005 runs in each of these tests, also during a zoom
- `accessibility.spec.ts` runs axe on the map view
- one manual check with the real tiles of OpenStreetMap showed the map of Barcelona with both pins

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- Groups of pins. A view with more than 100 restaurants shows only the first 100 of the sort.
- The position of the map is not in the URL. A reload opens Barcelona again.
- A paid tile server, when the app is commercial or has much more use. Only the two variables change.
- Leaflet 2 is in beta. It changes the internals that `MapView` replaces.
