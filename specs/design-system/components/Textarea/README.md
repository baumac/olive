# Textarea

A multi-line text field, for `description` and `feedback`.

From shadcn-vue (`@/components/ui/textarea`). Use with `v-model`. Same states as `Input`: focus `ring`, `aria-invalid` gives a `destructive` border.

- `description` and `feedback` are optional and allow at most 10,000 characters. Set `maxlength="10000"`.
- Minimum height is 64px. Let the person resize vertically.
- Show the character count in the field description only when the text is near the limit.
