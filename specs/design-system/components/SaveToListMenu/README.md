# SaveToListMenu

Adds a restaurant to the lists of the signed in user, or removes it. It is on the restaurant page, after the `FavoriteButton`.

Project component built from shadcn-vue `Popover`, `Button`, and `Input`. It calls the API, so it lives in `@/components/app`. Props: `restaurantId`.

- The trigger is an `outline` button "Save to list" with the `list-plus` icon. A signed out user goes to `/sign-in` with the current path as `redirect`.
- The popover has the title "Save to list", one checkbox for each list of the user, and a form with the field "New list" and a `secondary` button "Create".
- The favorites list is not in the menu. The `FavoriteButton` changes it.
- The lists load when the popover opens. While they load, show the text "Loading your lists".
- A checkbox changes at once. A failed request changes it back and shows the reason below the form with `role="alert"`.
- "Create" makes the list, adds the restaurant, and puts the new list first with a checked box. A name that the user already has shows "You have a list with this name."
- A user with the maximum number of lists gets "You have the maximum number of lists. Delete a list first." A full list gets "This list is full. Remove a restaurant first." `listFailureText` of `lib/lists.ts` selects the text by the `code` of the problem.
- If the list is made but the restaurant is not added, the new list stays first with an empty box. The text below the form tells the person to select it.
- The set has no `Checkbox` component. The menu uses the native checkbox with `accent-primary`. Add a `Checkbox` component when a second screen needs one.
