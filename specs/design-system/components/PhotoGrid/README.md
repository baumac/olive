# PhotoGrid

Shows the thumbnails of a record as a grid of buttons. Project component, not part of shadcn-vue.

Lives in `@/components/ui/photo-grid`. Props: `photos` (the `Photo` objects of the API), `label` (the accessible name of the list, such as "Photos of Great food"). Event: `open` with the index. `focus(index)` moves the focus to a thumbnail, for example after the viewer closes.

- A `<ul>` with 3 columns, 5 from `sm`, and a `space-2` gap. An empty list renders nothing.
- Each thumbnail is a square button with `radius-md` and `object-cover`. Its accessible name is the alt text, or "Photo 2 of 5" without one.
- Each `<img>` has `width`, `height` and `loading="lazy"` from the API. The page then does not move while images load.
- A thumbnail shows `Skeleton` until it loads. A thumbnail that fails to load shows the `muted` background with the image-off icon.
- The focus outline is the solid 2px `ring` of all controls.
