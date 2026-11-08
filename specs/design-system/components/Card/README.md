# Card

A bordered surface for one restaurant or one review in a list.

From shadcn-vue (`@/components/ui/card`). Parts: `CardHeader`, `CardTitle`, `CardDescription`, `CardAction`, `CardContent`, `CardFooter`. The consumer gives the content of each part.

- Restaurant card: `name` as the title, `description` as the description, the overall average and the review count, tags as `Badge`s, the creator and date in the footer. A restaurant without reviews says "No reviews yet".
- Review card: `title` as the title, the scores as `RatingDisplay`, `feedback` in the content, tags and the creator in the footer.
- `creator` can be `null` after the user is deleted. Show "Deleted user" in this case.
- Use the `card` token for the fill, `border` for the hairline, `radius-xl` and `shadow-sm`. Padding and gap are 24px (`space-6`).
- Stack cards with a 16px gap. Render the list as `<ul>`, one card per `<li>`.
