# Label

The visible name of a form field.

From shadcn-vue (`@/components/ui/label`, built on Reka UI `Label`). The consumer gives `for` (the field `id`) and the text.

- Every field has a visible label. Placeholders do not replace labels.
- Use sentence case and no colon: "Name", "Feedback".
- Inside `FormItem`, `FormLabel` sets `for` and turns `destructive` when the field has an error.
