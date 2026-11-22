# RatingDisplay

Shows a read-only score as a number. Project component, not part of shadcn-vue.

Lives in `@/components/ui/rating-display`. Props: `value` (number), `max`, `label`, `showLabel`, `size` (`sm`, `md` or `lg`).

- Shows only the number, such as "7.6". It has no pips and no "/ 10". See decision 00019.
- The whole element has `role="img"` and an `aria-label` such as "Food: 8 out of 10". The visible parts are `aria-hidden`.
- `showLabel` puts the label before the number in `muted-foreground`, such as "Food 8".
- A card shows only the overall score, in size `md`, next to the title or the review count.
- The page of a restaurant or a review shows the overall score in size `lg`. The text "out of 10" follows it one time. One line below has the four scores with labels: space, service, food, value.
- A restaurant shows its averages in the same order. The number keeps one decimal.
