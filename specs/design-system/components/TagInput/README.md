# TagInput

Adds and removes tags on a restaurant or a review. Project component, not part of shadcn-vue.

Lives in `@/components/ui/tag-input`. Props: `v-model` (string array), `max` (default 20). Build it from `Badge`-like chips and an `Input`.

- Enter or comma adds the typed tag. Backspace in an empty field removes the last tag. Each chip has a remove button with `aria-label` "Remove <tag>".
- It checks the API rules before the API does: 1 to 63 characters, pattern `^[a-z0-9]+(-[a-z0-9]+)*$`, unique, at most 20 tags.
- Do not change the input silently. Show a `FormMessage` that suggests the valid form, such as `wood-fired`.
- Chips are `secondary` with `radius-sm`. The field border is `input`, `destructive` when invalid.
- Announce added and removed tags in a polite live region.
