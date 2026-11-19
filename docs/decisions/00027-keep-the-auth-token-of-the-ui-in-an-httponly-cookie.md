---
description: The UI keeps its auth token in an HttpOnly cookie that the API sets, not in localStorage. Requests with the cookie get a check against cross-origin requests.
status: implemented
---

# Motivation
[motivation]: #motivation

The UI kept the auth token in `localStorage`.
A script that runs on the page can read `localStorage` and send the token to another site.
The token then works from any device until it expires or the user signs out.
The Content Security Policy of decision 00005 makes such a script unlikely. It does not make it impossible.

A cookie with the `HttpOnly` attribute is not visible to scripts.
A script on the page can still send requests as the user while the page is open.
It cannot take the token away.

Expected outcome:
- no script on the page can read the auth token of the UI
- clients that send a bearer token work as before
- a cross-origin page cannot use the cookie

# Scope
[scope]: #scope

In scope:
- the `cookie` query parameter of `signUp`, `signIn`, and `refreshToken`
- the `reviews_session` cookie, and the `cookieAuth` security scheme in the spec
- the check against cross-origin requests with the cookie
- the session of the UI, its sync between tabs, and the move of a stored token to the cookie
- the tests

Out of scope:
- the bearer token. Other clients and the end-to-end tests keep it.
- a sign out of one device. Decision 00015 has the reason.
- a cookie for the pages of the UI. The server renders no page that needs a user.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Sign in with the cookie

```
POST /v1/users/tokens?cookie=true
Content-Type: application/json

{"email": "ana@example.com", "password": "correct-horse-battery"}
```

The response is `200 OK` with this header:

```
Set-Cookie: reviews_session=eyJhbGciOi...; Path=/v1; Max-Age=604800; HttpOnly; Secure; SameSite=Strict
```

The body has `expires` and `user`. It has no `token`.
`signUp` and `refreshToken` take the same parameter.

## Requests with the cookie

The browser sends the cookie on each request to `/v1`.
The API then uses the token in the cookie as the bearer token.

- an `Authorization` header wins over the cookie. The API then ignores the cookie.
- a `refreshToken` with the cookie sets a new cookie. The body has no `token`, also without `cookie=true`.
- a `signOut` with the cookie deletes the cookie
- a `401` for a cookie that is not valid deletes the cookie. The next request then has no cookie.

A `refreshToken` with a bearer token and `cookie=true` moves the token into a cookie.
The UI uses this one time for a token that an older version stored in `localStorage`.

## Cross-origin requests

The API ignores the cookie and the `cookie` parameter on a cross-origin `POST`, `PATCH`, or `DELETE` request from a browser.
- a cross-origin read uses the cookie. CORS does not let the other page read the response.
- a write gets `401 Unauthorized`
- a sign in gets the token in the body and no cookie

## The UI

The UI keeps `expires` and `user` in `localStorage`. They are not secret.
The tabs of one browser stay equal as in decision 00015.
The cookie is the same for all tabs.

# Implementation Details
[implementation-details]: #implementation-details

## The cookie

- `Path=/v1` sends the cookie only to the API. The UI build, the photos in `<img>`, and the PocketBase routes do not get it.
- `HttpOnly` hides the cookie from scripts
- `SameSite=Strict` stops the browser from sending the cookie on a request from another site
- `Max-Age` is the time until the `exp` claim of the token. The browser deletes the cookie with the token.
- `Secure` is set when the request came over TLS.
  Behind the reverse proxy of decision 00022, the app gets plain HTTP. An `X-Forwarded-Proto` header then gives the scheme. Caddy sets it.
  Without this header, `Secure` is set when the Application URL starts with `https://` and has the host of the request.
  A client can send a wrong `X-Forwarded-Proto`. This changes only its own cookie.
  A `Secure` cookie on a plain HTTP request is not stored. Sign in then seems to work, but no request has the cookie.
  The end-to-end tests have an `https` Application URL with another host. They send requests to `http://127.0.0.1`. WebKit does not keep a `Secure` cookie for this address.
- the name has no `__Host-` prefix. This prefix needs `Path=/`.

## Cross-origin requests

`SameSite=Strict` does not stop a request from a different origin of the same site, such as another subdomain.
The API therefore checks each request that has the cookie or `cookie=true` with `http.CrossOriginProtection` of the Go standard library.

- a `Sec-Fetch-Site` header of `same-origin` or `none` passes. All current browsers send this header.
- without `Sec-Fetch-Site`, an `Origin` header with the host of the request passes. A request without `Origin` also passes. It does not come from a browser.
- `GET`, `HEAD`, and `OPTIONS` always pass. They do not change data.
  The CORS settings of PocketBase do not allow credentials. A cross-origin page cannot read a response to a request with the cookie.
- the check does not reject a request. It removes the cookie and the `cookie` parameter. No operation gets a new `403` response.
- a request with a bearer token has no check. A page cannot add an `Authorization` header to a request of another origin without CORS.

The check of `signIn` and `signUp` stops a login CSRF. A cross-origin page cannot sign the browser in as another user.

## The server

- `authenticate` in `app/internal/publicapi/server.go` reads the cookie when the request has no `Authorization` header
- the state of the request records that the token came from the cookie
- `SignIn`, `SignUp`, and `RefreshToken` set the `Set-Cookie` header of the spec and leave out `token`
- the 401 of `prepare` for a cookie that is not valid has a `Set-Cookie` header with `Max-Age=0`
- the `cookieAuth` scheme is in the spec of each operation that has `bearerAuth`. A test checks that the cookie name of the spec and of the code are equal.

## The UI

- `saveSession` keeps `expires` and `user`. `sessionToken` and the `accessToken` of the SDK configuration are gone.
- sign in, sign up, and the token refresh send `cookie=true`
- a stored session with a `token` is from an older version. The first refresh sends this token in the `Authorization` header with `cookie=true`. The session then has no token.
- a middleware sends requests with `credentials: 'omit'` while the UI has no session. The sign up, sign in, and sign out requests are exceptions.
  A cookie that a failed sign out left in the browser then has no effect on the pages.
- a `401` on a request with credentials ends the session. The middleware sends the request again with `credentials: 'omit'`.
- the sign out sends the cookie with `keepalive`. A `401` with `error="invalid_token"` still means success. See decision 00015.
  A `401` without it means that the request had no cookie. The UI then warns that other devices can stay signed in.

Accepted risks:
- a failed sign out leaves the cookie in the browser until it expires or the next sign in replaces it. The token stays valid on the server. Decision 00015 has the same risk for other devices.
  The UI does not send this cookie while it has no session.
- a script on the page can send requests as the user while the page is open. The Content Security Policy is the defence against such a script.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
