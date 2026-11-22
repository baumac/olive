---
description: The OpenAPI spec of the Public API is the contract between the UI and the app. It has the common rules of all operations. Both SDKs are generated from it, and the app validates each request and response against it.
status: implemented
---

# Motivation
[motivation]: #motivation

The UI and the app are developed apart.
They need one contract that both follow.
If each side writes its own types and checks, the two sides drift without a failed test.

`specs/api/public-openapi.yaml` is this contract.
Clients use only the Public API. They do not depend on PocketBase.
The SDKs are generated from the spec, so the compiler finds code that does not match it.
The app validates each request and each response against the spec at runtime, so the spec is the only copy of each rule.

Expected outcome:
- one spec defines the paths, the schemas, the auth, and the errors of the Public API
- all operations follow the same rules for formats, bodies, lists, and errors
- the UI calls the API only through a generated TypeScript client
- the app implements a generated Go server interface
- one `make` command generates both SDKs. A check fails when the generated code does not match the spec.
- a request that breaks the spec gets `400` or `422`. A response that breaks the spec does not reach the client.

# Scope
[scope]: #scope

In scope:
- the spec file, its tags, and its versions
- the common rules: auth basics, formats, text rules, request bodies, list operations, and errors
- the names of the restaurant and the recipe parts of the spec
- the TypeScript SDK and the Go SDK, and the `make` targets that generate and check them
- the validation of requests and responses at runtime

Out of scope:
- the operations of each topic. Decisions 00003, 00004, 00005, 00006, 00007, and 00008 have them.
- the auth tokens, the cookie, and the approval. Decision 00003 has them.
- the rate limits. Decision 00003 has them.
- all APIs of PocketBase. Clients use only the Public API.
- publishing the SDKs to npm or to another registry. Only this repository uses them.
- a Go client. The app needs only the server code.
- API documentation from the spec. The spec is the documentation.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The spec

The paths of the spec are relative to `/v1`. The spec has six tags:

| Tag | Operations | Decision |
|---|---|---|
| `users` | sign up, email confirmation, sign in, sign out, token refresh, password reset, and `/users/me` | 00003 |
| `restaurants` | restaurants, their reviews, their photos, and their versions | 00004, 00005, 00007 |
| `recipes` | recipes, their reviews, their photos, and their versions | 00004, 00005, 00007 |
| `reviewers` | the public profile of a user and the reviews of a user | 00006 |
| `places` | the place search | 00008 |
| `lists` | the lists of restaurants and recipes | 00006 |

The restaurant and the recipe parts have the same shape. Their names tell the kind:
- the operations and schemas of restaurant reviews have `restaurant`, such as `createRestaurantReview` and `RestaurantReview`
- the operations and schemas of recipe reviews have `recipe`, such as `createRecipeReview` and `RecipeReview`
- the counts of a reviewer and a list name their kind, such as `restaurant_review_count` and `recipe_count`
- the paths under `/restaurants/{restaurantId}/reviews` and `/recipes/{recipeId}/reviews` already name the kind
- `review_count` of a restaurant or a recipe, `ReviewId`, and `ReviewOverallFilter` are the same for both kinds

## Versions

`/v1` is the name of the API. It does not change with the version of the spec.
The `version` of `info` is semantic. Its first number changes when a change can break a client.
The description of the spec has one line for each version.

The UI is the only known client, and it ships in the same binary as the server.
A breaking change then changes the spec, the server, and the UI at the same time. The API keeps no aliases for old names.
A browser tab with the UI from before a deploy can fail until a reload.

## Auth basics

- read operations do not need a token. A visitor without an account can read restaurants, recipes, reviews, reviewers, and lists.
- the version lists and the place search need a token
- create, update, and delete operations need a token and an approved user. Decision 00003 has the approval.
- a request with an invalid token gets `401`, also on a read operation. The API does not treat it as an anonymous request.
- the UI sends the token in the `reviews_session` cookie. Other clients send a bearer token. Decision 00003 has both.

## Formats

- **Timestamp**: RFC 3339 in UTC with three decimals, such as `2024-11-04T01:26:55.002Z`. `new Date(value)` in JavaScript and the ISO 8601 parsers of Swift and Go read it.
- **ID**: the PocketBase record ID. It has 15 characters that match `^[a-z0-9]+$`, such as `k3x9m2p7q1w8z4a`. IDs are random and are not in the order of time.
- **Length**: a string length counts Unicode code points, not bytes. This matches `maxLength` of JSON Schema.
- **Date**: the `date` of a review has no time. Decision 00005 has its range.

