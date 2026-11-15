---
description: A person sets a preferred location in the browser. The restaurant list then shows the nearest restaurants first, and the maps open there. The default sort of the list is the highest rated.
status: implemented
---

# Motivation
[motivation]: #motivation

The restaurant list showed the newest restaurants first. A new restaurant is seldom the one that a person looks for.
The API had the `near` filter, but the UI never used it for the list. It also never asked the browser for the position of the device.
Each map opened at the center of Barcelona.

Expected outcome:
- the list shows the highest rated restaurants first
- a person sets a location one time. The list then shows the nearest restaurants first.
- the maps and the place search start at this location

# Scope
[scope]: #scope

In scope:
- the `distance` sort of `listRestaurants`, and a `near` parameter without a radius
- the preferred location of the UI, from the position of the device or from the place search
- the default sort of the restaurant list of the UI

Out of scope:
- a location in the user record. See the alternatives.
- the default sort of the API. It stays `-created`. Other clients do not change.
- a distance in the responses of the API. The UI computes it.
- a position that follows the device. The UI reads the position one time, when the person asks for it.
- the review lists and the lists of decision 00033

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

The restaurant page has a "Set location" button. Its menu has two ways to set the location.
- "Use my current location" asks the browser for the position of the device. The browser asks the person for permission.
- A signed in and approved user can search for a place, such as a city or an address.

The button then shows the name of the location. The menu also has the distance and "Clear location".

| Location | Distance | List | Default sort |
|---|---|---|---|
| none | | all restaurants | Highest rated |
| set | Within 1, 5, 10, 25, or 50 km | the restaurants in this distance | Nearest |
| set | Any distance | all restaurants | Nearest |

The first distance is 10 km. With a search text, the default sort is still "Best match".
Each restaurant card shows the distance from the location, such as `1.2 km`.
The map view opens at the location and shows it as a blue dot. The map has no distance filter. It shows the restaurants of its view.
A view with more than 100 restaurants shows the nearest ones for the "Nearest" sort.
The place search of the restaurant form prefers places near the location.

The API request of the list is, for example:

```
GET /v1/restaurants?sort=distance&near[latitude]=41.377&near[longitude]=2.189&near[radius]=10000
```

# Implementation Details
[implementation-details]: #implementation-details

## API

`sort` of `listRestaurants` has the new property `distance`. It needs `near`. Without `near`, the response is `400 Bad Request`.
`distance` puts the nearest restaurant first. `-distance` puts it last.
The `radius` of `near` is now optional. Without it, `near` selects all restaurants and only gives the point of the sort.
"Any distance" and the map view then also have the `distance` sort.
The first sort key puts restaurants without a location last in both directions.
The sort key is the squared distance of the `near` filter, from `squaredDistance` in `app/internal/publicapi/list.go`.
The filter and the sort then always agree. The order of a squared distance is the order of the distance.

A sort key is SQL text without parameters. `distanceSort` writes the numbers of the point into the text with `strconv.FormatFloat`.
The handler rejects NaN and infinite values first. The spec validation can be off.

The cursor has the sort values and the `id` of the last item. The fingerprint has the `near` values.
A cursor is then valid only for the same point and radius.

SQLite computes the distance for each restaurant that matches the filters and sorts them. No index gives this order.
With a radius, only the restaurants in its latitude range match. See the unresolved questions.

## Storage

`app/ui/src/lib/preferredLocation.ts` keeps the location in `localStorage` with the key `reviews-location`.
The value has the point, a label, and the radius. It is not in the URL. A shared link does not show the location of the sender.
Sign out does not remove it. It is a setting of the browser, not data of the account.

## Position of the device

The UI calls `navigator.geolocation.getCurrentPosition` only after a click on "Use my current location". A page load never asks.
It rounds the point to 3 decimals. This is about 100 meters. The API and its logs then do not get the exact position.
The cards therefore show "Less than 100 m" for a shorter distance.
The menu shows a message when the person refuses the permission, or when the device has no position.
Browsers give a position only to a page with HTTPS, or on `localhost`.
A `Permissions-Policy` header of a reverse proxy must not turn off `geolocation`.

## Sort of the list

`useListQuery` now takes the sort options and the default sort as getters.
The page adds "Nearest" as the first option and as the default only while the person has a location.
A link with `sort=distance` opens with the default sort for a person without a location.
The map view sends `near` without a radius, together with `within`.

## Alternatives

- **A location in the user record.** It follows the person to each device. It needs a spec change, a migration, and a place to edit it.
  The server then stores where each person is. A signed out reader also could not set it. The browser storage needs none of this.
- **Change the default sort of the API to `overall`.** Clients that page through the default order would get a new order. Only the UI needs the change.
- **Ask for the position when the page loads.** Browsers and people refuse a request without a visible reason.
- **Sort by distance in the UI.** The UI has only one page of the list. The order must come from the API.

## Corner cases

- The position is old after the person moves. "Use my current location" reads it again.
- A stored radius that is not an option becomes "Any distance".
- Two restaurants at the same distance are in the order of their `id`.
- A restaurant without a location matches no `near` filter with a radius. With "Any distance", it is last in the list.
- The list has no restaurant in the distance. The empty state then has a "Show any distance" button, also when filters are active.
- A second location replaces a position of the device that the browser gives later.
- A page without HTTPS gets a message that the position needs HTTPS.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- The `distance` sort reads each restaurant that matches the filters. A site with many restaurants can need an R-tree index.
