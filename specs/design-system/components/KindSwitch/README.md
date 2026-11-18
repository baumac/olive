# KindSwitch

Selects the restaurants or the recipes of a page with two parts: "My reviews", the reviews of a reviewer page, and a list page.

Project component. It knows the route, so it lives in `@/components/app`. Props: `kind` (`restaurant` or `recipe`), `label` (the name of the `nav`, such as "Reviews of").

- It is a `nav` with two links: "Restaurants" and "Recipes". They are links, not tabs. Each part has its own URL, and a shared link opens the same part.
- The recipes link adds `?kind=recipes` to the path of the page. The restaurants link has no query. A switch drops the filters and the sort of the other kind. Their score filters are different.
- The links sit on a `muted` track with 4px padding and the `lg` radius. The current link has `aria-current="page"`, the `background` fill, and `shadow-sm`. The other link is `muted-foreground` and turns `foreground` on hover.
- The switch replaces the history entry. Back then leaves the page, like a filter change.
- Put it above the list that it switches. `useKindQuery` of `@/composables/useKindQuery` reads the kind of the URL.
