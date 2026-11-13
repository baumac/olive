---
description: Give each restaurant a location, add a place search, and stop a second restaurant at the same place.
status: implemented
---

# Motivation
[motivation]: #motivation

A restaurant had only a name, a description, and tags.

- A person could not see where a restaurant is. Two restaurants with one name in different cities looked the same.
- Two users could add the same restaurant. Its reviews then split between two pages. Decision 00001 left this to superusers.
- The backlog has a map of the restaurants. A map needs a point for each restaurant.

Expected outcome:
- each new restaurant has a point and an address
- a user finds the place with a search and does not type coordinates
- the same place cannot get a second restaurant page
- the UI shows the restaurants near a new location before the user saves

# Scope
[scope]: #scope

In scope:
- the `location` of a restaurant in the Public API
- the `near` filter of `listRestaurants`
- the place search, `GET /places`
- the rule against a second restaurant at one place
- the restaurant form, the restaurant page, and the restaurant cards of the UI
- the tests

Out of scope:
- a map in the UI. The next decision adds it with a tile server. This decision makes its data.
- a filter by a bounding box and a sort by distance. The map decision defines them.
- a merge of the duplicate restaurants that exist. Superusers do it, as decision 00001 says.
- a location for the restaurants that exist. Their creators add it. See the guide-level explanation.
- the location of the user. The browser does not ask for it.
- opening hours, phone numbers, and other data of a place

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The location of a restaurant

```json
{
  "id": "k3x9m2p7q1w8z4a",
  "name": "Can Solé",
  "location": {
    "latitude": 41.3766,
    "longitude": 2.1893,
    "address": "Carrer de Sant Carles 4, 08003 Barcelona, España",
    "place_id": "osm:N1234567890"
  }
}
```

- `latitude` and `longitude` are degrees. `address` is one line of 1 to 255 characters with the rules of a name.
- `place_id` is optional. It is the ID of the place at the geocoder. It starts with the source, such as `osm`.
- `createRestaurant` needs a `location`. This breaks a client that sends none. The UI is the only client.
- a restaurant from before this decision has `"location": null`
- `updateRestaurant` takes a `location`. It replaces the complete stored location. `null` is not allowed, so a location cannot be removed.
- the restaurant search also finds words of the address

## Who can change the location

The rules extend decision 00016.

| Action | Who | Condition |
| --- | --- | --- |
| Add a location to a restaurant without one | the creator | none |
| Change a stored location | the creator | no other user has a review of the restaurant |

For a restaurant without a creator, "the creator" is each user who has a review of it.
A change that fails the condition gets `409 Conflict`, like a change of the name.
The reviews of other users are about the restaurant at this place.

## The place search

```
GET /v1/places?search=can+sole&latitude=41.3874&longitude=2.1686
Authorization: Bearer eyJhbGciOi...
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
      }
    },
    {
      "location": { "latitude": 41.377, "longitude": 2.19, "address": "Carrer de Sant Carles 6, 08003 Barcelona, España" }
    }
  ]
}
```

- the search needs a signed in user and at least 3 characters. It returns at most 10 places and has no pages.
- `latitude` and `longitude` are optional. The search then prefers places near that point. The UI sends the center of Barcelona.
- a client sends the `location` of the selected place in `createRestaurant` or `updateRestaurant`
- each user can send 30 searches per minute. More get `429 Too Many Requests`.
- the search returns `503 Service Unavailable` when it is off or when the geocoder does not answer. A client can then send a location that the user typed.

The app gets the places from a Photon geocoder. Photon uses the data of OpenStreetMap.
The default is the public server `https://photon.komoot.io`. It is free and has no key. Its terms are:
- the number of requests must be reasonable. The operator slows or blocks heavy use.
- it has no guarantee that it answers

