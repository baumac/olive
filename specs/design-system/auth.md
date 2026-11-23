# User authentication

How people sign up, sign in and stay signed in. The API is the Users API: the `users` tag in `specs/api/public-openapi.yaml`.

## Who needs an account

- Reading restaurants and reviews needs no account.
- Adding a restaurant and writing, editing or deleting a review needs an account.
- When a signed out person starts one of these actions, such as "Add review" or "Add a visit", go to `/sign-in?redirect=<the action's path>`. Do not show a disabled button.

## Approval

- A new account waits for the approval of an administrator. The `user` of the auth token has `approved`.
- While `approved` is `false`, `ApprovalBanner` shows below `AppHeader` on each page: an `Alert` with the title "Your account waits for approval", the reason, and an `outline` button "Check again".
- "Check again" calls `refreshToken` and stores the new `user`. It then shows a `Toast`: "Your account is approved" or "Your account is not approved yet". A failed request shows the destructive `Toast` "The check did not work".
- The person can read all pages. A link to a form that adds or changes data stays on the page and shows the `Toast` "Your account waits for approval". A direct load of such a form first gets the `user` again. If the account still waits, it opens `/`.
- A `403` with the `code` `not_approved` means that an administrator took the approval back. The UI stores `approved` as `false`, so the banner shows.
- The heart and "Save to list" show the same `Toast` and send no request. The journal of the person has no "New list" button.
- Do not show disabled buttons. The banner tells the reason one time.

## Pages

- `/sign-in`: `SignInPage` with `SignInForm`.
- `/sign-up`: `SignUpPage` with `SignUpForm`.
- `/confirm-email`: `ConfirmEmailPage`. The link of the confirmation email opens it.
- `/forgot-password`: `ForgotPasswordPage` with one "Email" field.
- `/reset-password`: `ResetPasswordPage`. The link of the password reset email opens it.
- `/account`: `AccountPage`. `UserMenu` has the "Account" item. The page needs an account.
- The links of the emails have the token after `#token=`. The page reads the token and then removes it from the address.
- Both pages take a `redirect` query and return to it after success. Accept only paths that start with `/` and not `//`. The default is `/`, the table.
- The two forms link to each other and keep the `redirect` query.

## Sign up

1. The person fills in `SignUpForm` and selects "Create account".
2. The UI calls `signUp` (`POST /v1/users`) with `name`, `email` and `password`.
3. The response is `202` for a new and for a used email address. The page shows the card "Check your email" with the email address. The person is not signed in.
4. The card has the `outline` button "Send the email again". It sends the same sign up again.
5. The person opens the link of the email, fills in the "Password" field with the password of the sign up, and selects "Confirm email address". `ConfirmEmailPage` then calls `confirmEmail` (`POST /v1/users/confirmations`) with the token and the password and stores `expires` and `user`. The page sends no request before the click, so a mail scanner does not use the link.
   - A wrong password is a `422` with the property `password`. Show "The password is not correct. Enter the password of your sign up." next to the field and keep the form. The link still works.
   - A `429` with the `code` `wrong_password_limit` shows "Too many wrong passwords. Try again later." next to the field.
   - The field has the link "Forgot your password?" below it. A password reset also confirms the email address.
6. The UI opens the `redirect` path of the sign up and shows the "Welcome to Sponga47 Reviews" `Toast`. For an account that is not approved, the `Toast` also tells that an administrator must approve it.
7. A link that is not valid, is too old, or was used shows the destructive `Alert` "This link does not work" and a link to the sign in page.

## Sign in

1. The person fills in `SignInForm` and selects "Sign in".
2. The UI calls `signIn` (`POST /v1/users/tokens`) with `email` and `password`.
3. The UI stores `token`, `expires` and `user`, then opens the `redirect` path.
4. A `403` with the `code` `email_not_confirmed` shows the destructive `Alert` "Confirm your email address first". The API sent a new link.
5. The form has the link "Forgot your password?" below the "Password" field.

## Password reset

1. The person enters the email address on `/forgot-password` and selects "Send the link".
2. The UI calls `requestPasswordReset` (`POST /v1/users/password-resets`). The response is `202` for all email addresses.
3. The page shows the card "Check your email". Its text starts with "If ... has an account".
4. The link of the email opens `/reset-password`. The person enters "New password" and "Confirm new password".
5. The UI calls `resetPassword` (`PUT /v1/users/password`), deletes the stored session, opens `/sign-in`, and shows the `Toast` "Your password is changed".
6. A token that the API refuses shows the destructive `Alert` "This link does not work" and the link "Ask for a new link".

## Account page

`AccountPage` has three cards.

- "Name": one field and the button "Save name". The button is disabled while the name is the stored name. It calls `updateUser` and stores the new `user`.
- "Password": "Current password", "New password", "Confirm new password", and the button "Change password". It calls `changePassword` with `cookie=true` and stores the new session. The other devices are signed out.
- "Delete account": a `destructive` button opens an alert dialog with a "Password" field. "Delete account" calls `deleteUser`, deletes the stored session, opens the home page, and shows the `Toast` "Your account is deleted".
- A wrong password is a `422`. Show "The password is not correct." next to the field.
- A message of the two new password fields goes away while the person types. A message that goes away on blur moves the button, and the click misses it.

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
- `401` on sign in means a wrong email or password.
- `429` on sign up, sign in, the confirmation, or the password reset means too many tries. Show the form's destructive `Alert` "Too many tries" / "Wait a minute and try again."
- Sign up and sign in ignore the stored token. Do not delete it before these calls.
- For `5xx` and network errors, show the form's destructive `Alert` and keep what the person typed.

## Not in the Users API

These have no API, so the UI does not show them: a change of the email address and sign in with other providers.
