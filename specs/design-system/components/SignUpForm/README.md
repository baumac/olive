# SignUpForm

Creates an account with an email address and a password, then signs the person in.

Project component built from `Card`, `Form`, `Input`, `Button` and `Alert`, like `SignInForm`. Lives in `@/components/ui/sign-up-form`. Props: `loading` (boolean), `errors` (field messages keyed by property), `error` (string or null). Event: `submit` with `{ name, email, password }`. The screen calls `POST /v1/users` (`signUp` in `public-openapi.yaml`). The response has an auth token, so the person is signed in. See the "User authentication" section.

- Title "Create an account". Description: "With an account you can add restaurants and write reviews."
- Fields, in this order:
  - "Name": `autocomplete="nickname"`, 1 to 255 characters, not only spaces. Help text: "Other people see this name on your restaurants and reviews." Required.
  - "Email": `type="email"`, `autocomplete="email"`, at most 255 characters. Required.
  - "Password": `autocomplete="new-password"`, 8 to 71 characters and at most 72 bytes in UTF-8. Help text: "8 to 71 characters. Show the password to check it." Required. Has the show/hide toggle from `SignInForm`.
- The form has no "Confirm password" field. The show/hide toggle lets the person check the password. See decision 00019.
- Check the fields when the person leaves each one and on submit, with the same rules as the API. Show the first error next to its field and move focus to the first invalid field.
- One `default` button "Create account", full width. While the request runs, disable it and show "Creating account…".
- Below the button: "Already have an account? Sign in", linking to the sign in page with the same `redirect` query.

Error texts. The UI checks each field with these rules. A `422` from the API names the field in `property`. Show the UI text for that field and fall back to the API `message`:

| Field | Problem | Text |
| --- | --- | --- |
| `name` | empty or only spaces | "Enter your name." |
| `name` | too long | "Use at most 255 characters." |
| `email` | empty | "Enter your email address." |
| `email` | not an email address | "Enter an email address, for example ana@example.com." |
| `password` | empty | "Enter a password." |
| `password` | too short or too long | "Use 8 to 71 characters." |

- `409` means another account has the email. Show a destructive `Alert` above the fields: "An account with this email already exists" / "Sign in with this email instead.", with "Sign in" as a link.
- Any other error shows the destructive `Alert` "Your account was not created" / "Try again in a moment."
- Email verification is not in the Users API. Do not tell people to check their email.
