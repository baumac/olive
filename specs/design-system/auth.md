# User authentication

How people sign up, sign in and stay signed in. The API is the User Auth API in `specs/api/users-openapi.yaml`. PocketBase serves it for the `users` collection.

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
2. The UI calls `signUp` (`POST /api/collections/users/records`) with `name`, `email`, `password` and `passwordConfirm`.
3. The response has no token. The UI calls `signIn` with the same email and password.
4. The UI stores the token and the user, opens the `redirect` path and shows the "Welcome to Sponga47 Reviews" `Toast`.

If step 3 fails, open `/sign-in` with the email filled in and show the "Account created" `Toast`.

## Sign in

1. The person fills in `SignInForm` and selects "Sign in".
2. The UI calls `signIn` (`POST /api/collections/users/auth-with-password`) with `identity` (the email) and `password`.
3. The UI stores `token` and `record`, then opens the `redirect` path.

## Stay signed in

- Store the token and the user record in `localStorage`, so a reload keeps the person signed in.
- Send the token as `Authorization: Bearer <token>` on create, update and delete calls to the Public API. Read calls do not need it.
- When the app starts with a stored token, call `refreshAuth` (`POST /api/collections/users/auth-refresh`). On `200`, store the new token and user. Until it answers, show `AppHeader` with the stored user.
- Refresh again when less than one day is left before the token expires. The expiry is the `exp` claim of the token.
- On `401` or `403` from `refreshAuth`, or `401` from the Public API: delete the stored token and user. If the page needs an account, go to `/sign-in?reason=session-ended&redirect=<current path>`. Otherwise stay on the page and show the "Sign in" button.
- Do not lose what the person typed. Before a form sends a request that needs an account, keep a draft of the fields in `sessionStorage` and restore it after sign in.

## Sign out

- `UserMenu` has "Sign out". The User Auth API has no sign out call.
- Delete the stored token and user, show the "Signed out" `Toast` and open the home page.

## Errors

Errors from the User Auth API use the PocketBase format, not Problem Details: `{ status, message, data }`. `data` has one key per invalid property, with a `code` and a `message`.

- Show field errors next to their field. `SignUpForm` lists the codes and their texts.
- A sign in `400` with empty `data` means a wrong email or password.
- For `5xx` and network errors, show the form's destructive `Alert` and keep what the person typed.
- Do not show the PocketBase `message` when there is a text for the code.

## Not in the User Auth API

These have no API yet, so the UI does not show them: password reset, email verification, changing the name, email or password, deleting an account, and sign in with other providers.
