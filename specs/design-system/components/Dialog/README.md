# Dialog

A modal window that asks the person to confirm a delete.

From shadcn-vue (`@/components/ui/dialog`, built on Reka UI `Dialog`). Parts: `DialogTrigger`, `DialogContent`, `DialogHeader`, `DialogTitle`, `DialogDescription`, `DialogFooter`, `DialogClose`. The consumer gives the title, the description and the footer buttons. For confirm flows, `AlertDialog` gives the `alertdialog` role.

- Title is a question: "Delete this review?". The description names the record and says that the delete is permanent.
- Footer: `outline` "Cancel" first, then `destructive` "Delete review". Focus starts on "Cancel".
- Escape and the close button close it. Focus is trapped inside and goes back to the trigger on close.
- The overlay is black at 50%. The panel is `background`, `border`, `radius`, `shadow-lg`, 24px padding.
