# SignInPage

The page at `/sign-in`. It shows `SignInForm` under `AppHeader`.

Page, not a component. Lives in `ui/src/pages/SignInPage.vue`.

- Route: `/sign-in`. Query `redirect`: the path to open after sign in, for example `/restaurants/k3x9m2p7q1w8z4a/reviews/new`. Accept only paths that start with `/` and not `//`. Without it, open `/restaurants`.
- Query `reason=session-ended`: show the "Your session has ended" notice. The app adds it when a token refresh fails.
- Page title: "Sign in · Sponga47 Reviews".
- `AppHeader` shows the "Sign in" button with `aria-current="page"`.
- The card is centered, at most 384px wide, with 32px space above. On small screens it fills the width with 16px side padding.
- A person who is already signed in and opens `/sign-in` goes straight to the `redirect` path.
- Focus the "Email" field when the page opens. With an email already filled in, focus "Password".
