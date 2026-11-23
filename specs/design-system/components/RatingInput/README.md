# RatingInput

Picks an integer score from 1 to a maximum. Project component, not part of shadcn-vue.

Lives in `@/components/ui/rating-input`. Props: `v-model` (number), `max`, `label`, `description`, `lowest`, `highest`. Build it on Reka UI `RadioGroup`.

- Reviews use four inputs with `max` 10. Restaurants: `space`, `service`, `food`, `price`. The label of `price` is "Value". Recipes: `taste`, `ease`, `instructions`, `value`.
- It follows the radio group pattern: Tab moves into the group, arrow keys change the value, the selected segment has `tabindex="0"`.
- `description` tells what the score rates. It is one line above the group, like `FormDescription`. `lowest` and `highest` are one or two words. They show in `caption` text below the two ends of the row, for example "poor value" and "great value". Hidden words make the sentence `1 means poor value. 10 means great value.` for a screen reader.
- The component links the description and the ends to the group with `aria-describedby`. `describedBy` adds the id of the error message.
- The value is required. Do not preselect a value. Show a `FormMessage` if the person submits with no value.
- Each segment shows its number. The selected segment is `primary` with `primary-foreground`. Others are `card` with an `input` border.
- The ten segments share the width of the row with a 4px gap. Each is at least 32 by 44px, with `radius-md`. See decision 00009.
- The review form shows the overall of the selected scores after the four inputs, before the save. See decision 00005.
