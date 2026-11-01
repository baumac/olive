# Alert

An inline message with an icon, a short title and optional detail, for information, success or an error.

- **You provide:** a `title` that states what happened, optional `children` with what to do next, and a `tone`: `info` (default, `accent`), `success` or `danger`.
- `danger` alerts use `role="alert"` and are announced at once. Other tones use `role="status"`.
- The icon and title carry the meaning together with the color, so state never depends on hue alone.
- Do place alerts at the top of the area they describe. Do write titles as plain facts ("Review published").
- Don't stack more than two alerts. Don't use an alert for marketing or tips.
