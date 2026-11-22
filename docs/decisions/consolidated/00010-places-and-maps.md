---
description: A place search of the app finds the point and the address of a restaurant through a Photon geocoder. The UI shows restaurants on Leaflet maps with the tiles of OpenStreetMap. A person sets a preferred location in the browser, and the restaurant list then shows the nearest restaurants first.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00005 gives each restaurant a point and an address.
A user who adds a restaurant should not type coordinates.
A person wants to see where a restaurant is, look for restaurants in one part of a city, and find the restaurants near them.

The first release uses free public servers. See decision 00000.
The public Photon server of Komoot finds the places. The tile server of OpenStreetMap gives the map tiles.
These servers get the search text and the IP address of users, and they have no guarantee.
Environment variables can move both to other servers, such as servers on the home server. The backlog has this under "For the Future".

Expected outcome:
- a user finds a place with a search and does not type coordinates
- the restaurant page shows the restaurant on a map, with a directions link
- the restaurants page has a map view with the restaurants of the current filters
- a user sets the point of a location with a click on a map
- a person sets a preferred location one time. The list then shows the nearest restaurants first, and the maps open there.
- the operator changes or turns off the geocoder and the tile server without a new build

# Scope
[scope]: #scope

In scope:
- `searchPlaces` and its Photon adapter, cache, and limits
- the `near` and `within` filters and the `distance` sort of `listRestaurants`
- the tile server settings, `/map-config.json`, and the tile origin of the Content Security Policy
- `MapView` and the maps of the restaurant page, the restaurants page, and the restaurant form
- the preferred location of the UI and the default sort of the restaurant list

Out of scope:
- the location of a restaurant and the duplicate check of a place. Decision 00005 has them.
- self-hosted maps and place search in the first release
- a proxy for the tiles, and a download of tiles for offline use
- a dark map. The tiles are light in both themes.
- a location in the user record, and a position that follows the device
- a distance in the responses of the API. The UI computes it.
- the review lists and the lists of decision 00008 with a distance
- maps of recipes

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The place search

```
GET /v1/places?search=can+sole&latitude=41.3874&longitude=2.1686
```

```json
{
  "items": [
    {
      "name": "Can Solé",
      "location": {
        "latitude": 41.3766,
        "longitude": 2.1893,
        "address": "Carrer de Sant Carles 4, 08003 Barcelona, España",
        "place_id": "osm:N1234567890"
      },
      "website": "https://cansole.example",
      "phone": "+34 932 21 50 12"
    }
  ]
}
```

- the search needs an approved user and at least 3 characters. It returns at most 10 places and has no pages.
- `latitude` and `longitude` are optional. The search then prefers places near that point.
- a place has `website`, `phone`, and `opening_hours` when the geocoder gives them. The public server does not give them.
- a client sends the `location` of the selected place in `createRestaurant` or `updateRestaurant`
- the search gets `503` when it is off or when the geocoder does not answer. A client can then send a location that the user typed.
- decision 00004 has the limit of each user

The browser does not call the geocoder:
- `connect-src` of the Content Security Policy stays `'self'`
- the geocoder gets the IP address of the server, not of the user
- the app can limit and cache the requests
- a different client uses the same search

## Filters and the distance sort

```
GET /v1/restaurants?near[latitude]=41.3766&near[longitude]=2.1893&near[radius]=75
GET /v1/restaurants?within[west]=2.1&within[south]=41.3&within[east]=2.2&within[north]=41.4
GET /v1/restaurants?sort=distance&near[latitude]=41.377&near[longitude]=2.189&near[radius]=10000
```

- `near` needs `latitude` and `longitude`. `radius` is optional, in meters from 1 to 50,000.
- `near` without `radius` selects all restaurants and only gives the point of the `distance` sort
- `within` needs all four sides. `south` more than `north` gets `400`. `west` more than `east` is a box across the 180th meridian.
- a restaurant without a location matches no `near` filter with a radius and no `within` filter
- `distance` puts the nearest restaurant first. It needs `near`, or gets `400`. Restaurants without a location are last in both directions.
- the filters work with the search, the other filters, the sorts, and the cursors

## The maps

Restaurant page:
- a map with one pin under the address, and "Open in OpenStreetMap"
- a "Directions" link next to the address. An Android device gets a `geo:` link. An Apple device gets Apple Maps. Other devices get the directions page of OpenStreetMap.

Restaurants page:
- "List" and "Map" select the view. The map view has `view=map` in the URL.
- the map shows the restaurants in its view that match the search and the filters. A move or a zoom loads the restaurants of the new view.
- a click on a pin, or Enter and Space, shows the card of the restaurant under the map
- pins that are less than 44 pixels apart join into one round pin with a count. A click on it zooms until its pins are apart. The selected pin stays out of a cluster.
- the map shows at most 100 restaurants. A line below the map then tells the person to zoom in.
- a restaurant without a location is only in the list. A text under the map says this.
- a pan or a zoom puts the position in the URL, such as `at=41.3874,2.1686,13`. A reload and a link then open this view.
- the map opens at the first of these views: the `at` position of the URL, the last view before the person left the map, the restaurants of a list filter, the preferred location, and the center of Barcelona

