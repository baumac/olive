# Badge

A short label for a tag or a status.

From shadcn-vue (`@/components/ui/badge`). Props: `variant` (`default`, `secondary`, `outline`, `destructive`). The consumer gives the text.

- Tags use `secondary`, which has the `accent` fill and `accent-foreground` text (7.5:1 light, 9.3:1 dark). Show the tag exactly as stored: lowercase, words joined by hyphens.
- Use `outline` for a quiet status label.
- A badge that filters the list when clicked renders as a link or button, with focus `ring`.
- Text is 12px semibold (`caption`), with `radius-md`.
