# Select

Picks one value from a short list, for sort order and filters.

From shadcn-vue (`@/components/ui/select`, built on Reka UI `Select`). Parts: `SelectTrigger`, `SelectValue`, `SelectContent`, `SelectItem`. The consumer gives the options and a `v-model`.

- Map each option to a `sort` value of the API, for example "Best food" to `-food`. The API accepts `space`, `service`, `food`, `price`, `title`, `created`, `updated` for reviews and `name`, `overall`, `created`, `updated` for restaurants. A `-` prefix sorts descending.
- Label options in plain words, not API names.
- The list is `popover` with `shadow-md`. The highlighted item is `accent`. The selected item shows a check icon.
- Arrow keys, Home, End and type-ahead work by Reka UI. Do not replace it with a native `<select>` styled by hand.