Restaurant form:
- a selected place shows on a small map
- "Enter the location by hand" has a map. A click sets the latitude and the longitude. The pin can also be dragged.
- the two number fields stay. They are the way for the keyboard, and the pin follows them.
- the search box searches 0.4 seconds after the last key, from 3 characters. A link under it credits OpenStreetMap.
- after a selection, the form shows the restaurants at most 75 meters away. See decision 00005.

The list view is the text alternative of the map view. Each map has a name for screen readers.

## The preferred location

The restaurants page has a "Set location" button with two ways to set it:
- "Use my current location" asks the browser for the position of the device. The browser asks the person for permission.
- an approved user can search for a place, such as a city or an address

| Location | Distance | List | Default sort |
|---|---|---|---|
| none | | all restaurants | Highest rated |
| set | Within 1, 5, 10, 25, or 50 km | the restaurants in this distance | Nearest |
| set | Any distance | all restaurants | Nearest |

- the first distance is 10 km. With a search text, the default sort is "Best match".
- each card shows the distance, such as `1.2 km`
- the map opens at the location and shows it as a blue dot. The map has no distance filter.
- the place search of the restaurant form prefers places near the location
- the menu shows the name of the location, the distance, and "Clear location"
- the location is not in the URL. A shared link does not show the location of the sender.

## The servers

The place search uses `https://photon.komoot.io` by default. Photon uses the data of OpenStreetMap.
The map tiles come from `https://tile.openstreetmap.org`. The browser of each user loads them.

The usage rules of these servers:
- the number of requests must be reasonable. The operators slow or block heavy use.
- the servers have no guarantee that they answer
- each map shows "© OpenStreetMap contributors" with a link, always visible
- the tile request has a `Referer` header. The UI must not have a `Referrer-Policy` of `no-referrer`.
- no download of tiles in advance

| Variable | Use |
|---|---|
| `REVIEWS_PLACES_URL` | the base URL of a Photon server, or `off` |
| `REVIEWS_MAP_TILE_URL` | a URL template with `{z}`, `{x}`, and `{y}`, and optionally `{r}`, or `off`. `off` turns all maps off. |
| `REVIEWS_MAP_ATTRIBUTION` | the credit of a tile server that is not from OpenStreetMap. The map shows it before the OpenStreetMap credit. |

A Photon server of the operator can give the `website`, `phone`, and `opening_hours` of a place, when it imports the data with `-extra-tags`.
A paid geocoder needs a new adapter in `places.go`, and its terms must allow the app to store the results.
A key in the tile URL of a paid server is visible to all users.

# Implementation Details
[implementation-details]: #implementation-details

## Place search

`app/api/internal/publicapi/places.go` has the search.
- it sends `GET <url>/api?q=<search>&limit=10` to Photon, with `lat` and `lon` for the point
- the point has one decimal, about 11 km. Users in one city then share the cache entries.
- the `User-Agent` is `reviews-app` with the Application URL of the PocketBase settings. The operator of the geocoder can then find the operator of the app.
- the request has a timeout of 5 seconds. The app reads at most 1 MB of the answer.
- the address is the street with the house number, the postcode with the city, the state, and the country. Missing parts are left out.
- the app removes the characters that `Address` refuses and cuts the text to 255 characters. A value that fails its schema is left out. The response then cannot fail the validation.
- `photonFeature` reads `properties.extra`: `website` or `contact:website`, `phone` or `contact:phone`, and `opening_hours`. The first value of a list with `;` is the phone. `;` in the opening hours becomes a line break.
- a place gets a `place_id` only when it has an `amenity` of `restaurant`, `cafe`, `fast_food`, `bar`, `pub`, `biergarten`, or `ice_cream`
- a place without a point or without text is left out. Equal places are returned one time.
- the cache keeps 1,000 answers for 24 hours in memory. A full cache removes the expired answers, then the answer that expires first.
- equal searches at one time cause one request to the geocoder
- all users together cause at most 60 requests to the geocoder per minute. More searches that are not in the cache get `503`. A `503` does not count for the limit of the user.
- the search text goes to the operator of the geocoder. It has no data of the user.

## Filters and the sort

- `near` and `within` are `deepObject` parameters with the schemas `Near` and `Box`
- `Latitude` and `Longitude` have `format: double`. The Go SDK would otherwise use `float32`, which has an error of more than 1 meter.
- `near` first selects a band of latitudes with `idx_restaurants_latitude`. It then compares the distance on a flat surface.
  One degree of latitude is 111,320 meters. One degree of longitude is that times the cosine of the latitude.
  The difference of the longitudes is the short way around the earth.
- the `distance` sort key is the squared distance of `squaredDistance` in `list.go`. The filter and the sort then always agree.
- `distanceSort` writes the numbers of the point into the SQL text with `strconv.FormatFloat`. The handler refuses NaN and infinite values first.
- the fingerprint has the `near` values, so a cursor is valid only for the same point and radius
- SQLite computes the distance for each matching restaurant and sorts them. No index gives this order. A site with many restaurants can need an R-tree index.

