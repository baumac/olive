# EntryCard

One review shown as one experience on the table or in a list of entries: the photo, the subject, the title, the reviewer, the date and the overall.

Project component. Lives in `@/components/app/EntryCard.vue`, because it links to the review. Props: `review` (a restaurant, recipe or pantry item review), `kind`, `variant` (`photo` or `row`).

- `photo`: the first photo of the review at full width, 224px high with `radius-2xl`, and the `md` `ScoreStamp` on it. Below it: the subject as a `kicker` in `primary`, the title in `entry-title`, and `small` metadata: "Priya · Visited on 20 Nov 2026".
- `row`: a `card` row with `radius-xl`. A 64px thumbnail of the first photo, or the utensils, chef hat or shopping basket icon on `accent`. Then the subject as a `kicker`, the title in the `name` style, and the metadata. The overall in `score-lg` sits at the right end.
- The table uses `photo` for an entry with photos and `row` for one without. See decision 00009.
- The photo uses the `srcset` of the `hero` size of `CoverPhoto`, so a dense phone screen loads `full`. The `<img>` has `alt=""`, because the card link names the review.
- The card is one link to the review page. Its name is the subject and the title.
- The date is the `date` of the review with "Visited on", "Cooked on" or "Tasted on", never relative time. See the "Dates, numbers and language" section.
- Entries stack with a 20px gap (`space-5`). Render them as `<ul>`, one entry per `<li>`.
- `creator` can be `null` after the user is deleted. Show "Deleted user" in this case.
