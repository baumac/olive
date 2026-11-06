# SignInForm

The sign in form for the Public UI. People must sign in to write, edit and delete reviews and restaurants.

Project component built from `Card`, `Form`, `Input`, `Button` and `Alert`. Lives in `@/components/ui/sign-in-form`. Props: `loading` (boolean), `error` (string or null). Event: `submit` with `{ email, password }`. The screen calls the API, stores the token and goes back to the page the person came from.

- Fields: "Email" (`type="email"`, `autocomplete="email"`) and "Password" (`type="password"`, `autocomplete="current-password"`). Both required.
- One `default` button "Sign in", full width. While the request runs, disable it and show "Signing in…".
- A wrong email or password shows one destructive `Alert` above the fields: "The email or password is not correct". Do not say which one is wrong.
- Center the card on the page, at most 384px wide.
- Let password managers fill the fields. Do not block paste.
- Sign up and password reset are not in the Public API yet. Add links here when they exist.
