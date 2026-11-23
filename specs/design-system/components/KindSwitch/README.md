# KindSwitch

Selects the kinds of the entries on the table and on a journal: "All", "Restaurants" or "Recipes".

Project component. It knows the route, so it lives in `@/components/app`. Props: `kind` (`all`, `restaurant` or `recipe`), `label` (the name of the group, such as "Show").

- It is a row of three links: "All", "Ate out" and "Cooked" on the table, and "All", "Restaurants" and "Recipes" on a journal. They are links, not tabs. Each part has its own URL, and a shared link opens the same part.
- "All" has no query. The other links add `?kind=restaurants` or `?kind=recipes` to the path of the page. A switch drops the filters and the sort of the other kind. Their score filters are different.
- With "All", the entries are always the newest first. The sort control and the score filters show only for one kind. See decision 00006.
- The links are choice pills, 44px high (`rv-choice` in the previews). The current link has `aria-current="page"` and the `primary` fill with `primary-foreground` text. The others are `card` with an `input` border.
- The switch replaces the history entry. Back then leaves the page, like a filter change.
- Put it above the entries that it switches. `useKindQuery` of `@/composables/useKindQuery` reads the kind of the URL.
