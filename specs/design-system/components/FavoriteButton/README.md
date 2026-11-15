# FavoriteButton

A heart that adds a restaurant to the favorites list of the signed in user, or removes it.

Project component. It calls the API, so it lives in `@/components/app`. Props: `restaurantId`, `name`, `favorite` (the `favorite` value of the restaurant), `labeled` (boolean). Event: `change`, sent with the new state after the API confirms it.

- It is a toggle button. The accessible name is "Favorite <name>" in the two states. `aria-pressed` tells the state. Do not change the name with the state.
- Not pressed: the outline of the `heart` icon. Pressed: the same icon with `fill-current`.
- On a restaurant card: a round `secondary` icon button at the top right of the cover, 8px from the edges. It is above the link layer of the card, so a click on it does not open the restaurant.
- On the restaurant page: `labeled`, an `outline` button with the heart and the text "Favorite", before "Save to list".
- The heart changes at once. A failed request changes it back and shows a destructive `Toast`, such as "Not added to favorites".
- A signed out user goes to `/sign-in` with the current path as `redirect`. The heart is never pressed for a signed out user.
