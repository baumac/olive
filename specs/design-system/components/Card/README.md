# Card

A bordered `surface-raised` panel that groups one item or one task, with an optional title, header actions and footer.

- **You provide:** `children` for the body, and optionally `title`, `actions` (small buttons or a Badge) and `footer` (buttons, aligned right).
- The card sets `space-4` padding, a `line` border and `radius-md`. It fills its container; you set the width and the grid.
- Do lay out card lists in a grid with `space-4` gaps. Do keep one primary action per card at most.
- Don't nest cards. Don't add shadows or a colored edge to make a card stand out; use a Badge instead.
