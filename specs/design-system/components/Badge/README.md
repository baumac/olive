# Badge

A short label for a tag or a status.

From shadcn-vue (`@/components/ui/badge`). Props: `variant` (`default`, `secondary`, `outline`, `destructive`). The consumer gives the text.

- Tags use `secondary`. Show the tag exactly as stored: lowercase, words joined by hyphens.
- Use `outline` for the review kind in a list of mixed kinds.
- A badge that filters the list when clicked renders as a link or button, with focus `ring`.
- Text is 12px medium (`caption`).
