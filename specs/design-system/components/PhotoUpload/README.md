# PhotoUpload

Selects one image for a record, shows a preview, and takes an optional description. Project component, not part of shadcn-vue.

Lives in `@/components/ui/photo-upload`. Props: `actionLabel` (default "Upload photo"), `busy`, `error` (the text of a failed upload). Event: `add` with the file and the description. The parent uploads the file and then calls `reset()`.

- The file input sits inside a label that looks like an `outline` button with the image-plus icon: "Add photo". The input is visually hidden but keeps the keyboard focus. The label shows the focus outline.
- It accepts JPEG, PNG and WebP. It checks the type and the size (at most 10 MB) before an upload. Error text: "Select a JPEG, PNG, or WebP image of at most 10 MB."
- The hint under the button says what to add: "JPEG, PNG, or WebP of at most 10 MB. Add only photos of food, rooms, and menus."
- After a selection, a bordered panel shows the preview (`radius-md`, at most 192px high), a "Description" `Input` with the help text "Optional. Describe the photo for people who cannot see it.", the main button and "Cancel".
- The preview is a `blob:` URL. The component frees it when the selection changes or the component goes away.
- While `busy`, the buttons are disabled and the main button says "Uploading…".
- The parent hides the component when the record has the most photos, and says why.