## Text rules

- `Name`, `Title`, and `LongText` refuse control characters, direction marks and overrides, such as U+202E and U+061C, and invisible characters.
  The invisible characters are the zero-width spaces, the soft hyphen, the invisible operators, and the Hangul fillers.
  A name with them can look like another name or show as blank space.
- the zero-width joiners stay allowed. Emoji and some scripts use them.
- `LongText`, such as `description` and `feedback`, allows line breaks and tabs.
- a schema with `x-trim: true` loses the ASCII spaces, tabs, and line breaks at the start and the end of its value.
  Names, titles, addresses, alt texts, ingredients, and steps have it.
  ` Luigi's ` and `Luigi's` are then the same name. Other whitespace, such as a no-break space, stays.
- a schema with `x-nfc: true` gets Unicode normalization form C before the validation, in the body and in the query. `Tag` has it.

## Request bodies

- a property that the schema does not define gets `400`. A read-only property also gets `400`.
- a value that breaks a rule of the schema gets `422`
- a create uses `application/json`. Its response is `201` with the new resource and a `Location` header.
- an update uses JSON Merge Patch, RFC 7396, with `application/merge-patch+json`
  - a missing property stays unchanged
  - `null` clears an optional property. `null` on a required property gets `422`.
  - `{}` is a valid patch. It changes nothing and keeps `updated`.
  - the response is `200` with the changed resource
- a delete gets `204` with an empty body
- another content type gets `415`

## List operations

### Filters

A filter is a query parameter `{property}[{operator}]={value}`. All filters of a request must match.

| Operator | Meaning | Example |
|---|---|---|
| `eq` | equal to | `food[eq]=5` |
| `lt`, `lte`, `gt`, `gte` | less than, less or equal, greater than, greater or equal | `created[gte]=2024-11-04T00:00:00.000Z` |
| `in` | equal to one of the values | `creator[in]=u5n1c8r3t6y0b2e&creator[in]=u2g6k0m4o8q1s5w` |
| `has` | the array has the value | `tags[has]=italian` |
| `has_any` | the array has one of the values | `tags[has_any]=italian&tags[has_any]=pasta` |
| `has_all` | the array has all of the values | `tags[has_all]=italian&tags[has_all]=pasta` |

- an operator with more than one value uses a repeated parameter
- an unknown property or operator gets `400`. An operator that a property does not allow also gets `400`.
- the tag operators compare whole tags. `tags[has]=italian` does not match `south-italian`.
- an unknown ID in a filter, such as `creator[eq]` or `list[eq]`, gives an empty list, not `404`

### Search

A list with a `search` parameter does a full-text search. Each topic decision tells the properties of its search.
- an item matches when it has all words of the search
- a word of two or more characters also matches the longer words that start with it. `carb` matches "carbonara".
- the search ignores case and accents. `cafe` matches "Café".
- the search splits words at spaces and punctuation. `date-night` matches the text "date night".
- the search has 1 to 255 characters and at most 10 words. A search without a letter or a digit gets `400`.
- `"`, `*`, and `OR` have no special meaning. The search is plain text.
- a `relevance` sort puts the best matches first. The score of an item adds a weight for each property that has all words of the search.
- the score depends only on the item. A change of another item does not move it, so the pages of a `relevance` sort stay stable.
- with `search` and without `sort`, the sort is `relevance`. A `relevance` sort without `search` gets `400`.

### Sorts

- `sort` is a comma-separated list of properties. A `-` prefix sorts from high to low. `sort=-food,title` sorts by food from high to low, then by title.
- the `name` and `title` sorts ignore the case of the ASCII letters `A` to `Z`. They do not use the rules of a language.
- the API adds `id` as the last sort key. The order is then stable for equal values.
- a sort with a repeated property gets `400`

### Pages

- `limit` sets the page size. The default is 20, and the maximum is 100. Another value gets `400`.
- `cursor` gets the next page. Clients do not read or build cursors.
- a cursor works only with the filters and the sort that made it. Other filters or another sort get `400`.
- the response is `{"items": [...], "next_cursor": "..."}`. `next_cursor` is `null` on the last page.
- a cursor is not a snapshot. An item that changes during the paging can move. A client can then miss it or get it two times.

## Errors

Errors are RFC 9457 Problem Details with `application/problem+json`.
A validation error has an `errors` array with one item for each invalid property.
An array item has a zero-based index, such as `tags[2]`.

