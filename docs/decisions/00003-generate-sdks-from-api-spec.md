---
description: Generate a TypeScript SDK and a Go SDK from the Public API spec.
status: implemented
---

# Motivation
[motivation]: #motivation

The Public API spec in `specs/api/public-openapi.yaml` is the contract between the `ui` app and the PocketBase app in `app`.
If each app writes its own types and HTTP code by hand, the apps can drift from the spec without notice.
A change to the spec then breaks the other app only at runtime.

We generate code from the spec for both apps.
A change to the spec then changes the generated code.
The compiler of each app finds the code that does not match the spec.

Expected outcome:
- The `ui` app calls the Public API only through a generated TypeScript client.
- The PocketBase app in `app` implements a generated Go server interface and uses generated Go types.
- One `make` command generates both SDKs from the spec.
- A check finds generated code that does not match the spec.

# Scope
[scope]: #scope

In scope:
- A TypeScript SDK in `sdk/typescript` with the client that the `ui` app uses. [OpenAPI Generator](https://github.com/OpenAPITools/openapi-generator) generates it.
- A Go SDK in `sdk/go` with the server interface and the Go types. [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) generates it.
- The generator settings, the `make` targets, and the check for out-of-date code
- Tests that show the generated code handles the difficult parts of the spec

Out of scope:
- Publishing the SDKs to npm or to other package registries. Only this repository uses them.
- A Go client. The PocketBase app in `app` is the only Go user, and it needs only the server code.
- API documentation from the spec. This stays in the backlog.
- Implementing the `app` and `ui` apps. Each SDK gives only the code that the spec defines.
- Request validation middleware that reads the spec at runtime

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

New concepts:
- **SDK**: a package of code generated from `specs/api/public-openapi.yaml`. The project has two SDKs.
- **TypeScript SDK**: the npm package `@reviews/sdk` in `sdk/typescript`. It has one API class per tag and one model per schema.
- **Go SDK**: the Go module `github.com/baumac/reviews-app/sdk/go`. Its package `reviewsapi` has the Go types and the strict server interface.
- **Strict server interface**: a Go interface with one method per operation. Each method gets a typed request and returns a typed response.

The spec is the only source.
Nobody edits the generated files.
To change the API, change the spec and run `make generate`.

```bash
make generate         # Generate the TypeScript SDK and the Go SDK
make check-generate   # Fail if the generated code does not match the spec
make test-sdk         # Run the tests of both SDKs
```

## TypeScript SDK

The `ui` app depends on `@reviews/sdk` through an npm workspace.
The SDK has a `UsersApi` class and a `RestaurantsApi` class, one for each tag.
Each method has the name of the `operationId` and takes one request object.

```ts
import { Configuration, RestaurantsApi, UsersApi } from '@reviews/sdk'

const config = new Configuration({
  basePath: '/v1',
  accessToken: async () => session.token ?? '',
})

const users = new UsersApi(config)
const auth = await users.signIn({
  credentials: { email: 'ana@example.com', password: 'correct-horse-battery' },
})

const restaurants = new RestaurantsApi(config)
const page = await restaurants.listReviews({
  restaurantId: 'k3x9m2p7q1w8z4a',
  rating: { gte: 4 },
  sort: '-rating,title',
})
```

A response with a status that is not `2xx` throws a `ResponseError`.
The UI reads the Problem Details body from `error.response`.
Each method also has a `Raw` form, such as `createRestaurantRaw`, that gives the response headers.

## Go SDK

The PocketBase app in `app` implements `reviewsapi.StrictServerInterface` and mounts the generated handler on the PocketBase router.

```go
type Server struct {
	app core.App
}

func (s *Server) GetRestaurant(
	ctx context.Context,
	req reviewsapi.GetRestaurantRequestObject,
) (reviewsapi.GetRestaurantResponseObject, error) {
	r, err := s.app.FindRecordById("restaurants", req.RestaurantId)
	if err != nil {
		return reviewsapi.GetRestaurant404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: reviewsapi.NotFoundApplicationProblemPlusJSONResponse{
				Type: "about:blank", Title: "Not Found", Status: 404,
			},
		}, nil
	}
	return reviewsapi.GetRestaurant200JSONResponse(toRestaurant(r)), nil
}
```

The compiler fails when the PocketBase app in `app` does not implement an operation of the spec.
It also fails when a handler returns a status or a body that the spec does not define for the operation.

# Implementation Details
[implementation-details]: #implementation-details

## Why these generators

| Option | Result |
| --- | --- |
| OpenAPI Generator `typescript-fetch` for the UI | Selected. It uses the browser `fetch` and has no runtime dependencies. It makes a TypeScript interface and `FromJSON` and `ToJSON` functions for each model. |
| oapi-codegen for the API | Selected. It makes idiomatic Go types and a strict server for `net/http`. It supports OpenAPI 3.1. |
| OpenAPI Generator `go-server` for the API | Not selected. Handlers return an untyped status and body, so the compiler does not check responses against the spec. It does not support `oneOf`, `allOf`, `not`, `null`, or `deepObject`, which the review kinds, merge patches, and filters need. It also adds a second router to the PocketBase app. |
| Hand-written types and clients | Not selected. They drift from the spec. |

## Layout

```
go.work                       # hand-written
package.json                  # hand-written npm workspace root
sdk/
  typescript/
    package.json              # hand-written
    tsconfig.json             # hand-written
    tsconfig.test.json        # hand-written
    templates/                # hand-written template overrides
    test/                     # hand-written tests
    src/                      # generated
  go/
    go.mod                    # hand-written
    oapi-codegen.yaml         # hand-written generator settings
    reviewsapi_test.go        # hand-written tests
    reviewsapi.gen.go         # generated
```

- generated files are committed. The `app` and `ui` apps build without the generators.
- `sdk/typescript/src` has only generated files. `make generate` deletes it before it runs the generator. The generator does not delete the files of removed schemas.
- the root `go.work` file has `sdk/go` and `app`. The `app` module also has a `replace` line for `sdk/go`. `go mod tidy` does not use `go.work`, and the SDK module is not published.
- the root `package.json` is an npm workspace with `sdk/typescript`. The `ui` app adds itself when it exists.

## TypeScript SDK settings

- generator: `typescript-fetch`
- the makefile pins the version of the `openapitools/openapi-generator-cli` Docker image. `make generate` runs it, so developers do not need Java.
- the makefile gives these options:
  - `supportsES6`: `true`
  - `importFileExtension`: `.js`. The package is an ES module.
  - `useSingleRequestParameter`: `true`
  - `withInterfaces`: `true`. Tests in the `ui` app can replace an API class with a fake.
  - `legacyDiscriminatorBehavior`: `false`. The `Review`, `ReviewCreate`, and `ReviewPatch` unions then select the model from `kind`.
  - `apiDocs` and `modelDocs`: `false`. The spec is the documentation.
- `stringEnums` stays `false`. With `true`, the enum type of `kind` does not match the literal types of the union, and `tsc` fails.
- `tsc` builds the package into `sdk/typescript/dist`. The `ui` app imports the built package.
- `package.json` is hand-written. The package name is `@reviews/sdk`.

## Go SDK settings

- generator: oapi-codegen v2
- the version is pinned in the makefile. `make install-tools` installs it into `./bin`, like vacuum.
- `oapi-codegen.yaml`:

```yaml
package: reviewsapi
output: reviewsapi.gen.go
generate:
  models: true
  std-http-server: true
  strict-server: true
output-options:
  nullable-type: true
```

- the server uses `net/http` because the PocketBase router uses `net/http` handlers
- the PocketBase app in `app` mounts the generated handler under `/v1` with `StdHTTPServerOptions.BaseURL`. PocketBase keeps its own routes under `/api` and `/_`.
- `nullable-type: true` makes `nullable.Nullable[T]` for nullable properties. A merge patch body then shows the difference between a missing property and `null`.
- the strict server calls hand-written error handlers for request decode errors and handler errors. These handlers return Problem Details, so all errors keep one format.

## Difficult parts of the spec

The generators do not support all parts of OpenAPI 3.1 in the same way.
Each part below has a test in the SDK that it affects.
`make test-sdk` runs the tests of both SDKs.
If a generator does not support a part, we change the settings or add a small hand-written fix.
We do not change the meaning of the spec to work around a generator without a new decision.

| Part of the spec | Result | Test |
| --- | --- | --- |
| `Review` is a `oneOf` with a `discriminator` on `kind` | TypeScript selects `BasicReview` or `JocDeCartesReview` from `kind`. Go makes a union type with `AsBasicReview`, `AsJocDeCartesReview`, and `ValueByDiscriminator` methods. | both SDKs |
| `OtherReview` uses `not` for kinds that a later release adds | TypeScript returns the JSON object without changes. Go `ValueByDiscriminator` returns an error, but `AsOtherReview` gets the common properties. | both SDKs |
| `const` values on `kind` | TypeScript and Go make an enum type with one value | both SDKs |
| Merge patch bodies with `application/merge-patch+json` | The client sends this content type. A `null` value stays in the body. A missing property is not in the body. | TypeScript |
| Merge patch bodies on the server | The handler sees the difference between a missing property and `null` | Go |
| `deepObject` filters, such as `rating` | The client sends `rating[gte]=4`. It encodes the brackets as `%5B` and `%5D`, which servers decode. The server reads the filter into a struct. | both SDKs |
| Array filters with `explode: true`, such as `tags[has_any]` | The client repeats the parameter for each value | both SDKs |
| `Timestamp` is a string with a `pattern`, not `date-time` | Both SDKs keep the value as a string. They do not parse it as a date. | both SDKs |
| `next_cursor` is `null` on the last page | Go uses `nullable.Nullable` and sends `null`. TypeScript gives `null`. | both SDKs |
| Responses with `application/problem+json` | Go makes a response type for each status. TypeScript throws `ResponseError` with the body. | both SDKs |
| `Location` and `Retry-After` response headers | Go response types have header fields. TypeScript gives the headers through the `Raw` methods. | both SDKs |
| Operations with `security: - {}` | The TypeScript client does not send the stored token on sign up and sign in | TypeScript |

### Workarounds

- `typescript-fetch` changes an optional `null` array with `uniqueItems` to `undefined`. A merge patch then cannot clear `tags`.
  `sdk/typescript/templates/modelGeneric.mustache` is a copy of the generator template with a fix in the `ToJSON` part. It keeps `null`.
  Update this copy when the generator version changes.
- `typescript-fetch` makes a `Set<string>` for `tags` because of `uniqueItems`. The UI changes it to an array where it needs one.
- both generators add the schema name `OtherReview` as a discriminator value, because the mapping has no entry for it.
  The TypeScript type of `kind` therefore lists `OtherReview`, not `string`.
  The UI handles a `kind` that it does not know in the `default` branch of a `switch`.
- the spec has these changes that do not change the API:
  - each `kind` with `const` also has `type: string`. Without it, `typescript-fetch` cannot read the discriminator.
  - the `deepObject` filters use the named schemas `RatingRange`, `ScoreRange`, and `TimestampRange`. Without them, the generated names come from the first operation that uses each filter.

## Out-of-date check

- `make check-generate` runs `make generate`. Then it fails if `git diff` shows changes in `sdk` or if `sdk` has files that git does not track.
- `make lint` runs `make check-generate` after `lint-spec`. The lint therefore needs Docker.
- a change to the spec without new generated code then fails the lint

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- OpenAPI Generator marks its OpenAPI 3.1 support as beta. If `typescript-fetch` cannot handle the spec, we compare other TypeScript generators in a new decision.
