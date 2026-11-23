# ScoreSummary

The score block of the page of a subject or a review: the overall in large type, and each score on a line with a dotted leader, as on a menu. Project component, not part of shadcn-vue.

Lives in `@/components/app/ScoreSummary.vue`, because it reads the score fields of `lib/reviewKinds.ts`. Props: `scores` (the `rating` of a subject or the scores of a review), `kind` (`restaurant` or `recipe`). Slot: the line under "out of 10", such as "6 reviews".

- Left: the overall in `score-xl` (60px serif), "out of 10" under it in `small` `muted-foreground`, and the slot. "out of 10" shows one time on the page. See decision 00009.
- Right: a `dl` with one line per score in the order of `lib/reviewKinds.ts`. The `dt` is the label, such as "Value". The `dd` is the value in `score-line` (18px serif). Between them is an `aria-hidden` element with a 2px dotted `border` bottom: the leader.
- The block is `card` with a `border` hairline, `radius-2xl` and 18px by 20px padding.
- A subject without reviews has no block. It shows "No reviews yet".
- The overall has the hidden word "Overall" before it.