The app is not commercial and has few users, so this is a good start.
`REVIEWS_PLACES_URL` sets a different Photon server. `REVIEWS_PLACES_URL=off` turns the search off.
A paid geocoder becomes necessary when the app is commercial, has much more use, or the public server fails often.
It needs a new adapter in `places.go`. Its terms must allow the app to store the results. Some free plans do not.

The browser does not call the geocoder. The reasons are:
- the Content Security Policy of decision 00005 stays `connect-src 'self'`
- the geocoder gets the IP address of the server, not of the user
- the app can limit and cache the requests
- a different client, such as a phone app, uses the same search

## One restaurant for one place

A place of one restaurant has a `place_id`. These are the OpenStreetMap places with `amenity` equal to `restaurant`, `cafe`, `fast_food`, `bar`, `pub`, `biergarten`, or `ice_cream`.
An address, a building, a street, or a city has no `place_id`. One building can have many restaurants.

Hard rule:
- only one restaurant can have a `place_id`
- a create or an update with the `place_id` of a different restaurant gets `409 Conflict`
- the problem then has the ID of that restaurant in `restaurant`. The UI shows a link to it.

```json
{
  "type": "about:blank",
  "title": "Conflict",
  "status": 409,
  "detail": "Another restaurant has this place.",
  "restaurant": "k3x9m2p7q1w8z4a"
}
```

Soft rule:
- after the user selects a location, the form gets the restaurants at most 75 meters from it with the `near` filter
- it shows them under "Is it one of these?" with links. The user can still save.

The hard rule does not stop each duplicate.
A user can select the address in place of the restaurant, or can type the location. The soft rule shows the duplicate in these cases.
A rule on the name and the distance would refuse real restaurants, such as two bars in one food court.

## The near filter

```
GET /v1/restaurants?near[latitude]=41.3766&near[longitude]=2.1893&near[radius]=75
```

- `radius` is in meters, from 1 to 50,000. All three properties are required.
- a restaurant without a location does not match
- the filter works with the search, the other filters, the sorts, and the cursors

## The restaurant form

- "Location" has a search box. The form searches 0.4 seconds after the last key, from 3 characters.
- the results are a list of buttons with the name and the address. A click selects the place.
- the name of the selected place goes into an empty "Name" field
- "Enter the location by hand" shows the fields "Address", "Latitude", and "Longitude". The form opens them when the search fails.
- a link under the search credits OpenStreetMap. The OpenStreetMap attribution guidelines require it for an app with such a search.
- the edit form of a restaurant with reviews of other users shows the location without the search, with the reason
- a restaurant without a location shows "Add the location" on its page to the user who can change it

# Implementation Details
[implementation-details]: #implementation-details

## Spec

- new schemas: `Latitude`, `Longitude`, `Address`, `PlaceId`, `Location`, `Near`, `Place`, and `PlaceList`
- `Latitude` and `Longitude` have `format: double`. The Go SDK otherwise uses `float32`, which has an error of more than 1 meter.
- `Restaurant.location` is required and can be `null`. `RestaurantCreate.location` is required. `RestaurantPatch.location` is optional and not `null`.
- `RestaurantPatch` is a JSON Merge Patch, but a `location` replaces the complete object. A location without `place_id` removes the stored one. A merge of single coordinates has no use.
- `near` is a `deepObject` parameter, like the comparison filters
- `searchPlaces` has the new tag `places` and the new response `503 ServiceUnavailable`
- `Problem` gets the optional property `restaurant`
- `x-rate-limit` gets the name `place-search` with the key `user`

## Data

- migration `1794268809_add_restaurant_location.go` adds the fields `latitude`, `longitude`, `address`, and `place_id` to `restaurants`
- a restaurant without a location has an empty `address`. A number field cannot be empty, and 0 is a valid coordinate.
- `idx_restaurants_place_id` is a unique index on `place_id` where it is not empty. It also stops a duplicate from the dashboard.
- `idx_restaurants_latitude` is on `(latitude, longitude)` for the `near` filter
- `restaurants_fts` gets the column `address`. FTS5 cannot add a column, so the migration makes the table again and fills it.
- the triggers of decision 00012 do not use the new columns. The migration does not change them.