```json
{
  "type": "about:blank",
  "title": "Unprocessable Content",
  "status": 422,
  "detail": "The request body has invalid properties.",
  "errors": [{ "property": "food", "message": "must be between 1 and 10" }]
}
```

| Status | Use |
|---|---|
| `400` | a malformed request, an invalid parameter, or an unknown or read-only property |
| `401` | a missing token on an operation that needs one, an invalid token, or a wrong email or password |
| `403` | a user without approval, a sign in before the email confirmation, or a change that only the creator can do |
| `404` | the resource or the parent in the path does not exist |
| `409` | a conflict with stored data, such as a place or a source that another record has, or a full list |
| `413` | a body over the size limit, such as a photo over 10 MiB |
| `415` | a wrong content type |
| `422` | an invalid value in the body |
| `429` | a rate limit. The response has a `Retry-After` header. Decision 00003 has the limits. |
| `503` | the place search is off, or its geocoder does not answer. See decision 00008. |

Each `403` and each `409` has a `code`. Clients select their texts by the `code`, not by the status or the `detail`.
A new reason needs a new value in the `code` enum of `Problem`.

| `code` | Status | Reason | Decision |
|---|---|---|---|
| `not_approved` | 403 | no superuser approved the user | 00003 |
| `email_not_confirmed` | 403 | a sign in before the email confirmation | 00003 |
| `not_creator` | 403 | the user is not the creator | 00004 |
| `favorites_list` | 403 | a rename or a delete of a favorites list | 00006 |
| `other_reviews` | 409 | a delete of a restaurant or a recipe with reviews of other users | 00004 |
| `place_taken` | 409 | a `location.place_id` that another restaurant has | 00004 |
| `source_taken` | 409 | a `source` that another recipe has | 00004 |
| `photo_limit` | 409 | a sixth photo of a review | 00007 |
| `list_name_taken` | 409 | a list name that another list of the user has | 00006 |
| `list_limit` | 409 | a list above the 50 lists of a user | 00006 |
| `list_restaurant_limit`, `list_recipe_limit` | 409 | a restaurant or a recipe above the limit of a list | 00006 |
| `wrong_password_limit` | 429 | too many wrong passwords for an email address | 00003 |

A conflict of one property also has `property`, such as `name`, `location`, or `source`.
A `place_taken` conflict has the `restaurant` that has the place. A `source_taken` conflict has the `recipe`.

## The SDKs

The spec is the only source. Nobody edits the generated files.
To change the API, change the spec and run `make generate`.

```bash
make generate         # Generate the TypeScript SDK and the Go SDK
make check-generate   # Fail if the generated code does not match the spec
make test-sdk         # Run the tests of both SDKs
```

The TypeScript SDK is the npm package `@reviews/sdk` in `app/sdk/typescript`.
It has one API class for each tag. Each method has the name of the `operationId` and takes one request object.

```ts
const restaurants = new RestaurantsApi(config)
const page = await restaurants.listRestaurantReviews({
  restaurantId: 'k3x9m2p7q1w8z4a',
  food: { gte: 7 },
  sort: '-food,title',
})
```

A response with a status that is not `2xx` throws a `ResponseError`. The UI reads the problem from `error.response`.
Each method has a `Raw` form that gives the response headers.

The Go SDK is the module `github.com/baumac/reviews-app/app/sdk/go`. Its package `reviewsapi` has the Go types and the strict server interface.
The strict server interface has one method for each operation. Each method gets a typed request and returns a typed response.
The compiler fails when the app does not implement an operation. It also fails when a handler returns a status or a body that the operation does not define.

# Implementation Details
[implementation-details]: #implementation-details

## Layout

```
go.work                       # hand-written
package.json                  # hand-written npm workspace root
app/sdk/
  typescript/
    package.json              # hand-written
    tsconfig.json             # hand-written
    tsconfig.test.json        # hand-written
    templates/                # hand-written template overrides
    test/                     # hand-written tests
    src/                      # generated
  go/
    go.mod                    # hand-written. A separate module inside app.
    oapi-codegen.yaml         # hand-written generator settings
    reviewsapi_test.go        # hand-written tests
    reviewsapi.gen.go         # generated
```

