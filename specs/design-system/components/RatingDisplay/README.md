# RatingDisplay

Shows a read-only score from 1 to a maximum. Project component, not part of shadcn-vue.

Lives in `@/components/ui/rating-display`. Props: `value` (number), `max`, `label`.

- Always shows the number and the maximum as text ("8 / 10"). The pips repeat the number for quick scanning.
- The whole element has `role="img"` and an `aria-label` such as "Food: 8 out of 10". The pips are `aria-hidden`.
- Filled pips are `primary`. Empty pips are `muted` with an `input` outline.
- Below the rows, show the legend in `muted-foreground`: "A higher score is always better. A high price score means good value for money."
- A review shows the overall row first, then five rows: space, service, food, price, cleanliness. The API computes the overall value from the five scores.
- A restaurant shows its averages in the same order. The pips round down, the text keeps the decimal.
- Proposed rendering: ADR 00002 does not fix the visual form.
