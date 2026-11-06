# Input

A single-line text field, for `name` and `title`.

From shadcn-vue (`@/components/ui/input`). Use with `v-model`. The consumer gives `id`, `name`, `maxlength` and the placeholder; it always sits inside a `Form` field with a `Label`.

- `name` and `title` allow 1 to 255 characters with at least one non-space character. Set `maxlength="255"` and `required`.
- The border is the `input` token. Focus adds a 2px `ring` outline.
- Set `aria-invalid="true"` when the field has an error. The border changes to `destructive`.
- Placeholders show an example, never the label. Placeholder text is `muted-foreground`.
- Text is 16px on small screens, 14px from `md` up. This stops iOS zoom on focus.
