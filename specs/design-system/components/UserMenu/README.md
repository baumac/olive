# UserMenu

The account menu at the right end of `AppHeader`.

Project component built from shadcn-vue `DropdownMenu` and `Avatar`. Lives in `@/components/ui/user-menu`. Props: `user` (the `user` from the auth token response, or `null`). Events: `signOut`.

- Signed out: an `outline` button "Sign in" that goes to `/sign-in` with the current path as `redirect`.
- Signed in: a round ghost button with the `Avatar`. Its `aria-label` starts with the visible initials: "<initials>, account menu for <name>", such as "AN, account menu for Ana". Speech input can then use the initials (WCAG 2.5.3 Label in Name).
- The menu label shows the `name` and, under it, the `email`. The API sends `email` for the signed in user.
- Items: "My reviews", "My lists", a separator, the "Theme" radio group with "Light", "Dark" and "System", a separator, "Sign out". "My reviews" opens `/my-reviews`. "My lists" opens the reviewer page of the user. Its "Lists" section has the lists.
- "Sign out" deletes the stored token and user, shows a "Signed out" `Toast` and goes to the home screen. The Users API has no sign out call, so this happens only in the browser.
- The Users API cannot change a name, email or password. Do not add an "Account" item until it can.
- The menu is `popover` with `shadow-md` and `z-popover`. The highlighted item is `accent`.
