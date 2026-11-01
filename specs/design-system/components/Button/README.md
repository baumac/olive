# Button

Starts an action; use one `primary` button per screen for the main action.

- **You provide:** a verb label as `children` ("Save", "Delete review"), an `onClick`, and `type="submit"` inside forms. The default type is `button`.
- **Variants:** `primary` (`brand` fill) for the main action, `secondary` (default) for everything else, `ghost` for low-weight actions in toolbars and cards, `danger` for destructive actions after a confirmation.
- **Sizes:** `md` (36px) by default; `sm` (28px) only in dense rows and card headers.
- Do keep labels in sentence case and short. Do disable while a request runs and change the label ("Saving").
- Don't put two `primary` buttons side by side. Don't use `danger` for "Cancel".
