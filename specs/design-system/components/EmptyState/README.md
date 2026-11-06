# EmptyState

Fills the place of a list that has no items, and says what to do next.

Project component. Lives in `@/components/ui/empty-state`. Props: `icon`, `title`, `description`. Slot: `action`, usually one `Button`.

- Two cases, two texts:
  - Nothing exists yet: "No reviews yet", "Be the first to review <restaurant>.", action "Write a review".
  - Filters match nothing: "No reviews match these filters", "Remove a filter or clear all filters.", action "Clear filters" as an `outline` button.
- The title is a short statement. The description says what to do, in one or two sentences.
- The box has a dashed `input` border, `radius-xl` and centered content. The icon sits in a `muted` square.
- Do not show `LoadMore` or `Skeleton` with an empty state.
- When filters change the list to empty, the result count says "0 reviews" in the `FilterBar` status region.
