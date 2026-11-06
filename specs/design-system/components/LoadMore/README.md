# LoadMore

Gets the next page of a list with the `next_cursor` value. Project component, not part of shadcn-vue.

Lives in `@/components/ui/load-more`. Props: `cursor` (string or `null`), `loading` (boolean). Event: `load`, sent with the cursor. The consumer calls the API and appends the items.

- Show the button only when `next_cursor` is not `null`. `null` means the last page.
- Send the cursor with the same filters and sort that made it. Change of filter or sort starts again from the first page.
- Never parse or build a cursor value.
- While loading, disable the button, show a spinner and keep the label width. Move focus to the first new item after load.
- On error, show a destructive `Alert` above the button and keep the button for a retry.
