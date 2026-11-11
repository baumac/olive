# RatingInput

Picks an integer score from 1 to a maximum. Project component, not part of shadcn-vue.

Lives in `@/components/ui/rating-input`. Props: `v-model` (number), `max`, `label`, `description`, `lowest`, `highest`. Build it on Reka UI `RadioGroup`.

- Reviews use five inputs with `max` 10: `space`, `service`, `food`, `price`, `cleanliness`.
- It follows the radio group pattern: Tab moves into the group, arrow keys change the value, the selected segment has `tabindex="0"`.
- Show help text below the group, like `FormDescription`. `description` tells what the score rates. `lowest` and `highest` tell what 1 and `max` mean, for example `1 means poor value for money. 10 means great value for money.` Without it, a person can read a high price score as expensive.
- The component links its help text to the group with `aria-describedby`. `describedBy` adds the id of the error message.
- The value is required. Do not preselect a value. Show a `FormMessage` if the person submits with no value.
- Each segment shows its number. The selected segment is `primary` with `primary-foreground`. Others are `background` with an `input` border.
- Proposed rendering: numbered segments, 32px square, 4px gap. ADR 00002 does not fix the visual form.
