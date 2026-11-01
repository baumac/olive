# TextField

A labelled single-line input, or a textarea with `multiline`, with an optional hint or error below it.

- **You provide:** a visible `label`, plus any native input props (`value`, `onChange`, `type`, `name`, `required`). Pass `id` only when something else must point at the field; one is made for you otherwise.
- `hint` explains the expected format. `error` replaces the hint, turns the border `danger` and sets `aria-invalid`.
- Do write errors that say how to fix the value ("Enter a full email address, like name@example.com.").
- Don't use the placeholder as the label. Don't show an error before the person has left the field or submitted.
- Stack fields with `space-4` between them; fields fill their container's width.
