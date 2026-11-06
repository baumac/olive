# Alert

A message box for errors from the API, confirmations that stay on the page, and page-level notes.

From shadcn-vue (`@/components/ui/alert`). Props: `variant` (`default`, `destructive`, `success`). `success` is a project variant: add it to the component's `cva` variants. Parts: an icon, `AlertTitle`, `AlertDescription`. The consumer gives all three.

- Use `destructive` for API errors that have no field: `401`, `403`, `404`, `409`, `5xx`, network errors.
- Use `success` for a confirmation that must stay on the page, such as after an account change. For a quick "Review saved", use `Toast`.
- Use `default` for neutral notes, with the info icon.
- The title says what happened. The description says what to do next. Use the problem `title` and `detail` only when they are written for people.
- Add `role="alert"` to errors that appear after an action, and `role="status"` to confirmations.
- Every variant has an icon and words. Color is never the only signal.
- An empty list uses `EmptyState`, not an alert.