- the generated files are committed. The app and the UI build without the generators.
- `make generate` deletes `app/sdk/typescript/src` before it runs the generator. The generator does not delete the files of removed schemas.
- the root `go.work` has `app/sdk/go` and `app/api`. The app module also has a `replace` line for the SDK. `go mod tidy` does not use `go.work`, and the SDK module is not published.
- the root `package.json` is an npm workspace with `app/sdk/typescript`, `app/ui`, and `tests`

## TypeScript SDK

- the generator is `typescript-fetch` of OpenAPI Generator. It uses the `fetch` of the browser and has no runtime dependencies.
- the makefile pins the version of the `openapitools/openapi-generator-cli` Docker image. Developers do not need Java.
- the options: `supportsES6: true`, `importFileExtension: .js`, `useSingleRequestParameter: true`, `withInterfaces: true`, `legacyDiscriminatorBehavior: false`, and no API and model docs
- `withInterfaces` lets the UI tests replace an API class with a fake
- `tsc` builds the package into `app/sdk/typescript/dist`. The UI imports the built package. `make build-ui` builds the SDK first.
- the SDK and the UI use one TypeScript version. `vue-tsc` needs the JavaScript API of the compiler, which TypeScript 7 does not have.

## Go SDK

- the generator is oapi-codegen v2. The makefile pins its version, and `make install-tools` installs it into `./bin`.
- `oapi-codegen.yaml` makes the models, a `net/http` server, and the strict server. It sets `nullable-type: true` and `embedded-spec: true`.
- the server uses `net/http`, because the PocketBase router uses `net/http` handlers. Decision 00002 mounts it under `/v1`.
- `nullable.Nullable[T]` tells a missing property from `null` in a merge patch
- the strict server calls hand-written error handlers for decode errors and handler errors. All errors then have one format.
- the embedded spec has gzip data. The Go version that builds `oapi-codegen` changes these bytes. `TOOLS_GO` of the makefile pins this Go version. See decision 00010.

## Hard parts of the spec

Each part has a test in the SDK that it affects. If a generator does not support a part, the settings change or a small fix is added.
The meaning of the spec does not change for a generator without a decision.

| Part | Result |
|---|---|
| merge patch bodies | the client sends `application/merge-patch+json` and keeps `null`. The handler tells a missing property from `null`. |
| `deepObject` filters, such as `food` | the client sends `food[gte]=4`. The server reads the filter into a struct. The filters use named schemas, such as `ScoreRange` and `TimestampRange`, for stable names. |
| array filters with `explode: true` | the client repeats the parameter for each value |
| `Timestamp` is a string with a `pattern` | both SDKs keep the value as a string |
| `next_cursor` is `null` on the last page | Go sends `null`. TypeScript gives `null`. |
| problem responses | Go has a response type for each status. TypeScript throws `ResponseError`. |
| `Location` and `Retry-After` headers | Go response types have header fields. TypeScript gives them through the `Raw` methods. |
| operations with `security: - {}` | the TypeScript client does not send the stored token |

Workarounds:
- `typescript-fetch` changes an optional `null` array with `uniqueItems` to `undefined`. A merge patch then cannot clear `tags`.
  `app/sdk/typescript/templates/modelGeneric.mustache` is a copy of the generator template with a fix in `ToJSON`. Update the copy when the generator version changes.
- `typescript-fetch` makes a `Set<string>` for `tags`, because of `uniqueItems`. The UI changes it to an array where it needs one.

## The out-of-date check

- `make check-generate` runs `make generate`. It fails when `git diff` shows changes in `app/sdk`, or when `app/sdk` has untracked files.
- `make lint` runs it after `lint-spec`. The lint then needs Docker.
- `make lint-spec` checks the spec with vacuum and the rules of `specs/api/vacuum-ruleset.yaml`

## Validation at runtime

`reviewsapi.GetSwagger()` gives the embedded spec as a kin-openapi document. `openapi.go` of `app/api/internal/publicapi` loads it one time.
The `prepare` middleware uses it for each request:

1. it finds the spec operation of the ServeMux pattern. The pattern also gives the path parameters, so the app has no second router.
2. after the rate limit and the token check, it validates the parameters. An invalid parameter gets `400`, such as `The query parameter limit is not valid.`
3. it checks the content type, the size, and the JSON object of the body. Then it validates the body. An invalid property gets `422`.
4. it applies `x-nfc` and `x-trim` to the body values of these schemas. The handlers get the changed body.
5. the handler writes its response into a buffered writer
6. it validates the buffered response. A valid response goes to the client with `Flush`. An invalid response gets a generic `500`, and the log gets the error.

