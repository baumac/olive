# PantryItemCard

The row of one pantry item in a list: the cover, the name, the category, the producer, the origin, the review count, the tags, the overall and the heart.

Project component. Lives in `@/components/app/PantryItemCard.vue`, because it links to the pantry item. It is `ParentCard` with the details of the kind in its slot. Props: `item` (a `PantryItem`), `heading` (`h2` or `h3`).

- The detail line has the icon of the category and "Wine · Weingut Example · Mosel, Germany". A detail without a value is not in the line. See decision 00012.
- The icon of the category is wine for wine, beer for beer, and the shopping basket for cheese and chocolate. The icon is `aria-hidden`, because the text names the category.
- An unknown category of a newer API shows as its text with the shopping basket.
- The rest is `ParentCard`: a 64px cover, the name in the `name` style, the review count and the tags in `small`, the overall in `score-lg`, and the heart of `FavoriteButton` above the link layer.
- The Pantry, the tag page, the list page and the journal use it. Render the cards as `<ul>`, one card per `<li>`.
