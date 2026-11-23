# CoverPhoto

Shows the cover photo of a restaurant in a box with a fixed aspect ratio. Project component, not part of shadcn-vue.

Lives in `@/components/ui/cover-photo`. Props: `photo` (the `cover` of a `Restaurant`, or `null`), `size` (`card` or `hero`), `alt`. Decision 00007 selects the cover.

- `card` shows the `thumb` image in a square box of 64px with `radius`. It loads lazily. It is the thumbnail at the start of a subject row.
- `hero` shows the `full` image, full width on a phone and with `radius-2xl` from `sm`. It is the first part of the restaurant page and of the review page. Its `srcset` lets a narrow screen load the `thumb` and a dense screen the `full`. The photo of an `EntryCard` uses the same `srcset`.
- The image fills the box with `object-cover`. The box has the `muted` background.
- It shows `Skeleton` until the image loads, and the image-off icon on `muted` when it fails, like `PhotoGrid`.
- Without a photo, it shows the utensils icon on `muted`. The cards of a row then line up.
- In a card, the `<img>` has `alt=""`. The name of the card link is the name of the restaurant.
- On the restaurant page, the cover is a button. Its name is "Open the cover photo", with the alt text after a colon. It opens the `PhotoViewer` of the gallery.
