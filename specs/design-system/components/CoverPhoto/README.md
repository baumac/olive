# CoverPhoto

Shows the cover photo of a restaurant in a box with a fixed aspect ratio. Project component, not part of shadcn-vue.

Lives in `@/components/ui/cover-photo`. Props: `photo` (the `cover` of a `Restaurant`, or `null`), `size` (`card` or `hero`), `alt`. Decision 00030 selects the cover.

- `card` shows the `thumb` image in a 16:9 box. It loads lazily. It is the first part of the restaurant `Card`, flush with its top and sides.
- `hero` shows the `full` image in a 16:9 box, 21:9 from `sm`. It is the first part of the restaurant page, with `radius-xl`. Its `srcset` lets a narrow screen load the `thumb`.
- The image fills the box with `object-cover`. The box has the `muted` background.
- It shows `Skeleton` until the image loads, and the image-off icon on `muted` when it fails, like `PhotoGrid`.
- Without a photo, it shows the utensils icon on `muted`. The cards of a row then line up.
- In a card, the `<img>` has `alt=""`. The name of the card link is the name of the restaurant.
- On the restaurant page, the cover is a button. Its name is "Open the cover photo", with the alt text after a colon. It opens the `PhotoViewer` of the gallery.
