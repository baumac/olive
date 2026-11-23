# UserMenu

The account menu at the right end of `AppHeader`, at every width. Below `md` it is the only way to the account page, the theme and "Sign out".

Project component built from shadcn-vue `DropdownMenu` and `Avatar`. Lives in `@/components/ui/user-menu`. Props: `user` (the `user` from the auth token response, or `null`). Events: `signOut`.

- Signed out: an `outline` button "Sign in" that goes to `/sign-in` with the current path as `redirect`.
- Signed in: a round ghost button with the `Avatar`. Its `aria-label` starts with the visible initials: "<initials>, account menu for <name>", such as "AN, account menu for Ana". Speech input can then use the initials (WCAG 2.5.3 Label in Name).
- The menu label shows the `name` and, under it, the `email`. The API sends `email` for the signed in user.
- Items: "You", "Account", a separator, the "Theme" radio group with "Light", "Dark" and "System", a separator, "Sign out". "You" opens the journal of the user at `/reviewers/{id}`, with the entries, the favorites and the lists. "Account" opens `/account`.
- "Sign out" calls `signOut`, which ends all sessions of the user. It then shows a "Signed out" `Toast` and goes to the table. See decision 00003.
- The menu is `popover` with `shadow-md` and `z-popover`. Each item is at least 44px high. The highlighted item is `accent`.
