# Select

A labelled native dropdown for picking one option from a short list.

- **You provide:** a visible `label` and `options` as `{ value, label, disabled? }`, plus native props (`value`, `onChange`, `name`). Add `placeholder` with `defaultValue: ''` when there is no sensible default.
- `hint` and `error` behave as in TextField.
- Do use it for 4 to 15 options. Use radio-style choices or Tabs for fewer, and a searchable field for more.
- Don't nest actions inside options. Don't auto-submit a form when the value changes.
