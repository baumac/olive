# Avatar

A round badge with a person's initials, next to their name.

From shadcn-vue (`@/components/ui/avatar`): `Avatar`, `AvatarImage`, `AvatarFallback`. The API has no profile pictures, so use `AvatarFallback` with initials. Props in our wrapper: `name` (string or null), `size` (`sm` 24px, `default` 32px, `lg` 40px).

- Initials are the first two letters of the creator `name`, in uppercase: "Ana" gives "AN".
- `creator` is `null` after a user is deleted. Show "?" with an outline and the text "Deleted user".
- Fill is `muted`, text is `foreground` (15.4:1 light, 13.8:1 dark).
- When the name shows beside it, the avatar is decorative: `aria-hidden="true"`. Alone, it has `role="img"` and an `aria-label` with the name.
- Use `default` in `UserMenu` and `lg` on the reviewer page.
- Use it only in `UserMenu` and at the top of the reviewer page. Cards and bylines show the name as text. See decision 00009.
