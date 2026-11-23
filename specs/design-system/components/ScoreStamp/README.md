# ScoreStamp

The overall of a review or a subject in a round saffron disc, on top of a photo. Project component, not part of shadcn-vue.

Lives in `@/components/ui/score-stamp`. Props: `value` (number), `size` (`sm` 36px or `md` 56px).

- `md` sits at the bottom right of the photo of an `EntryCard`, 12px from the edges. `sm` sits at the bottom right of a journal tile, 6px from the edges.
- The fill is `saffron` and the number is `saffron-foreground` in the serif: 9.6:1 light, 9.0:1 dark. `shadow-stamp` lifts it off light photos.
- Use it only on photos. A row without a photo shows the overall as `RatingDisplay`. The map pins of `MapView` have the same look with the `map-pin` tokens.
- The number keeps one decimal, such as "8.3". A whole value has none, such as "7".
- It has the hidden word "Overall" before the number, so a screen reader says "Overall 8.3".
