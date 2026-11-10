# PhotoViewer

Shows one photo at full size in a `Dialog`. Project component, not part of shadcn-vue.

Lives in `@/components/ui/photo-viewer`. Props: `photos`, `v-model:index` (the open photo, `null` when closed), `editable`, `saving`, `error`. Events: `saveAlt` with the photo and the text, `delete` with the photo, `close` with the index after the focus trap ends.

- The title is "Photo 2 of 5". The alt text is the description, and the image uses it as its `alt`.
- The image keeps its aspect ratio, at most 60% of the viewport high, on the `muted` background with `radius-md`.
- With more than one photo, `outline` buttons "Previous photo" and "Next photo" move through the photos. The arrow keys do the same. The list wraps at both ends.
- Escape and the close button close it. The parent then moves the focus back to the thumbnail of the shown photo.
- With `editable`, the creator gets "Edit description" and "Delete photo". Edit shows a "Description" `Input`, "Save description" and "Cancel". Delete asks "Delete this photo?" in an `AlertDialog`.
- Other people get no actions.
