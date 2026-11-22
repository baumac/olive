# TagInput

Adds and removes tags on a restaurant. Reviews have no tags. Project component, not part of shadcn-vue.

Lives in `@/components/ui/tag-input`. Props: `v-model` (string array), `max` (default 20). Build it from `Badge`-like chips and an `Input`.

- Enter or comma adds the typed tag. Backspace in an empty field removes the last tag. Each chip has a remove button with `aria-label` "Remove <tag>".
- It checks the API rules before the API does: 1 to 63 characters, the pattern of the `Tag` schema, unique, at most 20 tags.
- Tags have the lowercase letters and digits of all scripts, such as `café` or `ラーメン`. See decision 00005.
- Do not change the input silently. Show a `FormMessage` that suggests the valid form, such as `wood-fired`.
- The one exception is Unicode normalization form C. The tag looks the same after the change. The API does the same change.
- Chips are `secondary` with `radius-sm`. The field border is `input`, `destructive` when invalid.
- Announce added and removed tags in a polite live region.