## The app server

- `webui.Options` has `TileURL` and `TileAttribution`. `main.go` reads the variables.
- `webui.TileOrigin` checks the template and gives its origin. A URL without `{z}`, `{x}`, and `{y}`, or with another part in braces such as `{s}`, stops the start.
- a host with a space, a semicolon, a quote, or a part in braces is refused. It could change the policy.
- `img-src` of the Content Security Policy gets the tile origin. Without a tile URL it stays `'self' blob:`.
- the UI route serves `/map-config.json` with `tiles` and `attribution`. The UI build is the same for all settings. This file is not part of the Public API.

## MapView

- the map library is Leaflet 1.9.4. It needs no key and draws image tiles without WebGL.
- `MapView` in `app/ui/src/components/ui/map` is the only file that uses Leaflet. It is an async component, so pages without a map do not load Leaflet.
- `loadMapConfig` of `lib/mapConfig.ts` gets `/map-config.json` one time. Without it, the UI shows no map. A failed request is not kept, so the next map tries again.
- Leaflet sets its styles through the style object of each element. `style-src` then does not change.
- Leaflet stops the load of a tile with a `data:` image, which the policy blocks. `MapView` replaces the three methods of `L.TileLayer` that use it and uses a `blob:` image.
  These are internals of Leaflet. A Leaflet update needs a check. The end-to-end tests fail on a violation.
- the pins are `divIcon` markers with an inline SVG and fixed colors
- `MapView` selects a pin on Enter and Space. It keeps the marker of a restaurant that stays in the view, so a pin keeps the keyboard focus.
- the `cluster` and `clusterLabel` props make the groups from the pixel distance at the zoom level. No library is added. Only the map of the restaurants sets them.
- the map element has `isolation: isolate`. The z-index values of Leaflet stay under the header and the dialogs.
- the mouse wheel zooms only the map of the restaurants page
- `mapBox` of `lib/places.ts` changes the view to the `within` box. A map can show longitudes outside of -180 to 180.
- the restaurants page waits 0.3 seconds after a move before it loads, and ignores the answers of older requests
- the map view sends the selected sort, with `near` without a radius. With more than 100 restaurants, the sort selects the 100 that it shows.
- `formatMapPosition` and `parseMapPosition` write and read `at` with 5 decimals. A wrong value gives no position.
- `at` changes only after a move of the user. It replaces the history entry. A move of the app removes it.
- before a move of the app, `MapView` stops a running pan of the user. It finds it with the class `leaflet-pan-anim`. A Leaflet update needs a check.
- a click on the map of the form rounds the coordinates to 6 decimals
- `directionsLink` of `lib/places.ts` selects the link from the user agent. An iPad can send the user agent of a Mac, so a Mac also gets Apple Maps.

## The preferred location

- `lib/preferredLocation.ts` keeps the point, a label, and the radius in `localStorage` with the key `reviews-location`
- sign out does not remove it. It is a setting of the browser, not data of the account.
- the UI calls `navigator.geolocation.getCurrentPosition` only after a click. A page load never asks.
- it rounds the point to 3 decimals, about 100 meters. The API and its logs then do not get the exact position. The cards show "Less than 100 m" for a shorter distance.
- browsers give a position only to a page with HTTPS or on `localhost`. A `Permissions-Policy` of the reverse proxy must not turn off `geolocation`.
- `useListQuery` takes the sort options and the default sort as getters. "Nearest" is the first option and the default only while the person has a location.
- the default sort of the API stays `-created`. Only the UI changes its default.
- a list with no restaurant in the distance has a "Show any distance" button

## Tests

- `location_test.go` tests the filters, the date line, and the place search with a fake Photon server
- `tests/photon.ts` is the fake geocoder and the fake tile server of the end-to-end tests. Each search gets its own place, so parallel tests make no duplicates. The tests need no network.
- `tests/e2e/maps.spec.ts` tests the three maps, the keyboard, the click that sets the point, and `at`
- jsdom has no layout, so the UI tests do not test `MapView`

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| The browser calls the geocoder | Not selected. The policy, the privacy of the IP address, the limits, and the cache need the server. |
| Self-hosted Photon and tiles in the first release | Not selected. The public servers are free and good for a few users. The backlog has the move. |
| A proxy for the tiles | Not selected. The usage policy does not recommend one, and the app would carry all tile traffic. |
| A map library with WebGL or a key | Not selected. Leaflet needs neither. |
| A cluster library | Not selected. A small function in `MapView` makes the groups. |
| A location in the user record | Not selected. The server would store where each person is, and a visitor could not set it. |
| The `overall` default sort in the API | Not selected. Clients that page through the default order would get a new order. |
| Ask for the position when the page loads | Not selected. Browsers and people refuse a request without a visible reason. |
| The distance sort in the UI | Not selected. The UI has only one page of the list. |
| The newest restaurants first as the default of the UI | Removed. A new restaurant is seldom the one that a person looks for. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
