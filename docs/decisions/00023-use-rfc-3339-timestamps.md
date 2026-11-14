---
description: The Public API sends and accepts RFC 3339 timestamps. The space of the PocketBase format becomes a T.
status: implemented
---

# Motivation
[motivation]: #motivation

The API sent timestamps in the PocketBase format, such as `2024-11-04 01:26:55.002Z`.
This is not RFC 3339, so the date parsers of most platforms refuse it.
Each client needed its own function that changes the space to a `T`. The UI had `parseTimestamp` for this.
A change is cheap now. The UI is the only client. An iOS app is in the backlog.

Expected outcome:
- `new Date(value)` in JavaScript and the ISO 8601 parsers of Swift and Go read each timestamp of the API

# Scope
[scope]: #scope

In scope:
- the `Timestamp` schema of `specs/api/public-openapi.yaml`, its examples, and the generated SDKs
- the timestamps in the responses, and the values of the `created` and `updated` filters
- `parseTimestamp` of the UI

Out of scope:
- the format in the database. PocketBase keeps its format.
- `format: date-time` in the schema. The generators would then change the type from a string to a date object in each SDK.
- support for the old format in the filters. A value with a space gets `400`.
- time zones other than UTC and precisions other than milliseconds

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

A timestamp of the API is `2024-11-04T01:26:55.002Z`. It is always UTC with three decimals.

```
GET /v1/restaurants?created[gte]=2024-11-01T00:00:00.000Z
```

This is a breaking change of the API. No client other than the UI exists, and the UI ships in the same binary.

# Implementation Details
[implementation-details]: #implementation-details

- the pattern of `Timestamp` is `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`
- `apiTimestamp` in `app/internal/publicapi` formats the `created`, `updated`, and `expires` values
- `storedTimestamp` changes a filter value to the format of the database. The request validation of decision 00007 checks the pattern first.
- the two formats sort in the same order as text. The indexes and the cursors do not change.
- a cursor has the values of the database, not of the API. Cursors from before the change still work.
- `parseTimestamp` of the UI is `new Date(value)`

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
