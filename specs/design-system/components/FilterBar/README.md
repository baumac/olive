# FilterBar

Filters and sorts a list of restaurants or reviews. It turns the API's filter parameters into controls and chips.

Project component built from `Button`, `Select`, `Popover` and chip buttons. Lives in `@/components/ui/filter-bar`. Props: `filters` (object), `sortOptions`. Events: `update:filters`, `update:sort`. The screen maps filters to query parameters and refetches from the first page.

- Controls in a row: a search box for a `search` filter, a "Filters" button that opens a popover with the other filters, then sort.
- Review filters: a full-text search (`search`) and the five scores (`food[gte]` and so on). Restaurant filters: a full-text search (`search`), tags, and the overall average (`overall[gte]`). The overall filter says that restaurants without reviews do not match.
- Each active filter shows as a removable chip: "Tag: italian". The remove button's label is "Remove filter tag italian". "Clear all" removes every chip.
- Score filters say what they include: "Food: 8 or more".
- After a change, show the result count in a polite `role="status"` region, and start again from the first page. A cursor works only with the filters that made it.
- Keep filters in the URL query, so a filtered list can be shared and the back button works.
- Chips use `radius` full, an `input` border and `caption` text.
- A search box sends its text after a 300 ms pause in typing. Clear all, or a removed chip, also drops a search that waits. Its label is visually hidden, so it has a placeholder and a search icon. Its chip is "Search: pasta". While a search is active, the sort has a "Best match" option, and it is the default.
