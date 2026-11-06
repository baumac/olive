# RatingDisplay

Shows a read-only score from 1 to a maximum. Project component, not part of shadcn-vue.

Lives in `@/components/ui/rating-display`. Props: `value` (number), `max` (5 or 10), `label`.

- Always shows the number and the maximum as text ("5 / 5"). The pips repeat the number for quick scanning.
- The whole element has `role="img"` and an `aria-label` such as "Food: 8 out of 10". The pips are `aria-hidden`.
- Filled pips are `primary`. Empty pips are `muted` with an `input` outline.
- A `joc_de_cartes` review shows four rows: space, service, food, price.
- Proposed rendering: ADR 00002 does not fix the visual form.