## API

- `CreateRestaurant` and `UpdateRestaurant` check the place and save in one transaction. PocketBase has one SQLite writer, so two requests cannot both pass the check.
- `UpdateRestaurant` runs the check of the reviews of other users for a change of the name or of a stored location. A location that is equal to the stored one is not a change.
- the `near` filter first selects a band of latitudes. `idx_restaurants_latitude` is for this condition. It then compares the distance on a flat surface.
  - one degree of latitude is 111,320 meters. One degree of longitude is that times the cosine of the latitude of the point.
  - the difference of the longitudes is the short way around the earth. A point at longitude 180 finds restaurants on both sides.
  - the error is small for a radius of 50 km. It grows near the poles, where the flat surface is a bad model.
- the search scores of decision 00004 are now `name` 10, `tags` 4, `address` 2, and `description` 1

## Place search

`app/internal/publicapi/places.go` has the search.

- it sends `GET <url>/api?q=<search>&limit=10` to Photon, with `lat` and `lon` for the point
- the point has one decimal, about 11 km. Users in one city then share the cache entries.
- the `User-Agent` is `reviews-app` with the Application URL of the PocketBase settings. The operator of the geocoder can then find the operator of the app.
- the request has a timeout of 5 seconds. The app reads at most 1 MB of the answer.
- the address is the street with the house number, the postcode with the city, the state, and the country. Parts that Photon does not give are left out.
- the app removes the characters that the `Address` schema refuses and cuts the text to 255 characters. A response then cannot fail the response validation of decision 00007.
- a place without a point or without text is left out. Equal places are returned one time.
- the cache keeps 1,000 answers for 24 hours in memory. A full cache is emptied.
- all users together cause at most 60 requests to the geocoder per minute. More searches that are not in the cache get `503`.
- a search that gets `503` does not count for the limit of the user
- the search text goes to the operator of the geocoder. It has no data of the user.

Environment variables:
- `REVIEWS_PLACES_URL` is the base URL of a Photon server, or `off`
- `REVIEWS_PLACE_SEARCH_LIMIT` is the number of searches of each user per minute. The default is 30.

## UI

- `LocationField` in `app/ui/src/components/app` has the search and the fields by hand. It calls `searchPlaces`, so it is an app component.
- the fields by hand accept a comma or a point in a number. A changed location by hand has no `place_id`.
- `RestaurantFormPage` sends the `location` in a patch only when it changed and is not locked
- the check for near restaurants runs for each new location. A failed check shows nothing.
- `fieldErrors` maps an error of `location.address` to the `location` field
- the restaurant page and the restaurant cards show the address with a pin icon

## Tests

- `location_test.go` tests the location rules, the place conflict, the `near` filter with the date line, the address in the search, and the place search with a fake Photon server
- `migrations_test.go` tests the migration with a restaurant from before it, and the unique index
- `app/ui/test/location.test.ts` tests `LocationField` and the helpers
- `tests/photon.ts` is a fake Photon server for the end-to-end tests. `global-setup.ts` starts it on port 8096. Each search gets its own place, so parallel tests make no duplicates.
- the test API gives a restaurant without a location a random point in the South Pacific
- `restaurants.spec.ts` tests the search, the missing location, the second restaurant at a place, the location by hand, and the locked location
- `accessibility.spec.ts` runs axe on the form with search results and with the fields by hand

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- The map of the restaurants and its tile server. The plan is Leaflet with the tiles of `tile.openstreetmap.org`, loaded by the browser.
- A sort by distance. It needs a cursor with the distance.
- OpenStreetMap can delete a place or give it a new ID. The restaurant keeps the old `place_id`, and a second restaurant at the place is then possible.
- The restaurants from before this decision have no location until a user adds one.
- A paid geocoder. See the place search section for the conditions.
