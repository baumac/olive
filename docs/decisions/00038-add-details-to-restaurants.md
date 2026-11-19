---
description: Give restaurants an optional website, phone number, opening hours, and price level, and fill them from the place search when the geocoder has them.
status: implemented
---

# Motivation
[motivation]: #motivation

A restaurant has a name, a description, tags, and a location.
People who look at a restaurant also want to know how to call it, where its menu is, when it is open, and how much it costs.
They must now find these facts on a different site.
Decision 00017 put "opening hours, phone numbers, and other data of a place" out of scope.

Expected outcome:
- a restaurant page shows the website, the phone number, the opening hours, and the price level when they are known
- the user who adds a restaurant can type them
- the place search fills them when the geocoder has them
- search engines get the website and the phone number in the structured data of decision 00021

# Scope
[scope]: #scope

In scope:
- the properties `website`, `phone`, `opening_hours`, and `price_level` of a restaurant in the Public API
- the same properties, except `price_level`, in the results of the place search
- the restaurant form and the restaurant page of the UI
- the JSON-LD of restaurant pages
- the tests

Out of scope:
- opening hours that the app can read, such as "open now" or a filter for open restaurants. See the unresolved questions.
- a filter or a sort by price level
- a second request to OpenStreetMap for the tags of a place. See the place search section.
- the details of the restaurants that exist. Their creators add them.
- the details on the restaurant cards of the lists and on the map. A card stays short.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The details of a restaurant

```json
{
  "id": "k3x9m2p7q1w8z4a",
  "name": "Can Solé",
  "website": "https://cansole.example",
  "phone": "+34 932 21 50 12",
  "opening_hours": "Tu-Sa 13:00-16:00, 20:00-23:00\nSu 13:00-16:00",
  "price_level": 3
}
```

All four properties are optional. A restaurant without a value has no property in the response, like `description`.

| Property | Rules |
| --- | --- |
| `website` | An `http` or `https` URL of at most 2,048 characters. It has no spaces and no hidden characters. |
| `phone` | 3 to 20 digits. Spaces, parentheses, points, slashes, and hyphens can separate them. A `+` can come first. At most 40 characters. |
| `opening_hours` | A text of at most 500 characters. Line breaks are allowed. It has the other rules of `LongText`. |
| `price_level` | An integer from 1 to 4. 1 is inexpensive, 2 is moderate, 3 is expensive, and 4 is very expensive. |

- `createRestaurant` takes each property
- `updateRestaurant` takes each property. `null` removes the stored value, as for `description`.
- the same users as for the description can change the details. Reviews of other users do not lock them. They are facts that change, unlike the name and the place.
- the price level is not the `price` score of a review. The score tells if the price was fair. The level tells how much a meal costs.
- the API has no currency. The level is relative to the restaurants of the same city.

## The place search

```json
{
  "items": [
    {
      "name": "Can Solé",
      "location": { "latitude": 41.3766, "longitude": 2.1893, "address": "Carrer de Sant Carles 4, 08003 Barcelona, España", "place_id": "osm:N1234567890" },
      "website": "https://cansole.example",
      "phone": "+34 932 21 50 12",
      "opening_hours": "Tu-Sa 13:00-16:00, 20:00-23:00\nSu 13:00-16:00"
    }
  ]
}
```

A place has `website`, `phone`, and `opening_hours` when the geocoder gives them.
The UI puts them into the empty fields of the form when the user selects the place. The user can change them before the save.

The public Photon server of decision 00017 does not give these tags.
Photon gives the OpenStreetMap tags of a place in `properties.extra` only when its operator imports the data with `-extra-tags`.
A Photon server of the operator of the app, set with `REVIEWS_PLACES_URL`, can give them.
With the public server, the user types the details.

A request to the OpenStreetMap API or to Overpass for each selected place can get the tags.
This decision does not add it. It is a second external service with its own terms and its own limits.
Each select also sends a request from the server, and the place search does not need it to work.

