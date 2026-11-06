# UserMenu

The account menu at the right end of `AppHeader`.

Project component built from shadcn-vue `DropdownMenu` and `Avatar`. Lives in `@/components/ui/user-menu`. Props: `user` (`{ name, email }` or `null`). Events: `signOut`.

- Signed out: an `outline` button "Sign in" that goes to the sign in screen.
- Signed in: a round ghost button with the `Avatar`. Its `aria-label` is "Account menu for <name>".
- The menu shows the name and email, then "My reviews", "Account", a separator and "Sign out".
- "Sign out" removes the token, shows a "Signed out" `Toast` and goes to the home screen.
- The menu is `popover` with `shadow-md` and `z-popover`. The highlighted item is `accent`.
- The "Account" screen is not designed yet. Leave the item out until it exists.
