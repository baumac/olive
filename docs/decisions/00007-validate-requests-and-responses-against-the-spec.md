---
description: Validate the requests and the responses of the Public API against the spec at runtime with kin-openapi.
status: implemented
---

# Motivation
[motivation]: #motivation

The Public API had the rules of the spec two times:
- `specs/api/public-openapi.yaml` gives the types, lengths, patterns, and ranges
- `app/internal/publicapi/validate.go` and the body schemas in `body.go` gave the same rules in Go

The two copies could drift without a failed test.
A change to a rule needed two changes.
Nothing checked that the responses of the app match the spec.
The compiler checks the Go types of a response, but not the patterns, the formats, or the required properties.

Expected outcome:
- the spec is the only copy of each rule that the spec can describe
- a request that breaks a rule gets the same status and error format as before
- a response that breaks the spec does not reach the client

# Scope
[scope]: #scope

In scope:
- validation of the path parameters, the query parameters, and the request body
- validation of the status, the headers, and the body of each response
- the error messages of `422 Unprocessable Content`
- the removal of the Go copies of the rules

Out of scope:
- the rules that the spec cannot describe. They stay in Go:
  - repeated JSON properties, because the decoders keep only one copy
  - query parameters that the spec does not define, because OpenAPI does not refuse them
  - `5.0` for an integer, because JSON Schema allows it and the strict handler cannot decode it
  - a sort with a repeated property, a search without a letter or a digit, a search with more than 10 words, a `relevance` sort without a search, and the cursor fingerprint
  - a password of more than 72 bytes. PocketBase refuses it with `422`.
- validation in the UI. The UI keeps its own checks in `app/ui/src/lib/validation.ts`.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

The Go SDK embeds the spec. `reviewsapi.GetSwagger()` returns it as a kin-openapi document.
The `prepare` middleware of the Public API uses it for each request:

1. `prepare` finds the spec operation of the ServeMux pattern.
2. After the rate limit and the token check, it validates the parameters. An invalid parameter gets `400`, for example `The query parameter limit is not valid.`
3. It checks the content type, the size, and the JSON object of the body. Then it validates the body. An invalid property gets `422` with one error for each property.
4. The handler writes its response into a buffered writer.
5. `prepare` validates the buffered response. A valid response goes to the client with `Flush`. An invalid response gets a generic `500`, and the log has the validation error.

A rule change is now one change in the spec, followed by `make generate`.
For example, the hidden characters of decision 00001 are a `pattern` of the `Name`, `Title`, and `LongText` schemas.

The `x-pattern-message` extension gives the `422` message of a pattern:

```yaml
Tag:
  type: string
  pattern: '^[a-z0-9]+(-[a-z0-9]+)*$'
  x-pattern-message: must have lowercase letters, digits, and single hyphens between them
```

# Implementation Details
[implementation-details]: #implementation-details

- `app/sdk/go/oapi-codegen.yaml` has `embedded-spec: true`. The spec is outside the Go module of the app, so `go:embed` cannot read it.
- `app/internal/publicapi/openapi.go` loads the spec one time in `New`
- the spec also gives the token check and the rate limit of each operation. The `security` of the operation gives the token check. The `x-rate-limit` extension gives the rate limit. It was `x-rate-limited` before decision 00008. The app has no second table of operations.
- the ServeMux pattern gives the path parameters. The app then does not need a second router.
- the validation options:
  - `MultiError` gives all errors of a request in one response
  - `SkipSettingDefaults` keeps the request of the client. The defaults would change the cursor fingerprint.
  - `AuthenticationFunc` does nothing. `prepare` checks the bearer token before the validation.
  - the `email` format accepts only a bare address. `mail.ParseAddress` also accepts `Ana <ana@example.com>`.
- the response validation has its own options:
  - `IncludeResponseStatus` refuses a response status that the operation does not define
  - each stored email is valid. PocketBase checked it, also for a save in the dashboard. The bare address rule applies only to requests.
  - the log gets only the place and the rule of each error, such as `/user/email format`. The errors of kin-openapi show the values, such as tokens and emails. `DisableSchemaErrorDetails` does not work for OpenAPI 3.1 documents.
- `422` messages come from the schema keyword of each error, such as `must have 1 to 255 characters` for `maxLength`. A `pattern` error uses `x-pattern-message`.
- an array with more than 20 items gets only its `maxItems` error. The size of the response then does not grow with the request.
- the buffered writer keeps the status, the headers, and the body. Only `Flush` writes them to the underlying writer. The headers that PocketBase set before, such as CORS headers, stay.
- a panic in a handler happens before `Flush`. The client then gets only the `500` response, also after the handler wrote a status.
- the handlers no longer check IDs, tags, timestamps, scores, averages, or limits. The spec validation runs before each handler.

Switches:
- `REVIEWS_VALIDATE_RESPONSES=false` turns off the response validation. The responses then go to the client without the buffer. Use it if the validation slows the server, or if a spec mistake gives `500` for valid responses.
- `REVIEWS_VALIDATE_REQUESTS=false` turns off the validation of the parameters and the body values. The checks that the strict handler needs stay: the query names, the content type, the size, the JSON object, the unknown properties, and the integer literals.
- without the request validation, invalid values reach the handlers and can be saved. A response with a saved invalid value then fails the response validation and gets `500`. The server logs a warning at start.
- the handlers keep the guards that prevent a panic or an error without the request validation. `limit` stays from 1 to 100. An `overall` filter of `NaN` gets `400`.
- the default turns on both. An unset variable keeps the default. A value that is not `true` or `false` stops the server at start.

Changes for clients:
- the parameter checks run before the body checks. A request with an invalid parameter and an invalid body gets `400`.
- a `400` detail for a parameter names it, such as `The path parameter restaurantId is not valid.`
- a duplicate tag gives a `tags` error, not a `tags[2]` error
- each property has at most one error

Alternatives:

| Alternative | Result |
| --- | --- |
| The request validator middleware of oapi-codegen | Not selected. It also uses kin-openapi, but it does not validate responses. It also finds the operation with a second router. |
| Validation only in tests | Not selected. A response that breaks the spec in production would still reach the client. |
| A router from kin-openapi, such as `gorillamux` | Not selected. The ServeMux already found the operation. A second router could find a different one. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