OpenStreetMap has no common tag for a price level. The search does not fill it.

## The restaurant form

The form has a "Details" group under "Tags" with four optional fields:
- "Website". The form adds `https://` to a value without a scheme, such as `cansole.example`.
- "Phone"
- "Opening hours". A text field of 3 lines.
- "Price level". A select with "Not set" and the names of the four levels.

## The restaurant page

The page shows the details under the address in this order:
- the website as a link with the host name, such as `cansole.example`. The link opens in a new tab.
- the phone number as a `tel:` link
- the opening hours with their line breaks
- the price level as a word, such as "Expensive"

The page shows no currency sign. The API has no currency for a restaurant.

# Implementation Details
[implementation-details]: #implementation-details

## Spec

- new schemas: `Website`, `Phone`, `OpeningHours`, and `PriceLevel`
- `Restaurant`, `RestaurantCreate`, and `Place` get the optional properties. `Place` has no `price_level`.
- `RestaurantPatch` gets them with `null` allowed
- `Website` has the pattern `^https?://` and then no ASCII space, no control character, and no hidden character of the `Name` schema. The first character after `//` cannot be `/`, `?`, or `#`. A URL then always has a host.
- `Phone` has the pattern `^\+?\(?[0-9](?:[ ()./-]*[0-9]){2,19}$`

## Data

- migration `1794268816_add_restaurant_details.go` adds the fields `website`, `phone`, `opening_hours`, and `price_level` to `restaurants`
- the text fields have the patterns of the spec. The dashboard then cannot save a value that the API refuses. A bad value would break the response validation and the JSON-LD.
- the text fields are empty for a restaurant without the value. `price_level` is 0, which the API does not allow. 0 is "not set".
- the triggers of decision 00012 and the search tables do not use the new fields. The migration does not change them.

## API

- `toRestaurant` leaves out an empty text and a level of 0
- the patch uses the `changes` helper of the description. An equal value does not change `updated`.

## Place search

- `photonFeature` reads `properties.extra`. The keys are `website` or `contact:website`, `phone` or `contact:phone`, and `opening_hours`.
- the first value of a list with `;` is the phone number
- `;` in the opening hours becomes a line break. This makes the OpenStreetMap syntax easier to read.
- a value that fails the rules of its schema is left out. A response then cannot fail the response validation of decision 00007.
- the cache of decision 00017 keeps the details with the place

## Structured data

- the JSON-LD of a restaurant page gets `telephone` and `sameAs` with the website. schema.org uses `sameAs` for the official site of a thing. `url` stays the page of the app.
- `openingHours` of schema.org needs a fixed syntax. The opening hours are free text, so the JSON-LD leaves them out.
- `priceRange` of schema.org is a range of money or a number of currency signs. The API has no currency, so the JSON-LD leaves it out.
- the link preview of decision 00013 does not change. The details do not help a person decide to open a shared link.

## UI

- `lib/validation.ts` has the checks of the fields with the rules of the spec
- `lib/restaurantDetails.ts` has the change of a website without a scheme, the `tel:` link, and the names of the price levels
- `LocationField` sends the complete place with the `selected` event. `RestaurantFormPage` fills the empty name and the empty details from it.
- the form trims the phone and the opening hours before it checks them
- a draft keeps the details
- `RestaurantDetails` in `app/ui/src/components/app` shows the details on the restaurant page
- the website link has `rel="nofollow ugc noopener noreferrer"`. A user typed the URL.

## Tests

- `details_test.go` tests the create, the get, the patch with `null`, and the errors of each property
- `places_internal_test.go` tests the details of a Photon answer with `extra` tags
- `migrations_test.go` tests the migration with a restaurant from before it
- `pagemeta_test.go` tests the JSON-LD with the details
- `app/ui/test/restaurant-details.test.ts` tests the helpers, the form fields, and the page component
- `tests/photon.ts` gives `extra` tags to a search with the word "details"
- `restaurants.spec.ts` tests the details from the place search, a change, and a removal

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
