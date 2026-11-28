# CategoryChips

The row of chips that selects the category of the Pantry: "All", "Wine", "Beer", "Cheese" and "Chocolate".

Part of `@/pages/PantryPage.vue`. It knows the route. See decision 00012.

- It is a `nav` with the name "Category" and one link for each chip. They are links, not tabs. A shared link opens the same category.
- "All" has no `category`. The other links add `?category=wine` and the other values to the query of the page. The search, the filters and the sort of the URL stay.
- A change of the filters in `FilterBar` keeps the category. "Clear filters" of the empty state also clears the category.
- The chips are choice pills, 44px high, like `KindSwitch`. The current chip has `aria-current="page"` and the `primary` fill with `primary-foreground` text. The others are `card` with an `input` border.
- Each category chip has its 16px icon before the label: wine, beer, or the shopping basket. The icon is `aria-hidden`.
- The row wraps on a phone. Put it above `FilterBar`.
