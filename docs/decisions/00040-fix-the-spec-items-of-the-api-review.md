---
description: Fix the spec errors that the review of the Public API found. Each `403` and `409` problem gets a `code`. The API removes the spaces around names, titles, addresses, and alt texts.
status: implemented
---

# Motivation
[motivation]: #motivation

The review of the Public API found texts of the spec that did not agree with the server, and rules that only some operations followed.

- the texts about the cookie named `POST`, `PATCH`, and `DELETE`. The server also ignores the cookie on a cross-origin `PUT`.
- `getReviewer` and `refreshToken` had no `400` response. The server gives `400` for a malformed ID and for a malformed `cookie` parameter.
- the `password` of `Credentials` had no maximum length
- only list names lost the spaces at the start and the end
- the `favorites` alias did not tell why only two operations take it
- the list of versions in the description did not have the lists, the approval, the limits, and the restaurant details. It also did not tell why the path stays `/v1`.
- a client had to read the `detail` text to tell most `403` and `409` reasons apart
- `AGENTS.md` told that each write operation has a rate limit. Sign out and the token refresh have none.

# Scope
[scope]: #scope

In scope:
- the texts and the responses of `specs/api/public-openapi.yaml`
- the `code` of the `403` and `409` problems
- the spaces around names, titles, addresses, and alt texts

Out of scope:
- a rate limit for `signOut` and `refreshToken`. A user can then always sign out, and the UI calls `refreshToken` on each page load.
- the `favorites` alias on the read operations of `/lists`. `listReviewerLists` gives the ID of the favorites list.
- the other findings of the review

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## Codes

Each `403` and each `409` problem has a `code`.

| `code` | Status | Reason |
|---|---|---|
| `not_approved` | 403 | no superuser approved the user |
| `email_not_confirmed` | 403 | a sign in before the confirmation |
| `not_creator` | 403 | the user is not the creator. For a restaurant without a creator, the user has no review of it. |
| `favorites_list` | 403 | a rename or a delete of a favorites list |
| `review_exists` | 409 | a second review of one user for a restaurant |
| `other_reviews` | 409 | a delete, a rename, or a move of a restaurant with a review of another user |
| `place_taken` | 409 | a `location.place_id` that another restaurant has |
| `photo_limit` | 409 | a sixth photo of a review |
| `list_name_taken` | 409 | a list name that another list of the user has |
| `list_limit` | 409 | a list above the 50 lists of a user |
| `list_restaurant_limit` | 409 | a restaurant above the 500 restaurants of a list |

The `property` and `restaurant` properties of a problem do not change.

The UI selects its texts by the `code`, not by the status.
- `apiFailure` gives the problem of a failed call with its `code`
- `failureText` has the texts of `not_approved` and `not_creator` for a failure without a text of its own
- `listFailureText` has the texts of the list codes. A full list and the list limit had the `detail` text of the API before.
- a `403` with `not_approved` on a form now tells about the approval. It told before that the person is not the creator.
- a `403` or `409` with a code that the page does not know gets the general text of the page

## Spaces

- the API removes the ASCII spaces at the start and the end of a `Name`, a `Title`, an `Address`, and an `AltText`
- ` Luigi's ` and `Luigi's` are the same name. A patch with the first value does not change a restaurant with the second value.
- other whitespace stays. The patterns of these schemas allow a no-break space as a character of the value.
- a migration removes the spaces of the stored values. The `updated` time of the records stays.

## Versions

- the description of the spec has one list item for each version
- the `/v1` of the path is the name of the API. It does not follow the version of the document.
- version 3.0.0 also names the lists, the approval, and the limits. They came after 2.2.0 without a new version.
- version 4.0.0 also names the restaurant details of decision 00038 and the changes of this decision

# Implementation Details
[implementation-details]: #implementation-details

- `forbidden` and `conflict` of `problem.go` need a code, so a new reason cannot leave it out
- the `expect` helper of the tests fails for a `403` or `409` of `/v1` without a `code`
- the four schemas have the `x-trim: true` extension. `readBody` removes the spaces of each value of such a schema after the validation. The handlers then get the changed body, so a new operation needs no code for the rule.
- only the ASCII space goes away. `strings.TrimSpace` also removes a no-break space. A name of only this character passes the pattern and then became empty, with a `500` response. List names had this error before this decision.
- the migration `1794268817_trim_stored_names.go` uses `TRIM` of SQLite, which removes the same character. Without it, a stored value with spaces is not equal to the same value of a request.
- the cross-origin check is `http.CrossOriginProtection`. It covers all methods that are not safe, so only the text changed.
- the server already gave `400` for the two operations through the `default` response. Only the spec changed.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
