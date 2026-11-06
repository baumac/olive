# Toast

A short message that confirms an action, then goes away by itself.

From shadcn-vue: the `Sonner` component (`@/components/ui/sonner`, built on `vue-sonner`). Put one `<Toaster />` in the app root. Screens call `toast.success(title, { description })`, `toast.error(...)` or `toast(...)`.

- Show a toast after create, edit and delete: "Review saved", "Restaurant added", "Review deleted".
- The title is two or three words in the past tense. The description, if any, names the record.
- Success uses the `success` icon color. Error uses `destructive`. Neutral uses `foreground`.
- Place toasts at the bottom right on `md` and up, at the bottom center on small screens. Use `z-toast`.
- Success toasts close after 4 seconds. Error toasts stay until the person closes them.
- Toasts use `role="status"`, errors `role="alert"`. Pause the timer on hover and focus.
- Do not put the only copy of important information in a toast. A validation error goes next to its field.
