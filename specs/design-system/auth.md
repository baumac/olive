# User authentication

How people sign up, sign in and stay signed in. The API is the Users API: the `users` tag in `specs/api/public-openapi.yaml`.

## Who needs an account

- Reading restaurants and reviews needs no account.
- Adding a restaurant and writing, editing or deleting a review needs an account.
- When a signed out person starts one of these actions, such as "Write a review", go to `/sign-in?redirect=<the action's path>`. Do not show a disabled button.

## Pages

- `/sign-in`: `SignInPage` with `SignInForm`.
- `/sign-up`: `SignUpPage` with `SignUpForm`.
- Both pages take a `redirect` query and return to it after success. Accept only paths that start with `/` and not `//`. The default is `/restaurants`.
- The two forms link to each other and keep the `redirect` query.

## Sign up

1. The person fills in `SignUpForm` and selects "Create account".
2. The UI calls `signUp` (`POST /v1/users`) with `name`, `email` and `password`.
3. The response is an auth token, so the person is signed in. The UI stores `token`, `expires` and `user`.
4. The UI opens the `redirect` path and shows the "Welcome to Sponga47 Reviews" `Toast`.

## Sign in

1. The person fills in `SignInForm` and selects "Sign in".
2. The UI calls `signIn` (`POST /v1/users/tokens`) with `email` and `password`.
3. The UI stores `token`, `expires` and `user`, then opens the `redirect` path.

## Stay signed in

- Store `token`, `expires` and `user` in `localStorage`, so a reload keeps the person signed in.
- Send the token as `Authorization: Bearer <token>` on create, update and delete calls to the Public API. Read calls do not need it.
- When the app starts with a stored token, call `refreshToken` (`POST /v1/users/tokens/refresh`). On `200`, store the new `token`, `expires` and `user`. Until it answers, show `AppHeader` with the stored user.
- Refresh again when less than one day is left before `expires`. Do not parse the token.
- On `401` from `refreshToken` or from any other call that sends the token: delete the stored token and user. If the page needs an account, go to `/sign-in?reason=session-ended&redirect=<current path>`. Otherwise stay on the page and show the "Sign in" button.
- Do not lose what the person typed. Before a form sends a request that needs an account, keep a draft of the fields in `sessionStorage` and restore it after sign in.

## Sign out

- `UserMenu` has "Sign out". The Users API has no sign out call.
- Delete the stored token and user, show the "Signed out" `Toast` and open the home page.

## Errors

Errors use RFC 9457 Problem Details, like the rest of the Public API. A `422` has an `errors` array with one item per invalid property: `{ property, message }`.

- Check the fields in the UI with the same rules as the API, so most errors never reach the API.
- Show a `422` item next to the field that its `property` names. Use the UI text for that field and rule. Use the API `message` only when the UI has no text.
- `409` on sign up means another account has the email.
- `401` on sign in means a wrong email or password.
- `429` on sign up or sign in means too many tries. Show the form's destructive `Alert` "Too many tries" / "Wait a minute and try again."
- Sign up and sign in ignore the stored token. Do not delete it before these calls.
- For `5xx` and network errors, show the form's destructive `Alert` and keep what the person typed.

## Not in the Users API

These have no API yet, so the UI does not show them: password reset, email verification, changing the name, email or password, deleting an account, and sign in with other providers.
