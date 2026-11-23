# Card

A bordered surface on `card` paper: the score block, a side card, a group of settings.

From shadcn-vue (`@/components/ui/card`). Parts: `CardHeader`, `CardTitle`, `CardDescription`, `CardAction`, `CardContent`, `CardFooter`. The consumer gives the content of each part.

- Lists do not use `Card`. A subject is a row (`ParentCard`), and a review is an `EntryCard`. See decision 00009.
- The title is in the serif (`h3` style). The description is `muted-foreground`.
- Use the `card` token for the fill, `border` for the hairline and `radius-2xl`. Padding and gap are 20px (`space-5`). Cards have no shadow.
- `creator` can be `null` after the user is deleted. Show "Deleted user" in this case.
