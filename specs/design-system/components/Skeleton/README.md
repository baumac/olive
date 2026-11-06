# Skeleton

A placeholder block shown while a list or a record loads.

From shadcn-vue (`@/components/ui/skeleton`). The consumer gives the size with classes.

- Match the shape of the content that will load: one skeleton card per expected card, up to the page `limit`.
- Set `aria-busy="true"` on the region and give it an accessible name, such as "Loading restaurants".
- The fill is `accent` with a slow pulse. The pulse stops under `prefers-reduced-motion`.
- Show a skeleton only on the first load. `LoadMore` shows its own loading state.
