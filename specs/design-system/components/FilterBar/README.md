# FilterBar

Filters and sorts a list of restaurants or reviews. It turns the API's filter parameters into controls and chips.

Project component built from `Button`, `Select`, `Popover` and chip buttons. Lives in `@/components/ui/filter-bar`. Props: `filters` (object), `sortOptions`. Events: `update:filters`, `update:sort`. The screen maps filters to query parameters and refetches from the first page.

- Controls in a row: a "Filters" button that opens a popover with all filters, then the most used filters as `Select`s, then sort.
- Review filters: tags (`tags[has_any]`), kind (`kind[in]`), basic `rating` (`rating[gte]`), and for `joc_de_cartes` the four scores (`food[gte]` and so on). Restaurant filters: tags.
- Each active filter shows as a removable chip: "Tag: italian". The remove button's label is "Remove filter tag italian". "Clear all" removes every chip.
- Score filters say what they include: "Rating: 4 or more". A rating filter matches only `basic` reviews. Say so in the popover help text.
- After a change, show the result count in a polite `role="status"` region, and start again from the first page. A cursor works only with the filters that made it.
- Keep filters in the URL query, so a filtered list can be shared and the back button works.
- Chips use `radius` full, an `input` border and `caption` text.
