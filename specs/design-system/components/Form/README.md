# Form

Connects a field to its label, help text and error message.

From shadcn-vue (`@/components/ui/form`): `FormField`, `FormItem`, `FormLabel`, `FormControl`, `FormDescription`, `FormMessage`. The consumer gives the field `name` and the control in `FormControl`.

- Stack label, control, description and message with an 8px gap (`space-2`).
- `FormDescription` is `muted-foreground`. `FormMessage` is `destructive` text.
- `FormControl` sets `id`, `aria-describedby` and `aria-invalid` on the control.
- Show each API validation error next to its field. The `property` of each error in the `422` problem response selects the field (ADR 00001, Errors).
- An error that has no field goes into an `Alert` above the form.
- Error text tells the person what to do, not only what is wrong.
