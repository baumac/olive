# SignUpPage

The page at `/sign-up`. It shows `SignUpForm` under `AppHeader`.

Page, not a component. Lives in `ui/src/pages/SignUpPage.vue`.

- Route: `/sign-up`. Query `redirect`: the same rules as `SignInPage`.
- Page title: "Create an account · Sponga47 Reviews".
- `AppHeader` shows the "Sign in" button. The sign up page has no header link of its own. People reach it from the link on the sign in page and from "Create an account" prompts.
- The card layout is the same as `SignInPage`.
- After the account is created and the sign in works, open the `redirect` path and show a success `Toast`: "Welcome to Sponga47 Reviews" / "You are signed in as <name or email>."
- A person who is already signed in and opens `/sign-up` goes straight to the `redirect` path.
- Focus the "Name" field when the page opens.