- the spec also gives the token check and the rate limit of each operation: `security` gives the token check, and `x-rate-limit` gives the limit
- `x-allow-unapproved: true` lets a user without approval call an operation. Decision 00003 has the list.
- `x-pattern-message` gives the `422` message of a `pattern`. Other messages come from the keyword of the error, such as `must have 1 to 255 characters`.
- the validation options: `MultiError` gives all errors in one response. `SkipSettingDefaults` keeps the request of the client, because defaults would change the cursor fingerprint.
- the `email` format accepts only a bare address. `mail.ParseAddress` would also accept `Ana <ana@example.com>`. Stored emails skip this rule in responses.
- `IncludeResponseStatus` refuses a response status that the operation does not define
- the log gets only the place and the rule of each error, such as `/user/email format`. The errors of kin-openapi show the values, such as tokens.
- an array with more than 20 items gets only its `maxItems` error. The size of the response does not grow with the request.
- the buffered writer keeps the headers that PocketBase set before, such as CORS headers. A panic happens before `Flush`, so the client gets only the `500`.
- each property has at most one error. A duplicate tag gives a `tags` error, not a `tags[2]` error.
- `trimWhitespace` of `body.go` removes only `" \t\n\r"`. `strings.TrimSpace` would also remove a no-break space, and a name of only this character would become empty.
- `forbidden` and `conflict` of `problem.go` need a `code`. The `expect` helper of the tests fails for a `403` or `409` of `/v1` without a `code`.

Some rules stay in Go, because the spec cannot describe them:
- a repeated JSON property, because the decoders keep only one copy
- a query parameter that the spec does not define, because OpenAPI does not refuse it
- `5.0` for an integer, because JSON Schema allows it and the strict handler cannot decode it
- a sort with a repeated property, a search without a letter or a digit, a search with more than 10 words, a `relevance` sort without a search, and the cursor fingerprint
- a password of more than 72 bytes. PocketBase refuses it with `422`.

Switches:
- `REVIEWS_VALIDATE_RESPONSES=false` turns off the response validation. The responses then go to the client without the buffer. Use it if a spec mistake gives `500` for valid responses.
- `REVIEWS_VALIDATE_REQUESTS=false` turns off the validation of the parameters and the body values. The checks that the strict handler needs stay: the query names, the content type, the size, the JSON object, the unknown properties, and the integer literals.
- without the request validation, invalid values can be saved. A response with such a value then gets `500`. Turn off the request validation only with the response validation. The server logs a warning at start.
- the handlers keep the guards that stop a panic without the request validation. `limit` stays from 1 to 100. An `overall` filter of `NaN` gets `400`.
- an unset variable keeps the default, which is on. The value is a Go boolean: `true`, `false`, `1`, `0`, `t`, `f`, and their upper-case forms. Another value, such as `off` or `yes`, stops the server at start.

## Timestamps

- `apiTimestamp` of `app/api/internal/publicapi` changes the PocketBase format, `2024-11-04 01:26:55.002Z`, to RFC 3339. `storedTimestamp` changes a filter value back. Decision 00002 has the PocketBase mapping.
- the two formats sort in the same order as text. The indexes and the cursors use the stored values.
- `apiTimestamp` gives an empty string for a record without the value. The response validation then gives `500`.

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| OpenAPI Generator `go-server` for the app | Not selected. Its handlers return an untyped status and body, and it does not support `oneOf`, `null`, or `deepObject`. It also adds a second router. |
| Hand-written types and clients | Not selected. They drift from the spec. |
| The request validator middleware of oapi-codegen | Not selected. It does not validate responses, and it finds the operation with a second router. |
| A router of kin-openapi, such as `gorillamux` | Not selected. The ServeMux already found the operation. A second router could find a different one. |
| Validation only in tests | Not selected. A response that breaks the spec in production would reach the client. |
| A copy of the rules in Go | Removed. The two copies drifted, and a rule change needed two changes. |
| The PocketBase timestamp format with a space | Removed in version 2.0.0. It is not RFC 3339, and each client needed its own parser. |
| `format: date-time` for `Timestamp` | Not selected. The generators would change the type to a date object in each SDK. |
| Review kinds with a `kind` discriminator, such as `basic` with one rating | Removed. One kind is simpler for clients, filters, and sorts. |
| A new path, such as `/v2`, for each breaking change | Not selected. The UI is the only known client, and a new path changes each route. |
| Aliases for renamed operations and properties | Not selected. They double the operations for the short time until a reload. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
