# RatingDisplay

Shows a read-only score as a number in the serif. Project component, not part of shadcn-vue.

Lives in `@/components/ui/rating-display`. Props: `value` (number), `max`, `label`, `showLabel`, `size` (`sm`, `md` or `lg`).

- Shows only the number, such as "7.6", in Fraunces with lining tabular figures. It has no pips and no "/ 10". See decision 00009.
- The whole element has `role="img"` and an `aria-label` such as "Food: 8 out of 10". The visible parts are `aria-hidden`.
- `showLabel` puts the label before the number in `muted-foreground`, such as "Food 8".
- A row shows only the overall, in size `md` (`score-lg`, 24px), at its right end.
- On a photo, the overall is a `ScoreStamp`, not a `RatingDisplay`.
- The page of a subject or a review shows the overall and the scores in `ScoreSummary`.
- A restaurant shows its averages in the same order as a review. The number keeps one decimal.
