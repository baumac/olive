# Button

Starts an action: submit a form, open a dialog, delete a record.

From shadcn-vue (`@/components/ui/button`). Props: `variant` (`default`, `secondary`, `outline`, `ghost`, `link`, `destructive`), `size` (`default` 44px, `sm` 44px with less padding and 14px text, `lg` 52px, `icon` 44×44), `as` / `as-child` to render a `RouterLink`. The consumer gives the label as the default slot.

- Every button is a pill (`radius-full`) at least 44px high. See decision 00009.
- Use one `default` (primary) button per view, for the main action ("Save review", "Add a visit"). The round "Add review" of `BottomNav` is the one exception.
- Use `outline` for cancel and secondary paths, `ghost` for actions inside cards and lists.
- Use `destructive` only for delete. Always put it in a confirming `Dialog`.
- Write labels as a verb and an object, in sentence case: "Save review", "Delete restaurant".
- An `icon` button needs `aria-label`.
- Set `type="submit"` only on the submit button of a form. Set `disabled` while the request runs.
- Focus shows a 2px `ring` outline. Do not remove it.
