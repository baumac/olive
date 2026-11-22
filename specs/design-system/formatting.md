# Dates, numbers and language

Rules for how the UI shows values from the API. Format every value with the browser's `Intl` APIs. Numbers and relative times use the `en` locale. Dates use `en-GB`, which puts the day first as in "4 Nov 2024". `en` gives "Nov 4, 2024". Never build date or number text by hand.

## Dates and times

The API sends UTC timestamps in the RFC 3339 format, `2024-11-04T01:26:55.002Z`. `new Date()` reads the value. See decision 00001.

- Within the last 7 days, show relative time: "just now" (under 1 minute), "5 minutes ago", "3 hours ago", "yesterday", "4 days ago". Use `Intl.RelativeTimeFormat` with `numeric: "auto"`.
- Older than 7 days, show the day, the short month and the year: "4 Nov 2024". Use `Intl.DateTimeFormat` with `day: "numeric", month: "short", year: "numeric"`.
- Show the full date and time in the person's time zone on hover and to screen readers: wrap the text in `<time datetime="2024-11-04T01:26:55.002Z" title="4 November 2024, 02:26">`.
- When `updated` is later than `created`, add "edited" after the date: "4 Nov 2024 · edited". Do not show both dates.
- Metadata lines join parts with a middle dot and spaces: "Ana · 4 Nov 2024".

## Numbers and scores

- Review scores are integers. Show them with the maximum: "8 / 10". Use `tabular-nums`.
- Restaurant averages come from the API with one decimal: "7.6 / 10". Show a whole average without a decimal: "9 / 10".
- The `overall` of a recipe review has up to two decimals, such as 8.25. Show it with one decimal, like the averages: "8.3 / 10". See decision 00004.
- The total time of a recipe is in minutes. Show it with `Intl.NumberFormat` units and `unitDisplay: "short"`: "45 min", "1 hr", "1 hr 30 min". `formatMinutes` of `lib/format.ts` does this.
- Do not average or round scores in the UI. Show the values of the API.
- Counts use `Intl.NumberFormat` and the right plural: "1 review", "12 reviews", "1,204 reviews". Use `Intl.PluralRules` to pick the word.
- Zero is a word in empty states ("No reviews yet") and a number in counts ("0 reviews").

## Text from the API

- Show `name`, `title`, `description` and `feedback` as typed. Keep line breaks in `description` and `feedback` with `white-space: pre-line`. Do not render Markdown or HTML.
- Long text in a card stops after 3 lines with an ellipsis (`line-clamp-3`). The detail screen shows all of it.
- Tags show as stored: lowercase, with hyphens.
- A creator that is `null` shows as "Deleted user".

## Language

- The UI supports only English, with `<html lang="en">`. Other languages are out of scope (ADR 00009). Do not add a language switcher or translation files.
- Format numbers with the `en` locale and dates with `en-GB`, as in the examples above. The page language stays `en`.
