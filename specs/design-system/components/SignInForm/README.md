# SignInForm

Signs a person in with an email address and a password. People must sign in to add restaurants and to write, edit and delete reviews. Reading needs no account.

Project component built from `Card`, `Form`, `Input`, `Button` and `Alert`. Lives in `@/components/ui/sign-in-form`. Props: `loading` (boolean), `error` (string or null), `notice` (`"session-ended"` or null). Event: `submit` with `{ email, password }`. The screen calls `POST /v1/users/tokens` (`signIn` in `public-openapi.yaml`), stores the token and goes back to the page in the `redirect` query. See the "User authentication" section for the whole flow.

- Title "Sign in". Description: "Sign in to add restaurants and write reviews."
- Fields: "Email" (`type="email"`, `autocomplete="username"`) and "Password" (`type="password"`, `autocomplete="current-password"`). Both required. Check before sending that both have a value.
- The password field has a show/hide button inside it. It is a toggle: `aria-label="Show password"`, `aria-pressed` true while the text shows. It keeps focus in the field order after the input.
- One `default` button "Sign in", full width. While the request runs, disable it and show "Signing in…".
- `401` means the email and password do not match a user. Show one destructive `Alert` above the fields: "The email or password is not correct" / "Check both and try again." Do not say which one is wrong. Keep the email, clear the password and move focus to the password field.
- `422` is a field problem. Show each `errors` item next to the field that its `property` names.
- Any other error shows the destructive `Alert` "Sign in did not work" / "Try again in a moment."
- With `notice="session-ended"`, show a default `Alert` above the fields: "Your session has ended" / "Sign in again to continue."
- Below the button: "New to Olive? Create an account", linking to the sign up page with the same `redirect` query.
- Let password managers fill the fields. Do not block paste.
- Below the "Password" field: the link "Forgot your password?" to `/forgot-password`. The "User authentication" section has the password reset.
