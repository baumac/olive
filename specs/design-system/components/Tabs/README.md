# Tabs

Switches between a few views of the same page section; the selected tab has a `brand` underline.

- **You provide:** an accessible `label` for the tab list and `tabs` as `{ id, label, content }`. Use `defaultValue` for uncontrolled use, or `value` with `onChange` to keep the selection in your router or state.
- Arrow keys, Home and End move between tabs. Only the selected tab is in the tab order.
- Do use 2 to 6 short labels in sentence case. The list scrolls sideways on narrow screens.
- Don't use tabs for steps in a sequence. Don't put a form's Save button inside a single tab when it saves every tab.
