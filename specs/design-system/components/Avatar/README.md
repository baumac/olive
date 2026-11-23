# Avatar

A round badge with a person's initials, next to their name.

From shadcn-vue (`@/components/ui/avatar`): `Avatar`, `AvatarImage`, `AvatarFallback`. The API has no profile pictures, so use `AvatarFallback` with initials. Props in our wrapper: `name` (string or null), `size` (`sm` 28px, `default` 36px, `lg` 68px).

- Initials are the first two letters of the creator `name`, in uppercase: "Ana" gives "AN".
- `creator` is `null` after a user is deleted. Show "?" with an outline and the text "Deleted user".
- Fill is `accent`, text is `accent-foreground` in the serif (7.5:1 light, 9.3:1 dark).
- When the name shows beside it, the avatar is decorative: `aria-hidden="true"`. Alone, it has `role="img"` and an `aria-label` with the name.
- Use `default` in `UserMenu` and `lg` at the top of the journal, with the first letter of the name only.
- Use it only in `UserMenu` and at the top of the journal. Cards and bylines show the name as text. See decision 00009.
