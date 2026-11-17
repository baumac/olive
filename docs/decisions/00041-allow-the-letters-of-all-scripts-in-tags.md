---
description: Tags can have the lowercase letters and digits of all scripts, such as `café` or `ラーメン`. The API changes each tag to Unicode normalization form C.
status: implemented
---

# Motivation
[motivation]: #motivation

The `Tag` pattern of decision 00001 was `^[a-z0-9]+(-[a-z0-9]+)*$`.
A user could not add `café`, `crème-brûlée`, `ラーメン`, or `ψάρι`.
The UI suggested `caf` for `café`.
The names and texts of restaurants already allow all scripts. Only tags did not.

# Scope
[scope]: #scope

In scope:
- the `Tag` schema of `specs/api/public-openapi.yaml`
- the check of the server and the check of the UI
- the tag filters `tags[has]`, `tags[has_any]`, and `tags[has_all]`. They use the same schema.

Out of scope:
- tags with uppercase letters. `Café` and `café` stay one tag. The API refuses `Café`.
- tags that look the same in two scripts, such as a Latin `a` and a Cyrillic `а`. Only approved users add tags. See decision 00034.
- tags with an emoji, punctuation, or a space
- the zero-width joiner and the zero-width non-joiner. Some scripts, such as Persian, use them. The API refuses them, as before.
- a search that splits the words of scripts without spaces, such as Japanese. The FTS5 tokenizer keeps `ラーメン屋` as one word.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

A tag has words with single hyphens between them.
Each word starts with a lowercase letter, a letter without case, or a digit of any script.
Combining marks, such as the vowel signs of Devanagari, can follow.

The API changes each tag to Unicode normalization form C before the other checks.
`cafe` with a separate accent U+0301 becomes `café` with one character.
Some sources, such as macOS file names, give the separate form. A client then needs no own conversion.

| Tag | Result |
|---|---|
| `café`, `2-für-1`, `ψάρι`, `ラーメン`, `चाय` | valid |
| `cafe` + U+0301 | valid. The API saves `café`. |
| `café` and `cafe` + U+0301 in one request | `422`. The two values are the same tag. |
| `Café`, `Ψάρι` | `422`. Uppercase letters are not allowed. |
| `pasta` + U+FE0F, `pasta` + U+17B4, U+3164 | `422`. A default ignorable code point is invisible. |
| U+0301 + `a` | `422`. A word cannot start with a mark. |
| `dim` + U+00A0 + `sum` | `422`. A no-break space is not a letter. |

The tag filters change their values in the same way. A filter with an invalid tag gets `400`.

The search finds `café` with `cafe`. The FTS5 tokenizer of decision 00004 removes the diacritics.
The tag filters compare whole tags. `tags[has]=cafe` does not find `café`.

The UI also changes each typed tag and each tag of a link to Unicode normalization form C. Its check for repeated tags then works.
For an invalid tag, the UI suggests a valid form, such as `crème-brûlée` for `Crème Brûlée`, or `istanbul` for `İstanbul`.

Version 5.0.0 of the spec has this change.
A client that checks the responses against the old pattern refuses the new tags. The first number therefore changes.

# Implementation Details
[implementation-details]: #implementation-details

- the pattern is `^[\p{Ll}\p{Lm}\p{Lo}\p{Nd}][\p{Ll}\p{Lm}\p{Lo}\p{Mn}\p{Mc}\p{Nd}]*(-[\p{Ll}\p{Lm}\p{Lo}\p{Nd}][\p{Ll}\p{Lm}\p{Lo}\p{Mn}\p{Mc}\p{Nd}]*)*$`.
  Go `regexp` and JavaScript with the `u` flag both read it.
- `\p{Lu}` and `\p{Lt}` are not in the classes, so the pattern refuses uppercase letters.
- `\p{Lm}` is in the classes for the `ー` of `ラーメン`.
- a word cannot start with a mark. A mark after a hyphen has no letter to change.
- the `Tag` schema has the `x-nfc: true` extension. It works like `x-trim` of decision 00040, but before the validation.
  - `readBody` changes each string of a schema with `x-nfc` in the body. `changeValue` also goes into arrays, so it reaches each item of `Tags`.
  - `normalizeQuery` changes each value of a query parameter with `x-nfc` in its schema or in its items. The validation and the handler then read the changed query.
  - the validation comes after the change. `uniqueItems` and `maxLength` then count the changed tags. A few characters, such as U+0958, become two code points in this form.
- the `Tag` schema has `format: tag`. `checkTagFormat` of `openapi.go` refuses the code points with the Unicode property `Default_Ignorable_Code_Point`.
  Go has no table of this property. `checkTagFormat` uses `Other_Default_Ignorable_Code_Point` and `Variation_Selector`. The other parts of the property are in `Cf`, which the pattern refuses.
  The UI uses `\p{Default_Ignorable_Code_Point}`. Both sides use the Unicode property, so no list of characters is kept by hand.
- the pattern cannot refuse these code points. They are in `\p{Mn}` and `\p{Lo}`, and Go `regexp` has no class subtraction.
- the response validation has no validator for `tag`. kin-openapi ignores a format without a validator. The pattern still applies.
- the error messages are:
  - pattern: `must have words of lowercase letters, digits, and marks with single hyphens between them, and each word must start with a letter or a digit`
  - format: `must not have invisible characters`
- `maxLength` counts code points. The UI counts code points too, not UTF-16 units.
- `suggestTag` removes the combining dot that `toLowerCase` adds to `i` for `İ`.
- the stored tags are ASCII. No migration is necessary.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
