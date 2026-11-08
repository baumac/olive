# SignUpForm

Creates an account with an email address and a password, then signs the person in.

Project component built from `Card`, `Form`, `Input`, `Button` and `Alert`, like `SignInForm`. Lives in `@/components/ui/sign-up-form`. Props: `loading` (boolean), `errors` (field messages keyed by property), `error` (string or null). Event: `submit` with `{ name, email, password, passwordConfirm }`. The screen calls `POST /api/collections/users/records` (`signUp` in `users-openapi.yaml`). That response has no token, so the screen then calls `signIn` with the same email and password. See the "User authentication" section.

- Title "Create an account". Description: "With an account you can add restaurants and write reviews."
- Fields, in this order:
  - "Name (optional)": `autocomplete="nickname"`, at most 255 characters. Help text: "Other people see this name on your restaurants and reviews." An empty name shows as "Anonymous".
  - "Email": `type="email"`, `autocomplete="email"`, at most 255 characters. Required.
  - "Password": `autocomplete="new-password"`, 8 to 71 characters. Help text: "8 to 71 characters." Required. Has the show/hide toggle from `SignInForm`.
  - "Confirm password": `autocomplete="new-password"`, sent as `passwordConfirm`. Required. Has the show/hide toggle.
- Do not ask for `emailVisibility`. The UI never shows email addresses to other people, so always send the default `false`.
- Check the fields when the person leaves each one and on submit, with the same rules as the API. Show the first error next to its field and move focus to the first invalid field.
- One `default` button "Create account", full width. While the requests run, disable it and show "Creating account…".
- Below the button: "Already have an account? Sign in", linking to the sign in page with the same `redirect` query.

Error texts. A `400` response has one `data` key per invalid property. Map the `code` to these texts and fall back to the PocketBase `message`:

| Property | Code | Text |
| --- | --- | --- |
| `email` | `validation_required` | "Enter your email address." |
| `email` | `validation_is_email` | "Enter an email address, for example ana@example.com." |
| `email` | `validation_not_unique` | Destructive `Alert` above the fields: "An account with this email already exists" / "Sign in with this email instead.", with "Sign in" as a link. |
| `password` | `validation_required` | "Enter a password." |
| `password` | `validation_length_out_of_range` | "Use 8 to 71 characters." |
| `passwordConfirm` | `validation_values_mismatch` | "The passwords do not match." |
| `name` | `validation_length_out_of_range` | "Use at most 255 characters." |

- Any other error shows the destructive `Alert` "Your account was not created" / "Try again in a moment."
- If sign up works but the sign in after it fails, go to the sign in page. Show a success `Toast`: "Account created" / "Sign in to continue."
- Email verification is not in the User Auth API. Do not tell people to check their email.
